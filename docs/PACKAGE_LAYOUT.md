# waveshell — Package Directory Layout

> **Status:** Pre-development
> **Scope:** Canonical directory tree, per-package responsibilities, import constraints, and resolved structural questions.
> **Last updated:** June 2026

---

## Table of Contents

1. [Full Directory Tree](#1-full-directory-tree)
2. [Per-Package Responsibilities](#2-per-package-responsibilities)
3. [Resolved Structural Questions](#3-resolved-structural-questions)
4. [Test File Conventions](#4-test-file-conventions)

---

## 1. Full Directory Tree

```
cmd/
  waveshell/
    main.go
internal/
  model/        — domain types (Track, Album, Artist, enums); zero internal imports
  messages/     — all tea.Msg types and TickCmd; imports internal/model only
  config/       — Load, LoadFrom, Default, WriteConfig; imports internal/messages
  db/           — all SQLite query and write Cmds; imports internal/messages, internal/model
  scanner/      — filepath.WalkDir + dhowden/tag + db writes; imports internal/messages, internal/db
  mpv/          — subprocess launch, socket IPC, event loop goroutine; imports internal/messages
  tagger/       — atomic tag writes; imports internal/messages
  search/       — fuzzy index build and query; imports internal/messages, internal/model
  update/       — the BubbleTea Update and View functions, Model definition; imports all of the above
docs/
testdata/
  fixtures/     — silent audio files for format-specific tests (committed)
Makefile
go.mod
.golangci.yml
```

---

## 2. Per-Package Responsibilities

### `cmd/waveshell/`

**Responsibility:** Application entrypoint. Parses flags (none in MVP beyond `--config` and `--version`), calls `config.Load()`, opens the SQLite database, constructs the `Model`, starts the BubbleTea program.

**Exports:** `main()` only.

**Imports from internal:** `internal/config`, `internal/model` (to construct initial `Model`), `internal/update` (as `tea.Program` argument).

### `internal/model/`

**Responsibility:** Domain type definitions with zero internal package dependencies. This is a pure data package — no IO, no business logic, no BubbleTea imports.

**Exports:** `Track`, `Album`, `Artist`, `Playlist` and supporting enums (`BrowseMode`, `SortField`, `SortDirection`, `RepeatMode`, `PlaybackState`, `Pane`, `FocusLayer`, `ColumnID`).

**Imports from internal:** None. This is a leaf package.

**Notable:** Tag-related struct types (the clean tag values read from files before database insertion) live here as `RawTrackTags`. The `Track` struct is the canonical in-memory representation hydrated from SQLite rows.

### `internal/messages/`

**Responsibility:** The sole location for all `tea.Msg` types. Every package that needs to produce or consume Msgs imports this package. Prevents import cycles by keeping all Msg types in one leaf package.

**Exports:** All Msg struct types (listed in full in `docs/MSGS.md`), `TickCmd`, the `Cmd` constructor functions for each Msg, and `internal/mpv.Player` interface reference for the subscription pattern.

**Imports from internal:** `internal/model` only.

**Notable:** Go type aliases for message payloads (e.g., `type TrackListMsg []Track`) live here so the `Update` function can type-assert without importing `internal/model`.

### `internal/config/`

**Responsibility:** Load, validate, and write `config.toml`. Provides the `Config` struct that every other package reads during initialisation.

**Exports:** `Config`, `Load()`, `LoadFrom(path string)`, `Default()`, `WriteConfig(path string, cfg Config) error`, `Matches(key string) bool` helper for keybinding slices.

**Imports from internal:** `internal/messages` (for `ConfigWrittenMsg` emission after debounced writes).

**Notable:** `LoadFrom(path string)` exists exclusively for test isolation with `t.TempDir()`. The `Matches()` helper is used by `internal/update` to test keybinding slices without importing `internal/config`.

### `internal/db/`

**Responsibility:** All SQLite query and write operations. Does not export a `DB` type or raw query functions — the `*sql.DB` handle is passed as a parameter to the exported `Cmd` functions.

**Exports:** Cmd functions that return `tea.Cmd` (see `docs/MSGS.md` for full signatures): `QueryArtists`, `QueryAlbums`, `QueryTracks`, `InsertTrack`, `UpdateTrack`, `DeleteTrack`, `InsertPlayHistory`, etc.

**Imports from internal:** `internal/messages`, `internal/model`.

**Notable:** No `DB` struct, no service-object abstraction. The `*sql.DB` is stored in `Model` and threaded through as a parameter. This keeps the package as a thin layer over SQLite and prevents it from accumulating stateful abstractions.

### `internal/scanner/`

**Responsibility:** Walks configured library paths with `filepath.WalkDir`, reads audio file metadata via `dhowden/tag`, compares against the database, and emits insert/update Msgs for changed files.

**Exports:** `StartScanCmd(libraryPaths []string, db *sql.DB) tea.Cmd`.

**Imports from internal:** `internal/messages`, `internal/db`.

**Notable:** Uses Pattern A (Recursive Cmd, see `docs/MSGS.md` §3.1). Each scanned file emits a `ScanProgressMsg`. Errors per file emit `ScanFileErrorMsg` (non-fatal). Completion emits `ScanCompleteMsg`. The recursive tail-call pattern allows BubbleTea to yield between files, keeping the UI responsive during large scans.

### `internal/mpv/`

**Responsibility:** Launches mpv as a subprocess, manages the Unix socket IPC connection, implements the `Player` interface, runs an event-loop goroutine that reads mpv responses and dispatches typed Msgs to the BubbleTea subscription channel.

**Exports:** `Player` interface (see `docs/MPV_IPC.md` §6), `NewPlayer(socketPath string) (Player, error)`, `Launch() (Cmd, error)`, `Close()`.

**Imports from internal:** `internal/messages` (for Msg types sent to the subscription channel).

**Notable:** Uses Pattern B (Subscription Cmd, see `docs/MSGS.md` §3.2). The concrete `PlayerImpl` is not exported beyond construction; all consumption goes through the `Player` interface or the subscription channel.

### `internal/tagger/`

**Responsibility:** Atomic tag writes to audio files using a `.tmp` + `os.Rename` pattern. Never modifies the original file in place.

**Exports:** `WriteTags(path string, updates TagUpdate, db *sql.DB) tea.Cmd`, `BatchWriteTags(paths []string, updates TagUpdate, db *sql.DB) tea.Cmd`.

**Imports from internal:** `internal/messages`.

**Notable:** Uses Pattern A (Recursive Cmd) for batch writes, emitting `BatchTagWriteProgressMsg` per file. The `TagUpdate` struct specifies which fields to change (not a full replacement). Database updates are chained after each successful write so the UI reflects the change immediately.

### `internal/search/`

**Responsibility:** Builds and queries a fuzzy search index over tracks and albums. The index is rebuilt from SQLite data on startup (not persisted separately).

**Exports:** `BuildIndex(db *sql.DB) error`, `Search(query string) tea.Cmd`.

**Imports from internal:** `internal/messages`, `internal/model`.

**Notable:** Post-MVP (Milestone 6). In MVP, `internal/update` returns a no-op for search actions. The index is an in-memory fuzzy index built from SQLite data on startup using `sahilm/fuzzy`. The `internal/search` package wraps index construction, fuzzy query execution, and result hydration into typed Msgs.

### `internal/update/`

**Responsibility:** The BubbleTea `Model` definition and its `Init`, `Update`, and `View` methods. This is the largest package by surface area — it handles every Msg type, renders the full TUI, and decides what state transitions and side effects to produce.

**Exports:** `Model`, `InitialModel` (helper for test setup), `Init`, `Update`, `View`.

**Imports from internal:** Every other internal package. This is intentional — `internal/update` is the hub that all subsystems feed into.

**Notable:** `Model` implements `tea.Model` (see §3.1). `View` is a method on `Model` following standard BubbleTea convention. Rendering constants and lipgloss styling live here alongside business logic — no separate `internal/view` package is needed for MVP.

---

## 3. Resolved Structural Questions

### 3.1 Where `Model` is defined

`Model` is defined in `internal/update`. It implements `tea.Model` with `Init`, `Update`, and `View` methods in the same package. This is the simplest arrangement for MVP — no separate `internal/app` or `internal/view` package is needed.

```go
// internal/update/model.go
package update

type Model struct {
    Library LibraryState
    Player  PlayerState
    Queue   QueueState
    Search  SearchState
    UI      UIState
    Config  *config.Config
    DB      *sql.DB
    MPV     mpv.Player
}
```

If `internal/update` grows too large, the model can be extracted to `internal/app` in a refactoring pass. For MVP, keeping the model and update logic adjacent reduces indirection.

### 3.2 Single package design

`Model`, `Init`, `Update`, and `View` all live in `internal/update`. No `internal/app` or `internal/view` package. This is the standard BubbleTea convention — the model carries its own rendering and event handling.

The `Main` function in `cmd/waveshell/main.go` constructs the `tea.Program`:

```go
p := tea.NewProgram(update.Model{...}, tea.WithAltScreen())
```

### 3.3 Where sub-structs of `Model` live

All sub-structs (`LibraryState`, `PlayerState`, `QueueState`, `SearchState`, `UIState`) live in `internal/update` alongside `Model`:

```go
// internal/update/state.go
package update

type LibraryState struct { ... }
type PlayerState  struct { ... }
type QueueState   struct { ... }
type SearchState  struct { ... }
type UIState      struct { ... }
```

They are not scattered across subsystem packages. This avoids circular dependencies and keeps state initialisation in one place.

### 3.4 `internal/db` exports Cmd functions, not a DB type

The `internal/db` package does not export a `DB` struct, a connection pool wrapper, or raw query functions. Every database operation is exposed as a `func(db *sql.DB, args ...) tea.Cmd`. The `*sql.DB` handle lives in `Model` and is threaded through.

This prevents the package from growing a service-object abstraction and keeps all database interaction visible at the call site in `internal/update`.

```go
// internal/db/tracks.go
package db

func QueryTracks(db *sql.DB, albumID int64) tea.Cmd {
    return func() tea.Msg {
        rows, err := db.Query(`SELECT ...`, albumID)
        if err != nil { return messages.DBErrorMsg{Err: err} }
        // ...
        return messages.TrackListResultMsg{Tracks: tracks}
    }
}
```

### 3.5 `View` has no Msg knowledge

`View` is a method on `Model` in `internal/update`. It receives a fully-processed `Model` and renders it to a string. It never handles Msg types directly — all Msg processing has already happened in `Update` by the time `View` runs. This keeps rendering decoupled from the event system.

---

## 4. Test File Conventions

### Unit tests

All `_test.go` files live in the same package as the code they test (whitebox testing). Go's package-internal visibility is sufficient for all test scenarios in this codebase.

### Mock socket server

The mpv mock socket server lives at `internal/mpv/mock_test.go`. It is only compiled into test binaries. See `docs/MPV_IPC.md` §7 for its interface and usage.

### Test data fixtures

Silent audio files with fully populated tags are committed to `testdata/fixtures/`. Each format has its own subdirectory:

```
testdata/fixtures/
  flac/
    sample.flac         — 5-second silent FLAC with full Vorbis comments
  mp3/
    sample.mp3          — 5-second silent MP3 with full ID3v2.4 tags
  m4a/
    sample.m4a          — 5-second silent ALAC in MP4 container with full tags
  aiff/
    sample.aiff         — 5-second silent AIFF with ID3v2 tags
  wav/
    sample.wav          — 5-second silent WAV with ID3v2 tags
  ogg/
    sample.ogg          — 5-second silent OGG Vorbis with full Vorbis comments
```

Each fixture file has all tag fields populated: title, artist, album, album artist, genre, year, track number, disc number, label, grouping, composer, and ReplayGain tags where the format supports them. This allows the scanner, db, and tagger packages to test against every supported format in a single test run.

Fixtures are generated by `testdata/generate_fixtures.sh` (not committed — developers run it once after cloning) using `ffmpeg` and a metadata injection script. The output is deterministic (same input produces identical binary output).
