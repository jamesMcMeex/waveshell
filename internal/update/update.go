package update

import (
	"fmt"
	"log/slog"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jamesMcMeex/waveshell/internal/config"
	"github.com/jamesMcMeex/waveshell/internal/db"
	"github.com/jamesMcMeex/waveshell/internal/messages"
	"github.com/jamesMcMeex/waveshell/internal/model"
	"github.com/jamesMcMeex/waveshell/internal/mpv"
	"github.com/jamesMcMeex/waveshell/internal/scanner"
)

func InitialModel() Model {
	return Model{
		UI: UIState{
			BrowseMode: model.BrowseModeArtist,
			ActivePane: model.PaneLeft,
		},
		Player: PlayerState{
			Volume: 80,
		},
	}
}

func (m Model) Init() tea.Cmd {
	slog.Info("TUI init",
		"config_set", m.Config != nil,
		"db_set", m.DB != nil,
		"browse_mode", m.UI.BrowseMode,
	)

	var cmds []tea.Cmd

	if m.Player.MPVReady && m.Player.Events != nil {
		cmds = append(cmds, mpv.SubscribeCmd(m.Player.Events))
	}

	if m.Config != nil && m.Config.Library.ScanOnStartup && len(m.Config.Library.Paths) > 0 {
		slog.Info("init: queuing scan", "paths", m.Config.Library.Paths)
		cmds = append(cmds, func() tea.Msg {
			return messages.ScanStartedMsg{}
		})
	} else {
		scanOnStartup := m.Config != nil && m.Config.Library.ScanOnStartup
		pathsLen := 0
		if m.Config != nil {
			pathsLen = len(m.Config.Library.Paths)
		}
		slog.Info("init: scan not queued",
			"config_nil", m.Config == nil,
			"scan_on_startup", scanOnStartup,
			"paths_len", pathsLen,
		)
	}

	if q := queryLeftPaneCmd(m); q != nil {
		slog.Info("init: queuing left pane query")
		cmds = append(cmds, q)
	} else {
		slog.Info("init: no left pane query (nil db or unsupported mode)")
	}

	return tea.Batch(cmds...)
}

func queryLeftPaneCmd(m Model) tea.Cmd {
	if m.DB == nil {
		slog.Warn("queryLeftPaneCmd: db is nil")
		return nil
	}
	mode := m.UI.BrowseMode
	if mode == "" {
		mode = model.BrowseModeArtist
	}
	switch mode {
	case model.BrowseModeArtist:
		slog.Info("queryLeftPaneCmd: querying artists")
		return db.QueryArtistsCmd(m.DB)
	case model.BrowseModeLabel, model.BrowseModeGenre, model.BrowseModeYear:
		slog.Info("queryLeftPaneCmd: querying tag slice", "mode", mode)
		return db.QueryTagSliceCmd(m.DB, mode)
	default:
		slog.Warn("queryLeftPaneCmd: unsupported mode", "mode", mode)
		return nil
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// ── Scan messages (existing) ──────────────────────────────────────────

	case messages.ScanStartedMsg:
		if m.Library.Scanning {
			return m, nil
		}
		slog.Info("scan: starting", "paths", m.Config.Library.Paths)
		m.Library.Scanning = true
		m.Library.ScanComplete = false
		m.Library.ScanProcessed = 0
		m.Library.ScanSkipped = 0
		m.Library.ScanTotal = -1
		return m, scanner.StartScanCmd(m.Config.Library.Paths, m.DB)

	case messages.ScanProgressMsg:
		m.Library.Scanning = true
		m.Library.ScanProcessed = msg.Processed
		m.Library.ScanTotal = msg.Total
		m.Library.ScanCurrent = msg.CurrentPath
		return m, msg.NextCmd

	case messages.ScanFileErrorMsg:
		slog.Warn("scan: file error", "path", msg.Path, "error", msg.Err)
		m.Library.ScanSkipped++
		return m, msg.NextCmd

	case messages.ScanCompleteMsg:
		slog.Info("scan: complete", "processed", msg.Processed, "skipped", msg.Skipped)
		m.Library.Scanning = false
		m.Library.ScanComplete = true
		m.Library.ScanProcessed = msg.Processed
		m.Library.ScanSkipped = msg.Skipped
		var cmds []tea.Cmd
		if q := queryLeftPaneCmd(m); q != nil {
			cmds = append(cmds, q)
		}
		return m, tea.Batch(cmds...)

	// ── Window resize ─────────────────────────────────────────────────────

	case tea.WindowSizeMsg:
		m.UI.Width = msg.Width
		m.UI.Height = msg.Height
		return m, nil

	// ── DB query results ──────────────────────────────────────────────────

	case messages.ArtistListResultMsg:
		slog.Info("query: artist result", "count", len(msg.Artists))
		m.Library.Artists = msg.Artists
		m.Library.Albums = nil
		m.Library.Tracks = nil
		m.UI.LeftCursor = 0
		m.UI.LeftOffset = 0
		m.UI.MiddleCursor = 0
		m.UI.RightCursor = 0
		return m, nil

	case messages.TagSliceResultMsg:
		slog.Info("query: tag slice result", "count", len(msg.Values), "mode", msg.Mode)
		m.Library.TagSliceValues = msg.Values
		m.Library.Albums = nil
		m.Library.Tracks = nil
		m.UI.LeftCursor = 0
		m.UI.LeftOffset = 0
		m.UI.MiddleCursor = 0
		m.UI.RightCursor = 0
		return m, nil

	case messages.AlbumListResultMsg:
		slog.Info("query: album result", "count", len(msg.Albums), "key", msg.Key)
		m.Library.Albums = msg.Albums
		m.Library.Tracks = nil
		m.UI.MiddleCursor = 0
		m.UI.MiddleOffset = 0
		m.UI.RightCursor = 0
		if len(msg.Albums) > 0 {
			m.UI.ActivePane = model.PaneMiddle
		}
		return m, nil

	case messages.TrackListResultMsg:
		slog.Info("query: track result", "count", len(msg.Tracks))
		m.Library.Tracks = msg.Tracks
		m.UI.RightCursor = 0
		m.UI.RightOffset = 0
		if len(msg.Tracks) > 0 {
			m.UI.ActivePane = model.PaneRight
		}
		return m, nil

	case messages.DBErrorMsg:
		slog.Error("db error", "op", msg.Op, "error", msg.Err, "fatal", msg.Fatal)
		if msg.Fatal {
			return m, tea.Quit
		}
		return m, nil

	// ── mpv lifecycle ─────────────────────────────────────────────────────

	case messages.MPVReadyMsg:
		m.Player.Events = msg.Events
		m.Player.MPVReady = true
		return m, mpv.SubscribeCmd(msg.Events)

	case messages.MPVNotFoundMsg:
		m.MPVErr = fmt.Errorf("mpv not found on PATH — install mpv to enable playback")
		slog.Warn("mpv not found, playback disabled")
		return m, nil

	case messages.MPVConnectionLostMsg:
		m.Player.State = model.PlaybackStateStopped
		m.Player.CurrentTrack = nil
		m.Player.MPVReady = false
		m.Player.PositionSec = 0
		m.Player.DurationSec = 0
		slog.Warn("mpv connection lost", "error", msg.Err)
		return m, nil

	// ── mpv events (subscription) ─────────────────────────────────────────

	case messages.PlaybackStateChangedMsg:
		m.Player.State = msg.State
		if msg.State == model.PlaybackStatePlaying {
			m.Player.DisplayPositionSec = m.Player.PositionSec
			return m, tea.Batch(
				messages.TickCmd(),
				mpv.SubscribeCmd(m.Player.Events),
			)
		}
		return m, mpv.SubscribeCmd(m.Player.Events)

	case messages.TimePositionChangedMsg:
		m.Player.PositionSec = msg.PositionSec
		m.Player.DisplayPositionSec = msg.PositionSec
		return m, mpv.SubscribeCmd(m.Player.Events)

	case messages.DurationChangedMsg:
		m.Player.DurationSec = msg.DurationSec
		return m, mpv.SubscribeCmd(m.Player.Events)

	case messages.VolumeChangedMsg:
		m.Player.Volume = msg.Volume
		return m, mpv.SubscribeCmd(m.Player.Events)

	case messages.TrackEndedMsg:
		slog.Info("track ended", "reason", msg.Reason)
		m.Player.State = model.PlaybackStateStopped
		m.Player.CurrentTrack = nil
		m.Player.PositionSec = 0
		m.Player.DisplayPositionSec = 0
		return m, mpv.SubscribeCmd(m.Player.Events)

	// ── Playback tick ─────────────────────────────────────────────────────

	case messages.TickMsg:
		if m.Player.State == model.PlaybackStatePlaying {
			return m, messages.TickCmd()
		}
		return m, nil

	// ── Key events ────────────────────────────────────────────────────────

	case tea.KeyMsg:
		return handleKeyMsg(m, msg)
	}

	return m, nil
}

func handleKeyMsg(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// TODO: wire keybindings from cfg.Keybindings instead of hardcoding.
	// Each branch below should call cfg.Keybindings.CursorDown.Matches(key), etc.
	// See internal/config/config.go KeybindingsConfig.
	key := msg.String()

	// Help overlay routing
	if m.UI.ActiveOverlay == OverlayHelp {
		switch key {
		case "esc", "h", "q":
			m.UI.ActiveOverlay = OverlayNone
			m.Help.Active = false
			return m, nil
		default:
			updateHelpScroll(&m, msg)
			return m, nil
		}
	}

	// Browse mode picker routing
	if m.UI.ShowBrowsePicker {
		return handleBrowsePickerKey(m, key)
	}

	// Global keys (base layer)
	switch key {
	case "h":
		m.UI.ActiveOverlay = OverlayHelp
		initHelpOverlay(&m)
		return m, nil

	case "q":
		return m, tea.Quit

	case "esc":
		return m, nil

	case "c":
		if m.Config != nil {
			m.Config.UI.Theme = nextTheme(m.Config.UI.Theme)
			slog.Info("theme switched", "theme", m.Config.UI.Theme)
			if m.ConfigPath != "" {
				if err := config.WriteConfig(m.ConfigPath, *m.Config); err != nil {
					slog.Warn("failed to persist theme", "error", err)
				}
			}
		}
		return m, nil

	case "b":
		m.UI.ShowBrowsePicker = true
		m.UI.BrowsePickerCursor = browseModeIndex(m.UI.BrowseMode)
		return m, nil

	case "tab":
		m.UI.ActivePane = nextPane(m.UI.ActivePane)
		return m, nil

	case "shift+tab":
		m.UI.ActivePane = prevPane(m.UI.ActivePane)
		return m, nil

	case "j", "left":
		m.UI.ActivePane = prevPane(m.UI.ActivePane)
		return m, nil

	case "l", "right":
		m.UI.ActivePane = nextPane(m.UI.ActivePane)
		return m, nil
	}

	// Playback keys (always active)
	if m.Player.MPVReady {
		switch key {
		case "p":
			return handlePlayPause(m)
		case ",":
			return handlePrevTrack(m)
		case ".":
			return handleNextTrack(m)
		case "[":
			return handleSeek(m, -5)
		case "]":
			return handleSeek(m, 5)
		case "{":
			return handleSeek(m, -30)
		case "}":
			return handleSeek(m, 30)
		case "-":
			return handleVolume(m, -5)
		case "=":
			return handleVolume(m, 5)
		case "0":
			return handleVolume(m, -m.Player.Volume) // reset to 0
		}
	}

	// Pane-local keys
	switch m.UI.ActivePane {
	case model.PaneLeft:
		return handleLeftPaneKey(m, key)
	case model.PaneMiddle:
		return handleMiddlePaneKey(m, key)
	case model.PaneRight:
		return handleRightPaneKey(m, key)
	}

	return m, nil
}

func handleBrowsePickerKey(m Model, key string) (tea.Model, tea.Cmd) {
	modes := browseModes()
	switch key {
	case "esc":
		m.UI.ShowBrowsePicker = false
		return m, nil

	case "i", "up":
		if m.UI.BrowsePickerCursor > 0 {
			m.UI.BrowsePickerCursor--
		}
		return m, nil

	case "k", "down":
		if m.UI.BrowsePickerCursor < len(modes)-1 {
			m.UI.BrowsePickerCursor++
		}
		return m, nil

	case "enter":
		chosen := modes[m.UI.BrowsePickerCursor]
		if chosen == m.UI.BrowseMode {
			m.UI.ShowBrowsePicker = false
			return m, nil
		}
		m.UI.BrowseMode = chosen
		m.UI.ShowBrowsePicker = false
		m.Library.Artists = nil
		m.Library.Albums = nil
		m.Library.Tracks = nil
		m.Library.TagSliceValues = nil
		m.Library.SelectedArtistID = 0
		m.Library.SelectedAlbumID = 0
		m.Library.SelectedTagKey = ""
		m.UI.LeftCursor = 0
		m.UI.MiddleCursor = 0
		m.UI.RightCursor = 0
		m.UI.ActivePane = model.PaneLeft
		return m, queryLeftPaneCmd(m)
	}

	return m, nil
}

func handleLeftPaneKey(m Model, key string) (tea.Model, tea.Cmd) {
	items := leftPaneItems(m)
	maxIdx := len(items) - 1

	switch key {
	case "i", "up":
		if m.UI.LeftCursor > 0 {
			m.UI.LeftCursor--
			adjustOffset(&m.UI.LeftOffset, m.UI.LeftCursor, paneHeight(m))
		}
		return m, nil

	case "k", "down":
		if m.UI.LeftCursor < maxIdx {
			m.UI.LeftCursor++
			adjustOffset(&m.UI.LeftOffset, m.UI.LeftCursor, paneHeight(m))
		}
		return m, nil

	case "t", "home":
		m.UI.LeftCursor = 0
		m.UI.LeftOffset = 0
		return m, nil

	case "g", "end":
		if maxIdx >= 0 {
			m.UI.LeftCursor = maxIdx
			m.UI.LeftOffset = maxIdx - paneHeight(m) + 1
			if m.UI.LeftOffset < 0 {
				m.UI.LeftOffset = 0
			}
		}
		return m, nil

	case "ctrl+d":
		half := paneHeight(m) / 2
		m.UI.LeftCursor += half
		if m.UI.LeftCursor > maxIdx {
			m.UI.LeftCursor = maxIdx
		}
		adjustOffset(&m.UI.LeftOffset, m.UI.LeftCursor, paneHeight(m))
		return m, nil

	case "ctrl+u":
		half := paneHeight(m) / 2
		m.UI.LeftCursor -= half
		if m.UI.LeftCursor < 0 {
			m.UI.LeftCursor = 0
		}
		adjustOffset(&m.UI.LeftOffset, m.UI.LeftCursor, paneHeight(m))
		return m, nil

	case "enter":
		selID := selectedArtistID(m)
		if m.UI.BrowseMode == model.BrowseModeArtist && selID > 0 {
			m.Library.SelectedArtistID = selID
			return m, db.QueryAlbumsForArtistCmd(m.DB, selID)
		}
		tagKey := leftPaneSelectedKey(m)
		if tagKey != "" && m.UI.BrowseMode != model.BrowseModeArtist {
			m.Library.SelectedTagKey = tagKey
			return m, db.QueryAlbumsForTagCmd(m.DB, m.UI.BrowseMode, tagKey)
		}
		return m, nil
	}

	// Letter-jump
	if isLetter(key) {
		jumpToLetter(&m.UI.LeftCursor, &m.UI.LeftOffset, key, items, paneHeight(m))
		return m, nil
	}

	return m, nil
}

func handleMiddlePaneKey(m Model, key string) (tea.Model, tea.Cmd) {
	maxIdx := len(m.Library.Albums) - 1

	switch key {
	case "i", "up":
		if m.UI.MiddleCursor > 0 {
			m.UI.MiddleCursor--
			adjustOffset(&m.UI.MiddleOffset, m.UI.MiddleCursor, paneHeight(m))
		}
		return m, nil

	case "k", "down":
		if m.UI.MiddleCursor < maxIdx {
			m.UI.MiddleCursor++
			adjustOffset(&m.UI.MiddleOffset, m.UI.MiddleCursor, paneHeight(m))
		}
		return m, nil

	case "t", "home":
		m.UI.MiddleCursor = 0
		m.UI.MiddleOffset = 0
		return m, nil

	case "g", "end":
		if maxIdx >= 0 {
			m.UI.MiddleCursor = maxIdx
			m.UI.MiddleOffset = maxIdx - paneHeight(m) + 1
			if m.UI.MiddleOffset < 0 {
				m.UI.MiddleOffset = 0
			}
		}
		return m, nil

	case "ctrl+d":
		half := paneHeight(m) / 2
		m.UI.MiddleCursor += half
		if m.UI.MiddleCursor > maxIdx {
			m.UI.MiddleCursor = maxIdx
		}
		adjustOffset(&m.UI.MiddleOffset, m.UI.MiddleCursor, paneHeight(m))
		return m, nil

	case "ctrl+u":
		half := paneHeight(m) / 2
		m.UI.MiddleCursor -= half
		if m.UI.MiddleCursor < 0 {
			m.UI.MiddleCursor = 0
		}
		adjustOffset(&m.UI.MiddleOffset, m.UI.MiddleCursor, paneHeight(m))
		return m, nil

	case "enter":
		albumID := selectedAlbumID(m)
		if albumID > 0 {
			m.Library.SelectedAlbumID = albumID
			return m, db.QueryTracksCmd(m.DB, albumID)
		}
		return m, nil
	}

	items := middlePaneItems(m)
	if isLetter(key) {
		jumpToLetter(&m.UI.MiddleCursor, &m.UI.MiddleOffset, key, items, paneHeight(m))
		return m, nil
	}

	return m, nil
}

func handleRightPaneKey(m Model, key string) (tea.Model, tea.Cmd) {
	maxIdx := len(m.Library.Tracks) - 1

	switch key {
	case "i", "up":
		if m.UI.RightCursor > 0 {
			m.UI.RightCursor--
			adjustOffset(&m.UI.RightOffset, m.UI.RightCursor, paneHeight(m))
		}
		return m, nil

	case "k", "down":
		if m.UI.RightCursor < maxIdx {
			m.UI.RightCursor++
			adjustOffset(&m.UI.RightOffset, m.UI.RightCursor, paneHeight(m))
		}
		return m, nil

	case "t", "home":
		m.UI.RightCursor = 0
		m.UI.RightOffset = 0
		return m, nil

	case "g", "end":
		if maxIdx >= 0 {
			m.UI.RightCursor = maxIdx
			m.UI.RightOffset = maxIdx - paneHeight(m) + 1
			if m.UI.RightOffset < 0 {
				m.UI.RightOffset = 0
			}
		}
		return m, nil

	case "ctrl+d":
		half := paneHeight(m) / 2
		m.UI.RightCursor += half
		if m.UI.RightCursor > maxIdx {
			m.UI.RightCursor = maxIdx
		}
		adjustOffset(&m.UI.RightOffset, m.UI.RightCursor, paneHeight(m))
		return m, nil

	case "ctrl+u":
		half := paneHeight(m) / 2
		m.UI.RightCursor -= half
		if m.UI.RightCursor < 0 {
			m.UI.RightCursor = 0
		}
		adjustOffset(&m.UI.RightOffset, m.UI.RightCursor, paneHeight(m))
		return m, nil

	case "enter":
		tracks := trackList(m)
		if m.UI.RightCursor < len(tracks) {
			return playTrack(m, tracks[m.UI.RightCursor])
		}
		return m, nil
	}

	// Letter-jump by title
	tracks := trackList(m)
	titles := make([]string, len(tracks))
	for i, t := range tracks {
		titles[i] = t.Title
	}
	if isLetter(key) {
		jumpToLetter(&m.UI.RightCursor, &m.UI.RightOffset, key, titles, paneHeight(m))
		return m, nil
	}

	return m, nil
}

// ── Helpers ─────────────────────────────────────────────────────────────

func nextPane(p model.Pane) model.Pane {
	return (p + 1) % 3
}

func prevPane(p model.Pane) model.Pane {
	if p == 0 {
		return 2
	}
	return p - 1
}

func paneHeight(m Model) int {
	h := m.UI.Height - 5
	if h < 1 {
		h = 1
	}
	return h
}

func adjustOffset(offset *int, cursor int, height int) {
	if cursor < *offset {
		*offset = cursor
	}
	if cursor >= *offset+height {
		*offset = cursor - height + 1
	}
}

// isLetter returns true for single uppercase letters A-Z. Shift+Letter is the
// convention for letter-jumping through lists (browse pane, queue, etc.).
// Lowercase letters are reserved for navigation commands (e.g. g = bottom).
// See docs/INTERACTION_DESIGN.md §Keyboard Navigation.
func isLetter(key string) bool {
	if len(key) != 1 {
		return false
	}
	r := rune(key[0])
	return r >= 'A' && r <= 'Z'
}

func jumpToLetter(cursor *int, offset *int, letter string, items []string, height int) {
	if len(items) == 0 {
		return
	}
	upper := strings.ToUpper(letter)
	for i, item := range items {
		if len(item) > 0 && strings.EqualFold(string(item[0]), upper) {
			*cursor = i
			adjustOffset(offset, i, height)
			return
		}
	}
}

func browseModes() []model.BrowseMode {
	return []model.BrowseMode{
		model.BrowseModeArtist,
		model.BrowseModeLabel,
		model.BrowseModeGenre,
		model.BrowseModeYear,
	}
}

func browseModeIndex(mode model.BrowseMode) int {
	for i, m := range browseModes() {
		if m == mode {
			return i
		}
	}
	return 0
}

func themeNames() []string {
	return []string{"slate", "phosphor", "amber", "gameboy"}
}

func nextTheme(current string) string {
	names := themeNames()
	for i, name := range names {
		if name == current {
			return names[(i+1)%len(names)]
		}
	}
	return names[0]
}

// ── Playback helpers ──────────────────────────────────────────────────────────

func handlePlayPause(m Model) (tea.Model, tea.Cmd) {
	if m.Player.CurrentTrack == nil {
		tracks := trackList(m)
		if m.UI.RightCursor < len(tracks) {
			return playTrack(m, tracks[m.UI.RightCursor])
		}
		return m, nil
	}
	var err error
	if m.Player.State == model.PlaybackStatePlaying {
		err = m.MPV.Pause()
	} else {
		err = m.MPV.Play()
	}
	if err != nil {
		slog.Warn("playback: play/pause failed", "error", err)
	}
	return m, nil
}

func handlePrevTrack(m Model) (tea.Model, tea.Cmd) {
	if len(m.Library.Tracks) == 0 {
		return m, nil
	}
	track := m.Library.Tracks[m.UI.RightCursor]
	return playTrack(m, track)
}

func handleNextTrack(m Model) (tea.Model, tea.Cmd) {
	if len(m.Library.Tracks) == 0 {
		return m, nil
	}
	track := m.Library.Tracks[m.UI.RightCursor]
	return playTrack(m, track)
}

func handleSeek(m Model, seconds float64) (tea.Model, tea.Cmd) {
	if m.Player.CurrentTrack == nil {
		return m, nil
	}
	if err := m.MPV.SeekRelative(seconds); err != nil {
		slog.Warn("playback: seek failed", "error", err)
	}
	return m, nil
}

func handleVolume(m Model, delta int) (tea.Model, tea.Cmd) {
	if !m.Player.MPVReady {
		return m, nil
	}
	newVol := m.Player.Volume + delta
	if newVol < 0 {
		newVol = 0
	}
	if newVol > 100 {
		newVol = 100
	}
	if err := m.MPV.SetVolume(newVol); err != nil {
		slog.Warn("playback: volume failed", "error", err)
	}
	m.Player.Volume = newVol
	return m, nil
}

func playTrack(m Model, track model.Track) (tea.Model, tea.Cmd) {
	if !m.Player.MPVReady {
		return m, nil
	}

	slog.Info("playback: loading track", "path", track.FilePath)

	if err := m.MPV.LoadFile(track.FilePath); err != nil {
		slog.Warn("playback: load failed", "error", err, "path", track.FilePath)
		return m, nil
	}
	if err := m.MPV.Play(); err != nil {
		slog.Warn("playback: play failed", "error", err)
		return m, nil
	}

	// Drain stale events generated by LoadFile/Play (idle→loaded transition
	// end-file, etc.) so they don't immediately clear the current track.
	drainMPVEvents(m.Player.Events)

	m.Player.CurrentTrack = &track
	m.Player.State = model.PlaybackStatePlaying
	m.Player.PositionSec = 0
	m.Player.DisplayPositionSec = 0
	m.Player.DurationSec = float64(track.DurationMs) / 1000.0
	return m, tea.Batch(
		messages.TickCmd(),
		mpv.SubscribeCmd(m.Player.Events),
	)
}

func drainMPVEvents(ch <-chan tea.Msg) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
