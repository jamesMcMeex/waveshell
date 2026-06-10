# waveshell

waveshell is a keyboard-driven, privacy-first terminal music player for local audio libraries. Written in Go with BubbleTea (Elm Architecture).

## Stack

- **Language:** Go 1.22+
- **TUI:** BubbleTea + Lipgloss + Bubbles
- **Audio:** mpv (IPC via Unix socket)
- **Metadata:** dhowden/tag
- **Database:** modernc.org/sqlite (pure Go, zero CGo)
- **Config:** TOML (BurntSushi/toml)
- **Album art:** chafa (subprocess)
- **Audio formats scanned:** flac, alac, m4a, aiff, aif, mp3, wav, ogg

## Architecture

Elm Architecture: `Model` (state) → `Msg` (events) → `Model.Update(msg)` → `(Model, tea.Cmd)` → `Model.View()` → string.

All business logic lives in `Update` — pure, no IO — trivially unit-testable.

State is one struct, defined in `internal/update`:

```go
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

Two leaf packages are safe to import from anywhere: `internal/model` (domain types) and `internal/messages` (all Msg types). Everything else imports `internal/messages` to produce or consume Msgs — no cycles possible.

Two async patterns dominate: **Recursive Cmd** (scanner, batch writes — each step returns the next Cmd) and **Subscription Cmd** (mpv events — goroutine feeds a channel, Update re-schedules on receipt). See `docs/MSGS.md` §3.

## Key principles

- Keyboard-first — every action reachable without mouse
- Offline always — no network calls, no telemetry, ever
- Never mutate audio files silently — writes require explicit confirmation, use atomic `.tmp` + `os.Rename`
- Database is derived data — SQLite index is a cache, not source of truth
- `Esc` always dismisses the current overlay/dialog without side effects

## Docs (under `docs/`)

- `WAVESHELL_PRD.md` — full product spec: vision, milestones, metadata reference
- `INTERACTION_DESIGN.md` — UI behaviour: navigation, focus, browse modes, component patterns
- `SCHEMA.md` — SQLite schema DDL, indexes, migration strategy, query patterns
- `CONFIG.md` — full config.toml key reference, validation rules, Go struct hierarchy, built-in themes
- `MSGS.md` — all custom tea.Msg types, Cmd signatures, async patterns, Go type definitions
- `MPV_IPC.md` — mpv JSON IPC protocol, commands, events, Player interface, mock socket server
- `PACKAGE_LAYOUT.md` — directory tree, per-package responsibilities, import constraints

## Skills (`.agents/skills/`)

Skills are available for domain-specific coding tasks. Load the relevant skill when working in these areas:
- `bubbletea` — TUI development with BubbleTea, Lipgloss, and Bubbles components
- `golang-pro` — idiomatic Go patterns, concurrency, interfaces, testing
- `sqlite-database-expert` — SQLite schema design, migrations, FTS5, WAL mode

## Agent commands (`.opencode/commands/`)

| Command | Purpose |
|---|---|
| `arch` | Architecture discussion via Claude Sonnet 4.6 |
| `review` | Code review of uncommitted changes |
| `debug` | Read-only failure investigation |
| `scaffold` | Scaffold a new BubbleTea component |
| `test` | Run tests and fix failures (TDD) |
| `changelog` | Draft a CHANGELOG entry from git history |
| `commit` | Generate a conventional commit message from staged changes |

## Makefile targets

- `make test` — `go test ./... -race -coverprofile=coverage.out`
- `make lint` — `golangci-lint run`
- `make build` — `go build -o waveshell ./cmd/waveshell`
- `make coverage` — `go tool cover -html=coverage.out`
