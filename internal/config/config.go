// Package config loads, validates, and writes config.toml. It exports the
// Config struct that every other package reads during initialisation.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Library     LibraryConfig          `toml:"library"`
	Player      PlayerConfig           `toml:"player"`
	UI          UIConfig               `toml:"ui"`
	Themes      map[string]ThemeConfig `toml:"themes"`
	Keybindings KeybindingsConfig      `toml:"keybindings"`
	Logging     LoggingConfig          `toml:"logging"`
	EQ          EQConfig               `toml:"eq"`
	Hooks       HooksConfig            `toml:"hooks"`
	Scrobbling  ScrobblingConfig       `toml:"scrobbling"`
}

type LibraryConfig struct {
	Paths         []string `toml:"paths"`
	ScanOnStartup bool     `toml:"scan_on_startup"`
}

type PlayerConfig struct {
	MPVSocket           string `toml:"mpv_socket"`
	DefaultVolume       int    `toml:"default_volume"`
	DefaultPlaybackMode string `toml:"default_playback_mode"`
}

type UIConfig struct {
	Theme             string                  `toml:"theme"`
	DefaultBrowseMode string                  `toml:"default_browse_mode"`
	Columns           map[string]ColumnConfig `toml:"columns"`
}

type ColumnConfig struct {
	Visible []string `toml:"visible"`
}

type ThemeConfig struct {
	BG     string `toml:"bg"`
	FG     string `toml:"fg"`
	Accent string `toml:"accent"`
	Muted  string `toml:"muted"`
}

type KeybindingsConfig struct {
	CursorDown       KeyBindings `toml:"cursor_down"`
	CursorUp         KeyBindings `toml:"cursor_up"`
	CursorLeft       KeyBindings `toml:"cursor_left"`
	CursorRight      KeyBindings `toml:"cursor_right"`
	PageDown         KeyBindings `toml:"page_down"`
	PageUp           KeyBindings `toml:"page_up"`
	JumpToTop        KeyBindings `toml:"jump_to_top"`
	JumpToBottom     KeyBindings `toml:"jump_to_bottom"`
	FocusNextPane    KeyBindings `toml:"focus_next_pane"`
	FocusPrevPane    KeyBindings `toml:"focus_prev_pane"`
	ActionMenu       KeyBindings `toml:"action_menu"`
	QueueAdd         KeyBindings `toml:"queue_add"`
	QueueAddNext     KeyBindings `toml:"queue_add_next"`
	QueueAddAlbum    KeyBindings `toml:"queue_add_album"`
	SelectToggle     KeyBindings `toml:"select_toggle"`
	SelectAll        KeyBindings `toml:"select_all"`
	OpenSearch       KeyBindings `toml:"open_search"`
	OpenQueue        KeyBindings `toml:"open_queue"`
	OpenInfo         KeyBindings `toml:"open_info"`
	OpenHelp         KeyBindings `toml:"open_help"`
	OpenBrowsePicker KeyBindings `toml:"open_browse_picker"`
	OpenColumns      KeyBindings `toml:"open_columns"`
	OpenSortPicker   KeyBindings `toml:"open_sort_picker"`
	EditTags         KeyBindings `toml:"edit_tags"`
	Dismiss          KeyBindings `toml:"dismiss"`
	PlayPause        KeyBindings `toml:"play_pause"`
	NextTrack        KeyBindings `toml:"next_track"`
	PrevTrack        KeyBindings `toml:"prev_track"`
	SeekForward5     KeyBindings `toml:"seek_forward_5"`
	SeekBack5        KeyBindings `toml:"seek_back_5"`
	SeekForward30    KeyBindings `toml:"seek_forward_30"`
	SeekBack30       KeyBindings `toml:"seek_back_30"`
	VolumeDown       KeyBindings `toml:"volume_down"`
	VolumeUp         KeyBindings `toml:"volume_up"`
	VolumeReset      KeyBindings `toml:"volume_reset"`
	CyclePlayMode    KeyBindings `toml:"cycle_play_mode"`
	QueueMoveDown    KeyBindings `toml:"queue_move_down"`
	QueueMoveUp      KeyBindings `toml:"queue_move_up"`
	QueueRemove      KeyBindings `toml:"queue_remove"`
	QueueClear       KeyBindings `toml:"queue_clear"`
	ItemMoveDown     KeyBindings `toml:"item_move_down"`
	ItemMoveUp       KeyBindings `toml:"item_move_up"`
	ItemToggle       KeyBindings `toml:"item_toggle"`
	Quit             KeyBindings `toml:"quit"`
}

type KeyBindings []string

func (keys KeyBindings) Matches(key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

type LoggingConfig struct {
	Level string `toml:"level"`
	Path  string `toml:"path"`
}

type EQConfig struct {
	Enabled       bool                `toml:"enabled"`
	DefaultPreset string              `toml:"default_preset"`
	Presets       map[string]EQPreset `toml:"presets"`
}

type EQPreset struct {
	Bands []int `toml:"bands"`
}

type HooksConfig struct {
	OnTrackChange   string `toml:"on_track_change"`
	OnPlaybackStart string `toml:"on_playback_start"`
	OnPlaybackStop  string `toml:"on_playback_stop"`
}

type ScrobblingConfig struct {
	Enabled  bool   `toml:"enabled"`
	Username string `toml:"username"`
}

func Default() Config {
	return Config{
		Library: LibraryConfig{
			Paths:         []string{},
			ScanOnStartup: true,
		},
		Player: PlayerConfig{
			MPVSocket:           "/tmp/waveshell.sock",
			DefaultVolume:       100,
			DefaultPlaybackMode: "stop_at_end",
		},
		UI: UIConfig{
			Theme:             "slate",
			DefaultBrowseMode: "artist",
			Columns: map[string]ColumnConfig{
				"artist":   {Visible: []string{"track_number", "title", "format", "sample_rate", "bit_depth", "duration"}},
				"label":    {Visible: []string{"track_number", "title", "artist", "format", "duration"}},
				"genre":    {Visible: []string{"track_number", "title", "artist", "format", "duration"}},
				"year":     {Visible: []string{"track_number", "title", "artist", "format", "duration"}},
				"grouping": {Visible: []string{"track_number", "title", "artist", "format", "duration"}},
				"playlist": {Visible: []string{"playlist_position", "title", "artist", "album", "duration", "format"}},
			},
		},
		Themes: map[string]ThemeConfig{
			"gameboy":  {BG: "#0f380f", FG: "#9bbc0f", Accent: "#306230", Muted: "#8bac0f"},
			"phosphor": {BG: "#0a0a0a", FG: "#33ff33", Accent: "#00ff00", Muted: "#1a6b1a"},
			"amber":    {BG: "#0d0900", FG: "#ffb000", Accent: "#ff8c00", Muted: "#7a5200"},
			"slate":    {BG: "#1e2030", FG: "#c8d3f5", Accent: "#82aaff", Muted: "#545c7e"},
		},
		Keybindings: KeybindingsConfig{
			CursorDown:       []string{"j", "down"},
			CursorUp:         []string{"k", "up"},
			CursorLeft:       []string{"h", "left"},
			CursorRight:      []string{"l", "right"},
			PageDown:         []string{"ctrl+d"},
			PageUp:           []string{"ctrl+u"},
			JumpToTop:        []string{"g", "home"},
			JumpToBottom:     []string{"G", "end"},
			FocusNextPane:    []string{"tab"},
			FocusPrevPane:    []string{"shift+tab"},
			ActionMenu:       []string{"enter"},
			QueueAdd:         []string{"space"},
			QueueAddNext:     []string{"shift+space"},
			QueueAddAlbum:    []string{"a"},
			SelectToggle:     []string{"space"},
			SelectAll:        []string{"V"},
			OpenSearch:       []string{"/"},
			OpenQueue:        []string{"q"},
			OpenInfo:         []string{"i"},
			OpenHelp:         []string{"?"},
			OpenBrowsePicker: []string{"B"},
			OpenColumns:      []string{"C"},
			OpenSortPicker:   []string{"S"},
			EditTags:         []string{"e"},
			Dismiss:          []string{"esc"},
			PlayPause:        []string{"p"},
			NextTrack:        []string{"n"},
			PrevTrack:        []string{"b"},
			SeekForward5:     []string{"]"},
			SeekBack5:        []string{"["},
			SeekForward30:    []string{"}"},
			SeekBack30:       []string{"{"},
			VolumeDown:       []string{"-"},
			VolumeUp:         []string{"="},
			VolumeReset:      []string{"0"},
			CyclePlayMode:    []string{"m"},
			QueueMoveDown:    []string{"J"},
			QueueMoveUp:      []string{"K"},
			QueueRemove:      []string{"x"},
			QueueClear:       []string{"c"},
			ItemMoveDown:     []string{"J"},
			ItemMoveUp:       []string{"K"},
			ItemToggle:       []string{"space"},
			Quit:             []string{"ctrl+c"},
		},
		Logging: LoggingConfig{
			Level: "info",
			Path:  "",
		},
		EQ: EQConfig{
			Enabled:       false,
			DefaultPreset: "flat",
			Presets:       map[string]EQPreset{},
		},
		Hooks: HooksConfig{},
		Scrobbling: ScrobblingConfig{
			Enabled: false,
		},
	}
}

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

var validColumnIDs = map[string]bool{
	"track_number":      true,
	"playlist_position": true,
	"album_track":       true,
	"title":             true,
	"artist":            true,
	"album":             true,
	"year":              true,
	"genre":             true,
	"label":             true,
	"grouping":          true,
	"format":            true,
	"codec":             true,
	"sample_rate":       true,
	"bit_depth":         true,
	"bitrate":           true,
	"duration":          true,
	"file_size":         true,
	"date_added":        true,
}

var validPlaybackModes = map[string]bool{
	"stop_at_end":  true,
	"repeat_queue": true,
	"repeat_track": true,
	"shuffle":      true,
}

var validLogLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
}

var validBrowseModes = map[string]bool{
	"artist":   true,
	"label":    true,
	"genre":    true,
	"year":     true,
	"grouping": true,
}

var namedKeys = map[string]bool{
	"space":     true,
	"tab":       true,
	"enter":     true,
	"esc":       true,
	"backspace": true,
	"home":      true,
	"end":       true,
	"up":        true,
	"down":      true,
	"left":      true,
	"right":     true,
	"pgup":      true,
	"pgdown":    true,
	"insert":    true,
	"delete":    true,
}

var modifierRe = regexp.MustCompile(`^(ctrl|alt|shift)\+(.+)$`)

func isPrintableASCII(key string) bool {
	return len(key) == 1 && key[0] >= 32 && key[0] <= 126
}

func isValidBindingKey(key string) bool {
	if isPrintableASCII(key) {
		return true
	}
	if namedKeys[key] {
		return true
	}
	m := modifierRe.FindStringSubmatch(key)
	if m != nil {
		rest := m[2]
		if isPrintableASCII(rest) {
			return true
		}
		return namedKeys[rest]
	}
	return false
}

func (c Config) Theme() (ThemeConfig, error) {
	if t, ok := c.Themes[c.UI.Theme]; ok {
		return t, nil
	}
	return ThemeConfig{}, fmt.Errorf("theme %q not found", c.UI.Theme)
}

func ensureBuiltinThemes(cfg *Config) {
	builtins := Default().Themes
	for k, v := range builtins {
		if _, exists := cfg.Themes[k]; !exists {
			cfg.Themes[k] = v
		}
	}
}

func normalizeBrowseMode(mode string) string {
	if validBrowseModes[mode] {
		return mode
	}
	return "artist"
}

func Load() (Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return Config{}, fmt.Errorf("cannot determine user config dir: %w", err)
	}
	path := filepath.Join(configDir, "waveshell", "config.toml")
	return LoadFrom(path)
}

func LoadFrom(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}

	ensureBuiltinThemes(&cfg)

	if err := interpolateConfig(&cfg); err != nil {
		return Config{}, fmt.Errorf("interpolating config: %w", err)
	}

	applyTildeExpansion(&cfg)

	cfg.UI.DefaultBrowseMode = normalizeBrowseMode(cfg.UI.DefaultBrowseMode)

	if err := validateConfig(&cfg); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}

	filterLibraryPaths(&cfg)

	return cfg, nil
}

func WriteConfig(path string, cfg Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}
	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing config file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming config file: %w", err)
	}
	return nil
}

func validateConfig(cfg *Config) error {
	if cfg.Player.DefaultVolume < 0 || cfg.Player.DefaultVolume > 100 {
		return fmt.Errorf("player.default_volume must be between 0 and 100, got %d", cfg.Player.DefaultVolume)
	}

	if !validPlaybackModes[cfg.Player.DefaultPlaybackMode] {
		return fmt.Errorf("player.default_playback_mode %q is not valid; must be one of: stop_at_end, repeat_queue, repeat_track, shuffle", cfg.Player.DefaultPlaybackMode)
	}

	for i, p := range cfg.Library.Paths {
		if strings.HasPrefix(p, "~") {
			return fmt.Errorf("library.paths[%d]: %q could not be expanded (HOME not set?)", i, p)
		}
		if !strings.HasPrefix(p, "/") {
			return fmt.Errorf("library.paths[%d]: %q is not an absolute path", i, p)
		}
	}

	if _, err := cfg.Theme(); err != nil {
		return err
	}

	for name, theme := range cfg.Themes {
		if theme.BG == "" || theme.FG == "" || theme.Accent == "" || theme.Muted == "" {
			return fmt.Errorf("theme %q is missing required color fields (bg, fg, accent, muted)", name)
		}
		if !hexColorRe.MatchString(theme.BG) {
			return fmt.Errorf("theme %q: bg %q is not a valid hex color (must be #rrggbb)", name, theme.BG)
		}
		if !hexColorRe.MatchString(theme.FG) {
			return fmt.Errorf("theme %q: fg %q is not a valid hex color (must be #rrggbb)", name, theme.FG)
		}
		if !hexColorRe.MatchString(theme.Accent) {
			return fmt.Errorf("theme %q: accent %q is not a valid hex color (must be #rrggbb)", name, theme.Accent)
		}
		if !hexColorRe.MatchString(theme.Muted) {
			return fmt.Errorf("theme %q: muted %q is not a valid hex color (must be #rrggbb)", name, theme.Muted)
		}
	}

	if !validLogLevels[cfg.Logging.Level] {
		return fmt.Errorf("logging.level %q is not valid; must be one of: debug, info, warn, error", cfg.Logging.Level)
	}

	for _, colCfg := range cfg.UI.Columns {
		if len(colCfg.Visible) == 0 {
			return fmt.Errorf("ui.columns: visible array must not be empty")
		}
		for _, col := range colCfg.Visible {
			if !validColumnIDs[col] {
				return fmt.Errorf("ui.columns: %q is not a valid column identifier", col)
			}
		}
	}

	for name, preset := range cfg.EQ.Presets {
		if len(preset.Bands) != 10 {
			return fmt.Errorf("eq.presets.%s.bands must have exactly 10 values, got %d", name, len(preset.Bands))
		}
		for _, b := range preset.Bands {
			if b < -12 || b > 12 {
				return fmt.Errorf("eq.presets.%s.bands: value %d is outside range -12 to +12", name, b)
			}
		}
	}

	allBindings := []struct {
		name string
		keys KeyBindings
	}{
		{"cursor_down", cfg.Keybindings.CursorDown},
		{"cursor_up", cfg.Keybindings.CursorUp},
		{"cursor_left", cfg.Keybindings.CursorLeft},
		{"cursor_right", cfg.Keybindings.CursorRight},
		{"page_down", cfg.Keybindings.PageDown},
		{"page_up", cfg.Keybindings.PageUp},
		{"jump_to_top", cfg.Keybindings.JumpToTop},
		{"jump_to_bottom", cfg.Keybindings.JumpToBottom},
		{"focus_next_pane", cfg.Keybindings.FocusNextPane},
		{"focus_prev_pane", cfg.Keybindings.FocusPrevPane},
		{"action_menu", cfg.Keybindings.ActionMenu},
		{"queue_add", cfg.Keybindings.QueueAdd},
		{"queue_add_next", cfg.Keybindings.QueueAddNext},
		{"queue_add_album", cfg.Keybindings.QueueAddAlbum},
		{"select_toggle", cfg.Keybindings.SelectToggle},
		{"select_all", cfg.Keybindings.SelectAll},
		{"open_search", cfg.Keybindings.OpenSearch},
		{"open_queue", cfg.Keybindings.OpenQueue},
		{"open_info", cfg.Keybindings.OpenInfo},
		{"open_help", cfg.Keybindings.OpenHelp},
		{"open_browse_picker", cfg.Keybindings.OpenBrowsePicker},
		{"open_columns", cfg.Keybindings.OpenColumns},
		{"open_sort_picker", cfg.Keybindings.OpenSortPicker},
		{"edit_tags", cfg.Keybindings.EditTags},
		{"dismiss", cfg.Keybindings.Dismiss},
		{"play_pause", cfg.Keybindings.PlayPause},
		{"next_track", cfg.Keybindings.NextTrack},
		{"prev_track", cfg.Keybindings.PrevTrack},
		{"seek_forward_5", cfg.Keybindings.SeekForward5},
		{"seek_back_5", cfg.Keybindings.SeekBack5},
		{"seek_forward_30", cfg.Keybindings.SeekForward30},
		{"seek_back_30", cfg.Keybindings.SeekBack30},
		{"volume_down", cfg.Keybindings.VolumeDown},
		{"volume_up", cfg.Keybindings.VolumeUp},
		{"volume_reset", cfg.Keybindings.VolumeReset},
		{"cycle_play_mode", cfg.Keybindings.CyclePlayMode},
		{"queue_move_down", cfg.Keybindings.QueueMoveDown},
		{"queue_move_up", cfg.Keybindings.QueueMoveUp},
		{"queue_remove", cfg.Keybindings.QueueRemove},
		{"queue_clear", cfg.Keybindings.QueueClear},
		{"item_move_down", cfg.Keybindings.ItemMoveDown},
		{"item_move_up", cfg.Keybindings.ItemMoveUp},
		{"item_toggle", cfg.Keybindings.ItemToggle},
		{"quit", cfg.Keybindings.Quit},
	}

	for _, b := range allBindings {
		for _, k := range b.keys {
			if !isValidBindingKey(k) {
				return fmt.Errorf("keybindings.%s: %q is not a valid key", b.name, k)
			}
		}
	}

	return nil
}

// filterLibraryPaths removes library paths that do not exist or are not directories.
// The caller should log the dropped paths at warn level.
func filterLibraryPaths(cfg *Config) {
	var valid []string
	for _, p := range cfg.Library.Paths {
		info, err := os.Stat(p)
		if err != nil || !info.IsDir() {
			continue
		}
		valid = append(valid, p)
	}
	cfg.Library.Paths = valid
}
