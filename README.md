# waveshell

A keyboard-driven, privacy-first terminal music player for local audio libraries.

## Status

Pre-development. Initial milestone (M1 — Project Foundation) is the starting point.

## Features

- Three-pane library browser (Artists → Albums → Tracks)
- Browse modes: artist, label, genre, year, grouping
- mpv-backed playback with gapless support
- SQLite index with incremental rescan
- Configurable columns, colour themes, and keybindings
- Tag editing with atomic writes and before/after diff
- Full offline — no network, no telemetry, no surprises

See `docs/WAVESHELL_PRD.md` for the full milestone plan.

## Dependencies

- **Go 1.22+** — build toolchain
- **mpv** — runtime dependency for audio playback
- **chafa** — optional, for terminal album art (post-MVP)

## Quick start

```bash
# Build
make build

# Run
./waveshell

# With direct-launch path
./waveshell ~/Music/Artist/Album
```

## Configure

Config lives at `~/.config/waveshell/config.toml`. Missing file is not an error — all defaults apply. See `docs/CONFIG.md` for the full reference.

## Commands

| Command       | Description                                   |
| ------------- | --------------------------------------------- |
| `make build`  | `go build -o waveshell ./cmd/waveshell`       |
| `make test`   | `go test ./... -race -coverprofile=coverage.out` |
| `make lint`   | `golangci-lint run`                           |
| `make cov`    | `go tool cover -html=coverage.out`            |

## Project

- `docs/` — full specification: PRD, interaction design, schema, config, msgs, mpv IPC, package layout
- `AGENTS.md` — agent instructions and commands

## Licence

MIT
