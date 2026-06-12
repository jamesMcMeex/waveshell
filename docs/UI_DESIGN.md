# waveshell — UI Design

> **Companion to:** `docs/INTERACTION_DESIGN.md`
> **Scope:** Visual design of every screen, overlay, and dialog. ASCII mockups for all major states, component-to-use-case mapping, tab bar design, and the formal Contextual Dialog rule.
> This document answers _what things look like_. `INTERACTION_DESIGN.md` answers _how they behave_.
> **Status:** Draft — under collaborative refinement.
> **Last updated:** June 2026

---

## Table of Contents

1. [Design Principles](#1-design-principles)
2. [Keybinding Rules](#2-keybinding-rules)
3. [Component Library](#3-component-library)
4. [Layer System](#4-layer-system)
5. [Application Shell & Tab Bar](#5-application-shell--tab-bar)
6. [Now Playing Bar](#6-now-playing-bar)
7. [Layer 0A — Browser View](#7-layer-0a--browser-view)
8. [Layer 0B — All Tracks View](#8-layer-0b--all-tracks-view)
9. [Layer 0C — Player View](#9-layer-0c--player-view)
10. [Layer 1 — Overlays](#10-layer-1--overlays)
    - [10.1 Search Overlay](#101-search-overlay)
    - [10.2 Queue Overlay](#102-queue-overlay)
    - [10.3 Info Panel Overlay](#103-info-panel-overlay)
    - [10.4 Help Overlay](#104-help-overlay)
11. [Layer 2 — Dialogs](#11-layer-2--dialogs)
    - [11.1 Action Dialog](#111-action-dialog)
    - [11.2 Browse Mode Picker](#112-browse-mode-picker)
    - [11.3 Column Manager](#113-column-manager)
    - [11.4 Sort Picker](#114-sort-picker)
    - [11.5 Write Confirmation Dialog](#115-write-confirmation-dialog)
12. [Responsive Layout Variants](#12-responsive-layout-variants)
13. [Contextual Dialog Pattern Reference](#13-contextual-dialog-pattern-reference)
14. [Open Keybinding Conflicts](#14-open-keybinding-conflicts)

---

## 1. Design Principles

### 1.1 The Layer Contract (hard rule)

Every element of the UI lives at exactly one layer level at all times:

- **Layer 0** — the base view. Always visible. Two states are available simultaneously (Browser, All Tracks); a third (Player) is shown when navigated to. Only one Layer 0 state is active at a time.
- **Layer 1** — overlays. Rendered on top of Layer 0. The base layer remains visible but **dimmed** in the background; it does not receive input. At most one overlay is open at a time.
- **Layer 2** — dialogs. Small floating boxes rendered on top of whatever is beneath them (Layer 0 or Layer 1). At most one dialog is open at a time.

`Esc` always moves the user exactly one layer up without any side effect. This is unconditional.

**Layer 0 dimming when a Layer 1 overlay is open:** The base layer is rendered with reduced foreground brightness (e.g. `muted` palette applied to all text) while an overlay is active. This focuses attention on the overlay while preserving spatial context. Implementation: apply a dimming Lipgloss style to the base layer's rendered string before compositing via `rmhubbert/bubbletea-overlay`.

### 1.2 The Contextual Dialog Rule (hard rule)

> **All options menus, context menus, confirmations, and pickers must be rendered at Layer 2 (dialogs) or Layer 1 (overlays). Nothing contextual may be rendered inline within Layer 0.**

This means:
- Pressing `Enter` on a track opens an **Action Dialog (Layer 2)**, not inline playback.
- Browse mode selection opens a **Browse Mode Picker Dialog (Layer 2)**.
- Column configuration opens a **Column Manager Dialog (Layer 2)**.
- Confirmations (write diff, clear queue) are **Layer 2 dialogs**, never status bar toasts.
- Help is a **Layer 1 overlay** (full content, scrollable).

The user always knows they can press `Esc` to get back to exactly where they were.

### 1.3 Visual Hierarchy

Layers must be visually distinct at a glance:

| Layer | Border style          | Title                        | Background treatment           |
| ----- | --------------------- | ---------------------------- | ------------------------------ |
| 0     | `muted` (inactive pane) / `accent` (active pane) | `muted` uppercase | Normal |
| 1     | `accent` colour       | `accent` uppercase + padding | Layer 0 dimmed beneath         |
| 2     | `accent` colour       | `accent` uppercase           | Floating over L0 or L1         |

### 1.4 No Orphan State

Every context that opens (overlay or dialog) has a visible, consistent way to close it. Every Layer 1 overlay and Layer 2 dialog displays its dismiss hint in the key hint bar. Users are never left without an exit.

---

## 2. Keybinding Rules

These rules apply to all keybindings, including any future additions. Violations must be corrected before implementation. They are mirrored in `INTERACTION_DESIGN.md §9.1` and `§11`.

### Rule 1 — `A`–`Z` (uppercase) = letter-jump only

`Shift+A` through `Shift+Z` in any browser pane navigates to the first item whose name begins with that letter. No uppercase letter may ever be assigned to a command or function at the base layer. `J`/`K` are permitted inside dialogs and overlays for reorder operations, because letter-jump is a base-layer concept that does not apply within dialogs.

### Rule 2 — `a`–`z` (lowercase) = functions and actions only

Lowercase letters are for named actions: play, info, edit, sort, columns, search, etc. They must not be used for cursor movement, directional navigation, or letter-jump.

### Rule 3 — Cursor movement = arrow keys only

`↑` `↓` `Home` `End` `Ctrl+D` `Ctrl+U` handle all cursor and scroll movement. No alpha character (`a`–`z` or `A`–`Z`) may substitute for a directional key. This rule is unconditional.

### Rule 4 — `Ctrl+Letter` = configuration actions

Actions that open a persistent settings dialog (one that changes a global or per-mode configuration) use a `Ctrl+` prefix. This signals to the user that the action has lasting effect beyond the current session context.

| Ctrl binding | Action                  |
| ------------ | ----------------------- |
| `ctrl+b`     | Open Browse Mode Picker |
| `ctrl+t`     | Open Theme Picker       |
| `ctrl+a`     | Select all visible tracks |

### Rule 5 — `q` toggles the Queue overlay

`q` opens the Queue overlay from the base layer. All overlays and dialogs close exclusively with `Esc` — `q` is never a close/dismiss key. If any overlay is already open, `q` is a no-op. `Ctrl+C` is the only quit binding.

### Corrected keybindings summary

| Function                  | Previous (wrong)           | Corrected          |
| ------------------------- | -------------------------- | ------------------ |
| Open Column Manager       | `C` (uppercase)            | `c`                |
| Open Sort Picker          | `S` (uppercase)            | `s`                |
| Select all tracks         | `V` (uppercase)            | `ctrl+a`           |
| Toggle track selection    | `Space`                    | `v`                |
| Open Queue overlay        | `q` (was quit)             | `q` (queue toggle) |
| Quit                      | `q` / `Ctrl+C`             | `Ctrl+C` only      |
| Open Browse Mode          | `b` / `B`                  | `ctrl+b`           |
| Open Theme Picker         | `c` (cycle theme, removed) | `ctrl+t`           |
| Cursor up/down            | `i`/`↑`, `k`/`↓`           | `↑` / `↓` only     |
| Jump to top/bottom        | `t`/`Home`, `g`/`End`      | `Home` / `End` only |
| Pane focus left/right     | `h`/`l`, `Tab`/`Shift+Tab` | `←`/`→` (Tab/Shift+Tab as aliases) |
| Next / previous track     | `n` / `f`                  | `.` / `,`          |
| Play track                | `p`                        | `Space` (Tracks pane) |
| Enter (Left/Middle panes) | Select + move focus right  | Open Action Dialog (focus stays) |

---

## 3. Component Library

### Official `charmbracelet/bubbles`

| Component               | Use in waveshell                                              |
| ----------------------- | ------------------------------------------------------------- |
| `bubbles/textinput`     | Search input, tag edit fields, inline playlist name entry     |
| `bubbles/viewport`      | Queue overlay body, Info Panel body, Help overlay body        |
| `bubbles/progress`      | Playback progress bar — Now Playing bar and Player view       |
| `bubbles/stopwatch`     | Elapsed playback time (count-up, synced to mpv IPC position) |
| `bubbles/spinner`       | Library scan indicator in the status bar                     |
| `bubbles/help`          | Key hint bar — auto-generates from `key.Binding` definitions |
| `bubbles/key`           | All keybinding definitions; enables `config.toml` remapping  |

> **`stopwatch` not `timer`:** `bubbles/timer` counts down to zero; `bubbles/stopwatch` counts up. Elapsed playback position needs count-up. Use `stopwatch`, synced to mpv IPC position ticks.

### Community

| Component                          | Use in waveshell                                                     |
| ---------------------------------- | -------------------------------------------------------------------- |
| `evertras/bubble-table`            | Tracks pane (Browser), All Tracks view, Queue table in Player view   |
| `rmhubbert/bubbletea-overlay`      | Compositing Layer 1 overlays and Layer 2 dialogs over the base layer |

### Custom (Lipgloss, no external dependency)

| Component             | Description                                                                     |
| --------------------- | ------------------------------------------------------------------------------- |
| **Tab Bar**           | Layer 0 view switcher. Custom Lipgloss rendering — see §5.2.                    |
| Left / Middle panes   | Custom scrollable list with letter-jump and item counts. Not `bubbles/list`.    |
| Status Bar            | Single-line ambient info strip. Rendered as a styled Lipgloss string.           |

---

## 4. Layer System

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│  LAYER 2 — DIALOGS (floating, at most one at a time)                            │
│  ┌───────────────────────────────────────────────┐                              │
│  │  Action / Browse Mode / Column Manager /      │  Dismiss: Esc               │
│  │  Sort Picker / Write Confirm / Batch Confirm  │                              │
│  └───────────────────────────────────────────────┘                              │
│                                                                                 │
│  LAYER 1 — OVERLAYS (full-width, at most one at a time)                         │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  Search  /  Queue  /  Info Panel  /  Help                               │   │
│  │  (Layer 0 dimmed beneath; no input routed to it)                        │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                 │
│  LAYER 0 — BASE VIEWS (always visible, three states)                           │
│  ┌──────────────────────┐  ┌───────────────────────┐  ┌─────────────────────┐ │
│  │  0A: BROWSER         │  │  0B: ALL TRACKS       │  │  0C: PLAYER         │ │
│  │  3-pane hierarchy    │  │  Full-width flat table │  │  Immersive player   │ │
│  │  Left/Middle/Tracks  │  │  evertras/bubble-table │  │  Art + Queue table  │ │
│  └──────────────────────┘  └───────────────────────┘  └─────────────────────┘ │
│                                                                                 │
│  PERSISTENT CHROME (rendered below content, not a layer)                       │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  Tab Bar  ·  Status Bar  ·  Now Playing Bar (hidden in 0C)  ·  Hint Bar │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Application Shell & Tab Bar

### 5.1 Height allocation

```
TermHeight
  ─ 2  tab bar (label row + separator line) — hidden in Player view (0C)
  ─ 1  column header (Browser/All Tracks) or absent (Player)
  ─ 1  status bar
  ─ 2  now playing bar — absent in Player view (0C); Player uses this space
  ─ 1  hint bar (hidden if TermHeight < 10)
  ═ N  available content lines
```

### 5.2 Tab Bar Design

The tab bar is a custom Lipgloss component. It occupies 2 lines: one for the tab labels, one for the separator.

**Visual design (active = tab 1):**

```
 ┌─ 1 BROWSER ─┐   2 ALL TRACKS    3 PLAYER
 │              └───────────────────────────────────────────────────────────────────┐
```

**Visual design (active = tab 2):**

```
  1 BROWSER    ┌─ 2 ALL TRACKS ─┐   3 PLAYER
 ──────────────┘                 └──────────────────────────────────────────────────┐
```

**Visual design (active = tab 3):**

```
  1 BROWSER    2 ALL TRACKS    ┌─ 3 PLAYER ─────────────────────────────────────────┐
 ─────────────────────────────┘
```

**Styling rules:**

| State        | Lipgloss style                                                               |
| ------------ | ---------------------------------------------------------------------------- |
| Active tab   | Top + left + right border in `accent`, text `accent` bold, no bottom border  |
| Inactive tab | Plain text in `muted`, padding `0 1`, no border                              |
| Separator    | Full-width `─` in `muted` with the active tab's bottom-edge gap              |

**Implementation note:** The separator line is rendered as a string where the character columns beneath the active tab are blank (its open bottom), and `─` fills the rest. This is computed from the rendered widths of the tabs. The content area border's top-left corner connects to the right end of the active tab's open bottom. This is pure string manipulation — no external library required.

**Tab labels always show the numeric shortcut:**

```
 1 BROWSER   2 ALL TRACKS   3 PLAYER
```

The number is part of the label, not a separate prefix, so it reads naturally.

### 5.3 Full shell layout (≥ 120 cols, Browser active)

```
 ┌─ 1 BROWSER ─┐   2 ALL TRACKS    3 PLAYER
 │              └──────────────────────────────────────────────────────────────────┐
 │ ARTISTS (122)       │ ALBUMS (47)           │  #   TITLE                DUR  FMT│
 │ ─────────────────── │ ───────────────────── │  ──────────────────────────────── │
 │ Aphex Twin     127  │ Come to Daddy         │   1  Bucephalus...        3:52 FLAC│
 │▶Autechre        89  │▶Selected Ambient...   │  ▶2  Come to Daddy        5:24 FLAC│
 │ Boards of...    44  │ Amber                 │   3  Flim                 2:56 ALAC│
 │ Burial          31  │ LP5                   │   4  Milkman              3:19 FLAC│
 │ Clark           12  │ Oversteps             │   5  Fingerbib            3:16 FLAC│
 │ Goldie          23  │ Tri Repetae           │   6  Carn Marth           3:38 FLAC│
 │ LTJ Bukem       19  │                       │                                    │
 ├─────────────────────┴───────────────────────┴────────────────────────────────────┤
 │ By Artist  ·  Autechre  ·  6 tracks                                              │
 ├──────────────────────────────────────────────────────────────────────────────────┤
 │ ▓░▓░  ▶  Aphex Twin — Flim                    01:22 ████████████░░░░░░░░ 02:56  │
 │ ▓░░▓     Come to Daddy  ·  1997  ·  ALAC  ·  44.1 kHz  ·  16-bit    Vol 80%    │
 ├──────────────────────────────────────────────────────────────────────────────────┤
 │ [↵] action  [spc] play  [q] queue  [i] info  [v] select  [c] columns  [s] sort  [?]│
 └──────────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Now Playing Bar

A persistent 2-line bar rendered between the status bar and the hint bar. Always shown in Layer 0A (Browser) and Layer 0B (All Tracks). **Not shown in Layer 0C (Player)** — the Player view is the Now Playing view.

### 6.1 Full layout (≥ 120 cols, with album art)

Album art is rendered by `chafa` as a 4-column × 2-row block. Art zone shown only at `TermWidth ≥ 120`.

```
├──────────────────────────────────────────────────────────────────────────────────┤
│ ▓░▓░  ▶  Aphex Twin — Flim                  01:22 ████████████░░░░░░░░░░ 02:56  │
│ ▓░░▓     Come to Daddy  ·  1997  ·  ALAC  ·  44.1 kHz  ·  16-bit    Vol 80%    │
├──────────────────────────────────────────────────────────────────────────────────┤
```

**Zone breakdown:**

| Zone           | Content                                      | Component                |
| -------------- | -------------------------------------------- | ------------------------ |
| Art (≥120 only)| `chafa` 4×2 thumbnail                        | Subprocess output        |
| State icon     | `▶` playing / `⏸` paused / `·` stopped      | Lipgloss styled string   |
| Track info     | `Artist — Title` (truncated to fit)          | Lipgloss truncated       |
| Elapsed        | `01:22`                                      | Formatted from mpv IPC   |
| Progress bar   | Gradient fill, `accent` → `muted`            | `bubbles/progress`       |
| Remaining      | `02:56`                                      | Formatted from mpv IPC   |
| Album line     | `Album · Year · Format · Rate · Depth`       | Lipgloss `muted` string  |
| Volume         | `Vol 80%`                                    | Lipgloss string          |

### 6.2 Compact (80–119 cols, no art)

```
│ ▶  Aphex Twin — Flim              01:22 █████████░░░░░░ 02:56    Vol 80%      │
│    Come to Daddy  ·  1997  ·  ALAC  ·  44.1 kHz                 ▶ Normal     │
```

### 6.3 Minimal (< 80 cols)

```
│ ▶  Aphex Twin — Flim        01:22 ████░░░░░░ 02:56               │
│    ALAC · 44.1kHz · 16-bit                           Vol 80%      │
```

### 6.4 Idle state

```
│ ·  No track playing                                               │
│    Press [↵] on a track to play                      Vol 80%     │
```

---

## 7. Layer 0A — Browser View

The default view. Three panes: Left / Middle / Tracks. Pane labels are determined by the active browse mode. Active pane border uses `accent`; inactive panes use `muted`.

### 7.1 Pane layout (Artist mode, full width)

```
┌─────────────────────┬──────────────────────┬──────────────────────────────────────┐
│ ARTISTS (122)       │ ALBUMS (47)          │  #   TITLE               YEAR DUR FMT│
│ ─────────────────── │ ──────────────────── │  ──────────────────────────────────  │
│ Aphex Twin     127  │ Come to Daddy        │   1  Bucephalus...       1997 3:52 FLA│
│▶Autechre        89  │▶Selected Ambient...  │  ▶2  Come to Daddy       1997 5:24 FLA│
│ Boards of...    44  │ Amber                │   3  Flim                1997 2:56 ALA│
│ Burial          31  │ LP5                  │   4  Milkman             1997 3:19 FLA│
│ Clark           12  │ Oversteps            │   5  Fingerbib           1997 3:16 FLA│
│ Goldie          23  │ Tri Repetae          │   6  Carn Marth          1997 3:38 FLA│
│ LTJ Bukem       19  │                      │                                       │
└─────────────────────┴──────────────────────┴───────────────────────────────────────┘
```

### 7.2 Multi-select state

Selected tracks show `◆` in `accent`. The cursor moves freely through selected and unselected rows. Status bar and hint bar shift to batch context.

```
│  #   TITLE               YEAR DUR FMT │
│  ─────────────────────────────────── │
│   1  Bucephalus...       1997 3:52 FLA│
│◆ ▶2  Come to Daddy       1997 5:24 FLA│
│◆  3  Flim                1997 2:56 ALA│
│   4  Milkman             1997 3:19 FLA│
│◆  5  Fingerbib           1997 3:16 FLA│
```

Status bar: `By Artist · Autechre · 3 tracks selected`

Hint bar: `[v] toggle  [shift+↑/↓] extend range  [ctrl+a] select all  [↵] batch action  [e] edit tags  [esc] clear`

### 7.3 Playing track indicator

The currently playing track shows `▶` in `accent` in the `#` column. The cursor row uses a distinct highlight (reverse video or `accent` background row tint) so both states are visible simultaneously.

```
│   1  Bucephalus...       1997 3:52 FLA│  ← normal row
│  ▶   Come to Daddy       1997 5:24 FLA│  ← playing (▶ replaces #)
│  *3  Flim                1997 2:56 ALA│  ← cursor row (different highlight)
```

---

## 8. Layer 0B — All Tracks View

Full-width flat table of all library tracks. `evertras/bubble-table` handles rendering.

```
 1 BROWSER    ┌─ 2 ALL TRACKS ─┐   3 PLAYER
──────────────┘                 └─────────────────────────────────────────────────────┐
│  #     TITLE                        ARTIST              ALBUM           YEAR DUR FMT│
│  ─────────────────────────────────────────────────────────────────────────────────  │
│    1   Bucephalus Bouncing Ball     Aphex Twin          Come to Daddy   1997 3:52 FLA│
│   ▶    Come to Daddy               Aphex Twin          Come to Daddy   1997 5:24 FLA│
│    3   Flim                        Aphex Twin          Come to Daddy   1997 2:56 ALA│
│    4   Clipper                     Autechre            Incunabula       1994 4:48 FLA│
│    5   Bike                        Autechre            Incunabula       1994 6:21 FLA│
│    6   Basscad                     Autechre            Amber            1994 5:59 FLA│
│    7   Knife / Plug                Burial              Untrue           2007 4:59 FLA│
│    8   Archangel                   Burial              Untrue           2007 5:07 FLA│
│    9   Ghost Hardware              Burial              Untrue           2007 5:50 FLA│
│   10   Roygbiv                     Boards of Canada    Music Has...     1998 2:31 FLA│
├──────────────────────────────────────────────────────────────────────────────────────┤
│ All Tracks  ·  1,247 tracks  ·  sorted by Artist ↑                                  │
├──────────────────────────────────────────────────────────────────────────────────────┤
│ ▓░▓░  ▶  Aphex Twin — Flim                   01:22 ███████████░░░░░░░░ 02:56        │
│ ▓░░▓     Come to Daddy  ·  1997  ·  ALAC  ·  44.1 kHz  ·  16-bit         Vol 80%   │
├──────────────────────────────────────────────────────────────────────────────────────┤
│ [↵] action  [spc] play  [q] queue  [i] info  [v] select  [c] columns  [s] sort  [?]  │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

### Default column set

Always shows `ARTIST` and `ALBUM` by default since there is no hierarchy context (unlike the Browser's Tracks pane, where artist/album context comes from Left/Middle pane selections).

| Column   | Default visible | Notes                                          |
| -------- | --------------- | ---------------------------------------------- |
| `#`      | Yes             | Global index by current sort (1-based)         |
| `TITLE`  | Yes             | Flexible width, absorbs remaining space        |
| `ARTIST` | Yes             | Fixed width                                    |
| `ALBUM`  | Yes             | Fixed width                                    |
| `YEAR`   | Yes             | Fixed 6-char width                             |
| `DUR`    | Yes             | Fixed 6-char width                             |
| `FORMAT` | Yes             | e.g. `FLAC 44k`, `ALAC 44k`, `MP3 320`        |
| Others   | No (available)  | Genre, Label, Rate, Depth, Kbps, Date Added    |

### Default sort

Artist ↑ → Album ↑ → Track Number ↑. Natural library-browse ordering.

### Implementation note

Each row's hidden `RowData` carries a `trackID` field referencing the `model.Track` struct. `WithTargetHeight(availableLines)` fits the table to the content area exactly.

---

## 9. Layer 0C — Player View

An immersive full-screen view for active playback. Accessed via tab `3`. This view **replaces** the Now Playing bar — it IS the now playing view. The tab bar and hint bar are still shown; the Now Playing bar is not.

The Player view is intentionally beautiful: large album art, bold typography-scale text using character sizing, thick progress bar (double-height), and an integrated queue. A reserved area at the bottom is designated for a future spectrum analyser.

### 9.1 Full layout (≥ 120 cols) — Left/right split

Art occupies the left column. Track info, progress, and queue occupy the right.

```
  1 BROWSER    2 ALL TRACKS    ┌─ 3 PLAYER ───────────────────────────────────────────┐
 ──────────────────────────────┘
┌────────────────────────────────────┬─────────────────────────────────────────────────┐
│                                    │                                                  │
│  ▓▓▓▓▓░░░▓░▓▓▓░░▓▓▓▓▓▓░░░         │  Aphex Twin                                     │
│  ▓▓░░▓▓▓░░░▓▓▓▓▓░░▓▓░░░▓           │  Flim                                           │
│  ░░▓▓░░▓▓▓▓░░░▓▓▓░░▓▓▓▓░           │  Come to Daddy  ·  1997                         │
│  ▓░░▓▓▓░░░░▓▓▓▓░░░░▓░░▓▓           │                                                  │
│  ▓▓▓░░░▓▓░░░░▓▓▓░░▓▓░░░▓           │  ─────────────────────────────────────────────  │
│  ░▓▓▓▓░░▓▓▓░░░░▓▓▓░░▓▓▓░           │  01:22                                02:56     │
│  ▓░░░▓▓▓░░▓▓▓░░░░▓▓░░░▓▓           │  ████████████████████████░░░░░░░░░░░░░░░░░░░░  │
│  ░▓▓▓░░░▓▓░░▓▓▓░░░░▓▓▓░░           │  ████████████████████████░░░░░░░░░░░░░░░░░░░░  │
│  ▓░░░▓▓▓░░▓▓░░░▓▓░░░░▓▓▓           │  ─────────────────────────────────────────────  │
│  ░░▓▓▓░░▓▓░░░▓▓░░▓▓░░░░░           │  ALAC  ·  44.1 kHz  ·  16-bit  ·  641 kbps    │
│  ▓▓░░░▓▓░░░▓▓░░░▓▓░░▓▓░░           │  Vol 80%  ·  ▶ Normal                          │
│  ░▓▓▓░░░▓▓▓░░░░▓▓▓░░░▓▓░           │  ─────────────────────────────────────────────  │
│  ▓░░░▓▓░░░▓▓▓░░░░▓▓▓░░░▓           │  QUEUE  (12 tracks)                             │
│  ░░▓▓▓░░▓▓░░▓▓░░░░░▓▓▓░░           │  ─────────────────────────────────────────────  │
│  ▓▓░░░▓▓░░░▓▓░░░▓▓░░▓▓░░           │  ▶   Flim                    [playing]   2:56  │
│                                    │   1  Acrid Avid Jam Shred               3:19   │
│  (future: spectrum analyser        │   2  Come On You Slags!                 3:41   │
│   visualiser displayed here,       │   3  Autechre — Clipper                 4:48   │
│   below the album art)             │   4  Autechre — Bike                    6:21   │
│                                    │   5  Burial — Archangel                 5:07   │
│                                    │   6  Burial — Ghost Hardware            5:50   │
│                                    │                                                  │
└────────────────────────────────────┴─────────────────────────────────────────────────┘
│ All Tracks  ·  Playing: Come to Daddy  ·  Aphex Twin  ·  ALAC                        │
├──────────────────────────────────────────────────────────────────────────────────────┤
│ [p] pause  [.] next  [,] prev  [[] -5s  []] +5s  [-/=] vol  [m] mode  [?] help       │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

### 9.2 Full layout (80–119 cols) — Top/bottom split

At narrower terminals, art moves to the top and the queue to the bottom.

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                   │
│     ▓▓░░▓▓░░▓▓▓░░▓▓▓▓░░     Aphex Twin — Flim                                  │
│     ░░▓▓░░▓▓░░▓▓░░░▓▓░░░    Come to Daddy  ·  1997                              │
│     ▓▓░░▓▓░░░▓▓▓░░░▓▓▓░░                                                         │
│     ░░▓▓░░▓▓░░░░▓▓░░░▓▓░                                                         │
│                                                                                   │
│  01:22                                                                   02:56   │
│  ██████████████████████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │
│  ██████████████████████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │
│                                                                                   │
│  ALAC  ·  44.1 kHz  ·  16-bit  ·  Vol 80%  ·  ▶ Normal                         │
│                                                                                   │
├───────────────────────────────────────────────────────────────────────────────────┤
│  QUEUE  (12 tracks)                                                               │
│  ▶  Aphex Twin — Flim                                        [playing]   2:56    │
│   1  Aphex Twin — Acrid Avid Jam Shred                                   3:19    │
│   2  Aphex Twin — Come On You Slags!                                     3:41    │
│   3  Autechre — Clipper                                                   4:48   │
└───────────────────────────────────────────────────────────────────────────────────┘
│ Playing: Flim  ·  Aphex Twin  ·  ALAC                                             │
├───────────────────────────────────────────────────────────────────────────────────┤
│ [p] pause  [.] next  [,] prev  [[] -5s  []] +5s  [-/=] vol  [?] help             │
└───────────────────────────────────────────────────────────────────────────────────┘
```

### 9.3 Progress bar — double-height design

The Player view uses a double-height (2-row) progress bar to give it visual weight. Both rows render identically with the same fill ratio. Implemented using two consecutive `bubbles/progress` renders or a custom Lipgloss string.

```
  01:22                                                              02:56
  ████████████████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░
  ████████████████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░
```

### 9.4 Queue table in Player view

The queue in the Player view is a simplified `evertras/bubble-table` with a default column set of `#`, `TITLE`, `ARTIST`, `DUR`. Additional columns are available via the Column Manager (which operates independently for the Player queue from the All Tracks view).

The currently playing track is always shown at top, pinned, with `[playing]` badge and `▶` in the `#` column.

### 9.5 Player view — nothing queued / nothing playing state

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                                                                                   │
│                                                                                   │
│                         Nothing is playing.                                      │
│                                                                                   │
│                   Navigate to Browser or All Tracks,                             │
│                   select a track, and press [↵].                                 │
│                                                                                   │
│                                                                                   │
└───────────────────────────────────────────────────────────────────────────────────┘
```

### 9.6 Future: spectrum analyser

A reserved visual zone in the Player view (lower-left, below the album art in the left/right layout; lower portion of art area in top/bottom layout). Placeholder in current design. Will render a bar or waveform visualiser sourced from mpv audio data when implemented.

---

## 10. Layer 1 — Overlays

Overlays render over the full content area. Layer 0 is **dimmed** (reduced brightness) while an overlay is open; it receives no input. All overlays are visually distinguished by an `accent`-coloured border and padded title.

**Consistent overlay chrome:**

```
┌─ OVERLAY TITLE ──────────────────────────────────────────────────────────────────┐
│                                                                                   │
│  [ overlay content — scrollable via bubbles/viewport ]                           │
│                                                                                   │
└───────────────────────────────────────────────────────────────────────────────────┘
  [ hint bar reflects overlay-specific bindings + [esc] close ]
```

### 10.1 Search Overlay

Triggered by `/`. `bubbles/textinput` receives focus immediately. Background is dimmed Layer 0.

```
┌─ SEARCH ─────────────────────────────────────────────────────────────────────────┐
│                                                                                   │
│  ▸ metalheadz_______________________________________                             │
│                                                                                   │
│  ARTISTS  (0)                                                                     │
│                                                                                   │
│  ALBUMS  (4)                                                                      │
│  ▶ Timeless                                   Goldie                            │
│     Saturnz Return                             Goldie                            │
│     Inner City Life EP                         Goldie                            │
│     Prism                                      Various Artists                  │
│                                                                                   │
│  TRACKS  (18)                                                                     │
│     Inner City Life                  Goldie     ALAC 44k    6:30                │
│     Still Life                       Goldie     ALAC 44k    4:01                │
│     Angel                            Goldie     ALAC 44k    4:57                │
│     …  15 more                                                                   │
│                                                                                   │
└───────────────────────────────────────────────────────────────────────────────────┘
  [↵] action  [tab] next group  [ctrl+↵] results view  [esc] close
```

**Filter syntax highlight:** `label:`, `genre:`, `year:`, or `grouping:` prefix tokens are rendered in `accent` colour in the input field when typed.

### 10.2 Queue Overlay

Triggered by `~`. Body rendered via `bubbles/viewport`. Background is dimmed Layer 0.

```
┌─ QUEUE  (12 tracks) ─────────────────────────────────────────── ▶ Normal ───────┐
│                                                                                   │
│  ▶  Aphex Twin — Flim                                   [playing]   2:56        │
│  1  Aphex Twin — Acrid Avid Jam Shred                              3:19        │
│  2  Aphex Twin — Come On You Slags!                                3:41        │
│  3  Autechre — Clipper                                             4:48        │
│  4  Autechre — Bike                                                6:21        │
│  5  Burial — Archangel                                             5:07        │
│  6  Burial — Ghost Hardware                                        5:50        │
│  7  Burial — Near Dark                                             4:23        │
│  8  Boards of Canada — Roygbiv                                     2:31        │
│  …                                                                               │
│                                                                                   │
└───────────────────────────────────────────────────────────────────────────────────┘
  [↑/↓] navigate  [J/K] reorder  [x] remove  [c] clear  [esc] close
```

The playback mode is shown in the overlay title row. The currently playing track is pinned at top with `[playing]` badge.

**Clear confirmation (inline, replaces hint bar text):**

```
  Clear all queued tracks? The currently playing track is unaffected.
  [↵] confirm  [esc] cancel
```

### 10.3 Info Panel Overlay

Triggered by `i` from the Tracks pane, or via "View info" in the Action Dialog.

**View mode:**

```
┌─ INFO ─── Aphex Twin  /  Flim ───────────────────────────────────────────────────┐
│                                                                                   │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │ ▓░▓▓░▓░░▓▓░░▓▓▓░░▓▓    (album art — chafa, width ~20 cols × 8 rows)     │  │
│  │ ░▓▓░▓░▓░░▓▓▓░░░▓▓░░░                                                     │  │
│  │ ▓░░░▓▓░▓░░░░▓▓▓░░▓▓░                                                     │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                                                                   │
│  BASIC METADATA                                                                   │
│  Title          Flim                                                              │
│  Artist         Aphex Twin                                                        │
│  Album Artist   Aphex Twin                                                        │
│  Album          Come to Daddy                                                     │
│  Track          3 / 6                                                             │
│  Disc           1 / 1                                                             │
│  Year           1997                                                              │
│  Genre          Electronic                                                        │
│  Grouping       Warp Records                                                      │
│  Label          Warp Records                                                      │
│                                                                                   │
│  AUDIO                                                                            │
│  Format         ALAC                                                              │
│  Codec          alac                                                              │
│  Container      MP4                                                               │
│  Sample Rate    44,100 Hz                                                         │
│  Bit Depth      16-bit                                                            │
│  Bitrate        641 kbps                                                          │
│  Duration       2:56.441                                                          │
│  File Size      13.4 MB (14,073,291 bytes)                                        │
│                                                                                   │
│  LOUDNESS                                                                         │
│  RG Track Gain  -7.23 dB                                                          │
│  RG Track Peak  0.921631                                                          │
│  RG Album Gain  -8.12 dB                                                          │
│  RG Album Peak  0.981200                                                          │
│                                                                                   │
│  FILE                                                                             │
│  Path           ~/Music/Aphex Twin/Come to Daddy/Flim.m4a                        │
│  Last Modified  2024-03-15 11:42:07                                               │
│                                                                                   │
│  RAW TAGS                                                                         │
│  ©nam  Flim                                                                       │
│  ©ART  Aphex Twin                                                                 │
│  ©alb  Come to Daddy                                                              │
│  trkn  3/6                                                                        │
│  ©day  1997                                                                       │
│                                                                                   │
└───────────────────────────────────────────────────────────────────────────────────┘
  [e] edit  [↑/↓] scroll  [esc] close
```

**Edit mode** (triggered by `e`):

Editable fields become `textinput.Model` components. Unsaved-changes indicator `*` appears in the title. Live diff summary at the bottom of the viewport.

```
┌─ INFO  *  Aphex Twin  /  Flim ───────────────────────────────────────────────────┐
│                                                                                   │
│  BASIC METADATA                                                                   │
│  Title          [Flim_________________________________]                          │
│  Artist         [Aphex Twin___________________________]                          │
│  Album Artist   [Aphex Twin___________________________]                          │
│  Album          [Come to Daddy_______________________]                          │
│  Track          [3__]  /  [6__]                                                  │
│  Disc           [1__]  /  [1__]                                                  │
│  Year           [1997]                                                           │
│  Genre          [Electronic Ambient____________________]  ← edited              │
│  Grouping       [Warp Records_________________________]                          │
│  Label          [Warp Records_________________________]                          │
│                                                                                   │
│  ───────────────────────────────────────────────────────────────────────────────  │
│  UNSAVED CHANGES                                                                   │
│  Genre      Electronic  →  Electronic Ambient                                     │
│                                                                                   │
└───────────────────────────────────────────────────────────────────────────────────┘
  [↵] confirm write  [tab] next field  [esc] discard prompt
```

### 10.4 Help Overlay

Triggered by `?`. A `bubbles/viewport` listing all keybindings grouped by context. Content is generated dynamically from the active `key.Binding` definitions (supports `config.toml` remapping).

```
┌─ HELP ────────────────────────────────────────────────────────────────────────────┐
│                                                                                    │
│  NAVIGATION                                                                        │
│  ↓ / ↑        Cursor down / up                                                    │
│  → / ←        Focus next / previous pane                                         │
│  Tab          Focus next pane (alias for →)                                      │
│  Shift+Tab    Focus previous pane (alias for ←)                                  │
│  Ctrl+D       Scroll down half page                                               │
│  Ctrl+U       Scroll up half page                                                 │
│  Home         Jump to top                                                         │
│  End          Jump to bottom                                                      │
│  A–Z          Letter-jump to first item starting with that letter                 │
│                                                                                    │
│  VIEWS                                                                             │
│  1            Browser                                                             │
│  2            All Tracks                                                          │
│  3            Player                                                              │
│                                                                                    │
│  PLAYBACK                                                                          │
│  p            Play / Pause                                                        │
│  .            Next track                                                          │
│  ,            Previous track                                                      │
│  [ / ]        Seek -5s / +5s                                                      │
│  { / }        Seek -30s / +30s                                                    │
│  - / =        Volume down / up                                                    │
│  0            Reset volume to 100%                                                │
│  m            Cycle playback mode                                                 │
│                                                                                    │
│  LIBRARY (Tracks pane)                                                             │
│  ↵            Open action dialog                                                  │
│  Space        Play focused track                                                  │
│  v            Toggle selection on focused track                                   │
│  Shift+↓/↑    Extend selection range                                             │
│  ctrl+a       Select all visible tracks                                           │
│  i            Open info panel                                                     │
│  e            Edit tags (batch if selection active)                               │
│  c            Column Manager                                                      │
│  s            Sort picker                                                         │
│                                                                                    │
│  LIBRARY (Left / Middle panes)                                                     │
│  ↵            Open action dialog for highlighted artist / album                   │
│  a            Add highlighted album to queue                                      │
│                                                                                    │
│  CONFIGURATION                                                                     │
│  ctrl+b       Browse mode picker                                                  │
│  ctrl+t       Theme picker                                                        │
│                                                                                    │
│  OVERLAYS & APP                                                                    │
│  /            Search                                                              │
│  q            Queue (toggle)                                                      │
│  ?            Help (this screen)                                                  │
│  Esc          Close overlay / clear selection / dismiss dialog                    │
│  Ctrl+C       Quit                                                                │
│                                                                                    │
└────────────────────────────────────────────────────────────────────────────────────┘
  [↑/↓] scroll  [esc] close
```

---

## 11. Layer 2 — Dialogs

Small floating boxes rendered on top of whatever is beneath. Positioned near the cursor where practical. `accent`-coloured border, bold uppercase title.

**Consistent dialog chrome:**

```
┌─ DIALOG TITLE ───────────────────────────────────────────────┐
│  ──────────────────────────────────────────────────────────  │
│  ▶  Option One                                       [key]  │
│     Option Two                                       [key]  │
│  ──────────────────────────────────────────────────────────  │
│     Option Three                                     [key]  │
└──────────────────────────────────────────────────────────────┘
```

Rules: `↑`/`↓` navigate; `Enter` executes highlighted option; shortcut key executes directly; separator lines are not selectable; `Esc` closes without action.

### 11.1 Action Dialog

**Track:**

```
┌─ TRACK ──────────────────────────────────────────────────────┐
│  Aphex Twin — Flim                                            │
│  Come to Daddy  ·  1997  ·  ALAC  ·  2:56                    │
│  ──────────────────────────────────────────────────────────  │
│  ▶  Play now                                         [↵]    │
│     Play next                                        [n]    │
│     Add to queue                                    [spc]   │
│  ──────────────────────────────────────────────────────────  │
│     View info                                        [i]    │
│     Edit tags                                        [e]    │
│     Add to playlist…                                 [p]    │  ← post-MVP
└──────────────────────────────────────────────────────────────┘
```

**Album:**

```
┌─ ALBUM ──────────────────────────────────────────────────────┐
│  Aphex Twin — Come to Daddy                                   │
│  1997  ·  6 tracks  ·  ALAC                                  │
│  ──────────────────────────────────────────────────────────  │
│  ▶  Play album now                                   [↵]    │
│     Add album to queue                               [a]    │
│  ──────────────────────────────────────────────────────────  │
│     View in library                                  [l]    │
└──────────────────────────────────────────────────────────────┘
```

**Artist (from Left pane):**

```
┌─ ARTIST ─────────────────────────────────────────────────────┐
│  Aphex Twin  ·  127 tracks  ·  9 albums                       │
│  ──────────────────────────────────────────────────────────  │
│  ▶  Play all now                                     [↵]    │
│     Add all to queue                                 [a]    │
│  ──────────────────────────────────────────────────────────  │
│     Browse in library                                [l]    │
└──────────────────────────────────────────────────────────────┘
```

**Batch action dialog (multi-select active):**

```
┌─ BATCH ACTION ─── 3 tracks selected ─────────────────────────┐
│  ──────────────────────────────────────────────────────────  │
│  ▶  Play all now                                     [↵]    │
│     Play all next                                    [n]    │
│     Add all to queue                                 [a]    │
│  ──────────────────────────────────────────────────────────  │
│     Edit tags                                        [e]    │
│     Add to playlist…                                 [p]    │  ← post-MVP
└──────────────────────────────────────────────────────────────┘
```

### 11.2 Browse Mode Picker

Triggered by `ctrl+b`.

```
┌─ BROWSE BY ──────────────────────────────────────────────────┐
│  ──────────────────────────────────────────────────────────  │
│  ▶  Artist                                  [default]  [↵]  │
│     Label                                              [↵]  │
│     Genre                                              [↵]  │
│     Year                                               [↵]  │
│     Grouping                                           [↵]  │
│  ──────────────────────────────────────────────────────────  │
│     Playlists                                               │  ← post-MVP
└──────────────────────────────────────────────────────────────┘
```

### 11.2.1 Theme Picker

Triggered by `ctrl+t`. Selecting a theme applies it immediately as a live preview. `Esc` closes without changing the theme. The `[active]` badge marks the current theme.

```
┌─ THEME ───────────────────────────────────────────────────────┐
│  ─────────────────────────────────────────────────────────    │
│  ▶  default                                  [active]  [↵]   │
│     catppuccin-mocha                                   [↵]   │
│     catppuccin-latte                                   [↵]   │
│     gruvbox                                            [↵]   │
│     nord                                               [↵]   │
│     tokyo-night                                        [↵]   │
│     solarized-dark                                     [↵]   │
└───────────────────────────────────────────────────────────────┘
```

On selection: theme is applied immediately (live preview re-render), written to `config.toml`, dialog closes.

### 11.3 Column Manager

Triggered by `c`. Title shows the active browse mode (or "Player Queue" when invoked from the Player view queue).

```
┌─ COLUMNS ─── Artist mode ────────────────────────────────────┐
│                                                               │
│  Visible                                                      │
│  ▶  #   Track Number                                         │
│     t   Title                                                │
│     y   Year                                                 │
│     d   Duration                                             │
│     f   Format                                               │
│                                                               │
│  Available                                                    │
│     a   Artist                                               │
│     l   Album                                                │
│     g   Genre                                                │
│     r   Label                                                │
│     z   Sample Rate                                          │
│     k   Bit Depth                                            │
│     w   Bitrate                                              │
│     x   File Size                                            │
│     +   Date Added                                           │
│                                                               │
└───────────────────────────────────────────────────────────────┘
  [spc] toggle  [J/K] reorder  [esc] save & close
```

Column shortcut letters are single-character mnemonics for quick toggle — not bound globally, only within this dialog. `J/K` for reorder are uppercase and used only inside a dialog, so they do not conflict with the letter-jump rule (which applies to browser panes, not dialogs).

### 11.4 Sort Picker

Triggered by `s`. Shows only currently visible columns.

```
┌─ SORT BY ────────────────────────────────────────────────────┐
│  ──────────────────────────────────────────────────────────  │
│  ▶  Track Number                     [default]  ↑    [↵]   │
│     Title                                             [↵]   │
│     Year                                              [↵]   │
│     Duration                                          [↵]   │
│     Format                                            [↵]   │
└──────────────────────────────────────────────────────────────┘
```

Active sort shows direction `↑` / `↓`. Selecting active sort toggles direction. Selecting a new field sets it ascending.

### 11.5 Write Confirmation Dialog

Shown after `Enter` in tag edit mode.

```
┌─ WRITE TAGS ─── Aphex Twin  /  Flim ──────────────────────────┐
│                                                                 │
│  Genre      Electronic         →  Electronic Ambient           │
│  Label      —                  →  Warp Records                 │
│                                                                 │
│  1 file will be modified atomically.                            │
│                                                                 │
│  [↵] confirm write   [esc] cancel                              │
└─────────────────────────────────────────────────────────────────┘
```

---

## 12. Responsive Layout Variants

### ≥ 120 cols — Full layout

All features: album art in Now Playing bar and Info Panel, all three panes visible in Browser, full tab bar, Player view left/right split.

### 80–119 cols — Compact

No album art in Now Playing bar. Three browser panes, but narrower. Player view top/bottom split. Tab bar collapses inactive tab labels to just their number: `1  ─ 2 ALL TRACKS ─  3`.

### 60–79 cols — Two panes

Left pane hidden in Browser view. Breadcrumb appears at top of Middle pane:

```
 ─ 1 ─  ┌─ 2 ALL TRACKS ─┐  ─ 3 ─
 ───────┘                  └────────────────────────────────────────────────────────┐
│ [←] Artists                 │  #   TITLE                            DUR   FORMAT  │
│ ALBUMS (47)                  │  ─────────────────────────────────────────────────  │
│ ──────────────────────       │   1  Bucephalus Bouncing Ball         3:52  FLAC    │
│ Come to Daddy                │  ▶2  Come to Daddy                   5:24  FLAC    │
│▶Selected Ambient...          │   3  Flim                             2:56  ALAC    │
```

Tab labels collapse to numbers when `TermWidth < 80`: `─ 1 ─  ┌─ 2 ─┐  ─ 3 ─`

### < 60 cols — Single pane

```
─1─  ─2─  ┌─3─┐
           └────────────────────┐
│ Autechre › Selected Ambient   │
│  1  Basscad         5:59 FLAC │
│  2  Bike            6:21 FLAC │
│ ▶3  Clipper         4:48 FLAC │
└────────────────────────────────┘
  ▶ Aphex Twin — Flim  01:22 ██░ 02:56
  [spc] play  [q] queue  [?] help
```

---

## 13. Contextual Dialog Pattern Reference

### Decision tree

```
Does the interaction require user input or confirmation?
├── YES: Is it a full-featured scrollable view?
│   ├── YES → Layer 1 Overlay (viewport, accent border, Esc dismisses)
│   └── NO  → Layer 2 Dialog  (floating box, ↑/↓ navigate, Enter/Esc)
└── NO:  Show result inline or in the status bar. Never block input.
```

### What is NOT a dialog

- Status bar toast — informational, auto-dismisses, no input required
- Column sort indicator — inline state change
- Track selection `◆` — inline state indicator
- Playing `▶` — inline state indicator

### What IS a dialog (Layer 2)

Any menu with options, any confirmation of a destructive action, any picker (sort, browse mode, columns).

### What IS an overlay (Layer 1)

Any full-featured view with its own scrollable content (Help, Search, Queue, Info Panel).

---

## 14. Keybinding Decisions Log

A record of keybinding conflicts that existed in earlier drafts and how they were resolved. Kept here for audit trail.

| Conflict / Change | Resolution |
| ----------------- | ---------- |
| `i` = cursor up AND Info Panel | **Resolved:** cursor movement is arrow keys only. `i` = Info Panel. |
| `b` = Browse Mode Picker AND Previous track | **Resolved:** Browse Mode → `ctrl+b`. Previous track → `,`. |
| `j` = Focus prev pane AND cursor-down | **Resolved:** `j` freed. Cursor = `↓` only. Pane focus = `←`/`→` arrows. |
| `h`/`l` as pane focus aliases | **Resolved:** pane focus is `←`/`→` arrows (+ `Tab`/`Shift+Tab`). `h` and `l` freed. |
| `C` (uppercase) = Column Manager | **Resolved:** → `c`. `C` freed for letter-jump. |
| `S` (uppercase) = Sort Picker | **Resolved:** → `s`. `S` freed for letter-jump. |
| `V` (uppercase) = Select All | **Resolved:** → `ctrl+a`. `V` freed for letter-jump. |
| `Space` = toggle selection (conflicts with play) | **Resolved:** `Space` = play track. Selection toggled with `v`. |
| Enter = select + move focus (browser panes) | **Resolved:** cursor movement selects immediately. `Enter` = open Action Dialog in all panes. |
| `c` = Cycle Theme (displaced by Column Manager) | **Resolved:** Theme Picker → `ctrl+t`. Cycle-theme replaced by Theme Picker dialog. |
| `q` = quit (conflicts with queue) | **Resolved:** `q` = queue toggle. Quit = `Ctrl+C` only. |
| `n`/`f` for next/prev track | **Resolved:** `,` = previous track, `.` = next track. |
| `t`/`g` as alpha jump-to-top/bottom | **Resolved:** `Home` / `End` only. `t` and `g` freed. |
