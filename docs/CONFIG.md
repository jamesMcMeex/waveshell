# waveshell — Config Reference

> **Status:** Pre-development
> **Scope:** Canonical specification for `config.toml` — every supported key, its type, default value, and validation rules. The `internal/config` package unmarshals this file into the Go structs defined at the end of this document.
> **Last updated:** June 2026

---

## Table of Contents

1. [File Location](#1-file-location)
2. [Environment Variable Interpolation](#2-environment-variable-interpolation)
3. [library](#3-library)
4. [player](#4-player)
5. [ui](#5-ui)
6. [themes](#6-themes)
7. [keybindings](#7-keybindings)
8. [logging](#8-logging)
9. [Post-MVP Sections](#9-post-mvp-sections)
10. [Validation Rules](#10-validation-rules)
11. [Go Struct Reference](#11-go-struct-reference)
12. [Complete Annotated Example](#12-complete-annotated-example)

---

## 1. File Location

```
~/.config/waveshell/config.toml
```

The path is resolved via `os.UserConfigDir()`, which returns `$XDG_CONFIG_HOME` if set, otherwise `~/.config` on Linux and `~/Library/Application Support` on macOS. The config directory is created on first launch if it does not exist.

**Missing config file is not an error.** The application starts with all defaults applied. A missing `[library]` section means the app opens with an empty browser and a status bar message prompting the user to configure a library path.

**Malformed TOML is a hard error.** The app exits immediately with a message indicating the line and column of the parse failure.

---

## 2. Environment Variable Interpolation

String values in `config.toml` may reference environment variables using `${VAR}` syntax. Interpolation is performed after TOML parsing, before validation, on all string and string-array fields. The variable name must consist of ASCII letters, digits, and underscores only.

```toml
[library]
paths = ["${HOME}/Music", "${MUSIC_DIR}"]
```

Referencing an unset variable is a **hard error at startup**, not a silent empty string. This prevents subtle misconfiguration where a missing variable causes a path to be treated as a literal `${VAR}` string.

Interpolation is not recursive. `${${INNER}}` is not supported and will fail validation.

---

## 3. library

```toml
[library]
paths          = []        # required: at least one path before scanning is possible
scan_on_startup = true     # re-scan all paths on every launch
```

| Key               | Type       | Default | Required                            |
| ----------------- | ---------- | ------- | ----------------------------------- |
| `paths`           | `[]string` | `[]`    | No (but app is read-only until set) |
| `scan_on_startup` | `bool`     | `true`  | No                                  |

**`paths`** is an array of absolute directory paths. Each path is validated on startup: if it does not exist or is not a directory, the app logs the error and skips that path (non-fatal). If _all_ paths fail validation, the app opens with an empty browser.

Paths may use `${VAR}` interpolation. Relative paths are not supported — the config loader returns an error for any path that does not begin with `/` or `~` (after interpolation, `~` is expanded to the user home directory).

**`scan_on_startup`** controls whether a full incremental rescan is triggered on every launch. When `true` (the default), the scanner checks `mtime` for all known files and discovers new/deleted files. When `false`, the existing SQLite index is used as-is. The user can always trigger a manual rescan regardless of this setting (keybinding TBD, not yet specified in interaction design).

---

## 4. player

```toml
[player]
mpv_socket           = "/tmp/waveshell.sock"
default_volume       = 100
default_playback_mode = "stop_at_end"
```

| Key                     | Type     | Default                 | Valid values             |
| ----------------------- | -------- | ----------------------- | ------------------------ |
| `mpv_socket`            | `string` | `"/tmp/waveshell.sock"` | Any writable socket path |
| `default_volume`        | `int`    | `100`                   | `0`–`100`                |
| `default_playback_mode` | `string` | `"stop_at_end"`         | See below                |

**`mpv_socket`** is the Unix domain socket path passed to mpv via `--input-ipc-server`. Supports `${VAR}` interpolation. If the path's parent directory does not exist, the app will fail to launch mpv with a clear error message.

**`default_volume`** is the volume level applied at startup. This is the initial state; the user's last-used volume is not persisted between sessions.

**`default_playback_mode`** is the mode applied at startup. Valid values and their cycle order (when the user presses `m`):

| Value            | Display name           | Behaviour                                                   |
| ---------------- | ---------------------- | ----------------------------------------------------------- |
| `"stop_at_end"`  | _(no indicator shown)_ | Stop when the queue is exhausted. This is the default.      |
| `"repeat_queue"` | `Repeat Queue`         | Loop the queue indefinitely.                                |
| `"repeat_track"` | `Repeat Track`         | Repeat the current track indefinitely.                      |
| `"shuffle"`      | `Shuffle`              | Play queue tracks in random order; reshuffles on each loop. |

Cycle order: `stop_at_end` → `repeat_queue` → `repeat_track` → `shuffle` → `stop_at_end`.

---

## 5. ui

### 5.1 Top-level UI keys

```toml
[ui]
theme              = "slate"
default_browse_mode = "artist"
```

| Key                   | Type     | Default    | Valid values                                                          |
| --------------------- | -------- | ---------- | --------------------------------------------------------------------- |
| `theme`               | `string` | `"slate"`  | `"gameboy"` `"phosphor"` `"amber"` `"slate"` or any key in `[themes]` |
| `default_browse_mode` | `string` | `"artist"` | `"artist"` `"label"` `"genre"` `"year"` `"grouping"`                  |

`"playlist"` is not a valid value for `default_browse_mode` in MVP. It will be accepted without error (to avoid migration burden) but silently treated as `"artist"`.

### 5.2 Built-in theme presets

| Preset     | `bg`      | `fg`      | `accent`  | `muted`   | Character                   |
| ---------- | --------- | --------- | --------- | --------- | --------------------------- |
| `gameboy`  | `#0f380f` | `#9bbc0f` | `#306230` | `#8bac0f` | Four-green Game Boy palette |
| `phosphor` | `#0a0a0a` | `#33ff33` | `#00ff00` | `#1a6b1a` | Green phosphor CRT          |
| `amber`    | `#0d0900` | `#ffb000` | `#ff8c00` | `#7a5200` | Amber phosphor CRT          |
| `slate`    | `#1e2030` | `#c8d3f5` | `#82aaff` | `#545c7e` | Dark blue-grey; the default |

### 5.3 Column configuration

Each browse mode has an independent list of visible columns for the Tracks pane. The config keys mirror the mode names.

```toml
[ui.columns.artist]
visible = ["track_number", "title", "format", "sample_rate", "bit_depth", "duration"]

[ui.columns.label]
visible = ["track_number", "title", "artist", "format", "duration"]

[ui.columns.genre]
visible = ["track_number", "title", "artist", "format", "duration"]

[ui.columns.year]
visible = ["track_number", "title", "artist", "format", "duration"]

[ui.columns.grouping]
visible = ["track_number", "title", "artist", "format", "duration"]
```

The playlist mode column config is defined here for forward compatibility but is not used in MVP:

```toml
[ui.columns.playlist]
visible = ["playlist_position", "title", "artist", "album", "duration", "format"]
```

**Defaults** — if a mode's column config is absent from `config.toml`, the defaults above are applied in code. The config loader must not require all mode sections to be present.

**Valid column identifiers** — any value not in this list is a validation error at startup:

`track_number`, `playlist_position`, `album_track`, `title`, `artist`, `album`, `year`, `genre`, `label`, `grouping`, `format`, `codec`, `sample_rate`, `bit_depth`, `bitrate`, `duration`, `file_size`, `date_added`

**Minimum column constraint** — the `visible` array must contain at least one entry. An empty array is a validation error.

**Unsaved session changes** — column visibility and order changes made at runtime via the Column Manager (`C`) are written back to `config.toml` immediately. The in-memory `Config` struct is updated first; the file is written asynchronously via a debounced `tea.Cmd`. This means `config.toml` always reflects the last session's column state on next launch.

---

## 6. themes

User-defined theme presets. Any number of named themes may be defined. The name is the key used in `[ui] theme`.

```toml
[themes.my_dark]
bg     = "#1a1a2e"
fg     = "#e0e0e0"
accent = "#7c3aed"
muted  = "#4a4a6a"

[themes.high_contrast]
bg     = "#000000"
fg     = "#ffffff"
accent = "#ffff00"
muted  = "#888888"
```

| Key      | Type     | Required | Format                            |
| -------- | -------- | -------- | --------------------------------- |
| `bg`     | `string` | Yes      | 7-character hex colour: `#rrggbb` |
| `fg`     | `string` | Yes      | 7-character hex colour: `#rrggbb` |
| `accent` | `string` | Yes      | 7-character hex colour: `#rrggbb` |
| `muted`  | `string` | Yes      | 7-character hex colour: `#rrggbb` |

All four keys are required for a custom theme. A theme with any key missing is a validation error. The hex string must be exactly 7 characters (`#` plus 6 hex digits); shorthand (`#rgb`) is not supported.

A custom theme name that shadows a built-in preset name (`gameboy`, `phosphor`, `amber`, `slate`) overrides it without error.

---

## 7. keybindings

All keybindings are expressed as string arrays. Each action may have one or more key strings. An empty array disables the action entirely (no validation error — the user may intentionally unbind).

```toml
[keybindings]
# Navigation
cursor_down        = ["j", "down"]
cursor_up          = ["k", "up"]
cursor_left        = ["h", "left"]
cursor_right       = ["l", "right"]
page_down          = ["ctrl+d"]
page_up            = ["ctrl+u"]
jump_to_top        = ["t", "home"]
jump_to_bottom     = ["g", "end"]
focus_next_pane    = ["tab", "l"]
focus_prev_pane    = ["shift+tab", "j"]

# Library actions
action_menu        = ["enter"]
queue_add          = ["space"]
queue_add_next     = ["shift+space"]
queue_add_album    = ["a"]
select_toggle      = ["space"]      # same as queue_add; context-sensitive in Tracks pane
select_all         = ["V"]

# Overlays
open_search        = ["/"]
open_queue         = ["q"]
open_info          = ["i"]
cycle_theme        = ["c"]
open_help          = ["h"]
open_browse_picker = ["b"]
open_columns       = ["C"]
open_sort_picker   = ["S"]
edit_tags          = ["e"]
dismiss            = ["esc"]

# Playback (always active)
play_pause         = ["p"]
next_track         = ["n"]
prev_track         = ["b"]
seek_forward_5     = ["]"]
seek_back_5        = ["["]
seek_forward_30    = ["}"]
seek_back_30       = ["{"]
volume_down        = ["-"]
volume_up          = ["="]
volume_reset       = ["0"]
cycle_play_mode    = ["m"]

# Queue overlay
queue_move_down    = ["J"]
queue_move_up      = ["K"]
queue_remove       = ["x"]
queue_clear        = ["c"]

# Column manager / sort picker
item_move_down     = ["J"]
item_move_up       = ["K"]
item_toggle        = ["space"]

# Application
quit               = ["ctrl+c"]
```

### Key string format

| Format            | Example                                                                                                         | Meaning                             |
| ----------------- | --------------------------------------------------------------------------------------------------------------- | ----------------------------------- |
| Single character  | `"j"`                                                                                                           | The literal character j             |
| Named special key | `"space"`, `"tab"`, `"enter"`, `"esc"`, `"backspace"`, `"home"`, `"end"`, `"up"`, `"down"`, `"left"`, `"right"` | Special keys by name                |
| Modifier + key    | `"ctrl+c"`, `"ctrl+d"`, `"shift+tab"`, `"shift+space"`                                                          | Modifier prefix, lowercase          |
| Uppercase letter  | `"G"`, `"J"`, `"K"`, `"V"`                                                                                      | Distinct from lowercase counterpart |

The key strings map directly to BubbleTea's `tea.KeyType` and `tea.KeyMsg.String()` values. Any string that does not match a known key name or pattern is a validation error at startup.

### Context sensitivity

Several action names are context-sensitive: they route to different handlers depending on `ActiveOverlay`, `ActiveDialog`, and `ActivePane`. For example, `space` is bound to both `queue_add` and `select_toggle`, but only one fires depending on context. This is resolved in the `Update` function, not in the config layer. The config layer only validates that key strings are well-formed.

### Conflicts

The config loader does not validate for keybinding conflicts (the same key string bound to two different actions in the same context). Conflicts are the user's responsibility. If two actions share a key in the same context, the one matched first in the `Update` routing switch wins.

---

## 8. logging

```toml
[logging]
level = "info"
path  = ""
```

| Key     | Type     | Default  | Valid values                                    |
| ------- | -------- | -------- | ----------------------------------------------- |
| `level` | `string` | `"info"` | `"debug"` `"info"` `"warn"` `"error"`           |
| `path`  | `string` | `""`     | Any writable file path, or `""` for the default |

**`path`** defaults to `~/.config/waveshell/waveshell.log` when empty. Supports `${VAR}` interpolation. If the parent directory does not exist, the logger falls back to stderr with a warning — a logging misconfiguration must not prevent the app from starting.

**`level`** maps directly to `slog.Level`. At `"debug"`, every mpv IPC command and response is logged; this produces significant volume during playback and is intended for development only.

---

## 9. Post-MVP Sections

These sections are parsed and validated by the config loader but have no effect in MVP. They are defined here so the TOML format is stable before the features land.

### 9.1 EQ

```toml
[eq]
enabled        = false
default_preset = "flat"

[eq.presets.flat]
bands = [0, 0, 0, 0, 0, 0, 0, 0, 0, 0]

[eq.presets.electronic]
# 10 bands: 31Hz, 63Hz, 125Hz, 250Hz, 500Hz, 1kHz, 2kHz, 4kHz, 8kHz, 16kHz
bands = [4, 3, 0, -2, -3, 0, 2, 3, 4, 4]
```

| Key                       | Type     | Default  |
| ------------------------- | -------- | -------- |
| `eq.enabled`              | `bool`   | `false`  |
| `eq.default_preset`       | `string` | `"flat"` |
| `eq.presets.<name>.bands` | `[]int`  | —        |

Each `bands` array must contain exactly 10 integer values in the range `−12` to `+12` (dB). Any other length or value range is a validation error.

Built-in presets (`flat`, `rock`, `electronic`, `jazz`, `classical`) are defined in code and cannot be overridden via config. User presets in `[eq.presets]` are additive.

### 9.2 Script hooks

```toml
[hooks]
on_track_change    = ""
on_playback_start  = ""
on_playback_stop   = ""
```

Each value is a shell command string executed via `sh -c`. An empty string means the hook is disabled. The following environment variables are set when the hook fires:

| Variable         | Content                  |
| ---------------- | ------------------------ |
| `WS_TITLE`       | Track title              |
| `WS_ARTIST`      | Track artist             |
| `WS_ALBUM`       | Track album              |
| `WS_PATH`        | Full file path           |
| `WS_DURATION_MS` | Duration in milliseconds |

Hooks run asynchronously and their exit codes are ignored. A hook that exceeds 5 seconds is killed silently.

### 9.3 Scrobbling

```toml
[scrobbling]
enabled  = false
username = ""
```

| Key                   | Type     | Default | Notes                                 |
| --------------------- | -------- | ------- | ------------------------------------- |
| `scrobbling.enabled`  | `bool`   | `false` | Scrobbling only activates when `true` |
| `scrobbling.username` | `string` | `""`    | Last.fm username                      |

The API session key is stored in the OS keychain via `zalando/go-keyring`, never in `config.toml`. A `username` with no corresponding keychain entry causes a non-fatal error logged at `warn` level; scrobbling is silently disabled for that session.

---

## 10. Validation Rules

The config loader must enforce these rules and return typed errors (not log-and-continue) for any violation. The BubbleTea `Init` function receives a config-load result message and renders a blocking error dialog for hard errors.

| Rule                                                   | Severity | Behaviour                             |
| ------------------------------------------------------ | -------- | ------------------------------------- |
| Malformed TOML                                         | Hard     | Exit with parse error message         |
| Unrecognised top-level table key                       | Soft     | Log at `warn`, continue               |
| `${VAR}` references an unset env variable              | Hard     | Exit with message naming the variable |
| Library path does not exist or is not a directory      | Soft     | Log at `warn`, skip that path         |
| All library paths invalid                              | Soft     | Start with empty browser, show prompt |
| `default_volume` outside 0–100                         | Hard     | Exit with message                     |
| `default_playback_mode` not in allowed set             | Hard     | Exit with message                     |
| `theme` references unknown preset name                 | Hard     | Exit with message                     |
| Custom theme missing any of `bg`/`fg`/`accent`/`muted` | Hard     | Exit with message                     |
| Hex colour malformed (not `#rrggbb`)                   | Hard     | Exit with message                     |
| Column identifier not in valid set                     | Hard     | Exit with message                     |
| `visible` column array is empty                        | Hard     | Exit with message                     |
| `eq.presets.<name>.bands` not exactly 10 values        | Hard     | Exit with message                     |
| `eq.presets.<name>.bands` value outside −12 to +12     | Hard     | Exit with message                     |
| Keybinding key string not a recognised key             | Hard     | Exit with message                     |
| `logging.path` parent directory does not exist         | Soft     | Fall back to stderr, log warning      |

"Hard" means the application does not start. "Soft" means the application starts with a degraded or default state, and the issue is logged.

---

## 11. Go Struct Reference

The canonical Go representation of the config. Defined in `internal/config/config.go`. All fields use the `toml:` tag for `BurntSushi/toml` unmarshalling.

```go
type Config struct {
    Library     LibraryConfig            `toml:"library"`
    Player      PlayerConfig             `toml:"player"`
    UI          UIConfig                 `toml:"ui"`
    Themes      map[string]ThemeConfig   `toml:"themes"`
    Keybindings KeybindingsConfig        `toml:"keybindings"`
    Logging     LoggingConfig            `toml:"logging"`
    EQ          EQConfig                 `toml:"eq"`       // post-MVP; parsed but unused in MVP
    Hooks       HooksConfig              `toml:"hooks"`    // post-MVP
    Scrobbling  ScrobblingConfig         `toml:"scrobbling"` // post-MVP
}

type LibraryConfig struct {
    Paths         []string `toml:"paths"`
    ScanOnStartup bool     `toml:"scan_on_startup"`
}

type PlayerConfig struct {
    MPVSocket          string `toml:"mpv_socket"`
    DefaultVolume      int    `toml:"default_volume"`
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

// Theme resolves the active theme from Config.
// It checks Config.Themes[Config.UI.Theme] first, then the built-in presets.
// Returns an error if the name is not found in either source.
func (c Config) Theme() (ThemeConfig, error)

type KeybindingsConfig struct {
    CursorDown      KeyBindings `toml:"cursor_down"`
    CursorUp        KeyBindings `toml:"cursor_up"`
    CursorLeft      KeyBindings `toml:"cursor_left"`
    CursorRight     KeyBindings `toml:"cursor_right"`
    PageDown        KeyBindings `toml:"page_down"`
    PageUp          KeyBindings `toml:"page_up"`
    JumpToTop       KeyBindings `toml:"jump_to_top"`
    JumpToBottom    KeyBindings `toml:"jump_to_bottom"`
    FocusNextPane   KeyBindings `toml:"focus_next_pane"`
    FocusPrevPane   KeyBindings `toml:"focus_prev_pane"`
    ActionMenu      KeyBindings `toml:"action_menu"`
    QueueAdd        KeyBindings `toml:"queue_add"`
    QueueAddNext    KeyBindings `toml:"queue_add_next"`
    QueueAddAlbum   KeyBindings `toml:"queue_add_album"`
    SelectToggle    KeyBindings `toml:"select_toggle"`
    SelectAll       KeyBindings `toml:"select_all"`
    OpenSearch      KeyBindings `toml:"open_search"`
    OpenQueue       KeyBindings `toml:"open_queue"`
    OpenInfo        KeyBindings `toml:"open_info"`
    OpenHelp        KeyBindings `toml:"open_help"`
    OpenBrowsePicker KeyBindings `toml:"open_browse_picker"`
    OpenColumns     KeyBindings `toml:"open_columns"`
    OpenSortPicker  KeyBindings `toml:"open_sort_picker"`
    EditTags        KeyBindings `toml:"edit_tags"`
    Dismiss         KeyBindings `toml:"dismiss"`
    PlayPause       KeyBindings `toml:"play_pause"`
    NextTrack       KeyBindings `toml:"next_track"`
    PrevTrack       KeyBindings `toml:"prev_track"`
    SeekForward5    KeyBindings `toml:"seek_forward_5"`
    SeekBack5       KeyBindings `toml:"seek_back_5"`
    SeekForward30   KeyBindings `toml:"seek_forward_30"`
    SeekBack30      KeyBindings `toml:"seek_back_30"`
    VolumeDown      KeyBindings `toml:"volume_down"`
    VolumeUp        KeyBindings `toml:"volume_up"`
    VolumeReset     KeyBindings `toml:"volume_reset"`
    CyclePlayMode   KeyBindings `toml:"cycle_play_mode"`
    QueueMoveDown   KeyBindings `toml:"queue_move_down"`
    QueueMoveUp     KeyBindings `toml:"queue_move_up"`
    QueueRemove     KeyBindings `toml:"queue_remove"`
    QueueClear      KeyBindings `toml:"queue_clear"`
    ItemMoveDown    KeyBindings `toml:"item_move_down"`
    ItemMoveUp      KeyBindings `toml:"item_move_up"`
    ItemToggle      KeyBindings `toml:"item_toggle"`
    Quit            KeyBindings `toml:"quit"`
}

// KeyBindings is a named type for keybinding slices, enabling the Matches method.
type KeyBindings []string

// Matches reports whether the given tea.KeyMsg string matches any binding for this action.
// Used in Update: if cfg.Keybindings.CursorDown.Matches(msg.String()) { ... }
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
    Enabled       bool                    `toml:"enabled"`
    DefaultPreset string                  `toml:"default_preset"`
    Presets       map[string]EQPreset     `toml:"presets"`
}

type EQPreset struct {
    Bands [10]int `toml:"bands"`
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
```

### The `Load` function signature

```go
// Load reads and validates the config file at the XDG-compliant path.
// If the file does not exist, it returns a Config with all defaults applied and a nil error.
// If the file exists but is invalid, it returns a non-nil error describing the first failure.
// Env var interpolation is applied before validation.
func Load() (Config, error)

// LoadFrom reads from a specific path. Used in tests via t.TempDir().
func LoadFrom(path string) (Config, error)

// Default returns a Config with all fields set to their documented default values.
// Called by Load when no config file is present, and used as the merge base when
// the config file is present but omits optional fields.
func Default() Config
```

The separation between `Load` and `LoadFrom` is what makes `internal/config` testable with `t.TempDir()` without any mocking.

---

## 12. Complete Annotated Example

A full `config.toml` showing all supported keys with their defaults. This file can be copied to `~/.config/waveshell/config.toml` as a starting point.

```toml
# waveshell configuration
# Location: ~/.config/waveshell/config.toml
# All keys are optional unless noted. Omitted keys use the defaults shown here.

# ── Library ───────────────────────────────────────────────────────────────────

[library]
# At least one path is required for library scanning to function.
# ${HOME} and other environment variables are supported in string values.
paths = ["${HOME}/Music"]

# Check mtime for all files on every launch (recommended).
# Set to false only if the library is very large and startup time is a concern.
scan_on_startup = true


# ── Player ────────────────────────────────────────────────────────────────────

[player]
# Unix domain socket used to communicate with mpv.
mpv_socket = "/tmp/waveshell.sock"

# Volume at startup (0–100).
default_volume = 100

# Playback mode at startup.
# Options: "stop_at_end" | "repeat_queue" | "repeat_track" | "shuffle"
default_playback_mode = "stop_at_end"


# ── UI ────────────────────────────────────────────────────────────────────────

[ui]
# Active colour theme. Built-in: "gameboy" | "phosphor" | "amber" | "slate"
# Custom themes are defined in [themes.<name>] below.
theme = "slate"

# Browse mode shown at startup.
# Options: "artist" | "label" | "genre" | "year"
default_browse_mode = "artist"

# Columns shown in the Tracks pane, per browse mode.
# Changes made via the Column Manager (C) are written back here automatically.
[ui.columns.artist]
visible = ["track_number", "title", "format", "sample_rate", "bit_depth", "duration"]

[ui.columns.label]
visible = ["track_number", "title", "artist", "format", "duration"]

[ui.columns.genre]
visible = ["track_number", "title", "artist", "format", "duration"]

[ui.columns.year]
visible = ["track_number", "title", "artist", "format", "duration"]

[ui.columns.grouping]
visible = ["track_number", "title", "artist", "format", "duration"]


# ── Themes ────────────────────────────────────────────────────────────────────

# Custom theme example. Set [ui] theme = "my_theme" to activate.
# All four colour roles are required. Values must be 7-character hex (#rrggbb).

# [themes.my_theme]
# bg     = "#1a1a2e"
# fg     = "#e0e0e0"
# accent = "#7c3aed"
# muted  = "#4a4a6a"


# ── Keybindings ───────────────────────────────────────────────────────────────

# Each action accepts an array of key strings. An empty array disables the action.
# Omitting a key uses the default shown below.

[keybindings]
# Navigation
cursor_down        = ["k", "down"]
cursor_up          = ["i", "up"]
cursor_left        = ["j", "left"]
cursor_right       = ["l", "right"]
page_down          = ["ctrl+d"]
page_up            = ["ctrl+u"]
jump_to_top        = ["t", "home"]
jump_to_bottom     = ["g", "end"]
focus_next_pane    = ["tab"]
focus_prev_pane    = ["shift+tab"]

# Library actions
action_menu        = ["enter"]
queue_add          = ["space"]
queue_add_next     = ["shift+space"]
queue_add_album    = ["a"]
select_toggle      = ["space"]
select_all         = ["V"]

# Overlays & dialogs
open_search        = ["/"]
open_queue         = ["q"]
open_info          = ["i"]
cycle_theme        = ["c"]
open_help          = ["h"]
open_browse_picker = ["b"]
open_columns       = ["C"]
open_sort_picker   = ["S"]
edit_tags          = ["e"]
dismiss            = ["esc"]

# Playback (always active, regardless of overlay state)
play_pause         = ["p"]
next_track         = ["n"]
prev_track         = ["b"]
seek_forward_5     = ["]"]
seek_back_5        = ["["]
seek_forward_30    = ["}"]
seek_back_30       = ["{"]
volume_down        = ["-"]
volume_up          = ["="]
volume_reset       = ["0"]
cycle_play_mode    = ["m"]

# Queue overlay
queue_move_down    = ["J"]
queue_move_up      = ["K"]
queue_remove       = ["x"]
queue_clear        = ["c"]

# Column manager / sort picker
item_move_down     = ["J"]
item_move_up       = ["K"]
item_toggle        = ["space"]

# Application
quit               = ["ctrl+c"]


# ── Logging ───────────────────────────────────────────────────────────────────

[logging]
# Log level: "debug" | "info" | "warn" | "error"
level = "info"

# Log file path. Empty string uses the default: ~/.config/waveshell/waveshell.log
path = ""


# ── Post-MVP (parsed but inactive in MVP) ────────────────────────────────────

# [eq]
# enabled        = false
# default_preset = "flat"
#
# [eq.presets.electronic]
# bands = [4, 3, 0, -2, -3, 0, 2, 3, 4, 4]

# [hooks]
# on_track_change   = ""
# on_playback_start = ""
# on_playback_stop  = ""

# [scrobbling]
# enabled  = false
# username = ""
```
