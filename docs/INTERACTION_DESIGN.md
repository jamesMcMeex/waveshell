# waveshell — Interaction Design

> **Companion to:** `docs/WAVESHELL_PRD.md`
> **Scope:** How the UI behaves — navigation model, focus rules, browse modes, interaction flows, component patterns.
> The PRD answers _what_ the app does. This document answers _how it works_ when you're using or implementing it.
> **Last updated:** June 2026

---

## Table of Contents

1. [Purpose & Scope](#1-purpose--scope)
2. [Navigation Model — The Layer System](#2-navigation-model--the-layer-system)
3. [Browse Modes](#3-browse-modes)
4. [Focus & Input Routing](#4-focus--input-routing)
5. [Responsive Layout](#5-responsive-layout)
6. [Component Inventory](#6-component-inventory)
7. [Key Hint Bar](#7-key-hint-bar)
8. [Status Bar](#8-status-bar)
9. [Interaction Flows](#9-interaction-flows)
   - [9.1 Library Browser](#91-library-browser)
   - [9.2 Browse Mode Picker](#92-browse-mode-picker)
   - [9.3 Playlist Browse Mode](#93-playlist-browse-mode) _(post-MVP)_
   - [9.4 Search](#94-search)
   - [9.5 Action Dialog](#95-action-dialog)
   - [9.6 Column Management](#96-column-management)
   - [9.7 Column Sorting](#97-column-sorting)
   - [9.8 Multi-Select & Batch Operations](#98-multi-select--batch-operations)
   - [9.9 Metadata Info Panel](#99-metadata-info-panel)
   - [9.10 Tag Edit & Write Confirmation](#910-tag-edit--write-confirmation)
   - [9.11 Queue View](#911-queue-view)
   - [9.12 Help Overlay](#912-help-overlay)
10. [Error & Feedback Patterns](#10-error--feedback-patterns)
11. [Keybinding Contexts](#11-keybinding-contexts)

---

## 1. Purpose & Scope

This document specifies the interaction model for waveshell. It covers:

- How the application is structured spatially (the layer system)
- How browse modes allow the library to be sliced by any tag dimension
- How keyboard focus moves between components
- How the layout adapts to terminal size
- Detailed step-by-step flows for every non-trivial interaction
- What context-sensitive feedback the UI provides at all times
  It does not re-specify features or test coverage — those belong in the PRD. It does not specify visual styling beyond what is necessary to convey behaviour — that belongs in `config.toml` theme presets and the lipgloss constants in `internal/update`.

The intended reader is a developer implementing any milestone from Milestone 3 onward, or anyone evaluating whether a proposed UI change is consistent with the established patterns.

Sections marked _(post-MVP)_ describe interactions that are fully specified here for architectural continuity but will not be implemented until after v1.

---

## 2. Navigation Model — The Layer System

waveshell is a single-view application, not a multi-page one. There is no router, no history stack, no back button. The library browser is always present; everything else is rendered on top of it.

The UI is organised into two layers.

### Layer 0 — The Base (always visible)

The library browser: three panes (Left / Middle / Tracks) and the Now Playing bar. The pane labels change with browse mode, but the structure is constant. This layer is never replaced, hidden, or blanked. It is the spatial anchor that tells the user where they are.

### Layer 1 — Overlays

Panels that render on top of Layer 0. The base layer remains visible in the background but does not receive input while an overlay is open. There is at most one overlay open at a time.

| Overlay    | Trigger | Dismiss      |
| ---------- | ------- | ------------ |
| Search     | `/`     | `Esc`        |
| Queue      | `q`     | `Esc`        |
| Info Panel | `i`     | `Esc`        |
| Help       | `h`   | `Esc` or `q` |

Overlays are full-width or near-full-width panels anchored to the top of the content area. They are visually distinct from the base layer via a border and title header using the `accent` colour.

### Layer 2 — Dialogs

Small floating boxes rendered on top of either layer. Used for action menus, confirmations, tag write diffs, and pickers. There is at most one dialog open at a time.

| Dialog                       | Trigger                                | Dismiss         |
| ---------------------------- | -------------------------------------- | --------------- |
| Action menu                  | `Enter` on a track, album, or playlist | `Esc`           |
| Browse mode picker           | `B`                                    | `Esc`           |
| Column manager               | `C`                                    | `Esc`           |
| Sort picker                  | `S`                                    | `Esc`           |
| Write confirmation / diff    | `Enter` in tag edit mode               | `Esc` to cancel |
| Batch operation confirmation | `Enter` during multi-select edit       | `Esc` to cancel |

Dialogs are positioned near the cursor where practical.

### The "Esc contract"

`Esc` always moves the user one layer up without any side effect. From a dialog → closes the dialog, no write, state unchanged. From an overlay → closes the overlay, cursor position in the base layer is preserved. From the base layer → clears multi-select if active; no-op otherwise.

This contract is unconditional. Nothing should ever require two `Esc` presses to dismiss a single element.

---

## 3. Browse Modes

### Concept

The library browser is a generic **Left pane → Middle pane → Tracks pane** browser. The panes are not hardwired to "Artists" and "Albums" — they are determined by the active **Browse Mode**. Changing the browse mode re-indexes the same underlying SQLite library through a different dimensional lens.

This is the architectural foundation for tag-slice views and playlists. All modes use the same three-pane shell, the same navigation mechanics, and the same column, sort, and multi-select behaviour. Only the data and pane labels change. There are no special-cased layout variants per mode.

### Available modes

| Mode       | Left pane | Middle pane        | Tracks pane                            | MVP?     |
| ---------- | --------- | ------------------ | -------------------------------------- | -------- |
| `artist`   | Artists   | Albums by artist   | Tracks on album                        | ✓        |
| `label`    | Labels    | Albums on label    | Tracks on album                        | ✓        |
| `genre`    | Genres    | Albums in genre    | Tracks on album                        | ✓        |
| `year`     | Years     | Albums in year     | Tracks on album                        | ✓        |
| `playlist` | Playlists | Albums in playlist | Tracks in playlist (filtered by album) | post-MVP |

### Mode state in the Model

```go
type BrowseMode string

const (
    BrowseModeArtist   BrowseMode = "artist"
    BrowseModeLabel    BrowseMode = "label"
    BrowseModeGenre    BrowseMode = "genre"
    BrowseModeYear     BrowseMode = "year"
    BrowseModePlaylist BrowseMode = "playlist"
)
```

`BrowseMode` lives in `UIState`. All database queries for the left and middle panes are parameterised on this value. No other part of the architecture changes when the mode changes — pane navigation, column configuration, sort, multi-select, and search all behave identically.

### Column configuration is per-mode

Each browse mode carries its own column configuration (see §9.6). This allows `label` mode to have the `label` column visible while `artist` mode doesn't need it. Columns and sort state reset to the mode's saved configuration when switching modes.

### Persistence

The active browse mode and per-mode column/sort configuration persist for the session. On next launch, the app opens in `artist` mode unless a default is set in `config.toml`:

```toml
[ui]
default_browse_mode = "artist"

[ui.columns.artist]
visible = ["track_number", "title", "format", "sample_rate", "bit_depth", "duration"]

[ui.columns.playlist]
visible = ["playlist_position", "title", "artist", "album", "duration", "format"]
```

### Contextual mode switch from the Info Panel _(post-MVP)_

When viewing a track's info panel, the user can jump directly to a tag-filtered view by focusing a field (Label, Genre, Year) and pressing `Enter`. This opens the Browse Mode Picker pre-selected on the matching mode, with the Left pane cursor pre-positioned on that value. The Info Panel field highlight interaction should be designed to accommodate this from Milestone 7.

---

## 4. Focus & Input Routing

All key events pass through a single `Update` function. The routing decision is made at the top of that function based on `model.UI.ActiveOverlay` and `model.UI.ActiveDialog`.

```
KeyMsg received
     │
     ├─ model.UI.ActiveDialog != DialogNone
     │       └─ route to dialog handler
     │
     ├─ model.UI.ActiveOverlay != OverlayNone
     │       └─ route to overlay handler (Search / Queue / InfoPanel / Help)
     │
     └─ base layer
             └─ route to active pane handler (Left / Middle / Tracks)
```

There is no ambiguity. A key never reaches two handlers simultaneously.

### ActiveOverlay type

```go
type ActiveOverlay int

const (
    OverlayNone ActiveOverlay = iota
    OverlaySearch
    OverlayQueue
    OverlayInfoPanel
    OverlayHelp
)
```

### ActiveDialog type

```go
type ActiveDialog int

const (
    DialogNone ActiveDialog = iota
    DialogAction        // context menu on track/album/playlist
    DialogBrowseMode    // browse mode picker
    DialogColumns       // column manager
    DialogSort          // sort field picker
    DialogWriteDiff     // tag write confirmation
    DialogBatchDiff     // batch tag write confirmation
)
```

Both fields live in `UIState`. Resetting either field to `None` is the only dismiss mechanism — no separate `visible bool` flags.

---

## 5. Responsive Layout

BubbleTea fires `tea.WindowSizeMsg{Width, Height}` on every terminal resize. `UIState` stores the current dimensions:

```go
type UIState struct {
    TermWidth      int
    TermHeight     int
    ActiveOverlay  ActiveOverlay
    ActiveDialog   ActiveDialog
    ActivePane     Pane
    BrowseMode     BrowseMode
    ColumnConfig   map[BrowseMode][]ColumnID
    SortState      map[BrowseMode]SortState
    // ...
}
```

All widths and heights in `View` are computed from these values. No dimension is ever hardcoded.

### Three-pane column widths

```go
leftWidth   = model.UI.TermWidth * 22 / 100
middleWidth = model.UI.TermWidth * 28 / 100
tracksWidth = model.UI.TermWidth - leftWidth - middleWidth - 4 // 4 = border chars
```

These ratios are defaults. They are identical for all browse modes including playlist — there is no special-cased layout for playlist mode.

### Responsive breakpoints

| Terminal width | Layout                                                                                                              |
| -------------- | ------------------------------------------------------------------------------------------------------------------- |
| ≥ 120 cols     | Full three-pane layout                                                                                              |
| 80–119 cols    | Three panes, left column narrowed                                                                                   |
| 60–79 cols     | Two panes: Middle + Tracks. Left hidden; a `[←] {Left pane label}` breadcrumb appears at the top of the Middle pane |
| < 60 cols      | Single pane: Tracks only. Breadcrumb header shows the active path (e.g. `Aphex Twin › Selected Ambient Works`)      |

### Height allocation

```
total height
  - 1 line  : top border
  - 1 line  : Tracks pane column header row
  - N lines  : content panes (fills remaining space)
  - 2 lines  : Now Playing bar
  - 1 line  : key hint bar
  - 1 line  : bottom border
```

`N` is computed as `TermHeight - 6`. If the terminal is too short to show at least 4 tracks (< 10 lines total), the key hint bar is hidden.

---

## 6. Component Inventory

waveshell uses `bubbles` components for all stateful leaf elements. Each is embedded in the relevant state struct and updated via delegation in the parent `Update`.

| Component         | `bubbles` type      | Used in                     |
| ----------------- | ------------------- | --------------------------- |
| Search input      | `textinput.Model`   | `SearchState.Input`         |
| Queue list        | `viewport.Model`    | `QueueState.Viewport`       |
| Info panel body   | `viewport.Model`    | `InfoPanelState.Viewport`   |
| Tag edit fields   | `[]textinput.Model` | `InfoPanelState.EditFields` |
| Playback progress | `progress.Model`    | `PlayerState.ProgressBar`   |
| Scan progress     | `progress.Model`    | `LibraryState.ScanProgress` |
| Scan spinner      | `spinner.Model`     | `LibraryState.Spinner`      |
| Help overlay body | `viewport.Model`    | `HelpState.Viewport`        |

The three browser panes (Left, Middle, Tracks) are **custom list components**, not `bubbles/list`. Multi-select, the column header row, column-aware rendering, and sort indicators require a delegate interface that `bubbles/list` makes unnecessarily complex. Custom pane components implement their own `j/k` cursor, scroll offset, and selection state — all pure data in the Model.

---

## 7. Key Hint Bar

A persistent one-line bar rendered between the content area and the Now Playing bar. It shows 4–6 context-relevant key bindings at all times. Its content changes based on `ActiveOverlay`, `ActiveDialog`, `ActivePane`, `BrowseMode`, and whether a multi-select is in progress.

### Format

```
[↵] play  [spc] queue  [i] info  [C] columns  [S] sort  [/] search  [?] help
```

Keys are shown in brackets. Actions are lowercase. The bar is rendered in `muted` colour.

### Content rules by context

| Context                                     | Hint bar content                                                                   |
| ------------------------------------------- | ---------------------------------------------------------------------------------- |
| Base — Left pane focused                    | `[↵] select  [b] browse mode  [c] theme  [tab] →  [/] search  [h] help`                       |
| Base — Middle pane focused                  | `[↵] select  [tab] →  [a] add album  [/] search  [h] help`                         |
| Base — Tracks pane, no selection            | `[↵] action  [spc] queue  [i] info  [v] select  [C] columns  [S] sort  [/] search` |
| Base — Tracks pane, items selected          | `[e] edit tags  [spc] add to queue  [V] select all  [esc] clear`                   |
| Base — Playlist left pane                   | `[↵] open  [n] new playlist  [B] browse mode  [tab] →`                             |
| Overlay: Search                             | `[↵] action  [tab] next group  [ctrl+↵] results view  [esc] close`                 |
| Overlay: Queue                              | `[J/K] reorder  [x] remove  [c] clear  [esc] close`                                |
| Overlay: Info Panel (view mode)             | `[e] edit tags  [j/k] scroll  [esc] close`                                         |
| Overlay: Info Panel (edit mode)             | `[↵] confirm  [tab] next field  [esc] cancel`                                      |
| Dialog: Browse mode / Sort / Column manager | `[j/k] navigate  [↵] confirm  [esc] cancel`                                        |
| Dialog: Write diff                          | `[↵] write  [esc] cancel`                                                          |

---

## 8. Status Bar

A second line rendered immediately above the Now Playing bar. Provides ambient state information.

### Layout

```
By Label  ·  Metalheadz  ·  23 albums  ·  Shuffle  ·  Scanning…
```

Left-aligned fields, separated by `·` in `muted` colour.

### Fields

| Field                  | When shown                          | Content                                            |
| ---------------------- | ----------------------------------- | -------------------------------------------------- |
| Browse mode            | When not default `artist` mode      | `By Label` / `By Genre` / `By Year` / `Playlists`  |
| Current selection path | When a left/middle item is selected | The selected value (label name, genre, year, etc.) |
| Track / album count    | Always                              | `1,247 tracks` / `23 albums` / context-specific    |
| Sort indicator         | When not the mode default           | `sorted by Year ↓`                                 |
| Playback mode          | When not `StopAtEnd`                | `Repeat Track` / `Repeat Queue` / `Shuffle`        |
| Scan indicator         | During scan                         | `Scanning… 312 tracks` with spinner                |
| Selection count        | When items selected                 | `3 tracks selected`                                |
| Error toast            | On non-critical error               | Transient message (see §10)                        |

---

## 9. Interaction Flows

### 9.1 Library Browser

The base layer. Three panes: Left / Middle / Tracks (labels determined by browse mode — e.g. Artists / Albums / Tracks in default `artist` mode).

**Pane focus model:**

Active pane border renders in `accent` colour. Inactive pane borders render in `muted`. Focus cycles rightward with `Tab` and leftward with `Shift+Tab`. `l` and `j` also shift focus right and left respectively.

**Filtering behaviour:**

Selecting an item in the Left pane filters the Middle pane to show only items related to that selection. Selecting an item in the Middle pane filters the Tracks pane. Changing the Left selection immediately re-filters both right panes.

**Cursor behaviour on filter change:**

When a right pane's contents change due to a selection in the left pane, that pane's cursor resets to position 0 and scroll resets to the top.

**Letter-jump:**

In any pane, pressing a letter key (range depends on context — `A`–`S` for pane movement letters, but all panes handle their own active range) moves the cursor to the first item whose display name begins with that letter. Works in all three panes across all browse modes.

> **Rule:** `Shift+Letter` (`A`–`Z`) is always letter-jump. Lowercase letters are reserved for navigation commands (e.g. `g` = jump to bottom, `t` = jump to top). Never assign lowercase letters to letter-jump or uppercase letters to navigation — this would break the convention.

**Scroll:**

Each pane scrolls independently. `Ctrl+D` / `Ctrl+U` scroll by half a page.

---

### 9.2 Browse Mode Picker

`B` opens the browse mode picker from anywhere in the base layer.

```
┌──────────────────────────────┐
│  BROWSE BY                   │
│  ────────────────────────    │
│  ▶  Artist       [default]   │
│     Label                    │
│     Genre                    │
│     Year                     │
│     Grouping                 │
│  ────────────────────────    │
│     Playlists                │ ← post-MVP
└──────────────────────────────┘
```

The currently active mode is highlighted with `▶`. The separator between standard modes and Playlists is not selectable.

Selecting a mode:

1. `ActiveDialog` resets to `DialogNone`.
2. `UIState.BrowseMode` is updated.
3. A `tea.Cmd` fires a database query to populate the new left pane.
4. Left and middle pane cursors reset to position 0.
5. Column configuration and sort state load from the saved config for that mode.
6. The status bar reflects the new mode immediately.
   `Esc` closes without changing mode.

---

### 9.3 Playlist Browse Mode _(post-MVP)_

Playlist mode uses the standard three-pane layout. All three panes behave identically to other browse modes. No special-cased layout or mechanics.

```
┌──────────────────┬──────────────────────┬──────────────────────────────────────┐
│  PLAYLISTS       │  ALBUMS              │  #    TITLE             DUR   FORMAT  │
│  ──────────────  │  ──────────────────  │  ──────────────────────────────────  │
│  Late Night  12  │  Timeless            │   1   Goldie — Inner City Life  6:30  │
│▶ Drum & Bass 47  │▶ Saturnz Return      │   2   Goldie — Angel       4:57       │
│  Ambient     23  │  Inner City Life EP  │   3   LTJ Bukem — Horizons 9:12       │
│  Work Focus  31  │  Foley Room          │  …                                    │
│                  │  Isam                │                                       │
├──────────────────┴──────────────────────┴──────────────────────────────────────┤
│ ▶  Goldie — Inner City Life                  01:22 ────●──── 06:30             │
│    Timeless · ALAC · 44.1kHz · 24bit                            Vol: 80%       │
└─────────────────────────────────────────────────────────────────────────────────┘
```

**Middle pane (Albums in playlist):** Shows the distinct albums represented by tracks in the selected playlist. Selecting an album filters the Tracks pane to only tracks from that album within the playlist. Selecting the implicit `All` entry at the top of the Middle pane clears the album filter and shows all tracks in the playlist.

**Track numbering:** The `#` column in playlist mode shows **playlist position** (1-based), not album track number. An optional `album_track` column is available if the user wants to see album track numbers alongside playlist positions (see §9.6).

**Default sort:** Playlist position. This is the user-defined order. Sort by any other field is a view-only override — the underlying `position` values in `playlist_tracks` are unchanged.

**Sorting in playlist mode:** Any sort other than the default shows a note in the Tracks pane column header:

```
  #    TITLE              ARTIST ↑   DUR      FORMAT
                          (playlist order suspended)
```

Selecting `[default]` in the sort picker restores playlist position order and clears the note.

**Reordering tracks:** `J` / `K` in the Tracks pane moves the selected track's playlist position up or down by one. This writes a new `position` value to `playlist_tracks` immediately (no confirmation). Reordering is only available when sort is set to default (playlist position); if a non-default sort is active, `J` / `K` are not bound to reorder, preventing accidental mutation of a sorted view.

#### Playlist management

`Enter` on a playlist in the Left pane opens the action dialog:

```
┌──────────────────────────────────────┐
│  Drum & Bass Best  (47 tracks)       │
│  ────────────────────────────────    │
│  ▶  Open                        [↵]  │
│     Add all to queue            [a]  │
│  ────────────────────────────────    │
│     Rename playlist             [r]  │
│     Delete playlist             [d]  │
└──────────────────────────────────────┘
```

**Creating a playlist:** `n` from the Left pane (in playlist mode) opens an inline `textinput` — a new row appears at the bottom of the playlist list with an edit cursor. `Enter` saves; `Esc` cancels.

**Adding to a playlist from the library:** In any non-playlist browse mode, the track Action Dialog includes an "Add to playlist…" option that opens a secondary dialog listing all playlists. Selecting one appends the track with a position after the last existing track.

---

### 9.4 Search

The search overlay is the primary discovery mechanism, operating in two modes: **quick search** (overlay) and **results view** (persistent).

#### Quick Search (overlay)

1. User presses `/` from anywhere in the base layer.
2. `ActiveOverlay` is set to `OverlaySearch`. The search overlay renders over the base layer.
3. `SearchState.Input` receives focus immediately.
4. As the user types, `Update` fires a fuzzy search `tea.Cmd` against the in-memory index, returning ranked results.
5. Results render below the input, grouped with counts:

```
┌─────────────────────────────────────────────────────┐
│  SEARCH                                             │
│  ▸ metalheadz__________________________________     │
│                                                     │
│  ARTISTS (1)                                        │
│    Goldie                                           │
│                                                     │
│  ALBUMS (3)                                         │
│  ▶ Timeless                                         │
│    Saturnz Return                                   │
│    Inner City Life EP                               │
│                                                     │
│  TRACKS (14)                                        │
│    Inner City Life                    ALAC 44.1k    │
│    Still Life                         ALAC 44.1k    │
│    …                                                │
│                                                     │
│  [↵] action  [tab] next group  [ctrl+↵] results view  [esc] close │
└─────────────────────────────────────────────────────┘
```

6. `j/k` moves the cursor through results. `Tab` jumps to the next group.
7. `Enter` on a result opens the Action Dialog for that item.
8. `Esc` closes the overlay. `SearchState` is cleared. Base layer cursor is unchanged.

#### Label/grouping filter syntax

If the query contains a `label:`, `grouping:`, `genre:`, or `year:` prefix token (e.g. `label:Metalheadz 1995`), the search engine filters results to items matching that tag field before fuzzy-matching the remainder. The filter token is highlighted in the input field in `accent` colour.

#### Results View (persistent)

1. `Ctrl+Enter` while the overlay is open with a non-empty query.
2. The overlay closes. The base layer enters **search results mode**: the Tracks pane is replaced by a flat list of all matching tracks. Left and Middle panes are visually dimmed.
3. Status bar shows: `Search Results · "metalheadz" · 47 tracks · sorted by Relevance`.
4. Multi-select, queue operations, tag editing, column management, and sorting all work on this list identically to the normal Tracks pane.
5. The default sort for search results is relevance score. The column configuration used is the one for the active browse mode.
6. `Esc` exits search results mode and restores the library browser. Cursor and browse mode are preserved.

---

### 9.5 Action Dialog

Pressing `Enter` on any actionable item opens a context menu rather than immediately executing a potentially destructive action. The dialog also surfaces keybindings, teaching them over time.

#### Track action dialog

```
┌────────────────────────────────┐
│  Aphex Twin — Flim             │
│  ──────────────────────────    │
│  ▶  Play now              [↵]  │
│     Play next             [N]  │
│     Add to queue        [Spc]  │
│  ──────────────────────────    │
│     View info             [i]  │
│     Edit tags             [e]  │
│     Add to playlist…      [p]  │ ← post-MVP
└────────────────────────────────┘
```

#### Album action dialog

```
┌────────────────────────────────┐
│  Aphex Twin — Selected Ambient │
│  ──────────────────────────    │
│  ▶  Play album now        [↵]  │
│     Add album to queue    [a]  │
│  ──────────────────────────    │
│     View in library       [L]  │
└────────────────────────────────┘
```

#### Behaviour

- `j/k` navigates options. `Enter` or the bracketed shortcut key executes that option directly.
- Separator lines are not selectable — cursor skips them.
- `Esc` closes without action.
- The dialog is dismissed automatically after any action executes, except "View info" and "Edit tags", which transition to the Info Panel overlay.

---

### 9.6 Column Management

Each browse mode has an independent list of visible columns that determines what the Tracks pane displays and what sort fields are available. Column configuration is a first-class part of the app state, not just a config-file concern.

#### Column identifier registry

All columns available across all modes:

| Column ID           | Display label | Available in                                                         |
| ------------------- | ------------- | -------------------------------------------------------------------- |
| `playlist_position` | `#`           | Playlist mode only                                                   |
| `track_number`      | `#`           | All modes except playlist                                            |
| `album_track`       | `TRK`         | Playlist mode (shows album track number alongside playlist position) |
| `title`             | `TITLE`       | All                                                                  |
| `artist`            | `ARTIST`      | All                                                                  |
| `album`             | `ALBUM`       | All                                                                  |
| `year`              | `YEAR`        | All                                                                  |
| `genre`             | `GENRE`       | All                                                                  |
| `label`             | `LABEL`       | All                                                                  |
| `grouping`          | `GROUPING`    | All                                                                  |
| `format`            | `FORMAT`      | All                                                                  |
| `codec`             | `CODEC`       | All                                                                  |
| `sample_rate`       | `RATE`        | All                                                                  |
| `bit_depth`         | `DEPTH`       | All                                                                  |
| `bitrate`           | `KBPS`        | All                                                                  |
| `duration`          | `DUR`         | All                                                                  |
| `file_size`         | `SIZE`        | All                                                                  |
| `date_added`        | `ADDED`       | All                                                                  |

Columns marked "Playlist mode only" do not appear in the Column Manager when not in playlist mode, and vice versa for `track_number`.

#### Column header row

The Tracks pane has a persistent header row rendered above the track list. It is styled in `muted` and is not a selectable list item. The column whose sort is active is highlighted in `accent` with a direction indicator:

```
  #    TITLE                    YEAR ↓    DUR     FORMAT
  01   Wooden Toy               2005      4:32    FLAC 44k
▶ 02   Lost & Safe              2005      5:12    FLAC 44k
  03   Stealth                  2005      3:45    FLAC 44k
```

Column widths are computed proportionally from the Tracks pane width and the number of visible columns. `TITLE` is always the flexible column that absorbs remaining space. All other columns use fixed widths appropriate to their content.

#### Column Manager dialog

`C` opens the Column Manager from the Tracks pane (or Search Results view).

```
┌──────────────────────────────────────┐
│  COLUMNS — Artist mode               │
│  ──────────────────────────────      │
│  Visible                             │
│  ▶  #  Track Number                  │
│     T  Title                         │
│     Y  Year                          │
│     D  Duration                      │
│     F  Format                        │
│                                      │
│  Available                           │
│     A  Artist                        │
│     L  Album                         │
│     G  Genre                         │
│     B  Label                         │
│     R  Sample Rate                   │
│     K  Bit Depth                     │
│     Z  Bitrate                       │
│     S  File Size                     │
│     +  Date Added                    │
│                                      │
│  [spc] toggle  [J/K] reorder  [esc] save & close │
└──────────────────────────────────────┘
```

- The dialog header shows the current browse mode so it is always clear which mode's columns are being edited.
- `j/k` navigates the list. `Space` toggles the highlighted column: visible → available or available → visible. The change takes effect immediately — the Tracks pane behind the dialog re-renders live.
- `J` / `K` reorder within the Visible section. A column in the Available section cannot be reordered until it is made visible.
- `Esc` closes the dialog. Changes are already applied (live preview) and persisted to `config.toml`.
- There is no Cancel. The dialog has no destructive actions; the user can always re-toggle a column they accidentally hid.
- At least one column must remain visible. `Space` on the last visible column is a no-op.

#### Sort follows visible columns

The sort picker (§9.7) only shows columns that are currently visible. This creates a clear constraint: to sort by a field, make it a column. If the active sort column is hidden via the Column Manager, the sort resets to the mode default and the status bar briefly shows: `Sort reset (column hidden)`.

---

### 9.7 Column Sorting

Sorting is available in the Tracks pane (all browse modes) and in Search Results view.

#### Sort state

```go
type SortState struct {
    Field     SortField
    Direction SortDirection  // Ascending | Descending
}

type SortField string

const (
    SortDefault          SortField = ""                   // mode-specific default
    SortPlaylistPosition SortField = "playlist_position"
    SortTrackNumber      SortField = "track_number"
    SortAlbumTrack       SortField = "album_track"
    SortTitle            SortField = "title"
    SortArtist           SortField = "artist"
    SortAlbum            SortField = "album"
    SortYear             SortField = "year"
    SortDuration         SortField = "duration"
    SortBitrate          SortField = "bitrate"
    SortSampleRate       SortField = "sample_rate"
    SortBitDepth         SortField = "bit_depth"
    SortFileSize         SortField = "file_size"
    SortLabel            SortField = "label"
    SortGenre            SortField = "genre"
    SortGrouping         SortField = "grouping"
    SortDateAdded        SortField = "date_added"
)
```

`SortState` is stored per browse mode in `UIState.SortState` (a `map[BrowseMode]SortState`). Switching modes loads the saved sort state for that mode rather than always resetting.

#### Default sort per mode

| Browse mode    | Default sort                        |
| -------------- | ----------------------------------- |
| `artist`       | Track number (within album context) |
| `label`        | Track number (within album context) |
| `genre`        | Track number (within album context) |
| `year`         | Track number (within album context) |
| `grouping`     | Track number (within album context) |
| `playlist`     | Playlist position                   |
| Search results | Relevance score                     |

#### Sort picker dialog

`S` opens the sort picker. It shows **only currently visible columns** for the active mode.

```
┌────────────────────────────────┐
│  SORT BY                       │
│  ──────────────────────────    │
│  ▶  Track Number  [default] ↑  │
│     Title                      │
│     Year                       │
│     Duration                   │
│     Format                     │
└────────────────────────────────┘
```

- The active sort field is highlighted with `▶` and its direction shown (↑ / ↓).
- Selecting the active field **toggles direction**.
- Selecting a new field sets it ascending.
- Selecting `[default]` when not already active resets to the mode default.
- `Esc` closes without change.
- If no columns are visible beyond `title` and `#`, the picker shows only those entries.

#### Sort indicator in the column header

The sorted column header is highlighted in `accent` with a direction arrow, replacing the previous "pane title + sort indicator" pattern. The column header row is the single source of sort truth.

---

### 9.8 Multi-Select & Batch Operations

Multi-select is available in the Tracks pane across all browse modes and in Search Results view.

#### Entering and managing a selection

| Key     | Action                                                       |
| ------- | ------------------------------------------------------------ |
| `Space` | Toggle selection on the current track. Cursor does not move. |
| `V`     | Select all currently visible tracks.                         |
| `Esc`   | Clear all selections.                                        |

A selected track displays a `◆` prefix in `accent` colour. The count appears in the status bar: `3 tracks selected`. Standard navigation continues to work during selection.

#### Batch operations

| Key           | Action                                          |
| ------------- | ----------------------------------------------- |
| `e`           | Open batch tag editor (see §9.10)               |
| `Space`       | Add all selected tracks to the end of the queue |
| `Shift+Space` | Insert all selected tracks as play-next         |
| `Esc`         | Clear selection                                 |

---

### 9.9 Metadata Info Panel

Opened with `i` from the Tracks pane or via "View info" in the Action Dialog. Renders as a full-height overlay using `bubbles/viewport`.

#### Panel sections

```
┌───────────────────────────────────────────────────┐
│  INFO — Aphex Twin / Flim                         │
│  ─────────────────────────────────────────────    │
│                                                   │
│  BASIC METADATA                                   │
│  Title          Flim                              │
│  Artist         Aphex Twin                        │
│  Album Artist   Aphex Twin                        │
│  Album          Come to Daddy                     │
│  Track          3 / 6                             │
│  Disc           1 / 1                             │
│  Year           1997                              │
│  Genre          Electronic                        │
│  Grouping       Warp Records                      │
│  Label          Warp Records                      │
│                                                   │
│  AUDIO                                            │
│  Format         ALAC                              │
│  Codec          alac                              │
│  Container      MP4                               │
│  Sample Rate    44,100 Hz                         │
│  Bit Depth      16-bit                            │
│  Bitrate        641 kbps                          │
│  Duration       2:56.441                          │
│  File Size      13.4 MB (14,073,291 bytes)        │
│                                                   │
│  LOUDNESS                                         │
│  RG Track Gain  -7.23 dB                          │
│  RG Track Peak  0.921631                          │
│  RG Album Gain  -8.12 dB                          │
│  RG Album Peak  0.981200                          │
│  R128 Track     -1.23                             │
│                                                   │
│  ARTWORK                                          │
│  Present        Yes                               │
│  Dimensions     600 × 600 px                      │
│  Format         JPEG                              │
│  Size           87 KB                             │
│                                                   │
│  FILE                                             │
│  Path           ~/Music/Aphex Twin/Come to Daddy/ │
│                 Flim.m4a                          │
│  Last Modified  2024-03-15 11:42:07               │
│                                                   │
│  RAW TAGS                                         │
│  ©nam  Flim                                       │
│  ©ART  Aphex Twin                                 │
│  …                                                │
│                                                   │
│  [e] edit tags  [j/k] scroll  [esc] close         │
└───────────────────────────────────────────────────┘
```

- Fields with no value show `—` in `muted` colour.
- Absent loudness fields are omitted entirely.
- The Raw Tags section shows every tag in the file in its native encoding key. It is always last.
- Long file paths wrap and align to the label column width.

---

### 9.10 Tag Edit & Write Confirmation

Entered from the Info Panel with `e`. The panel transitions from view mode to edit mode in-place.

#### Edit mode

All editable fields become `textinput.Model` components. `Tab` / `Shift+Tab` moves between fields. Unsaved changes are indicated by a `*` prefix in `accent` colour. A live diff panel appears at the bottom:

```
  UNSAVED CHANGES
  Genre       Electronic Ambient  →  Ambient
  Label       —                   →  Warp Records
```

#### Confirmation

`Enter` advances to write confirmation. A diff dialog renders over the panel:

```
┌──────────────────────────────────────────────────┐
│  WRITE TAGS — Aphex Twin / Flim                  │
│                                                  │
│  Genre       Electronic Ambient  →  Ambient      │
│  Label       —                   →  Warp Records │
│                                                  │
│  1 file will be modified atomically.             │
│                                                  │
│  [↵] confirm write   [esc] cancel                │
└──────────────────────────────────────────────────┘
```

`Enter` executes the write (`.tmp` → `os.Rename`). On success: dialog closes, panel returns to view mode. On error: original confirmed untouched; panel stays in edit mode with changes preserved for retry.

`Esc` at the diff dialog returns to edit mode. `Esc` in edit mode prompts: `Discard unsaved changes? [↵] discard  [esc] keep editing`.

#### Batch tag editing

When multi-select is active and `e` is pressed, the panel opens in batch edit mode. Header reads `EDITING N TRACKS`. Fields where values differ across selected tracks show `<mixed>` in `muted`. Editing a `<mixed>` field sets it on all selected files; leaving it untouched makes no change to that field on any file. The write diff dialog lists every file and every change before a single file is touched. Files are written one at a time; a write failure halts remaining writes without rolling back already-completed ones.

---

### 9.11 Queue View

Opened with `q`. Renders as a full-height overlay using `bubbles/viewport`.

```
┌──────────────────────────────────────────────────────┐
│  QUEUE (12 tracks)                        Shuffle    │
│  ────────────────────────────────────────────────    │
│                                                      │
│  ▶  Aphex Twin — Flim                    [playing]   │
│  1  Aphex Twin — Acrid Avid Jam Shred    02:23       │
│  2  Aphex Twin — Come On You Slags!      03:41       │
│  3  Autechre — Clipper                   05:12       │
│  …                                                   │
│                                                      │
│  [J/K] reorder  [x] remove  [c] clear  [esc] close  │
└──────────────────────────────────────────────────────┘
```

`J` / `K` moves the selected item down / up. `x` removes it. If the removed item is currently playing, playback advances to the next track. `c` clears all except the playing track (inline confirmation). `Esc` closes.

---

### 9.12 Help Overlay

Opened with `?`. A `bubbles/viewport` overlay listing all keybindings grouped by context: Navigation · Playback · Queue · Search · Browse Modes · Columns & Sorting · Metadata · Application.

Content is generated from the same keybinding configuration used at runtime, so it reflects any customisation in `config.toml`. `Esc` or `q` closes.

---

## 10. Error & Feedback Patterns

### Non-critical errors — status bar toast

For errors that do not require immediate user action (mpv socket timeout, scan skipped a file):

1. The error message replaces the normal status bar content temporarily.
2. Rendered in a distinct warm colour.
3. Auto-clears after 4 seconds or on next keypress.
4. Also written to `waveshell.log`.

### Critical errors — blocking dialog

For errors that prevent the app from functioning (mpv not installed, no library path, database unreadable):

1. A Layer 2 dialog renders with the error and a suggested resolution.
2. No key events reach the base layer.
3. If unrecoverable, only `q` / `Ctrl+C` is offered.

### Write errors — in-panel

Tag write errors surface inside the Info Panel in context (see §9.10). The original file is confirmed untouched before the message is shown.

### Scan progress — status bar

- Spinner animates during scan: `Scanning… 1,247 tracks`.
- On completion: `Scan complete · 1,247 tracks` for 2 seconds, then cleared.
- Skipped files are counted: `Scan complete · 1,247 tracks · 3 skipped`.

### Sort reset notification

When a visible column is hidden via the Column Manager and it was the active sort column, the sort silently resets to default and the status bar briefly shows: `Sort reset (column hidden)`.

### Confirmations — inline, not toast

Destructive confirmations (clear queue, discard unsaved edits) use an inline prompt within the relevant overlay or dialog. The toast auto-dismisses; a confirmation must not.

---

## 11. Keybinding Contexts

All bindings are configurable in `config.toml`. The defaults below are the shipped values.

### Base layer — Library Browser

| Key           | Action                                | Notes                                |
| ------------- | ------------------------------------- | ------------------------------------ |
| `k` / `↓`     | Cursor down                           |                                      |
| `i` / `↑`     | Cursor up                             |                                      |
| `Ctrl+D`      | Scroll down half page                 |                                      |
| `Ctrl+U`      | Scroll up half page                   |                                      |
| `t` / `Home`  | Jump to top                           |                                      |
| `g` / `End`   | Jump to bottom                        |                                      |
| `A`–`Z`       | Letter-jump                           | First item starting with that letter |
| `Tab`         | Focus next pane (right)               | Wraps: Tracks → Left                 |
| `Shift+Tab`   | Focus previous pane (left)            | Wraps: Left → Tracks                 |
| `j`           | Focus previous pane                   |                                      |
| `l`           | Focus next pane                       |                                      |
| `Enter`       | Open Action Dialog                    |                                      |
| `Space`       | Toggle track selection                | Tracks pane                          |
| `Shift+Space` | Insert selected as play-next          | Tracks pane                          |
| `a`           | Add album to queue                    | From any pane                        |
| `V`           | Select all visible tracks             | Tracks pane                          |
| `b`           | Open Browse Mode Picker               | Any pane                             |
| `C`           | Open Column Manager                   | Tracks pane                          |
| `S`           | Open Sort Picker                      | Tracks pane / Search Results         |
| `i`           | Open Info Panel                       | Tracks pane                          |
| `e`           | Edit tags (batch if selection active) | Tracks pane                          |
| `/`           | Open Search overlay                   |                                      |
| `q`           | Open Queue overlay                    |                                      |
| `m`           | Cycle playback mode                   |                                      |
| `c`           | Cycle theme                           |                                      |
| `h`           | Open Help overlay                     |                                      |
| `Esc`         | Clear selection / no-op               |                                      |
| `Ctrl+C`      | Quit                                  |                                      |

### Playback (always active)

| Key       | Action                           |
| --------- | -------------------------------- |
| `p`       | Toggle play / pause              |
| `n`       | Next track                       |
| `b`       | Previous track                   |
| `[` / `]` | Seek -5s / +5s                   |
| `{` / `}` | Seek -30s / +30s                 |
| `-` / `=` | Volume down / up (5% increments) |
| `0`       | Reset volume to 100%             |

### Playlist mode — Left pane (additional bindings)

| Key | Action                                       |
| --- | -------------------------------------------- |
| `n` | New playlist (inline name input)             |
| `r` | Rename selected playlist                     |
| `d` | Delete selected playlist (with confirmation) |

### Playlist mode — Tracks pane (additional binding, default sort only)

| Key | Action                                     |
| --- | ------------------------------------------ |
| `J` | Move selected track down in playlist order |
| `K` | Move selected track up in playlist order   |

### Column Manager dialog

| Key       | Action                                  |
| --------- | --------------------------------------- |
| `j` / `↓` | Navigate list                           |
| `k` / `↑` | Navigate list                           |
| `Space`   | Toggle column visible / hidden          |
| `J`       | Move column down (Visible section only) |
| `K`       | Move column up (Visible section only)   |
| `Esc`     | Save and close                          |

### Search overlay

| Key          | Action                                   |
| ------------ | ---------------------------------------- |
| `j` / `↓`    | Cursor down through results              |
| `k` / `↑`    | Cursor up through results                |
| `Tab`        | Jump to next result group                |
| `Shift+Tab`  | Jump to previous result group            |
| `Enter`      | Open Action Dialog for selected result   |
| `Ctrl+Enter` | Switch to persistent Search Results view |
| `Esc`        | Close overlay                            |

### Queue overlay

| Key       | Action                            |
| --------- | --------------------------------- |
| `j` / `↓` | Cursor down                       |
| `k` / `↑` | Cursor up                         |
| `J`       | Move selected item down           |
| `K`       | Move selected item up             |
| `x`       | Remove selected item              |
| `c`       | Clear queue (inline confirmation) |
| `Esc`     | Close overlay                     |

### Info Panel — view mode

| Key       | Action          |
| --------- | --------------- |
| `j` / `↓` | Scroll down     |
| `k` / `↑` | Scroll up       |
| `e`       | Enter edit mode |
| `Esc`     | Close overlay   |

### Info Panel — edit mode

| Key         | Action                        |
| ----------- | ----------------------------- |
| `Tab`       | Focus next field              |
| `Shift+Tab` | Focus previous field          |
| `Enter`     | Advance to write confirmation |
| `Esc`       | Prompt to discard changes     |

### Browse Mode Picker / Sort Picker / Action Dialog

| Key          | Action                       |
| ------------ | ---------------------------- |
| `j` / `↓`    | Next option                  |
| `k` / `↑`    | Previous option              |
| `Enter`      | Execute selected option      |
| Shortcut key | Execute that option directly |
| `Esc`        | Close without action         |
