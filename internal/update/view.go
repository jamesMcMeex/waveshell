package update

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jamesMcMeex/waveshell/internal/model"
)

func (m Model) View() string {
	if m.UI.Width == 0 {
		slog.Debug("view: waiting for window size")
		return "Loading waveshell...\n"
	}

	base := renderBase(m)

	if m.UI.ActiveOverlay == OverlayHelp {
		base = renderHelpOverlay(m, base)
	}

	if m.UI.ShowBrowsePicker {
		base = renderBrowsePicker(m, base)
	}

	return base
}

func renderBase(m Model) string {
	theme := ResolveTheme(m.Config)

	// Account for 2 border chars per pane (left + right) = 6 total for 3 panes
	borderChars := 6
	leftWidth := (m.UI.Width - borderChars) * 22 / 100
	middleWidth := (m.UI.Width - borderChars) * 28 / 100
	tracksWidth := (m.UI.Width - borderChars) - leftWidth - middleWidth

	if leftWidth < 12 {
		leftWidth = 12
	}
	if middleWidth < 14 {
		middleWidth = 14
	}
	if tracksWidth < 22 {
		tracksWidth = 22
	}

	contentLines := paneHeight(m)

	left := renderLeftPane(m, theme, leftWidth, contentLines)
	middle := renderMiddlePane(m, theme, middleWidth, contentLines)
	tracks := renderTracksPane(m, theme, tracksWidth, contentLines)

	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, middle, tracks)

	nowPlaying := renderNowPlaying(m, theme)
	statusBar := renderStatusBar(m, theme)
	hints := renderKeyHints(m, theme)

	return lipgloss.JoinVertical(lipgloss.Left, panes, nowPlaying, statusBar, hints)
}

func renderLeftPane(m Model, th Theme, width int, height int) string {
	title := leftPaneLabel(m)
	items := leftPaneItems(m)
	inner := width - 2

	var lines []string
	start := m.UI.LeftOffset
	for i := start; i < start+height && i < len(items); i++ {
		line := items[i]
		if len(line) > inner {
			line = line[:inner]
		}
		if i == m.UI.LeftCursor {
			lines = append(lines, th.CursorLine().Render(fmt.Sprintf("▶ %-*s", inner-2, line)))
		} else {
			lines = append(lines, th.NormalLine().Render(fmt.Sprintf("  %-*s", inner-2, line)))
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", inner))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	titleLine := th.PaneTitle().Render(fmt.Sprintf(" %-*s ", inner-2, title))
	paddedTitle := fmt.Sprintf("%-*s", width, titleLine)
	border := th.PaneBorder(m.UI.ActivePane == model.PaneLeft)
	return lipgloss.JoinVertical(lipgloss.Left, paddedTitle, border.Width(width).Render(content))
}

func renderMiddlePane(m Model, th Theme, width int, height int) string {
	title := middlePaneLabel()
	items := middlePaneItems(m)
	inner := width - 2

	var lines []string
	start := m.UI.MiddleOffset
	for i := start; i < start+height && i < len(items); i++ {
		line := items[i]
		if len(line) > inner {
			line = line[:inner]
		}
		if i == m.UI.MiddleCursor {
			lines = append(lines, th.CursorLine().Render(fmt.Sprintf("▶ %-*s", inner-2, line)))
		} else {
			lines = append(lines, th.NormalLine().Render(fmt.Sprintf("  %-*s", inner-2, line)))
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", inner))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	titleLine := th.PaneTitle().Render(fmt.Sprintf(" %-*s ", inner-2, title))
	paddedTitle := fmt.Sprintf("%-*s", width, titleLine)
	border := th.PaneBorder(m.UI.ActivePane == model.PaneMiddle)
	return lipgloss.JoinVertical(lipgloss.Left, paddedTitle, border.Width(width).Render(content))
}

func renderTracksPane(m Model, th Theme, width int, height int) string {
	cols := visibleColumns(m)
	inner := width - 2
	colWidths := columnWidths(m, cols, inner)
	header := renderColumnHeader(cols, colWidths, th)

	paddedHeader := fmt.Sprintf("%-*s", width, header)

	tracks := trackList(m)
	var lines []string
	start := m.UI.RightOffset
	for i := start; i < start+height && i < len(tracks); i++ {
		row := renderTrackRow(tracks[i], cols, colWidths)
		if len(row) > inner {
			row = row[:inner]
		}
		line := fmt.Sprintf(" %-*s", inner-1, row)
		if i == m.UI.RightCursor {
			lines = append(lines, th.CursorLine().Render(line))
		} else {
			lines = append(lines, th.NormalLine().Render(line))
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", inner))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	border := th.PaneBorder(m.UI.ActivePane == model.PaneRight)
	return lipgloss.JoinVertical(lipgloss.Left, paddedHeader, border.Width(width).Render(content))
}

func renderColumnHeader(cols []string, colWidths map[string]int, th Theme) string {
	labels := map[string]string{
		"track_number": "#",
		"title":        "TITLE",
		"duration":     "DUR",
		"format":       "FORMAT",
		"sample_rate":  "RATE",
		"bit_depth":    "DEPTH",
		"bitrate":      "KBPS",
		"artist":       "ARTIST",
		"album":        "ALBUM",
		"year":         "YEAR",
		"genre":        "GENRE",
		"label":        "LABEL",
		"grouping":     "GROUP",
	}
	var parts []string
	for _, col := range cols {
		label := labels[col]
		if label == "" {
			label = col
		}
		w := colWidths[col]
		if len(label) > w {
			label = label[:w]
		}
		parts = append(parts, fmt.Sprintf("%-*s", w, label))
	}
	return th.PaneTitle().Render(fmt.Sprintf(" %-*s", totalWidth(cols, colWidths), strings.Join(parts, " ")))
}

func totalWidth(cols []string, colWidths map[string]int) int {
	total := 0
	for _, col := range cols {
		total += colWidths[col]
	}
	return total + len(cols) - 1
}

func visibleColumns(m Model) []string {
	browseMode := string(m.UI.BrowseMode)
	if m.Config != nil {
		if cc, ok := m.Config.UI.Columns[browseMode]; ok && len(cc.Visible) > 0 {
			return cc.Visible
		}
	}
	return defaultColumns(browseMode)
}

func defaultColumns(mode string) []string {
	switch mode {
	case string(model.BrowseModeArtist):
		return []string{"track_number", "title", "format", "sample_rate", "bit_depth", "duration"}
	case string(model.BrowseModeLabel):
		return []string{"track_number", "title", "artist", "format", "duration"}
	case string(model.BrowseModeGenre):
		return []string{"track_number", "title", "artist", "format", "duration"}
	case string(model.BrowseModeYear):
		return []string{"track_number", "title", "artist", "format", "duration"}
	default:
		return []string{"track_number", "title", "format", "duration"}
	}
}

func columnWidths(m Model, cols []string, innerWidth int) map[string]int {
	widths := make(map[string]int)
	fixedWidths := map[string]int{
		"track_number": 3,
		"duration":     6,
		"format":       16,
		"sample_rate":  6,
		"bit_depth":    5,
		"bitrate":      5,
		"year":         5,
		"label":        12,
		"grouping":     8,
		"genre":        10,
		"artist":       12,
		"album":        12,
	}

	avail := innerWidth - len(cols)

	for _, col := range cols {
		if col == "title" {
			continue
		}
		w, ok := fixedWidths[col]
		if !ok {
			w = 8
		}
		avail -= w
		widths[col] = w
	}
	if avail < 5 {
		avail = 5
	}
	widths["title"] = avail

	return widths
}

func renderStatusBar(m Model, th Theme) string {
	var parts []string

	if m.Library.Scanning {
		total := m.Library.ScanTotal
		if total < 0 {
			total = 0
		}
		processed := m.Library.ScanProcessed
		if processed > total {
			total = processed
		}
		parts = append(parts, fmt.Sprintf("Scanning: %d/%d files", processed, total))
	} else if m.UI.BrowseMode != model.BrowseModeArtist {
		parts = append(parts, fmt.Sprintf("By %s", browseModeLabel(m.UI.BrowseMode)))
	}

	selectedKey := leftPaneSelectedKey(m)
	if selectedKey != "" {
		parts = append(parts, selectedKey)
	}

	trackCount := len(m.Library.Tracks)
	if trackCount > 0 {
		parts = append(parts, fmt.Sprintf("%d tracks", trackCount))
	} else if len(m.Library.Albums) > 0 {
		parts = append(parts, fmt.Sprintf("%d albums", len(m.Library.Albums)))
	} else if !m.Library.Scanning && m.Config != nil && len(m.Config.Library.Paths) == 0 {
		parts = append(parts, "no library paths configured — edit config.toml")
	}

	if len(parts) == 0 {
		parts = append(parts, "waveshell")
	}

	return th.StatusBar().Render(strings.Join(parts, " · "))
}

func browseModeLabel(m model.BrowseMode) string {
	switch m {
	case model.BrowseModeLabel:
		return "Label"
	case model.BrowseModeGenre:
		return "Genre"
	case model.BrowseModeYear:
		return "Year"
	default:
		return "Artist"
	}
}

func renderNowPlaying(m Model, th Theme) string {
	if m.Player.State == model.PlaybackStateStopped || m.Player.CurrentTrack == nil {
		return "" // no playing track
	}

	width := m.UI.Width
	trk := m.Player.CurrentTrack

	// --- Line 1: track info + progress ---
	posSec := m.Player.DisplayPositionSec
	if posSec > m.Player.DurationSec && m.Player.DurationSec > 0 {
		posSec = m.Player.DurationSec
	}
	elapsed := formatDuration(int(posSec * 1000))
	total := formatDuration(int(m.Player.DurationSec * 1000))

	trackLabel := fmt.Sprintf("%s — %s", trk.Artist, trk.Title)
	if len(trackLabel) > width-30 {
		trackLabel = trackLabel[:width-33] + "..."
	}

	progressBar := renderProgressBar(posSec, m.Player.DurationSec, 10)

	rightInfo := fmt.Sprintf("%s %s %s", elapsed, progressBar, total)

	line1 := th.NowPlayingAccent().Render("▶") +
		th.NowPlayingMuted().Render("  ") +
		th.NowPlaying().Render(trackLabel) +
		th.NowPlayingMuted().Render(
			strings.Repeat(" ", width-len(trackLabel)-len(rightInfo)-3),
		) +
		th.NowPlaying().Render(rightInfo)

	// --- Line 2: album + format badge + volume ---
	formatBadge := th.FormatBadge().Render(FormatBadge(*trk))
	albumLabel := fmt.Sprintf("%s · %s", trk.Album, formatBadge)

	volStr := fmt.Sprintf("Vol: %d%%", m.Player.Volume)

	var stateIcon string
	switch m.Player.State {
	case model.PlaybackStatePlaying:
		stateIcon = "▶"
	case model.PlaybackStatePaused:
		stateIcon = "⏸"
	default:
		stateIcon = " "
	}

	// Volume on the right, album on the left
	padding := width - len(albumLabel) - len(volStr) - 3
	if padding < 1 {
		padding = 1
		albumLabel = albumLabel[:width-len(volStr)-6] + "..."
	}

	line2 := fmt.Sprintf(" %s %s%s%s",
		stateIcon,
		albumLabel,
		strings.Repeat(" ", padding),
		volStr,
	)
	line2 = th.NowPlayingMuted().Render(line2)

	return lipgloss.JoinVertical(lipgloss.Left, line1, line2)
}

func renderProgressBar(posSec, durSec float64, barWidth int) string {
	if durSec <= 0 {
		return strings.Repeat("─", barWidth)
	}
	ratio := posSec / durSec
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("─", barWidth-filled)
	return " " + bar + " "
}

func renderKeyHints(m Model, th Theme) string {
	var hints []string
	if m.MPVErr != nil {
		hints = append(hints, "[!] install mpv for playback")
	} else if m.Player.MPVReady {
		hints = append(hints, "[p] play")
		hints = append(hints, "[,/.] prev/next")
	}
	hints = append(hints, "[i/k] navigate")
	hints = append(hints, "[j/l] pane")
	hints = append(hints, "[b] mode")
	hints = append(hints, "[c] theme")
	hints = append(hints, "[h] help")
	hints = append(hints, "[q] quit")
	return th.KeyHintBar().Render(strings.Join(hints, "  "))
}

func renderBrowsePicker(m Model, base string) string {
	th := ResolveTheme(m.Config)
	modes := browseModes()

	var lines []string
	lines = append(lines, th.PaneTitle().Render("BROWSE BY"))
	lines = append(lines, strings.Repeat("─", 24))
	for i, mode := range modes {
		label := browseModeLabel(mode)
		if mode == model.BrowseModeArtist {
			label += " [default]"
		}
		if i == m.UI.BrowsePickerCursor {
			lines = append(lines, th.CursorLine().Render("▶ "+label))
		} else {
			lines = append(lines, th.NormalLine().Render("  "+label))
		}
	}

	picker := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Accent).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	extraHints := th.KeyHintBar().Render("  [i/k] navigate  [enter] select  [esc] cancel")

	return lipgloss.JoinVertical(lipgloss.Left, base, "\n", picker, extraHints)
}

// ── Help overlay ─────────────────────────────────────────────────────

func renderHelpOverlay(m Model, base string) string {
	th := ResolveTheme(m.Config)
	maxHeight := m.UI.Height - 6
	if maxHeight < 6 {
		maxHeight = 6
	}
	helpBox := buildHelpBox(m, th, m.Help.ScrollOffset, maxHeight)

	statusBar := renderStatusBar(m, th)
	hints := renderKeyHints(m, th)

	bodyHeight := m.UI.Height - lipgloss.Height(statusBar) - lipgloss.Height(hints)
	body := lipgloss.Place(
		m.UI.Width,
		bodyHeight,
		lipgloss.Center,
		lipgloss.Center,
		helpBox,
	)

	return lipgloss.JoinVertical(lipgloss.Left, body, statusBar, hints)
}

func buildHelpBox(m Model, th Theme, scrollOffset int, maxHeight int) string {
	var b strings.Builder
	header := th.HelpTitle().Render("HELP")
	items := []helpItem{
		{s: "NAVIGATION", header: true},
		{k: "i / ↑", d: "Cursor up"},
		{k: "k / ↓", d: "Cursor down"},
		{k: "j / ←", d: "Focus previous pane"},
		{k: "l / →", d: "Focus next pane"},
		{k: "Tab / Shift+Tab", d: "Cycle pane focus"},
		{k: "t / Home", d: "Jump to top"},
		{k: "g / End", d: "Jump to bottom"},
		{k: "Ctrl+D / Ctrl+U", d: "Scroll half page"},
		{k: "A–Z", d: "Jump to first matching item"},
		{s: "", header: false},
		{s: "LIBRARY", header: true},
		{k: "b", d: "Browse mode picker"},
		{k: "Enter", d: "Select item"},
		{s: "", header: false},
		{s: "APPLICATION", header: true},
		{k: "c", d: "Cycle theme"},
		{k: "h", d: "Help overlay"},
		{k: "q", d: "Quit"},
		{k: "Esc", d: "Dismiss overlay / cancel"},
		{s: "", header: false},
		{s: "PLAYBACK", header: true},
		{k: "p", d: "Play / pause"},
		{k: ",", d: "Previous track"},
		{k: ".", d: "Next track"},
		{k: "[ / ]", d: "Seek -5s / +5s"},
		{k: "{ / }", d: "Seek -30s / +30s"},
		{k: "- / =", d: "Volume down / up"},
		{k: "0", d: "Reset volume"},
		{s: "", header: false},
		{s: "COMING SOON", header: true},
		{k: "/", d: "Search"},
		{k: "i", d: "Info panel"},
		{k: "Space", d: "Add to queue"},
	}

	b.WriteString(header + "\n")
	b.WriteString(th.MutedText().Render(strings.Repeat("─", 30)) + "\n")
	for _, item := range items {
		if item.header {
			b.WriteString("\n" + th.HelpTitle().Render(item.s) + "\n")
			continue
		}
		if item.k == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("  %-18s %s", th.HelpKey().Render(item.k), th.HelpContent().Render(item.d)) + "\n")
	}

	content := b.String()
	boxLines := strings.Split(content, "\n")
	boxWidth := longestLine(boxLines)

	// Clamp visible lines to maxHeight (accounting for 2 border lines)
	visibleHeight := maxHeight - 2
	if visibleHeight < 4 {
		visibleHeight = 4
	}
	totalLines := len(boxLines)
	if totalLines > visibleHeight {
		endIdx := scrollOffset + visibleHeight
		if endIdx > totalLines {
			endIdx = totalLines
		}
		boxLines = boxLines[scrollOffset:endIdx]
		// Append scroll hint footer if content is clipped
		if endIdx < totalLines || scrollOffset > 0 {
			hint := th.MutedText().Render("[i/k] scroll  [h/esc] close")
			boxLines = append(boxLines, "", hint)
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Accent).
		Width(boxWidth + 2).
		Render(strings.Join(boxLines, "\n"))
}

type helpItem struct {
	s      string
	k      string
	d      string
	header bool
}

func longestLine(lines []string) int {
	max := 0
	for _, l := range lines {
		w := lipgloss.Width(l)
		if w > max {
			max = w
		}
	}
	return max
}
