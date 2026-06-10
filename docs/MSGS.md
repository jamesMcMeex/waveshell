# waveshell — Msg & Cmd Reference

> **Status:** Pre-development
> **Scope:** All custom `tea.Msg` types, every `tea.Cmd` signature, the two core async patterns, and the consolidated Go type definitions for `internal/messages`. This document is the API contract between every async subsystem and the BubbleTea Update loop.
> **Last updated:** June 2026

---

## Table of Contents

1. [Why This Document Exists](#1-why-this-document-exists)
2. [Package Strategy](#2-package-strategy)
3. [Core Async Patterns](#3-core-async-patterns)
   - [Pattern A: Recursive Cmd (scanner, batch writes)](#31-pattern-a-recursive-cmd)
   - [Pattern B: Subscription Cmd (mpv event stream)](#32-pattern-b-subscription-cmd)
4. [Msg Types by Subsystem](#4-msg-types-by-subsystem)
   - [Config](#41-config)
   - [Scanner](#42-scanner)
   - [Database](#43-database)
   - [mpv Process](#44-mpv-process)
   - [mpv Events](#45-mpv-events)
   - [Playback Tick](#46-playback-tick)
   - [Search](#47-search)
   - [Tag Editor](#48-tag-editor)
   - [Config Writer](#49-config-writer)
   - [Play History](#410-play-history)
5. [tea.Cmd Signatures](#5-teacmd-signatures)
6. [Go Type Definitions](#6-go-type-definitions)
7. [Quick-Reference Table](#7-quick-reference-table)

---

## 1. Why This Document Exists

The Elm Architecture enforces a clean separation: the `Update` function is pure and synchronous; all IO happens via `tea.Cmd`. The Msg sum type is therefore the seam between the pure core and every impure subsystem. Two things follow from this:

**Testability.** Every test in this codebase calls `Update(msg, model)` and asserts on the returned `(Model, Cmd)`. Tests are fast, synchronous, and require no subprocess or goroutine plumbing — but only if the Msg types are designed with testing in mind. Msgs that carry typed, assertable data (not raw `[]byte` or `interface{}`) make test assertions readable.

**Import cycle prevention.** Several packages need to produce Msgs: `internal/scanner`, `internal/mpv`, `internal/db`, `internal/tagger`. The `Update` function in `internal/update` needs to consume all of them. If each package defined its own Msg types, import cycles would be unavoidable. The solution is a single leaf package — `internal/messages` — that owns all Msg types. Every producer imports it; the consumer (`internal/update`) imports it. No cycles are possible.

---

## 2. Package Strategy

Two leaf packages carry no internal imports and are safe to import from anywhere:

```
internal/model     — domain types: Track, Album, Artist, and supporting enums
internal/messages  — all tea.Msg types; imports internal/model, nothing else
```

Every other package imports `internal/messages` to produce or consume Msgs. The dependency graph is:

```
internal/model
      ↑
internal/messages
      ↑
internal/config  internal/scanner  internal/mpv  internal/db  internal/tagger  internal/search
      ↑                ↑               ↑              ↑              ↑                ↑
                           internal/update
                                  ↑
                            cmd/waveshell
```

`internal/model` is defined separately from `internal/messages` because domain types (Track, Album, etc.) are also used by packages that never produce Msgs — for example, `internal/db` uses Track as a return type from query functions before wrapping results in a Msg.

---

## 3. Core Async Patterns

All async work in waveshell uses one of two patterns. Understanding them before reading the Msg list is essential.

### 3.1 Pattern A: Recursive Cmd

Used when work is long-running and should emit progress incrementally: the library scanner, and batch tag writes.

The pattern is: a Cmd does a unit of work, returns a progress Msg **and** the next Cmd. The Update function receives the Msg, updates the model, and schedules the next Cmd. Scanning continues until the final Cmd returns a `ScanCompleteMsg` with no follow-up Cmd.

```go
// In internal/scanner:

// StartScanCmd is the public entry point. Called from Update on app startup or manual rescan.
func StartScanCmd(paths []string, db *sql.DB) tea.Cmd {
    state := newScanState(paths, db)
    return continueScanCmd(state)
}

// continueScanCmd is unexported. It processes one batch and recurses.
func continueScanCmd(state *scanState) tea.Cmd {
    return func() tea.Msg {
        if state.done() {
            return messages.ScanCompleteMsg{
                Processed: state.processed,
                Skipped:   state.skipped,
            }
        }
        err := state.processNext()
        if err != nil {
            // Non-fatal: record the error and continue
            return messages.ScanFileErrorMsg{Path: state.current, Err: err, NextCmd: continueScanCmd(state)}
        }
        return messages.ScanProgressMsg{
            Processed:   state.processed,
            Total:       state.total,
            CurrentPath: state.current,
            NextCmd:     continueScanCmd(state),  // carry the next step forward
        }
    }
}
```

`ScanProgressMsg` carries `NextCmd tea.Cmd` so that Update can schedule it. Update's handling:

```go
case messages.ScanProgressMsg:
    model.Library.ScanProcessed = msg.Processed
    model.Library.ScanTotal = msg.Total
    return model, msg.NextCmd   // schedule the next batch

case messages.ScanCompleteMsg:
    model.Library.Scanning = false
    model.Library.ScanProcessed = msg.Processed
    return model, nil           // done; no follow-up Cmd
```

**Key property:** because all state lives in `scanState` (a value passed between Cmds, not a shared goroutine), the scanner is testable by constructing a scanState with fixture data and calling continueScanCmd directly.

---

### 3.2 Pattern B: Subscription Cmd

Used when an external process pushes events asynchronously: the mpv IPC event stream.

The pattern is: a goroutine reads from a channel and returns each event as a Msg. After delivering a Msg, Update immediately re-schedules the same subscription Cmd — creating a self-renewing listener. The goroutine itself is long-lived and communicates via a channel owned by `internal/mpv`.

```go
// In internal/mpv:

// SubscribeCmd returns a Cmd that waits for the next event from the mpv IPC reader
// and returns it as a Msg. Update re-schedules this Cmd after every event.
func SubscribeCmd(events <-chan tea.Msg) tea.Cmd {
    return func() tea.Msg {
        return <-events  // blocks until mpv pushes an event
    }
}
```

The goroutine that feeds the channel:

```go
// StartEventLoop runs in a goroutine after the mpv socket is connected.
// It reads raw JSON events from the socket, converts them to typed Msgs,
// and sends them to the events channel.
func (c *Conn) StartEventLoop(events chan<- tea.Msg) {
    for {
        raw, err := c.readLine()
        if err != nil {
            events <- messages.MPVConnectionLostMsg{Err: err}
            return
        }
        events <- c.parseEvent(raw)
    }
}
```

Update's handling:

```go
case messages.TimePositionChangedMsg:
    model.Player.PositionSec = msg.PositionSec
    return model, mpv.SubscribeCmd(model.Player.Events)  // re-subscribe immediately

case messages.MPVConnectionLostMsg:
    model.Player.State = model.PlaybackStateStopped
    model.Player.Error = msg.Err
    return model, nil   // do NOT re-subscribe; connection is gone
```

**Key property:** the channel is the boundary between the goroutine world and the Elm world. The goroutine never touches the Model; the Model never touches the goroutine directly. Tests mock the channel by pre-loading it with known Msgs.

---

## 4. Msg Types by Subsystem

### 4.1 Config

Produced by `internal/config`. Delivered on app startup before `Init` returns.

```go
// ConfigLoadedMsg is sent when config.toml has been read and validated successfully.
// The loaded Config is attached so Update can store it in the Model.
type ConfigLoadedMsg struct {
    Config config.Config
}

// ConfigErrorMsg is sent when config loading fails with a hard error
// (malformed TOML, unresolvable ${VAR}, invalid value).
// Renders as a blocking error dialog. The app cannot proceed.
type ConfigErrorMsg struct {
    Err error
}
```

**Error severity:** `ConfigErrorMsg` → blocking dialog (fatal). Config errors are always fatal because the app's keybindings, theme, and library paths all depend on a valid config.

---

### 4.2 Scanner

Produced by `internal/scanner`. Uses Pattern A (recursive Cmd).

```go
// ScanStartedMsg is sent immediately when a scan is triggered,
// before any files are processed. Used to show the spinner in the status bar.
type ScanStartedMsg struct{}

// ScanProgressMsg is sent after each file is processed successfully.
// NextCmd carries the continuation; Update must schedule it.
type ScanProgressMsg struct {
    Processed   int
    Total        int      // -1 until the directory walk completes and total is known
    CurrentPath string    // path of the file just processed; for status bar display
    NextCmd      tea.Cmd  // the next unit of work; must be returned from Update
}

// ScanFileErrorMsg is sent when a single file cannot be processed
// (unreadable, unsupported format, corrupt tags).
// Non-fatal: scanning continues. The file is counted in ScanCompleteMsg.Skipped.
// NextCmd carries the continuation, same as ScanProgressMsg.
type ScanFileErrorMsg struct {
    Path    string
    Err     error
    NextCmd tea.Cmd
}

// ScanCompleteMsg is the terminal event. No NextCmd.
type ScanCompleteMsg struct {
    Processed int
    Skipped   int
}
```

**Note on Total:** the scanner uses `filepath.WalkDir`, which does not know the total file count upfront. `Total` is set to `-1` during the walk phase and updated to the actual count once the walk completes and batch processing begins. The status bar renders `Scanning… 312 tracks` during the walk and `Scanning… 312 / 1,247` once the total is known.

---

### 4.3 Database

Produced by `internal/db`. One-shot async queries; no recursion.

```go
// ArtistListResultMsg carries the full artist list for the left pane
// in artist browse mode.
type ArtistListResultMsg struct {
    Artists []model.Artist
}

// TagSliceResultMsg carries distinct values for the left pane
// in label, genre, year, and grouping browse modes.
type TagSliceResultMsg struct {
    Mode   model.BrowseMode
    Values []string   // distinct non-null values, ordered alphabetically (or numerically for year)
}

// AlbumListResultMsg carries albums for the middle pane.
// Key is the selected left-pane value (artist ID cast to string for artist mode;
// label/genre/grouping string or year string for the other modes).
type AlbumListResultMsg struct {
    Mode   model.BrowseMode
    Key    string
    Albums []model.Album
}

// TrackListResultMsg carries tracks for the tracks pane.
type TrackListResultMsg struct {
    AlbumID int64
    Tracks  []model.Track
}

// DBErrorMsg is sent when a database query or write fails.
// Fatal = true for errors that prevent the app from functioning
// (database file unreadable, schema migration failure).
// Fatal = false for query errors that can be surfaced as a status bar toast.
type DBErrorMsg struct {
    Op    string  // human-readable description: "query artists", "write play history"
    Err   error
    Fatal bool
}
```

**Error severity:** `DBErrorMsg{Fatal: true}` → blocking dialog. `DBErrorMsg{Fatal: false}` → status bar toast, auto-clears after 4 seconds.

---

### 4.4 mpv Process

Produced by `internal/mpv`. These cover the lifecycle of the mpv subprocess.

```go
// MPVReadyMsg is sent after the mpv subprocess has started AND
// the Unix socket connection has been established successfully.
// Only after this Msg may playback commands be sent.
// Events is the channel to pass to SubscribeCmd.
type MPVReadyMsg struct {
    Events <-chan tea.Msg
}

// MPVNotFoundMsg is sent when the mpv binary cannot be located on PATH.
// Renders as a blocking dialog with installation instructions.
type MPVNotFoundMsg struct{}

// MPVConnectionLostMsg is sent when the socket connection drops unexpectedly
// (mpv crashed, was killed externally, or timed out on connect).
type MPVConnectionLostMsg struct {
    Err error
}
```

**Error severity:** `MPVNotFoundMsg` → blocking dialog (fatal). `MPVConnectionLostMsg` → status bar toast + reset PlayerState to Stopped (non-fatal; user can attempt reconnect).

---

### 4.5 mpv Events

Produced by `internal/mpv`'s event loop goroutine. Delivered via Pattern B (subscription). These represent mpv pushing state changes over the IPC socket.

```go
// PlaybackStateChangedMsg is sent when mpv's pause property changes.
type PlaybackStateChangedMsg struct {
    State model.PlaybackState   // Playing | Paused
}

// TimePositionChangedMsg is sent when mpv's time-pos property changes.
// mpv is configured to observe this property at 1-second intervals.
type TimePositionChangedMsg struct {
    PositionSec float64
}

// DurationChangedMsg is sent when mpv's duration property is first available
// after a new file loads. mpv does not know the duration until demuxing begins.
type DurationChangedMsg struct {
    DurationSec float64
}

// VolumeChangedMsg is sent when mpv's volume property changes.
// This fires both from waveshell commands and from external mpv control.
type VolumeChangedMsg struct {
    Volume int   // 0–100
}

// TrackEndedMsg is sent when mpv's end-file event fires.
// Update uses this to advance the queue.
type TrackEndedMsg struct {
    Reason model.TrackEndReason   // EOF | Stopped | Error
}
```

**On TimePositionChangedMsg frequency:** mpv property observation for `time-pos` is registered with `observe_property` at socket connect time. To avoid flooding the event loop, `internal/mpv` registers with a minimum interval of 1 second. The debounce happens at the mpv level, not in Go.

---

### 4.6 Playback Tick

A recurring timer used to keep the Now Playing bar's elapsed time display responsive when mpv events are delayed or absent (e.g. during seeks). Distinct from mpv's `TimePositionChangedMsg` — the tick drives the optimistic display; the mpv event provides the accurate position.

```go
// TickMsg is sent once per second while playback is active.
// Update uses it to increment the displayed elapsed time between
// genuine TimePositionChangedMsgs from mpv.
type TickMsg struct {
    Time time.Time
}
```

```go
// TickCmd schedules a single 1-second tick. Update re-schedules it
// on receipt while PlayerState is Playing.
func TickCmd() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return TickMsg{Time: t}
    })
}
```

Update handling:

```go
case messages.TickMsg:
    if model.Player.State == model.PlaybackStatePlaying {
        model.Player.DisplayPositionSec++   // optimistic; corrected by next TimePositionChangedMsg
        return model, messages.TickCmd()    // re-schedule
    }
    return model, nil   // do not re-schedule when paused or stopped
```

---

### 4.7 Search

Produced by `internal/search`. One-shot async; fuzzy matching runs in a goroutine.

```go
// SearchResultsMsg carries the ranked results for the current query.
// Sent after every keystroke (debounced inside FuzzySearchCmd, not in Update).
type SearchResultsMsg struct {
    Query   string
    Results model.SearchResults
}
```

---

### 4.8 Tag Editor

Produced by `internal/tagger`. The write path is always: write `.tmp` file → `os.Rename` → emit result Msg.

```go
// TagWriteCompleteMsg is sent after a successful atomic tag write.
// The database record for this track should be refreshed after receipt.
type TagWriteCompleteMsg struct {
    Path string
}

// TagWriteErrorMsg is sent if the write fails at any stage.
// OriginalIntact is always verified true before this Msg is sent —
// if the original file cannot be confirmed intact, the error message says so.
type TagWriteErrorMsg struct {
    Path           string
    Err            error
    OriginalIntact bool
}

// BatchTagWriteProgressMsg is sent after each file in a batch write completes
// (success or failure). Uses Pattern A; carries NextCmd.
type BatchTagWriteProgressMsg struct {
    Path      string
    Succeeded bool
    Err       error    // nil on success
    Done      int      // files completed so far (success + failure)
    Total     int
    NextCmd   tea.Cmd
}

// BatchTagWriteCompleteMsg is the terminal event for a batch write.
type BatchTagWriteCompleteMsg struct {
    Succeeded int
    Failed    int
    Errors    []BatchWriteError   // one entry per failed file
}

type BatchWriteError struct {
    Path string
    Err  error
}
```

---

### 4.9 Config Writer

Produced when the Column Manager writes column changes back to `config.toml`. Runs asynchronously to avoid blocking the UI.

```go
// ConfigWrittenMsg is sent after config.toml has been written successfully.
// No action required in Update; it exists for logging and test assertions.
type ConfigWrittenMsg struct{}

// ConfigWriteErrorMsg is sent if the config file write fails.
// Non-fatal: the in-memory Config is already updated; only the file write failed.
// Rendered as a status bar toast.
type ConfigWriteErrorMsg struct {
    Err error
}
```

---

### 4.10 Play History

Produced when a track crosses the 50% completion threshold. Fire-and-forget write to SQLite.

```go
// PlayHistoryWrittenMsg confirms the play_history row was inserted.
// Primarily used in tests to assert the DB write was attempted.
type PlayHistoryWrittenMsg struct {
    TrackID int64
}

// PlayHistoryWriteErrorMsg is sent if the insert fails.
// Non-fatal: play history is convenience data, not critical state.
// Logged at warn level; not surfaced in the UI.
type PlayHistoryWriteErrorMsg struct {
    TrackID int64
    Err     error
}
```

---

## 5. tea.Cmd Signatures

These are the exported Cmd constructor functions called from `Update`. Each lives in the package responsible for the work. All return `tea.Cmd`.

```go
// ── internal/config ──────────────────────────────────────────────────────────

// LoadConfigCmd reads ~/.config/waveshell/config.toml (XDG-resolved).
// Returns ConfigLoadedMsg or ConfigErrorMsg.
func LoadConfigCmd() tea.Cmd

// LoadConfigFromCmd reads from a specific path. Used in tests with t.TempDir().
// Returns ConfigLoadedMsg or ConfigErrorMsg.
func LoadConfigFromCmd(path string) tea.Cmd

// WriteConfigCmd writes the given Config back to disk asynchronously.
// Returns ConfigWrittenMsg or ConfigWriteErrorMsg.
func WriteConfigCmd(cfg config.Config) tea.Cmd


// ── internal/scanner ─────────────────────────────────────────────────────────

// StartScanCmd begins a full incremental scan of the given paths.
// Returns ScanStartedMsg immediately, then a chain of ScanProgressMsg /
// ScanFileErrorMsg, ending with ScanCompleteMsg. Uses Pattern A.
func StartScanCmd(paths []string, db *sql.DB) tea.Cmd


// ── internal/db ──────────────────────────────────────────────────────────────

// QueryArtistsCmd fetches all artists ordered by name_sort.
// Returns ArtistListResultMsg or DBErrorMsg.
func QueryArtistsCmd(db *sql.DB) tea.Cmd

// QueryTagSliceCmd fetches distinct non-null values for the given browse mode's
// left-pane dimension (label, genre, year, or grouping).
// Returns TagSliceResultMsg or DBErrorMsg.
func QueryTagSliceCmd(db *sql.DB, mode model.BrowseMode) tea.Cmd

// QueryAlbumsForArtistCmd fetches albums by artist ID.
// Returns AlbumListResultMsg or DBErrorMsg.
func QueryAlbumsForArtistCmd(db *sql.DB, artistID int64) tea.Cmd

// QueryAlbumsForTagCmd fetches albums matching a tag-slice left-pane value
// (label name, genre name, year string, or grouping name).
// Returns AlbumListResultMsg or DBErrorMsg.
func QueryAlbumsForTagCmd(db *sql.DB, mode model.BrowseMode, key string) tea.Cmd

// QueryTracksCmd fetches tracks for an album, ordered by disc_number, track_number.
// Returns TrackListResultMsg or DBErrorMsg.
func QueryTracksCmd(db *sql.DB, albumID int64) tea.Cmd

// WritePlayHistoryCmd inserts a play_history row for the given track.
// Returns PlayHistoryWrittenMsg or PlayHistoryWriteErrorMsg.
func WritePlayHistoryCmd(db *sql.DB, trackID int64, percentComplete int) tea.Cmd


// ── internal/mpv ─────────────────────────────────────────────────────────────

// StartMPVCmd launches the mpv subprocess and connects to its IPC socket.
// Returns MPVReadyMsg (with the event channel) on success,
// or MPVNotFoundMsg / MPVConnectionLostMsg on failure.
func StartMPVCmd(socketPath string) tea.Cmd

// SubscribeCmd waits for the next event on the given channel and returns it.
// Update must re-schedule this Cmd after every event to keep the loop alive.
// Uses Pattern B.
func SubscribeCmd(events <-chan tea.Msg) tea.Cmd

// SendCommandCmd sends a single fire-and-forget JSON command to mpv.
// Errors from individual commands (e.g. seek out of range) are logged
// but not surfaced as Msgs unless they indicate a lost connection.
func SendCommandCmd(conn *mpv.Conn, cmd mpv.Command) tea.Cmd


// ── internal/tagger ──────────────────────────────────────────────────────────

// WriteTagsCmd performs an atomic tag write for a single file.
// Returns TagWriteCompleteMsg or TagWriteErrorMsg.
func WriteTagsCmd(path string, edits map[string]string) tea.Cmd

// WriteBatchTagsCmd performs atomic tag writes for multiple files sequentially.
// Returns a chain of BatchTagWriteProgressMsg, ending with BatchTagWriteCompleteMsg.
// Uses Pattern A. Halts on first error but reports all results in the terminal Msg.
func WriteBatchTagsCmd(writes []tagger.TagWrite) tea.Cmd


// ── internal/search ──────────────────────────────────────────────────────────

// FuzzySearchCmd runs the fuzzy search against the in-memory index.
// Dispatched as a Cmd on every keystroke to keep Update pure.
// Returns SearchResultsMsg.
func FuzzySearchCmd(index *search.Index, query string) tea.Cmd


// ── internal/messages ────────────────────────────────────────────────────────

// TickCmd schedules a single 1-second tick for the Now Playing bar.
// Update re-schedules this while PlayerState is Playing.
// Returns TickMsg.
func TickCmd() tea.Cmd
```

---

## 6. Go Type Definitions

All types below are defined in `internal/messages`. Supporting domain types are defined in `internal/model` and imported here.

```go
package messages

import (
    "time"

    tea "github.com/charmbracelet/bubbletea"

    "github.com/<user>/waveshell/internal/config"
    "github.com/<user>/waveshell/internal/model"
)

// ── Config ───────────────────────────────────────────────────────────────────

type ConfigLoadedMsg    struct{ Config config.Config }
type ConfigErrorMsg     struct{ Err error }
type ConfigWrittenMsg   struct{}
type ConfigWriteErrorMsg struct{ Err error }

// ── Scanner ──────────────────────────────────────────────────────────────────

type ScanStartedMsg struct{}

type ScanProgressMsg struct {
    Processed   int
    Total        int     // -1 until walk phase completes
    CurrentPath string
    NextCmd      tea.Cmd
}

type ScanFileErrorMsg struct {
    Path    string
    Err     error
    NextCmd tea.Cmd
}

type ScanCompleteMsg struct {
    Processed int
    Skipped   int
}

// ── Database ─────────────────────────────────────────────────────────────────

type ArtistListResultMsg struct {
    Artists []model.Artist
}

type TagSliceResultMsg struct {
    Mode   model.BrowseMode
    Values []string
}

type AlbumListResultMsg struct {
    Mode   model.BrowseMode
    Key    string
    Albums []model.Album
}

type TrackListResultMsg struct {
    AlbumID int64
    Tracks  []model.Track
}

type DBErrorMsg struct {
    Op    string
    Err   error
    Fatal bool
}

// ── mpv Process ──────────────────────────────────────────────────────────────

type MPVReadyMsg struct {
    Events <-chan tea.Msg
}

type MPVNotFoundMsg struct{}

type MPVConnectionLostMsg struct {
    Err error
}

// ── mpv Events ───────────────────────────────────────────────────────────────

type PlaybackStateChangedMsg struct {
    State model.PlaybackState
}

type TimePositionChangedMsg struct {
    PositionSec float64
}

type DurationChangedMsg struct {
    DurationSec float64
}

type VolumeChangedMsg struct {
    Volume int
}

type TrackEndedMsg struct {
    Reason model.TrackEndReason
}

// ── Tick ─────────────────────────────────────────────────────────────────────

type TickMsg struct {
    Time time.Time
}

func TickCmd() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return TickMsg{Time: t}
    })
}

// ── Search ───────────────────────────────────────────────────────────────────

type SearchResultsMsg struct {
    Query   string
    Results model.SearchResults
}

// ── Tag Editor ───────────────────────────────────────────────────────────────

type TagWriteCompleteMsg struct {
    Path string
}

type TagWriteErrorMsg struct {
    Path           string
    Err            error
    OriginalIntact bool
}

type BatchTagWriteProgressMsg struct {
    Path      string
    Succeeded bool
    Err       error
    Done      int
    Total     int
    NextCmd   tea.Cmd
}

type BatchTagWriteCompleteMsg struct {
    Succeeded int
    Failed    int
    Errors    []BatchWriteError
}

type BatchWriteError struct {
    Path string
    Err  error
}

// ── Play History ─────────────────────────────────────────────────────────────

type PlayHistoryWrittenMsg struct {
    TrackID int64
}

type PlayHistoryWriteErrorMsg struct {
    TrackID int64
    Err     error
}
```

### Supporting enums (internal/model)

These are referenced by Msgs above and belong in `internal/model`, not `internal/messages`:

```go
package model

type PlaybackState int

const (
    PlaybackStateStopped PlaybackState = iota
    PlaybackStatePlaying
    PlaybackStatePaused
)

type TrackEndReason int

const (
    TrackEndedEOF     TrackEndReason = iota  // natural end of file
    TrackEndedStopped                         // explicit stop command sent
    TrackEndedError                           // mpv reported a playback error
)

type BrowseMode string

const (
    BrowseModeArtist   BrowseMode = "artist"
    BrowseModeLabel    BrowseMode = "label"
    BrowseModeGenre    BrowseMode = "genre"
    BrowseModeYear     BrowseMode = "year"
    BrowseModeGrouping BrowseMode = "grouping"
    BrowseModePlaylist BrowseMode = "playlist"  // post-MVP
)
```

---

## 7. Quick-Reference Table

| Msg type                   | Produced by         | Pattern            | Milestone | Fatal?  |
| -------------------------- | ------------------- | ------------------ | --------- | ------- |
| `ConfigLoadedMsg`          | `internal/config`   | One-shot           | M1        | —       |
| `ConfigErrorMsg`           | `internal/config`   | One-shot           | M1        | **Yes** |
| `ConfigWrittenMsg`         | `internal/config`   | One-shot           | M3        | —       |
| `ConfigWriteErrorMsg`      | `internal/config`   | One-shot           | M3        | No      |
| `ScanStartedMsg`           | `internal/scanner`  | —                  | M2        | —       |
| `ScanProgressMsg`          | `internal/scanner`  | **Recursive**      | M2        | —       |
| `ScanFileErrorMsg`         | `internal/scanner`  | **Recursive**      | M2        | No      |
| `ScanCompleteMsg`          | `internal/scanner`  | Recursive terminal | M2        | —       |
| `ArtistListResultMsg`      | `internal/db`       | One-shot           | M3        | —       |
| `TagSliceResultMsg`        | `internal/db`       | One-shot           | M3        | —       |
| `AlbumListResultMsg`       | `internal/db`       | One-shot           | M3        | —       |
| `TrackListResultMsg`       | `internal/db`       | One-shot           | M3        | —       |
| `DBErrorMsg`               | `internal/db`       | One-shot           | M2        | Varies  |
| `MPVReadyMsg`              | `internal/mpv`      | One-shot           | M4        | —       |
| `MPVNotFoundMsg`           | `internal/mpv`      | One-shot           | M4        | **Yes** |
| `MPVConnectionLostMsg`     | `internal/mpv`      | One-shot           | M4        | No      |
| `PlaybackStateChangedMsg`  | `internal/mpv`      | **Subscription**   | M4        | —       |
| `TimePositionChangedMsg`   | `internal/mpv`      | **Subscription**   | M4        | —       |
| `DurationChangedMsg`       | `internal/mpv`      | **Subscription**   | M4        | —       |
| `VolumeChangedMsg`         | `internal/mpv`      | **Subscription**   | M4        | —       |
| `TrackEndedMsg`            | `internal/mpv`      | **Subscription**   | M4        | —       |
| `TickMsg`                  | `internal/messages` | Recurring          | M4        | —       |
| `SearchResultsMsg`         | `internal/search`   | One-shot           | M6        | —       |
| `TagWriteCompleteMsg`      | `internal/tagger`   | One-shot           | M7        | —       |
| `TagWriteErrorMsg`         | `internal/tagger`   | One-shot           | M7        | No      |
| `BatchTagWriteProgressMsg` | `internal/tagger`   | **Recursive**      | M7        | —       |
| `BatchTagWriteCompleteMsg` | `internal/tagger`   | Recursive terminal | M7        | —       |
| `PlayHistoryWrittenMsg`    | `internal/db`       | One-shot           | M5        | —       |
| `PlayHistoryWriteErrorMsg` | `internal/db`       | One-shot           | M5        | No      |

**Fatal** means the Msg renders a blocking error dialog and halts normal input routing. **No** means the Msg is surfaced as a status bar toast and the app continues. **—** means informational; no error handling required.
