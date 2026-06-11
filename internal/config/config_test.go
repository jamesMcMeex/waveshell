package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDefault(t *testing.T) {
	cfg := Default()
	if len(cfg.Library.Paths) != 0 {
		t.Errorf("expected empty library paths, got %v", cfg.Library.Paths)
	}
	if cfg.Player.DefaultVolume != 100 {
		t.Errorf("expected default volume 100, got %d", cfg.Player.DefaultVolume)
	}
	if cfg.Player.DefaultPlaybackMode != "stop_at_end" {
		t.Errorf("expected stop_at_end, got %s", cfg.Player.DefaultPlaybackMode)
	}
	if cfg.UI.Theme != "slate" {
		t.Errorf("expected slate theme, got %s", cfg.UI.Theme)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("expected info log level, got %s", cfg.Logging.Level)
	}
	if !cfg.Library.ScanOnStartup {
		t.Error("expected scan_on_startup to default to true")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := LoadFrom("/nonexistent/path/config.toml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.UI.Theme != "slate" {
		t.Errorf("expected defaults, got theme=%s", cfg.UI.Theme)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "")
	cfg, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.UI.Theme != "slate" {
		t.Errorf("expected defaults for empty file, got theme=%s", cfg.UI.Theme)
	}
}

func TestLoad_ValidFull(t *testing.T) {
	dir := t.TempDir()
	musicDir := filepath.Join(dir, "music")
	if err := os.MkdirAll(musicDir, 0755); err != nil {
		t.Fatal(err)
	}
	data := `
[library]
paths = ["` + musicDir + `"]
scan_on_startup = false

[player]
mpv_socket = "/tmp/waveshell.sock"
default_volume = 80
default_playback_mode = "repeat_queue"

[ui]
theme = "phosphor"
default_browse_mode = "label"

[ui.columns.artist]
visible = ["track_number", "title", "format", "duration"]

[themes.custom]
bg = "#000000"
fg = "#ffffff"
accent = "#ff0000"
muted = "#888888"

[logging]
level = "debug"
path = "/tmp/waveshell.log"
`
	writeConfig(t, dir, data)
	cfg, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Library.Paths) != 1 || cfg.Library.Paths[0] != musicDir {
		t.Errorf("library paths: got %v", cfg.Library.Paths)
	}
	if cfg.Library.ScanOnStartup {
		t.Error("expected scan_on_startup false")
	}
	if cfg.Player.DefaultVolume != 80 {
		t.Errorf("expected volume 80, got %d", cfg.Player.DefaultVolume)
	}
	if cfg.Player.DefaultPlaybackMode != "repeat_queue" {
		t.Errorf("expected repeat_queue, got %s", cfg.Player.DefaultPlaybackMode)
	}
	if cfg.UI.Theme != "phosphor" {
		t.Errorf("expected phosphor theme, got %s", cfg.UI.Theme)
	}
	if cfg.UI.DefaultBrowseMode != "label" {
		t.Errorf("expected label browse mode, got %s", cfg.UI.DefaultBrowseMode)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("expected debug log level, got %s", cfg.Logging.Level)
	}
}

func TestLoad_Partial(t *testing.T) {
	dir := t.TempDir()
	data := `
[player]
default_volume = 50
`
	writeConfig(t, dir, data)
	cfg, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Player.DefaultVolume != 50 {
		t.Errorf("expected volume 50, got %d", cfg.Player.DefaultVolume)
	}
	if cfg.UI.Theme != "slate" {
		t.Errorf("expected default theme slate, got %s", cfg.UI.Theme)
	}
}

func TestLoad_MalformedToml(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `invalid toml {{{`)
	_, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err == nil {
		t.Fatal("expected error for malformed TOML")
	}
}

func TestLoad_RelativeLibraryPath(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
[library]
paths = ["./Music"]
`)
	_, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err == nil {
		t.Fatal("expected error for relative library path")
	}
}

func TestLoad_VolumeOutOfRange(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
[player]
default_volume = 200
`)
	_, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err == nil {
		t.Fatal("expected error for volume out of range")
	}
}

func TestLoad_InvalidPlaybackMode(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
[player]
default_playback_mode = "turbo"
`)
	_, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err == nil {
		t.Fatal("expected error for invalid playback mode")
	}
}

func TestLoad_UnknownTheme(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
[ui]
theme = "nonexistent"
`)
	_, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err == nil {
		t.Fatal("expected error for unknown theme")
	}
}

func TestLoad_CustomThemeMissingColor(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
[themes.custom]
bg = "#000000"
fg = "#ffffff"
accent = "#ff0000"
`)
	_, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err == nil {
		t.Fatal("expected error for incomplete custom theme")
	}
}

func TestLoad_InvalidHexColor(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
[themes.custom]
bg = "#000000"
fg = "#fff"
accent = "#ff0000"
muted = "#888888"
`)
	_, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err == nil {
		t.Fatal("expected error for invalid hex color")
	}
}

func TestLoad_InvalidColumnID(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
[ui.columns.artist]
visible = ["track_number", "made_up_column"]
`)
	_, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err == nil {
		t.Fatal("expected error for invalid column ID")
	}
}

func TestLoad_EmptyColumns(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
[ui.columns.artist]
visible = []
`)
	_, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err == nil {
		t.Fatal("expected error for empty columns")
	}
}

func TestInterpolateBasic(t *testing.T) {
	musicDir := t.TempDir()
	t.Setenv("TEST_WAVESHELL_HOME_ARBITRARY", musicDir)
	dir := t.TempDir()
	writeConfig(t, dir, `
[library]
paths = ["${TEST_WAVESHELL_HOME_ARBITRARY}"]
`)
	cfg, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Library.Paths) != 1 || cfg.Library.Paths[0] != musicDir {
		t.Errorf("expected interpolated path %q, got %v", musicDir, cfg.Library.Paths)
	}
}

func TestInterpolateUnsetVar(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
[library]
paths = ["${UNDEFINED_VARIABLE_XYZ}/Music"]
`)
	_, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err == nil {
		t.Fatal("expected error for unset env var")
	}
}

func TestThemeResolveBuiltin(t *testing.T) {
	cfg := Default()
	cfg.UI.Theme = "gameboy"
	theme, err := cfg.Theme()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if theme.BG != "#0f380f" {
		t.Errorf("expected gameboy bg #0f380f, got %s", theme.BG)
	}
}

func TestThemeResolveCustom(t *testing.T) {
	cfg := Default()
	cfg.UI.Theme = "mydark"
	cfg.Themes["mydark"] = ThemeConfig{
		BG:     "#111111",
		FG:     "#eeeeee",
		Accent: "#ff6600",
		Muted:  "#555555",
	}
	theme, err := cfg.Theme()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if theme.BG != "#111111" {
		t.Errorf("expected custom bg, got %s", theme.BG)
	}
}

func TestThemeResolveUnknown(t *testing.T) {
	cfg := Default()
	cfg.UI.Theme = "nope"
	_, err := cfg.Theme()
	if err == nil {
		t.Fatal("expected error for unknown theme")
	}
}

func TestKeyBindingsMatches(t *testing.T) {
	kb := KeyBindings{"j", "down"}
	if !kb.Matches("j") {
		t.Error("expected 'j' to match")
	}
	if !kb.Matches("down") {
		t.Error("expected 'down' to match")
	}
	if kb.Matches("up") {
		t.Error("expected 'up' not to match")
	}
}

func TestWriteConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := Default()
	cfg.Player.DefaultVolume = 42

	if err := WriteConfig(path, cfg); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	got, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom after write: %v", err)
	}
	if got.Player.DefaultVolume != 42 {
		t.Errorf("expected volume 42 after roundtrip, got %d", got.Player.DefaultVolume)
	}
}

func TestLibraryPaths_AllInvalid(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
[library]
paths = ["/nonexistent1", "/nonexistent2"]
`)
	cfg, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Library.Paths) != 0 {
		t.Errorf("expected all invalid paths to be filtered, got %v", cfg.Library.Paths)
	}
}

func TestKeybindings_InvalidKey(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
[keybindings]
cursor_down = ["not_a_valid_key!"]
`)
	if _, err := LoadFrom(filepath.Join(dir, "config.toml")); err == nil {
		t.Fatal("expected error for invalid keybinding key")
	}
}

func TestEQPreset_InvalidBands(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
[eq.presets.test]
bands = [1, 2, 3]
`)
	_, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err == nil {
		t.Fatal("expected error for wrong EQ bands count")
	}
}

func TestDefaultsColumnConfigPresent(t *testing.T) {
	cfg := Default()
	for _, mode := range []string{"artist", "label", "genre", "year", "grouping"} {
		cc, ok := cfg.UI.Columns[mode]
		if !ok {
			t.Errorf("missing column config for mode %s", mode)
			continue
		}
		if len(cc.Visible) == 0 {
			t.Errorf("empty columns for mode %s", mode)
		}
	}
}

func TestThemeBuiltinsSurviveUserDefined(t *testing.T) {
	dir := t.TempDir()
	data := `
[themes.custom]
bg = "#000000"
fg = "#ffffff"
accent = "#ff0000"
muted = "#888888"

[ui]
theme = "slate"
`
	writeConfig(t, dir, data)
	cfg, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cfg.Themes["slate"]; !ok {
		t.Error("built-in theme 'slate' missing after loading with custom theme")
	}
	if _, ok := cfg.Themes["gameboy"]; !ok {
		t.Error("built-in theme 'gameboy' missing after loading with custom theme")
	}
	theme, err := cfg.Theme()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if theme.BG != "#1e2030" {
		t.Errorf("expected slate bg, got %s", theme.BG)
	}
}

func TestBrowseMode_Valid(t *testing.T) {
	for _, mode := range []string{"artist", "label", "genre", "year", "grouping"} {
		dir := t.TempDir()
		writeConfig(t, dir, "[ui]\ndefault_browse_mode = \""+mode+"\"\n")
		cfg, err := LoadFrom(filepath.Join(dir, "config.toml"))
		if err != nil {
			t.Fatalf("browse mode %q should be valid: %v", mode, err)
		}
		if cfg.UI.DefaultBrowseMode != mode {
			t.Errorf("expected %q, got %q", mode, cfg.UI.DefaultBrowseMode)
		}
	}
}

func TestBrowseMode_PlaylistNormalized(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "[ui]\ndefault_browse_mode = \"playlist\"\n")
	cfg, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatalf("playlist should be accepted: %v", err)
	}
	if cfg.UI.DefaultBrowseMode != "artist" {
		t.Errorf("expected 'artist' (normalized from 'playlist'), got %q", cfg.UI.DefaultBrowseMode)
	}
}

func TestBrowseMode_UnknownNormalized(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "[ui]\ndefault_browse_mode = \"xyzzy\"\n")
	cfg, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatalf("unknown browse mode should be accepted and normalized: %v", err)
	}
	if cfg.UI.DefaultBrowseMode != "artist" {
		t.Errorf("expected 'artist' (normalized), got %q", cfg.UI.DefaultBrowseMode)
	}
}

func TestVolumeBoundaries(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "[player]\ndefault_volume = 0\n")
	cfg, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatalf("volume 0 should be valid: %v", err)
	}
	if cfg.Player.DefaultVolume != 0 {
		t.Errorf("expected volume 0, got %d", cfg.Player.DefaultVolume)
	}

	dir2 := t.TempDir()
	writeConfig(t, dir2, "[player]\ndefault_volume = 100\n")
	cfg2, err := LoadFrom(filepath.Join(dir2, "config.toml"))
	if err != nil {
		t.Fatalf("volume 100 should be valid: %v", err)
	}
	if cfg2.Player.DefaultVolume != 100 {
		t.Errorf("expected volume 100, got %d", cfg2.Player.DefaultVolume)
	}
}

func TestVolumeNegative(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "[player]\ndefault_volume = -1\n")
	_, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err == nil {
		t.Fatal("expected error for negative volume")
	}
}

func TestKeyBindingsEmpty(t *testing.T) {
	var kb KeyBindings
	if kb.Matches("j") {
		t.Error("empty KeyBindings should not match anything")
	}
}

func TestLibraryPaths_MixedValidity(t *testing.T) {
	dir := t.TempDir()
	musicDir := filepath.Join(dir, "music")
	if err := os.MkdirAll(musicDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, dir, "[library]\npaths = [\""+musicDir+"\", \"/nonexistent_path_xyz\"]\n")
	cfg, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Library.Paths) != 1 || cfg.Library.Paths[0] != musicDir {
		t.Errorf("expected only valid path, got %v", cfg.Library.Paths)
	}
}

func TestIsValidBindingKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{"j", true},
		{"down", true},
		{"ctrl+d", true},
		{"alt+1", true},
		{"shift+home", true},
		{"ctrl+alt+x", false},
		{"ctrl+{", true},
		{"", false},
		{"ctrl+€", false},
		{"ctrl+ctrl+d", false},
		{"not_a_valid_key!", false},
	}
	for _, tt := range tests {
		got := isValidBindingKey(tt.key)
		if got != tt.valid {
			t.Errorf("isValidBindingKey(%q) = %v, want %v", tt.key, got, tt.valid)
		}
	}
}

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if got := expandTilde("~"); got != home {
		t.Errorf("expandTilde('~') = %q, want %q", got, home)
	}
	if got := expandTilde("~/Music"); got != filepath.Join(home, "Music") {
		t.Errorf("expandTilde('~/Music') = %q, want %q", got, filepath.Join(home, "Music"))
	}
	if got := expandTilde("/absolute/path"); got != "/absolute/path" {
		t.Errorf("expandTilde('/absolute/path') = %q, want %q", got, "/absolute/path")
	}
	if got := expandTilde(""); got != "" {
		t.Errorf("expandTilde('') = %q, want ''", got)
	}
}

func TestInterpolate(t *testing.T) {
	t.Setenv("TEST_VAR", "hello")
	if got, err := interpolate("${TEST_VAR}"); err != nil || got != "hello" {
		t.Errorf("interpolate('${TEST_VAR}') = %q, %v, want 'hello', nil", got, err)
	}
	t.Setenv("TEST_VAR2", "world")
	if got, err := interpolate("${TEST_VAR} ${TEST_VAR2}"); err != nil || got != "hello world" {
		t.Errorf("interpolate two vars = %q, %v, want 'hello world', nil", got, err)
	}
	if got, err := interpolate("static"); err != nil || got != "static" {
		t.Errorf("interpolate('static') = %q, %v, want 'static', nil", got, err)
	}
	if got, err := interpolate("${TEST_VAR}-suffix"); err != nil || got != "hello-suffix" {
		t.Errorf("interpolate('${TEST_VAR}-suffix') = %q, %v", got, err)
	}
	if got, err := interpolate("prefix-${TEST_VAR}"); err != nil || got != "prefix-hello" {
		t.Errorf("interpolate('prefix-${TEST_VAR}') = %q, %v", got, err)
	}
}

func TestInterpolateUnset(t *testing.T) {
	_, err := interpolate("${UNSET_VAR}")
	if err == nil {
		t.Fatal("expected error for unset var")
	}
}

func TestHexColorRe(t *testing.T) {
	tests := []struct {
		s     string
		match bool
	}{
		{"#000000", true},
		{"#ffffff", true},
		{"#FFAABB", true},
		{"#GGGGGG", false},
		{"#0000000", false},
		{"000000", false},
		{"#", false},
	}
	for _, tt := range tests {
		got := hexColorRe.MatchString(tt.s)
		if got != tt.match {
			t.Errorf("hexColorRe.MatchString(%q) = %v, want %v", tt.s, got, tt.match)
		}
	}
}

func TestNormalizeBrowseMode(t *testing.T) {
	if got := normalizeBrowseMode("artist"); got != "artist" {
		t.Errorf("normalizeBrowseMode('artist') = %q, want 'artist'", got)
	}
	if got := normalizeBrowseMode("playlist"); got != "artist" {
		t.Errorf("normalizeBrowseMode('playlist') = %q, want 'artist'", got)
	}
	if got := normalizeBrowseMode("bogus"); got != "artist" {
		t.Errorf("normalizeBrowseMode('bogus') = %q, want 'artist'", got)
	}
}

func TestIsValidBindingKeyModifierNamedKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{"ctrl+home", true},
		{"alt+end", true},
		{"shift+tab", true},
		{"ctrl+space", true},
		{"alt+enter", true},
		{"shift+delete", true},
		{"ctrl+pgup", true},
		{"ctrl+pgdown", true},
		{"ctrl+insert", true},
		{"alt+backspace", true},
		{"shift+esc", true},
		{"ctrl+up", true},
		{"ctrl+down", true},
		{"ctrl+left", true},
		{"ctrl+right", true},
	}
	for _, tt := range tests {
		got := isValidBindingKey(tt.key)
		if got != tt.valid {
			t.Errorf("isValidBindingKey(%q) = %v, want %v", tt.key, got, tt.valid)
		}
	}
}

func TestLibraryPaths_NonDirectory(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "notadir.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	musicDir := filepath.Join(dir, "music")
	if err := os.MkdirAll(musicDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, dir, "[library]\npaths = [\""+filePath+"\", \""+musicDir+"\"]\n")
	cfg, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Library.Paths) != 1 || cfg.Library.Paths[0] != musicDir {
		t.Errorf("expected only the directory, got %v", cfg.Library.Paths)
	}
}

func TestInterpolateMultipleUnset(t *testing.T) {
	_, err := interpolate("${A}/${B}")
	if err == nil {
		t.Fatal("expected error for unset vars")
	}
}

func TestLogLevel_Invalid(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "[logging]\nlevel = \"trace\"\n")
	_, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
}

func TestEQPreset_Valid(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "[eq.presets.test]\nbands = [0, 1, -1, 2, -2, 3, -3, 4, -4, 5]\n")
	cfg, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatalf("unexpected error for valid EQ preset: %v", err)
	}
	if len(cfg.EQ.Presets["test"].Bands) != 10 {
		t.Errorf("expected 10 bands, got %d", len(cfg.EQ.Presets["test"].Bands))
	}
}

func TestEQPreset_BandBoundaries(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "[eq.presets.test]\nbands = [-12, -12, -12, -12, -12, 12, 12, 12, 12, 12]\n")
	_, err := LoadFrom(filepath.Join(dir, "config.toml"))
	if err != nil {
		t.Fatalf("boundary values -12 and 12 should be valid: %v", err)
	}
}
