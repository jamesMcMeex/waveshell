# waveshell

A keyboard-driven, privacy-first terminal music player for local audio libraries.

## Status

Milestone 1 — Project Foundation. Config loading, structured logging, build tooling.
See `docs/WAVESHELL_PRD.md` for the full milestone plan.

## Dependencies

- **Go 1.22+** — build toolchain
- **mpv** — runtime dependency for audio playback (not required for M1)
- **golangci-lint** — `brew install golangci-lint` (for `make lint`)

## Quick start

```bash
# Build
make build

# Run (prints confirmation; no TUI yet)
./waveshell

# Run with config path
./waveshell --config ~/.config/waveshell/config.toml

# Print version
./waveshell --version
```

## Commands

| Command      | Description                                   |
| ------------ | --------------------------------------------- |
| `make build` | `go build -o waveshell ./cmd/waveshell`       |
| `make test`  | `go test ./... -race -coverprofile=coverage.out` |
| `make lint`  | `golangci-lint run`                           |
| `make cov`   | `go tool cover -html=coverage.out`            |
| `make run`   | Build and print startup message               |

## First setup

1. Install Go 1.22+: `brew install go`
2. Install golangci-lint: `brew install golangci-lint`
3. Run `go mod tidy` to download dependencies and generate `go.sum`
4. Run `make build` to compile
5. Create `~/.config/waveshell/config.toml` (optional — all defaults apply)

## Configure

Config lives at `~/.config/waveshell/config.toml` (resolved via `os.UserConfigDir()` —
`~/.config/` on Linux, `~/Library/Application Support/` on macOS). Missing file is
not an error — all defaults apply. See `docs/CONFIG.md` for the full reference.

## Project

- `docs/` — full specification: PRD, interaction design, schema, config, msgs, mpv IPC, package layout
- `AGENTS.md` — agent instructions and commands

## Licence

MIT
