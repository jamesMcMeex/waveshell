# waveshell — mpv IPC Reference

> **Status:** Pre-development
> **Scope:** Canonical specification for the mpv JSON IPC protocol as used by waveshell — every JSON command the app sends, every event it handles, the `internal/mpv` Player interface, and the mock socket server test pattern.
> **Last updated:** June 2026

---

## Table of Contents

1. [Socket Connection](#1-socket-connection)
2. [Protocol Format](#2-protocol-format)
3. [Commands](#3-commands)
4. [Events](#4-events)
5. [Error Handling](#5-error-handling)
6. [Player Interface](#6-player-interface)
7. [Mock Socket Server](#7-mock-socket-server)

---

## 1. Socket Connection

mpv is launched as a subprocess with the following flags:

```
mpv --input-ipc-server=/tmp/waveshell.sock --idle=yes --no-terminal
```

The socket path is configurable via `config.toml` `[player].mpv_socket` (default: `/tmp/waveshell.sock`).

The connection is a **Unix domain socket** (not TCP). The `internal/mpv` package opens the socket immediately after mpv confirms it is ready. Both directions use newline-delimited JSON — one JSON object per line, terminated by `\n`.

The socket path directory must exist before mpv starts. The `internal/mpv` package removes any existing socket file at the configured path before launching mpv, ensuring a clean state.

---

## 2. Protocol Format

### Request (waveshell → mpv)

```json
{ "command": ["command_name", arg1, arg2], "request_id": 1 }
```

- `command` — an array where the first element is the command name and subsequent elements are positional arguments
- `request_id` — an integer that mpv echoes back in the response, used to correlate requests with responses. waveshell uses an incrementing counter starting at 1.

### Response (mpv → waveshell)

```json
{ "error": "success", "data": <value>, "request_id": 1 }
```

- `error` — always a string. `"success"` means no error; any other value is an error description.
- `data` — the return value, type varies by command. Absent for void commands.
- `request_id` — echoes the value from the request.

mpv responses are not guaranteed to arrive in request order. The `request_id` field is the sole correlation mechanism.

---

## 3. Commands

All commands below are sent as the `command` array inside a request object. The `request_id` is omitted from the examples for brevity but is always present.

### 3.1 `loadfile` — load and play a file

```json
{ "command": ["loadfile", "/path/to/track.flac", "replace"] }
```

Arguments:
- `path` — absolute filesystem path to the audio file
- `mode` — always `"replace"` (replaces the current file; append and insert modes are not used)

Response: `{ "error": "success" }` or `{ "error": "..." }`.

### 3.2 `set_property pause` — pause / resume

```json
{ "command": ["set_property", "pause", true] }
{ "command": ["set_property", "pause", false] }
```

- `true` pauses playback
- `false` resumes playback

### 3.3 `set_property volume` — set volume

```json
{ "command": ["set_property", "volume", 75] }
```

Volume range is 0–100. Waveshell enforces this range in `internal/update` before sending.

### 3.4 `seek` — seek within current track

```json
{ "command": ["seek", 15.5, "relative"] }
{ "command": ["seek", 120.0, "absolute"] }
```

- Relative seek: positive values seek forward, negative values seek backward
- Absolute seek: seeks to a position in seconds from the start of the track

### 3.5 `stop` — stop playback

```json
{ "command": ["stop"] }
```

Stops playback and unloads the current file. mpv emits an `end-file` event with reason `"stop"`.

### 3.6 `get_property time-pos` — query current position

```json
{ "command": ["get_property", "time-pos"] }
```

Response `data`: `float64` (seconds), or `null` if no file is loaded.

### 3.7 `get_property duration` — query track duration

```json
{ "command": ["get_property", "duration"] }
```

Response `data`: `float64` (seconds), or `null` if the duration is not yet known.

### 3.8 `observe_property` — subscribe to property updates

```json
{ "command": ["observe_property", 1, "time-pos"] }
{ "command": ["observe_property", 2, "pause"] }
{ "command": ["observe_property", 3, "duration"] }
{ "command": ["observe_property", 4, "volume"] }
```

Once registered, mpv emits `property-change` events (see §4.1) whenever the property value changes. The `id` is an integer chosen by waveshell and must be unique per property.

Observation IDs:
| Property   | ID | Purpose                                          |
| ---------- | -- | ------------------------------------------------ |
| `time-pos` | 1  | Update the now-playing progress bar and ticker    |
| `pause`    | 2  | Track play/pause state changes (user or EOF)      |
| `duration` | 3  | Capture duration when it becomes available        |
| `volume`   | 4  | Respond to external volume changes (e.g. via mpv OSC) |

Waveshell does not use `unobserve_property`. Observations are registered once and remain active for the lifetime of the mpv process.

---

## 4. Events

mpv pushes events unprompted over the socket. Events are JSON objects with an `"event"` field; they have no `request_id`.

### 4.1 `property-change`

```json
{ "event": "property-change", "id": 1, "name": "time-pos", "data": 42.5 }
{ "event": "property-change", "id": 2, "name": "pause", "data": true }
```

Fields:
- `id` — the observation ID from the `observe_property` request
- `name` — the property name (for verification)
- `data` — the new value; type depends on the property (`float64` for `time-pos`, `bool` for `pause`, etc.)

Waveshell maps each observation ID to a typed Msg:

| ID | Event                           | Msg                      |
| -- | ------------------------------- | ------------------------ |
| 1  | `property-change` `time-pos`    | `TimePositionChangedMsg` |
| 2  | `property-change` `pause`       | `PlaybackStateChangedMsg` |
| 3  | `property-change` `duration`    | `DurationChangedMsg`     |
| 4  | `property-change` `volume`      | `VolumeChangedMsg`       |

### 4.2 `end-file`

```json
{ "event": "end-file", "reason": "eof" }
{ "event": "end-file", "reason": "stop" }
{ "event": "end-file", "reason": "error" }
```

Reason values waveshell handles:
- `"eof"` — track finished naturally. Triggers queue advance to the next track.
- `"stop"` — user explicitly stopped playback. No queue advance.
- `"error"` — mpv could not load or play the file. The error details may be in a subsequent log message but are not reliably available in the IPC event. Waveshell logs the event and advances the queue (or stops if the queue is empty).

### 4.3 `start-file`

```json
{ "event": "start-file" }
```

Emitted when mpv begins loading a new file. Waveshell does not currently take action on this event but it is documented for future use (e.g., resetting the time-position display to 0:00 while duration is unknown).

---

## 5. Error Handling

### Error response values

| Error string                       | Severity   | Handling                                      |
| ---------------------------------- | ---------- | --------------------------------------------- |
| `"success"`                        | No error   | Proceed normally                              |
| `"invalid parameter"`              | Fatal      | Programming error — log and surface in status bar |
| `"property unavailable"`           | Ignorable  | Property is not available for this file (e.g. `duration` before load completes). Retry later. |
| `"access denied"`                  | Fatal      | Socket permissions misconfiguration — log and surface |
| `"unknown command"`                | Fatal      | Programming error — log and surface           |

Any unrecognised error string is treated as fatal and surfaced in the status bar.

### Connection loss

A socket read returning `io.EOF` or a dial failure means the mpv process has died or is unreachable. Waveshell handles this by:
1. Setting `PlayerState.Playing = false` and clearing the now-playing track
2. Emitting an `MPVConnectionLostMsg`
3. Setting a flag to attempt reconnection on the next user playback action

### mpv not found at startup

If the `mpv` binary is not found on `$PATH` at startup, waveshell emits `MPVNotFoundMsg` (fatal) and displays an error dialog prompting the user to install mpv. The application does not start playback without mpv; all other features (browse, search, tag editing) remain available.

---

## 6. Player Interface

The `internal/mpv` package exposes a `Player` interface. This is the seam that `internal/update` depends on — the concrete mpv implementation and any test mock both satisfy it.

```go
// Package mpv provides the Player interface and its mpv-backed implementation.
package mpv

// Player is the interface that internal/update depends on for playback control.
// Property observation events arrive via the subscription channel (see MSGS.md §3.2),
// not through this interface.
type Player interface {
    LoadFile(path string) error
    Play() error
    Pause() error
    Stop() error
    SeekRelative(seconds float64) error
    SeekAbsolute(seconds float64) error
    SetVolume(vol int) error
}
```

Method semantics:

| Method             | Behaviour                                                                 |
| ------------------ | ------------------------------------------------------------------------- |
| `LoadFile(path)`   | Sends `loadfile` command. Returns error if path does not exist or mpv rejects it. |
| `Play()`           | Sends `set_property pause false`. No-op if already playing.               |
| `Pause()`          | Sends `set_property pause true`. No-op if already paused.                 |
| `Stop()`           | Sends `stop` command. Clears current file in mpv.                         |
| `SeekRelative(s)`  | Sends `seek` with `"relative"`. Positive = forward, negative = backward.  |
| `SeekAbsolute(s)`  | Sends `seek` with `"absolute"`. Clamped to `[0, duration]` by mpv.        |
| `SetVolume(vol)`   | Sends `set_property volume`. Clamped to `[0, 100]` by waveshell before sending. |

### Connection / event lifecycle

The `Player` interface covers only imperative playback commands. The connection lifecycle and event dispatch are handled by the `mpv` package internals:

1. `mpv.Launch()` starts the mpv subprocess, connects to the Unix socket, registers property observations, and starts a goroutine that reads the socket, dispatches events to the BubbleTea subscription channel, and handles connection loss.
2. `mpv.Close()` sends `stop`, unregisters property observations (best-effort), closes the socket, and terminates the mpv process via `os.Process.Kill()`.

These are not part of the `Player` interface because they are lifecycle concerns, not playback commands. They are used by `internal/update` during app initialisation and teardown, not during normal playback interaction.

---

## 7. Mock Socket Server

For integration tests of the `internal/mpv` package, a mock socket server stands in for a real mpv process. It lives in `internal/mpv/mock_test.go` and is only compiled into test binaries.

### Pattern

```go
// mockServer listens on a Unix socket and responds to mpv IPC commands
// with configurable responses. It verifies that expected commands are received.
type mockServer struct {
    t        testing.TB
    path     string
    ln       net.Listener
    received []map[string]any         // all commands received, in order
    responses []map[string]any         // responses to send, consumed in order
    events   []map[string]any          // events to push at connection time
}

// newMockServer creates a mock server on a temp socket path.
func newMockServer(t testing.TB) *mockServer

// Close shuts down the listener.
func (s *mockServer) Close()

// NextCommand returns the nth received command (0-indexed), failing the test
// if fewer than n+1 commands have been received.
func (s *mockServer) NextCommand(n int) map[string]any
```

### Usage in tests

```go
func TestLoadFile(t *testing.T) {
    s := newMockServer(t)
    defer s.Close()

    s.responses = []map[string]any{
        {"error": "success", "request_id": 1},
    }

    p := NewPlayer(s.path) // Player connects to mock socket
    defer p.Close()

    err := p.LoadFile("/test/path.flac")
    assert.NoError(t, err)

    cmd := s.NextCommand(0)
    assert.Equal(t, "loadfile", cmd["command"].([]any)[0])
    assert.Equal(t, "/test/path.flac", cmd["command"].([]any)[1])
    assert.Equal(t, "replace", cmd["command"].([]any)[2])
}
```

### Pushing events

The mock server pushes events from its `events` slice after accepting a connection, simulating mpv's property-change subscriptions. This lets tests verify that `TimePositionChangedMsg` etc. are correctly dispatched to the subscription channel:

```go
s.events = []map[string]any{
    {"event": "property-change", "id": 1, "name": "time-pos", "data": 0.0},
    {"event": "property-change", "id": 2, "name": "pause", "data": false},
    {"event": "start-file"},
}
```

### Error simulation

To test error handling, set a response with an error string:

```go
s.responses = []map[string]any{
    {"error": "invalid parameter", "request_id": 1},
}

err := p.LoadFile("/bad/path.flac")
assert.Error(t, err)
assert.Contains(t, err.Error(), "invalid parameter")
```

### Connection loss simulation

Close the listener during a test to simulate mpv process death. The player's read goroutine will receive `io.EOF`, triggering reconnection logic.
