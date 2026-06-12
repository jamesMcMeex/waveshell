# waveshell — Development Plan

_Generated from codebase + documentation research. Current version: 0.2.0_

---

## State of the Roadmap

| Milestone | Status | Notes |
|---|---|---|
| M1 Foundation | Complete | Config, logging, Makefile, env interpolation |
| M2 Scanner & SQLite | Complete | WalkDir, dhowden/tag, binary inspection, WAL mode, incremental rescan |
| M3 TUI Browser | ~90% | Responsive breakpoints missing, Grouping mode unreachable, no multi-select |
| M4 Playback Engine | ~85% | Prev/next re-uses cursor track (no real queue), no gapless playback |
| **M5 Queue Management** | **Not started** | Queue overlay, repeat modes, play_history table |
| **M6 Search** | **Not started** | sahilm/fuzzy not in go.mod, no search overlay |
| **M7 Metadata Panel** | **Not started** | No info panel, no tag editing, no internal/tagger package |

---

## Key Findings

### charmbracelet/bubbles — absent
`charmbracelet/bubbles` is not in `go.mod` and not imported anywhere, despite the bubbletea skill
listing it as a required dependency. Every list, scroll, progress bar, and input field in the app
is hand-rolled, leading to duplicated scroll/cursor management across all three panes.

### charmbracelet/lipgloss — partially used
`lipgloss` v1.1.0 is present and used in `theme.go` and `view.go`. However, styles are created
fresh via `lipgloss.NewStyle()` on every render call rather than as package-level variables,
causing unnecessary allocations on every frame.

### Bubbles components that should replace hand-rolled code

| Component | What it replaces | Impact |
|---|---|---|
| `bubbles/list` | Three panes each copy-paste cursor+offset int scroll logic (~150 lines × 3) | High |
| `bubbles/viewport` | Help overlay has manual `ScrollOffset int` + line-slicing (~30 lines) | High |
| `bubbles/progress` | `renderProgressBar` is a 15-line `strings.Repeat("█"/"─")` hack | Medium |
| `bubbles/spinner` | Scan indicator is a static `"Scanning: X/Y"` text counter | Low–Medium |
| `bubbles/textinput` | Needed for M6 search — `SearchState.Query` is a placeholder today | Medium (future) |

For the three panes, the approach is a **thin wrapper**: use `bubbles/list` for cursor and scroll
management but render rows ourselves to preserve the `▶` cursor style and lipgloss theming.

### Skill gaps in current code

**bubbletea skill violations:**
- `bubbles` not in go.mod (Critical)
- All keyboard handling inlined in `update.go` — should live in `update_keyboard.go`
- `update.go` is 830 lines; skill maximum is 800, target <500
- Lipgloss styles created fresh per render — should be package-level vars in `styles.go`
- Text truncation uses raw byte slicing (`line[:inner]`) — not Unicode-safe; drops multi-byte runes

**golang-pro skill violations:**
- Keyboard bindings are hardcoded strings — a `// TODO: wire keybindings from cfg.Keybindings`
  comment already acknowledges this in `update.go`
- No `context.Context` on DB query functions or mpv IPC calls
- Missing godoc on exported symbols (`FormatBadge`, `ResolveTheme`, etc.)

---

## Execution Plan

### Phase 0 — Codebase tightening (no new features, pure debt reduction)

> Aligns the codebase with skill recommendations before building on top of it.

1. Add `charmbracelet/bubbles` to `go.mod`
2. Extract Lipgloss styles to `internal/update/styles.go` as package-level vars
3. Split `update.go` → `update.go` + `update_keyboard.go` (target: both files under 500 lines)
4. Fix Unicode-unsafe truncation → proper `truncateString(s string, maxLen int) string` with `…`
5. Wire `cfg.Keybindings` — remove hardcoded key strings (the TODO is already there)
6. Add `context.Context` to `db/commands.go` query funcs
7. Add godoc comments to all exported symbols

### Phase 1 — Bubbles integration (replaces hand-rolled components)

> Replace duplicated scroll/progress/spinner implementations with Bubbles components.

8. Replace 3-pane manual scroll with a thin `bubbles/list`-backed wrapper component
   - Cursor and scroll managed by `list.Model`; row rendering stays custom (lipgloss theming)
9. Replace help overlay scroll with `bubbles/viewport`
10. Replace `renderProgressBar` with `bubbles/progress`
11. Add `bubbles/spinner` to the scan status bar

### Phase 2 — M5 Queue Management

> First unbuilt milestone. `QueueState` struct and `RepeatMode` enum already exist in the model.

12. Queue overlay view (`q` key) — shows current queue, highlights playing track
13. `Space` / `Shift+Space` / `a` — add track / add as next / add album to queue
14. `J` / `K` — reorder queue entries; `x` — remove; `c` — clear queue
15. Playback mode cycling (`m`): StopAtEnd → RepeatQueue → RepeatTrack → Shuffle
16. Fix `handlePrevTrack` / `handleNextTrack` to navigate real queue (not cursor track)
17. Add `play_history` SQLite table (migration v2)
18. Write play-history entry at 50% completion via async Cmd

### Phase 3 — M6 Search

> Full-text fuzzy search across the library.

19. Add `sahilm/fuzzy` to `go.mod`
20. Search overlay (`/` key) with `bubbles/textinput` for query input
21. `SearchResultsMsg` — results grouped by Artists / Albums / Tracks with relevance ranking
22. `label:`, `genre:`, `year:` filter prefix syntax
23. `Ctrl+Enter` — persist results as a Search Results view mode

### Phase 4 — M7 Metadata Panel & Tag Editing

> Metadata inspection and in-place tag editing. Never mutates files without explicit confirmation.

24. Info panel overlay (`i` key) — sections: Basic, Audio, Loudness, Artwork, File, Raw Tags
25. Tag editing (`e`) — `bubbles/textinput` fields per editable tag
26. `internal/tagger` package — atomic write via `.tmp` + `os.Rename`
27. Before/after diff view before write confirmation dialog
28. Batch tag editing for multi-selected tracks

---

## M3/M4 Remaining Gaps (fix alongside Phase 0–1)

- **Grouping browse mode** — defined in `model.BrowseMode` but absent from `browseModes()` in
  `update.go`; add it to the slice and wire `QueryTagSliceCmd` for grouping
- **Responsive breakpoints** — always renders three panes regardless of terminal width; add
  2-pane (< ~110 cols) and 1-pane (< ~70 cols) layout modes per `INTERACTION_DESIGN.md`
- **Action dialog** — `Enter` in the tracks pane currently plays immediately; it should open a
  small action dialog (Play Now / Add to Queue / Add as Next / Info) per `INTERACTION_DESIGN.md`
- **Multi-select** — `Space` toggle, `V` select all, `Esc` clear (prerequisite for batch tag edit)
