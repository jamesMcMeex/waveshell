# waveshell — Product Requirements Document

> **Status:** Pre-development · Working title confirmed
> **Stack:** Go 1.22+ · BubbleTea · Elm Architecture
> **Last updated:** June 2026

---

## Table of Contents

1. [Vision](#1-vision)
2. [Design Principles](#2-design-principles)
3. [Tech Stack](#3-tech-stack)
4. [Architecture Overview](#4-architecture-overview)
5. [Security & Privacy Principles](#5-security--privacy-principles)
6. [Data Safety Principles](#6-data-safety-principles)
7. [Scope](#7-scope)
8. [MVP Milestones](#8-mvp-milestones)
9. [Post-MVP Backlog](#9-post-mvp-backlog)
10. [Testing Strategy](#10-testing-strategy)
11. [Metadata Reference](#11-metadata-reference)

---

## 1. Vision

waveshell is a keyboard-driven, privacy-first terminal music player for people who care deeply about their local audio library. It is honest about your audio — surfacing real codec data, loudness measurements, and bit depth rather than abstracting them away. It is safe with your files, never mutating anything silently. It works identically with no internet connection.

**Target user:** A technically proficient listener with a large, well-tagged local library in lossless or high-quality formats, a workflow already built around the terminal, and strong opinions about metadata fidelity.

---

## 2. Design Principles

1. **Keyboard first** — every action reachable without a mouse.
2. **Trust the user** — surface the real data (format, bit depth, sample rate, loudness). Don't hide it.
3. **Never surprise** — no silent writes, no background network calls, no data loss, ever.
4. **Offline always** — identical behaviour with no internet connection. The only optional exception is scrobbling (post-MVP), which requires explicit opt-in.
5. **Testable by design** — the Elm Architecture enforces pure Update and View functions. Business logic has no UI dependency and is trivially unit-testable.
6. **Earn trust with writes** — every mutation to an audio file requires explicit user confirmation. Bulk operations run in dry-run first and show a diff.
7. **The library is the source of truth** — the SQLite index is derived data. If it is deleted, the app rescans. Audio files are never modified without explicit instruction.

---

## 3. Tech Stack

| Concern        | Tool                  | Licence | Notes                                                     |
| -------------- | --------------------- | ------- | --------------------------------------------------------- |
| Language       | Go 1.22+              | —       | Single binary, fast compile, excellent stdlib             |
| TUI framework  | `bubbletea`           | MIT     | Elm Architecture (Model / Msg / Update / View)            |
| Styling        | `lipgloss`            | MIT     | CSS-like layout, colour themes, borders                   |
| Components     | `bubbles`             | MIT     | List, textinput, progress bar, viewport, spinner          |
| Audio playback | mpv (IPC socket)      | GPL     | JSON socket protocol; handles ALAC, AIFF, FLAC, MP3       |
| Metadata       | `dhowden/tag`         | BSD     | ID3v1/v2, MP4/ALAC, FLAC, AIFF tags                       |
| Database       | `modernc.org/sqlite`  | MIT     | Pure Go SQLite — zero CGo, no native dependencies         |
| Config         | `BurntSushi/toml`     | MIT     | Human-readable config; env var interpolation via `${VAR}` |
| Album art      | `chafa` (subprocess)  | LGPL    | Terminal image rendering; auto-detects protocol           |
| Testing        | `testing` + `testify` | MIT     | Table-driven tests, assertions, mocks                     |
| Linting        | `golangci-lint`       | MIT     | Wraps staticcheck, vet, errcheck, and others              |

**Distribution:** `go build` produces a single self-contained binary. Cross-compilation: `GOOS=darwin GOARCH=arm64 go build`. The only runtime dependency is mpv for audio playback.

---

## 4. Architecture Overview

waveshell uses the Elm Architecture as implemented by BubbleTea. For developers coming from React/Redux, the mental model maps directly:

| Redux              | BubbleTea            | Role                                                               |
| ------------------ | -------------------- | ------------------------------------------------------------------ |
| Store              | `Model`              | All application state in one struct                                |
| Action             | `Msg`                | Every possible event: keypresses, mpv events, scan results, errors |
| Reducer            | `Update(msg, model)` | Pure function — takes Msg + Model, returns new Model               |
| Middleware / thunk | `tea.Cmd`            | Side effects: file IO, mpv IPC, database queries                   |
| `render()`         | `View(model)`        | Pure function — Model → string                                     |

```
┌──────────────────────────────────────────────────────┐
│  Model  (one struct — complete application state)    │
│                                                      │
│  type Model struct {                                 │
│    Library    LibraryState                           │
│    Player     PlayerState                            │
│    Queue      QueueState                             │
│    Search     SearchState                            │
│    UI         UIState                                │
│    Config     Config                                 │
│  }                                                   │
└──────────────┬───────────────────────────────────────┘
               │
     ┌─────────▼──────────┐
     │  Msg  (sum type)   │  ← keypresses, mpv events,
     │                    │    scan progress, DB results
     └─────────┬──────────┘
               │
     ┌─────────▼──────────────────────────────────┐
     │  Update(msg Msg, model Model)              │
     │    → (Model, tea.Cmd)                      │
     │                                            │
     │  Pure function. No side effects.           │
     │  All IO happens via returned Cmds.         │
     └─────────┬──────────────────────────────────┘
               │
     ┌─────────▼──────────────────────────────────┐
     │  View(model Model) → string                │
     │                                            │
     │  Pure function. lipgloss for styling.      │
     └────────────────────────────────────────────┘
```

**TDD implication:** all business logic is exercised by calling `Update(msg, model)` and asserting on the returned Model. No TUI instantiation, no subprocess, no goroutine plumbing required in tests.

### UI Layout

```
┌─────────────────────────────────────────────────────────────────┐
│  ARTISTS                │  ALBUMS            │  TRACKS          │
│  ─────────────────────  │  ────────────────  │  ──────────────  │
│  Amon Tobin             │  Foley Room        │  01 Wooden Toy   │
│▶ Aphex Twin             │▶ Isam              │▶ 02 Lost & Safe  │
│  Boards of Canada       │  Out From Out...   │  03 Stealth      │
│  Goldie                 │  Supermodified     │  04 At the End   │
│  LTJ Bukem              │                    │                  │
│  Squarepusher           │                    │                  │
├─────────────────────────────────────────────────────────────────┤
│ ▶  Amon Tobin — Lost & Safe              02:14 ────●──── 04:38  │
│    Isam · ALAC · 44.1kHz · 24bit                     Vol: 80%  │
└─────────────────────────────────────────────────────────────────┘
```

### Theme System

Four named colour roles defined in the config. The constraint enforces visual consistency and enables strong palette identities.

| Role     | Purpose                                 |
| -------- | --------------------------------------- |
| `bg`     | Background                              |
| `fg`     | Primary text                            |
| `accent` | Selection, active track, progress bar   |
| `muted`  | Secondary text, borders, inactive items |

**Built-in presets:** `gameboy`, `phosphor`, `amber`, `slate`. Users may define custom presets in `~/.config/waveshell/config.toml` as hex values.

### Keybinding Conventions

| Key                | Action                            |
| ------------------ | --------------------------------- |
| `i k j l` / arrows | Navigate (up, down, left, right)   |
| `Tab`              | Shift focus between panes         |
| `Enter`            | Play selected                     |
| `Space`            | Add to queue                      |
| `Shift+Space`      | Play next                         |
| `a`                | Add album to queue                |
| `/`                | Open fuzzy search                 |
| `i`                | Open metadata info panel          |
| `e`                | Edit metadata (within info panel) |
| `q`                | Queue view                        |
| `m`                | Cycle playback mode               |
| `h`                | Help / keybinding reference       |
| `Esc`              | Close overlay / cancel            |

All keybindings are configurable via `config.toml`.

---

## 5. Security & Privacy Principles

**Data locality.** The app is entirely offline. No network calls, no analytics, no crash reporting, nothing phoning home. The only optional exception is scrobbling (post-MVP), which requires explicit user opt-in and user-supplied credentials.

**Credentials.** Any future API keys (Last.fm etc.) are stored in the system keychain via `zalando/go-keyring`, never in the config file in plaintext. Config values may reference environment variables using `${VAR}` syntax.

**File permissions.** The library scanner operates read-only by default. Write access (tag editing, file operations) is a separately-gated capability. The app never requests broader permissions than it needs.

**Config and database location.** All app data lives under `~/.config/waveshell/` (XDG-compliant). The user owns and can inspect all of it. No hidden directories, no opaque binary formats.

**No silent network.** `net/http` and socket calls are confined to the future scrobbling package, making the network surface area auditable by inspection.

**Safe default on first run.** Until a library path has been explicitly configured, the app operates in read-only mode. No writes of any kind are possible.

---

## 6. Data Safety Principles

The golden rule: **never mutate source audio files silently.**

**Atomic tag writes.** `dhowden/tag` writes to a `.tmp` file alongside the original, then uses `os.Rename` to move it into place. If the write fails mid-way, the original is untouched. This is the only permitted pattern for tag editing.

**Diff before write.** Any tag edit — single field or bulk — shows a before/after diff in the UI before writing. `Enter` confirms, `Esc` cancels cleanly. Cancellation leaves both the file and the Model unchanged.

**Bulk operations run dry-run first.** Any operation affecting multiple files shows the full set of proposed changes and requires explicit confirmation before a single file is touched.

**Database is a cache.** The SQLite index is treated as derived data. Deleting it causes the app to rescan on next launch. Audio files are the source of truth, not the database.

**WAL mode.** SQLite runs in Write-Ahead Logging mode. A crash during a write cannot corrupt the database.

**File operations (post-MVP).** Moving or renaming files uses a staging step; the original path is preserved in the database until the operation is confirmed complete. Deletion moves to the system trash via the XDG trash specification — never `os.Remove`.

---

## 7. Scope

### Permanently out of scope

- Streaming service integration of any kind
- Cloud sync or remote library access
- Telemetry, analytics, or crash reporting
- Any background network activity

### Out of scope for MVP — revisit post-v1

- Smart playlists and filter expressions
- Scrobbling (Last.fm)
- File and folder management (move, rename, delete)
- EQ and audio processing
- Spectrum visualiser
- Synced lyrics
- Daemon mode and IPC remote control
- Plugin or hook system

---

## 8. MVP Milestones

---

### Milestone 1 — Project Foundation

**Goal:** a compiling, linting, tested Go project skeleton. No UI.

**Deliverables:**

- `go.mod` with all dependencies pinned to specific versions
- `.golangci.yml` linter configuration
- `Makefile` with targets: `make test`, `make lint`, `make build`, `make run`, `make coverage`
- Config loader: reads `~/.config/waveshell/config.toml`, unmarshals to a typed Go struct, validates that all configured library paths exist on disk
- `${VAR}` environment variable interpolation in config string values
- XDG-compliant config path resolution (macOS + Linux)
- Structured logging to `~/.config/waveshell/waveshell.log` using stdlib `slog`
- README scaffold with build instructions
  **Test coverage targets:** config loading happy path, missing config file, malformed TOML, invalid library path, env var interpolation — all table-driven.

**Go concepts introduced:** structs, interfaces, `os.UserConfigDir()`, `testing.T`, table-driven test patterns, `t.TempDir()`.

---

### Milestone 2 — Library Scanner & SQLite Index

**Goal:** scan a directory tree, build a SQLite index, query it. Still no UI.

**Deliverables:**

- Recursive directory walker using `filepath.WalkDir` — discovers audio files by extension: `.flac`, `.alac`, `.m4a`, `.aiff`, `.aif`, `.mp3`, `.wav`, `.ogg`
- `dhowden/tag` metadata extraction per track:
  - Title, artist, album artist, album, track number, disc number, year, genre
  - Duration, format, codec, container
  - Sample rate, bit depth, bitrate (actual)
  - File size, last modified timestamp
  - Grouping (`TIT1` / `©grp`), Publisher/Label (`TPUB` / `LABEL`)
  - ReplayGain track gain + peak, album gain + peak
  - R128 integrated loudness (`R128_TRACK_GAIN`) if present
  - Embedded artwork: present/absent, dimensions, format (JPEG/PNG), size in bytes
- SQLite schema: `tracks`, `albums`, `artists` tables; both `grouping` and `label` indexed as separate columns
- WAL mode enabled at database initialisation
- Incremental rescan: only reprocesses files whose `mtime` has changed since last scan
- Scanner implemented as a `tea.Cmd` that emits `ScanProgress`, `ScanComplete`, and `ScanError` messages to the Update loop
- Direct-launch support: `waveshell ~/Music/Artist/Album` resolves the path, finds matching tracks, and queues them for immediate playback without entering the library browser
  **Test coverage targets:** scanner finds correct files and skips non-audio, metadata extracted accurately per format using fixture files, incremental scan respects mtime, corrupt or unreadable files are handled gracefully (error message, no crash), WAL mode confirmed, both grouping and label fields read correctly.

**Go concepts introduced:** `filepath.WalkDir`, goroutines and channels, `tea.Cmd` as the command pattern, `modernc.org/sqlite` basics.

---

### Milestone 3 — TUI Shell & Library Browser

**Goal:** a navigable three-pane browser. No playback yet.

**Deliverables:**

- BubbleTea app scaffold with `Model`, `Init`, `Update`, `View` wired up
- `UIState` in the Model tracks: active pane (Artist / Album / Track), cursor position per pane, scroll offset per pane
- Three-pane layout via `lipgloss` — Artists → Albums → Tracks — resizes correctly when the terminal is resized
- `hjkl` and arrow key navigation; `Tab` shifts focus between panes; selection in left panes filters the pane to the right
- Track list columns: track number, title, duration, format badge — configurable via `columns` in `config.toml`
- Format badge display: `ALAC 44.1k 24bit`, `FLAC 96k 24bit`, `MP3 320`, etc.
- Status bar at the bottom — static placeholder for now
- 4-colour theme system: `bg`, `fg`, `accent`, `muted` as named `lipgloss.Color` variables in a `Theme` struct; built-in presets: `gameboy`, `phosphor`, `amber`, `slate`; user-defined presets in config
- Help overlay: `?` opens a keybinding reference; `Esc` closes it
- Artist list: pressing a letter key (`A`–`Z`) jumps to the first artist starting with that letter
  **Test coverage targets:** `Update` with navigation `Msg`s produces correct cursor positions and scroll offsets; pane focus transitions; theme colours resolve correctly per preset; format badge string is correct for each file format; letter-jump positions cursor on the correct artist.

**Go concepts introduced:** Elm Architecture fully exercised for the first time, `lipgloss` layout system, `tea.KeyMsg` pattern matching.

---

### Milestone 4 — Playback Engine

**Goal:** play music.

**Deliverables:**

- `mpv` IPC wrapper in its own `internal/mpv` package: launches mpv as a subprocess with `--input-ipc-server=/tmp/waveshell.sock`, communicates over a Unix domain socket using mpv's JSON IPC protocol
- Commands: `Play`, `Pause`, `Stop`, `Next`, `Prev`, `Seek` (relative and absolute), `SetVolume`
- Event listener runs as a goroutine, feeds typed `Msg`s back to the BubbleTea event loop via a `tea.Cmd` subscription
- `PlayerState` in Model: `Playing | Paused | Stopped`, current track, elapsed time, total duration, volume level
- Now Playing bar in View: track title, artist, album, elapsed / total time, Unicode progress bar, volume indicator
- Format badge in Now Playing bar: `ALAC 44.1k 24bit`, `FLAC 96k 24bit`, `MP3 320`, etc.
- Gapless playback using mpv's native playlist handling
- Error surface: mpv not installed, socket connection timeout, and unexpected process exit each produce a typed error `Msg` that renders an actionable message in the UI
- Direct-launch path from Milestone 2: if launched with a path argument, the app skips the library browser, builds a track list from the path, and begins playback immediately
  **Test coverage targets:** IPC wrapper unit-tested against a mock Unix socket server (no real mpv required); JSON command serialisation round-trip; mpv event parsing; `PlayerState` machine transitions via `Update` calls; mpv-not-installed error message surfaces correctly.

**Go concepts introduced:** Unix domain sockets, subprocess management (`os/exec`), goroutine + channel patterns for event subscription.

---

### Milestone 5 — Queue Management

**Goal:** deliberate control over what plays.

**Deliverables:**

- `QueueState` in Model: `[]Track`, `currentIndex int`, `PlaybackMode`
- `Space` adds the selected track to the end of the queue; `Shift+Space` inserts as play-next; `a` adds the entire selected album in track order
- Queue view: `q` opens an overlay using `bubbles/viewport`, showing all upcoming tracks with the current track highlighted
- Reorder within the queue: `J` / `K` moves the selected item down / up; `x` removes it
- Clear queue command
- Playback modes: `StopAtEnd`, `RepeatTrack`, `RepeatQueue`, `Shuffle` — cycled with `m`
- Current playback mode displayed in the status bar
- Play history: every completed track (played to at least 50% of its duration) written to a `play_history` table in SQLite with timestamp
  **Test coverage targets:** queue CRUD operations via `Update`; reorder logic; `RepeatQueue` wraps correctly at end of queue; `Shuffle` produces a valid permutation of the correct length (with mocked `rand`); empty queue edge cases; `Update` emits the correct `Cmd` to trigger next-track playback.

---

### Milestone 6 — Search

**Goal:** find anything in the library instantly.

**Deliverables:**

- `/` opens a search overlay; `Esc` closes it; `Enter` confirms the selected result
- `bubbles/textinput` component for the search input field
- Fuzzy matching across artists, albums, and track titles simultaneously using `sahilm/fuzzy` (MIT)
- Results grouped by type — Artists / Albums / Tracks — ranked by relevance score within each group
- `Tab` within results toggles between two actions: "jump to in library browser" and "add to queue"
- Case-insensitive matching; diacritic-tolerant via `golang.org/x/text/unicode/norm`
- Filter by label / grouping field inline: `label:Metalheadz` or `grouping:Metalheadz` syntax in the search input
  **Test coverage targets:** fuzzy scorer returns expected rankings for known inputs; diacritic normalisation (e.g. "Bjork" matches "Björk"); empty query returns no results; `Update` transitions Model state correctly through search open / type / confirm / close; performance test with 10,000 mock tracks completes in under 50ms.

---

### Milestone 7 — Metadata Panel & Tag Editing

**Goal:** see and safely edit the real data.

**Deliverables:**

**Info panel (`i` to open):** renders as a `bubbles/viewport` overlay with the following sections:

_Basic metadata:_ title, artist, album artist, album, track number, disc number, year, genre, grouping, label/publisher.

_Audio technical data:_

- Format, codec, container format
- Sample rate, bit depth, bitrate (actual decoded value, not nominal)
- File size (bytes and human-readable), precise duration
- Gapless playback tags (iTunSMPB / gapless data) if present
  _Loudness data:_
- ReplayGain track gain and peak
- ReplayGain album gain and peak
- R128 integrated loudness (`R128_TRACK_GAIN`) if present
  _Embedded artwork:_
- Present / absent
- Pixel dimensions
- Format (JPEG / PNG)
- Size in kilobytes
  _File information:_
- Full file path
- Last modified timestamp
- All raw tag fields in a scrollable section
  **Tag editing (`e` from within the info panel):**
- Fields become editable via `bubbles/textinput` components
- Unsaved changes highlighted; a diff showing before → after values is displayed at the bottom of the panel
- `Enter` confirms and writes; `Esc` cancels — cancellation leaves both the file and the Model completely unchanged
- Write path: write to a `.tmp` file alongside the original, then `os.Rename` — atomic on POSIX systems
- Any error during the write operation surfaces as a typed error `Msg`; the original file is verified untouched before the error message is shown
  **Configurable columns (track list):** the columns displayed in the track list pane are user-configurable in `config.toml`:

```toml
[ui]
columns = ["track_number", "title", "format", "sample_rate", "bit_depth", "duration"]
```

Available column identifiers: `track_number`, `title`, `artist`, `album`, `year`, `genre`, `format`, `codec`, `sample_rate`, `bit_depth`, `bitrate`, `duration`, `file_size`, `label`, `grouping`.

**Test coverage targets:** metadata reads correctly for each supported format using committed fixture files (FLAC, ALAC, AIFF, MP3); diff generates correct before/after for changed fields; atomic write sequence verified (temp file exists during write, is absent after, original replaced); cancellation produces no file change; write error leaves original intact and untouched; R128 tag read correctly from fixture; both `grouping` and `label` fields read and displayed distinctly.

**Test fixture files:** a set of silent 5-second audio files in each supported format, with known metadata, generated once with ffmpeg and committed to the repository under `testdata/fixtures/`. Reused across all format-specific tests in Milestones 2, 4, and 7.

---

## 9. Post-MVP Backlog

Items are listed in approximate priority order within each category.

### Playback & Audio

| Feature               | Notes                                                                                                                        |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| 10-band graphic EQ    | mpv IPC `af set equalizer=...` — no new dependency; presets stored in config TOML; block-character UI in a dedicated overlay |
| EQ preset library     | Built-in: Flat, Rock, Electronic, Jazz, Classical; user-editable in config                                                   |
| Per-track EQ override | Store per-track EQ settings in SQLite; auto-applied on playback                                                              |
| Spectrum visualiser   | Block-character FFT display in the Now Playing bar                                                                           |

### Discovery & Navigation

| Feature                  | Notes                                                                                                                                   |
| ------------------------ | --------------------------------------------------------------------------------------------------------------------------------------- |
| Search-first launch mode | `waveshell --search` opens directly into the fuzzy finder rather than the library browser; alternative entry point for quick playback   |
| Depth search             | `F1`–`F4` scopes fuzzy search to a specific depth in the directory hierarchy; useful for well-organised `Artist/Album/Track` structures |
| Vim-style marks          | `m{a-z}` to bookmark a library position; `'{a-z}` to jump back; stored per-session and optionally persisted in SQLite                   |
| Recently played view     | A browsable virtual playlist of the last N played tracks, sourced from the `play_history` table                                         |
| Smart playlists          | Filter expressions over the SQLite index; e.g. "unplayed ALAC albums", "tracks with R128 loudness below −14 LUFS"                       |
| Letter-jump in all panes | Already implemented for artists in Milestone 3; extend to album and track panes                                                         |

### Metadata & Library

| Feature      | Notes                                                                                                             |
| ------------ | ----------------------------------------------------------------------------------------------------------------- |
| Watcher mode | `fsnotify` for real-time detection of library changes; triggers incremental rescan automatically                  |
| BPM display  | Read from tag if present (`TBPM` / `BPM` Vorbis comment); display in info panel and as optional track list column |
| Playlists    | Create and edit M3U playlists; internal playlist format stored in SQLite                                          |

### Integration & Automation

| Feature                  | Notes                                                                                                                                                                                    |
| ------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Daemon mode + IPC remote | `waveshell --daemon` for headless playback; `waveshell remote play/pause/next/status` from another terminal; enables Waybar/Sketchybar integration and media key daemons                 |
| Script hooks             | Shell commands fired on events: `on_track_change`, `on_playback_start`, `on_playback_stop`; configured in `config.toml`; covers 90% of automation use cases without a full plugin system |
| Scrobbling               | Last.fm opt-in; credentials stored via `zalando/go-keyring` (OS keychain), never in config file                                                                                          |
| MPRIS support            | Linux desktop integration for hardware media keys and `playerctl`                                                                                                                        |

### Visual

| Feature             | Notes                                                                                                                                                                          |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Album art rendering | `chafa` subprocess integration; auto-detects terminal protocol at startup (Kitty → iTerm2 → Sixel → half-block Unicode fallback); displays in a panel alongside the track list |
| Waveform display    | Block-character waveform in the Now Playing bar                                                                                                                                |

---

## 10. Testing Strategy

### Philosophy

The Elm Architecture makes this straightforward: `Update` is a pure function, so business logic tests are just function calls with assertions. No TUI instantiation, no mocking a framework, no async ceremony.

### Test Categories

**Unit tests** cover pure `Update(msg, model)` → `(Model, Cmd)` call chains, all data model operations (queue, scanner output, metadata parsing), and the mpv IPC command serialisation and event deserialisation logic. These should run in milliseconds and require no external processes.

**Integration tests** cover the library scanner against a real temporary directory of fixture audio files, the IPC wrapper against a mock Unix socket server, and all SQLite operations against an in-memory database opened with `modernc.org/sqlite` in `:memory:` mode.

**No end-to-end tests in MVP.** Testing that mpv actually produces audio is a manual smoke test, not a CI concern.

### Fixture Audio Files

A set of silent 5-second files in each supported format with fully populated known metadata, generated once:

```bash
ffmpeg -f lavfi -i anullsrc=r=44100:cl=stereo -t 5 -ar 44100 fixture.flac
# repeated for .mp3, .m4a (ALAC), .aiff, .wav, .ogg
```

Committed to the repository under `testdata/fixtures/`. Tags applied with `ffmpeg -metadata` at generation time. These files are the single source of truth for all format-specific test assertions.

### Coverage Targets

| Package                           | Target                          |
| --------------------------------- | ------------------------------- |
| `internal/config`                 | 95%                             |
| `internal/scanner`                | 95%                             |
| `internal/mpv`                    | 90%                             |
| `internal/db`                     | 90%                             |
| `internal/tagger`                 | 95%                             |
| `internal/update` (Elm Update fn) | 90%                             |
| `internal/update` (rendering)     | Exempt from hard target         |
| **Overall**                       | **80% minimum, enforced in CI** |

### Makefile Targets

```makefile
make test        # go test ./... -race -coverprofile=coverage.out
make coverage    # go tool cover -html=coverage.out
make lint        # golangci-lint run
make build       # go build -o waveshell ./cmd/waveshell
make build-all   # cross-compile: macOS ARM, macOS Intel, Linux x86_64
```

---

## 11. Metadata Reference

### Audio Format Support

| Format     | Extension(s)    | Metadata library | Notes                  |
| ---------- | --------------- | ---------------- | ---------------------- |
| FLAC       | `.flac`         | `dhowden/tag`    | Vorbis comments        |
| ALAC       | `.m4a`, `.alac` | `dhowden/tag`    | MP4 atoms              |
| AIFF       | `.aiff`, `.aif` | `dhowden/tag`    | ID3v2 embedded in AIFF |
| MP3        | `.mp3`          | `dhowden/tag`    | ID3v2.3 / ID3v2.4      |
| WAV        | `.wav`          | `dhowden/tag`    | ID3v2 or INFO chunk    |
| OGG Vorbis | `.ogg`          | `dhowden/tag`    | Vorbis comments        |

### Label / Publisher Field Mapping

The app surfaces both the Grouping and Label/Publisher fields distinctly. They are separate columns in the SQLite schema and separate rows in the info panel.

| Format              | Grouping field | Tag ID     | Label / Publisher field | Tag ID                        |
| ------------------- | -------------- | ---------- | ----------------------- | ----------------------------- |
| MP3 / AIFF (ID3v2)  | Content Group  | `TIT1`     | Publisher               | `TPUB`                        |
| FLAC / OGG (Vorbis) | Grouping       | `GROUPING` | Label                   | `LABEL`                       |
| MP4 / M4A / ALAC    | Grouping       | `©grp`     | Label (freeform)        | `----:com.apple.iTunes:LABEL` |

**Background:** `TPUB` (Publisher) is the correct ID3v2 field for record label data. Apple Music does not expose this field in its UI, leaving `TIT1` (Grouping / Content Group) as the only accessible freeform field — which is why some library workflows store label names there. waveshell reads and displays both fields without preference, enabling the user to see the state of their library accurately and edit either field via the tag editor.

### Loudness Tag Reference

| Tag key                 | Standard       | Scope                                |
| ----------------------- | -------------- | ------------------------------------ |
| `REPLAYGAIN_TRACK_GAIN` | ReplayGain 2.0 | Per-track gain in dB                 |
| `REPLAYGAIN_TRACK_PEAK` | ReplayGain 2.0 | Per-track sample peak (0.0–1.0)      |
| `REPLAYGAIN_ALBUM_GAIN` | ReplayGain 2.0 | Per-album gain in dB                 |
| `REPLAYGAIN_ALBUM_PEAK` | ReplayGain 2.0 | Per-album sample peak (0.0–1.0)      |
| `R128_TRACK_GAIN`       | EBU R128       | Per-track integrated loudness offset |
| `R128_ALBUM_GAIN`       | EBU R128       | Per-album integrated loudness offset |

All six are read and displayed in the info panel if present. None are written by the app unless explicitly edited via the tag editor.
