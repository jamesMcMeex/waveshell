# waveshell — SQLite Schema Reference

> **Status:** Pre-development
> **Scope:** Canonical DDL for all tables, indexes, and migration versioning.
> The scanner, db, search, tagger, and queue packages all derive their data contracts from this document.
> **Last updated:** June 2026

---

## Table of Contents

1. [Design Decisions](#1-design-decisions)
2. [Initialisation](#2-initialisation)
3. [Schema Version Table](#3-schema-version-table)
4. [Core Tables](#4-core-tables)
   - [artists](#41-artists)
   - [albums](#42-albums)
   - [tracks](#43-tracks)
5. [Play History](#5-play-history)
6. [Playlists (post-MVP)](#6-playlists-post-mvp)
7. [Indexes](#7-indexes)
8. [Migration Strategy](#8-migration-strategy)
9. [Query Patterns](#9-query-patterns)

---

## 1. Design Decisions

These decisions are recorded here so they do not get re-litigated during implementation.

**Duration is stored as INTEGER milliseconds, not REAL seconds.** Floating-point duration comparisons are a source of subtle bugs. Milliseconds are exact integers and are what `dhowden/tag` returns after conversion.

**Timestamps are INTEGER Unix seconds (UTC).** No timezone handling in the database layer. The view layer formats them for display. `last_modified` comes from `os.FileInfo.ModTime().Unix()`. `date_added` is set once on first scan and never updated on rescan.

**Artwork is stored at the track level.** Each audio file embeds its own artwork independently. When displaying album art, the application queries the first track in the album. No separate artwork blob table — only dimensions, format, and size are stored; the raw bytes are read from the file on demand.

**`label` and `grouping` are separate columns on both `tracks` and `albums`.** They are never merged or inferred from each other. See the PRD §11 for the tag field mapping rationale.

**`albums` is a first-class table, not a view.** The middle pane in all browse modes queries albums directly. Storing album-level data (especially album ReplayGain) in a dedicated table avoids repeated aggregation at query time. Album rows are created and updated by the scanner.

**`album_artist` is stored on `tracks` as a plain TEXT field and also as a foreign key to `artists`.** The `artists` table is the source for the left pane in `artist` browse mode. The `album_artist` text column allows the raw tag value to be displayed without a join.

**Bitrate is the actual decoded bitrate, not the nominal/header value.** The scanner computes this from file size and duration where the tag library does not expose it directly.

**`bit_depth` is nullable.** Lossy formats (MP3, OGG, AAC) do not have a meaningful bit depth. The info panel renders `—` for NULL.

**No foreign key enforcement at the SQLite level in production.** `PRAGMA foreign_keys = ON` is enabled in tests against `:memory:` databases, where it catches schema regressions. In production it is left off for performance. The application code is responsible for maintaining referential integrity during scan writes.

---

## 2. Initialisation

The following PRAGMAs are set once at database open time, before any DDL or DML:

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous   = NORMAL;
PRAGMA foreign_keys  = OFF;   -- see design decisions above
PRAGMA busy_timeout  = 5000;  -- 5 seconds; prevents immediate SQLITE_BUSY on concurrent access
```

WAL mode is persistent once set — it survives database close/reopen. `synchronous = NORMAL` is safe with WAL and significantly faster than `FULL`.

---

## 3. Schema Version Table

A single-row table tracks the schema version. The application reads this on startup and applies any outstanding migrations before opening the rest of the database.

```sql
CREATE TABLE IF NOT EXISTS schema_version (
    version     INTEGER NOT NULL,
    applied_at  INTEGER NOT NULL   -- Unix seconds
);
```

The initial schema (this document) is **version 1**. Insert on first run:

```sql
INSERT INTO schema_version (version, applied_at) VALUES (1, unixepoch());
```

---

## 4. Core Tables

### 4.1 artists

The left pane source for `artist` browse mode. One row per distinct album artist value encountered during scanning.

```sql
CREATE TABLE IF NOT EXISTS artists (
    id          INTEGER PRIMARY KEY,
    name        TEXT    NOT NULL UNIQUE,
    name_sort   TEXT    NOT NULL        -- e.g. "Aphex Twin" → "Aphex Twin"; "The Orb" → "Orb, The"
);
```

**`name_sort`** is used for display ordering in the left pane. The scanner derives it by stripping a leading "The ", "A ", or "An " and appending it as a suffix with a comma. This is applied consistently at scan time; the user never sets it directly. It is not exposed in the info panel.

---

### 4.2 albums

One row per distinct (album title, album artist) pair. Created and updated by the scanner.

```sql
CREATE TABLE IF NOT EXISTS albums (
    id                  INTEGER PRIMARY KEY,
    title               TEXT    NOT NULL,
    artist_id           INTEGER REFERENCES artists(id),   -- NULL for compilations with no single album artist
    album_artist        TEXT,                              -- raw tag value; NULL if not set
    year                INTEGER,
    genre               TEXT,
    label               TEXT,
    grouping            TEXT,
    track_count         INTEGER NOT NULL DEFAULT 0,        -- updated by scanner on each rescan
    disc_count          INTEGER NOT NULL DEFAULT 1,

    -- Album-level ReplayGain (from any track in the album; all tracks share the same values)
    rg_album_gain       REAL,
    rg_album_peak       REAL,
    r128_album_gain     REAL,

    UNIQUE (title, artist_id)
);
```

**`artist_id` is nullable** to support compilations where `album_artist` may be "Various Artists" or absent. The scanner creates an artist row for "Various Artists" if that tag value is present.

**`track_count` and `disc_count`** are maintained by the scanner and used for display in the middle pane (e.g. `Isam · 12 tracks`). They are derived data, not a source of truth.

**Album ReplayGain** is stored on the album row as a convenience. The scanner writes it from the first track it processes for that album. All tracks in a correctly-tagged album carry identical album gain values; if they do not, the value from the last-processed track wins. This is a known limitation with no planned resolution.

---

### 4.3 tracks

The primary table. One row per audio file.

```sql
CREATE TABLE IF NOT EXISTS tracks (
    id                  INTEGER PRIMARY KEY,

    -- File identity
    file_path           TEXT    NOT NULL UNIQUE,
    file_size_bytes     INTEGER NOT NULL,
    last_modified       INTEGER NOT NULL,              -- Unix seconds; used for incremental rescan
    date_added          INTEGER NOT NULL,              -- Unix seconds; set once, never updated on rescan

    -- Relationships
    album_id            INTEGER REFERENCES albums(id),
    artist_id           INTEGER REFERENCES artists(id), -- the track artist, not necessarily album artist

    -- Core tags
    title               TEXT    NOT NULL,
    artist              TEXT    NOT NULL,              -- track artist, raw tag value
    album_artist        TEXT,                          -- raw tag value; may differ from albums.album_artist
    album               TEXT,                          -- raw tag value; denormalised for display without join
    track_number        INTEGER,
    disc_number         INTEGER,
    year                INTEGER,
    genre               TEXT,
    grouping            TEXT,                          -- TIT1 / ©grp
    label               TEXT,                          -- TPUB / LABEL / ----:com.apple.iTunes:LABEL

    -- Audio stream
    duration_ms         INTEGER NOT NULL,              -- milliseconds; exact integer
    format              TEXT    NOT NULL,              -- FLAC | ALAC | MP3 | AIFF | WAV | OGG
    codec               TEXT    NOT NULL,              -- flac | alac | mp3 | pcm | vorbis | aac
    container           TEXT    NOT NULL,              -- FLAC | MP4 | MP3 | AIFF | WAV | OGG
    sample_rate         INTEGER NOT NULL,              -- Hz (e.g. 44100, 48000, 96000)
    bit_depth           INTEGER,                       -- NULL for lossy formats
    bitrate             INTEGER NOT NULL,              -- kbps, actual decoded value

    -- ReplayGain
    rg_track_gain       REAL,                          -- dB, e.g. -7.23
    rg_track_peak       REAL,                          -- 0.0–1.0
    rg_album_gain       REAL,                          -- dB
    rg_album_peak       REAL,                          -- 0.0–1.0
    r128_track_gain     REAL,                          -- EBU R128 offset
    r128_album_gain     REAL,                          -- EBU R128 album offset

    -- Embedded artwork (presence and metadata only; raw bytes read from file on demand)
    has_artwork         INTEGER NOT NULL DEFAULT 0,    -- 0 | 1
    artwork_width       INTEGER,                       -- px; NULL if no artwork
    artwork_height      INTEGER,                       -- px; NULL if no artwork
    artwork_format      TEXT,                          -- JPEG | PNG | NULL
    artwork_size_bytes  INTEGER                        -- NULL if no artwork
);
```

**Format/codec/container mapping for reference:**

| Format label | Codec value | Container value | Extension(s)    |
| ------------ | ----------- | --------------- | --------------- |
| `FLAC`       | `flac`      | `FLAC`          | `.flac`         |
| `ALAC`       | `alac`      | `MP4`           | `.m4a`, `.alac` |
| `MP3`        | `mp3`       | `MP3`           | `.mp3`          |
| `AIFF`       | `pcm`       | `AIFF`          | `.aiff`, `.aif` |
| `WAV`        | `pcm`       | `WAV`           | `.wav`          |
| `OGG`        | `vorbis`    | `OGG`           | `.ogg`          |

These are the normalised string values the application writes and reads. No other variants are valid.

**Fields that are NOT editable via the tag editor** — they are derived from the audio stream, not from writable tag frames, and must not appear as editable `textinput` components in the info panel:

- `duration_ms`, `format`, `codec`, `container`, `sample_rate`, `bit_depth`, `bitrate`
- `file_size_bytes`, `last_modified`, `date_added`
- `has_artwork`, `artwork_width`, `artwork_height`, `artwork_format`, `artwork_size_bytes`
  All other non-ID, non-relationship fields are editable.

---

## 5. Play History

Written when a track is played to at least 50% of its duration. Never updated — append-only.

```sql
CREATE TABLE IF NOT EXISTS play_history (
    id          INTEGER PRIMARY KEY,
    track_id    INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    played_at   INTEGER NOT NULL,   -- Unix seconds
    percent_complete INTEGER NOT NULL  -- 0–100; the actual percentage reached at stop/skip
);
```

`percent_complete` records the actual value rather than a boolean so that future smart playlist features ("unplayed", "never completed") have the data they need without a schema migration.

`ON DELETE CASCADE` means history rows are removed if the track is removed from the library (e.g. file deleted and library rescanned). This is the correct behaviour — orphaned history rows for deleted files have no utility.

---

## 6. Playlists (post-MVP)

These tables are included in the initial schema so that the version 1 DDL is the complete baseline. They will not be queried by any MVP code path.

```sql
CREATE TABLE IF NOT EXISTS playlists (
    id          INTEGER PRIMARY KEY,
    name        TEXT    NOT NULL UNIQUE,
    created_at  INTEGER NOT NULL,   -- Unix seconds
    updated_at  INTEGER NOT NULL    -- Unix seconds; updated on any track add/remove/reorder
);

CREATE TABLE IF NOT EXISTS playlist_tracks (
    id          INTEGER PRIMARY KEY,
    playlist_id INTEGER NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
    track_id    INTEGER NOT NULL REFERENCES tracks(id)    ON DELETE CASCADE,
    position    INTEGER NOT NULL,   -- 1-based; the user-defined order
    UNIQUE (playlist_id, track_id),
    UNIQUE (playlist_id, position)
);
```

**`position` is 1-based** and is the displayed `#` value in playlist browse mode. Reorder operations (`J`/`K` in the tracks pane) update `position` values directly with a gap-closing approach: swap the target track's position with its neighbour rather than renumbering the whole playlist.

**`UNIQUE (playlist_id, track_id)`** prevents the same track appearing twice in one playlist. If this constraint needs to be relaxed in future (intentional duplicates), it requires a migration.

---

## 7. Indexes

```sql
-- Incremental rescan: check file_path and last_modified before re-extracting metadata
CREATE INDEX IF NOT EXISTS idx_tracks_file_path      ON tracks (file_path);
CREATE INDEX IF NOT EXISTS idx_tracks_last_modified  ON tracks (last_modified);

-- Artist browse mode: left pane and middle pane population
CREATE INDEX IF NOT EXISTS idx_tracks_artist_id      ON tracks (artist_id);
CREATE INDEX IF NOT EXISTS idx_tracks_album_id       ON tracks (album_id);

-- Tag-slice browse modes: left pane population for label, genre, grouping, year
CREATE INDEX IF NOT EXISTS idx_tracks_label          ON tracks (label);
CREATE INDEX IF NOT EXISTS idx_tracks_genre          ON tracks (genre);
CREATE INDEX IF NOT EXISTS idx_tracks_grouping       ON tracks (grouping);
CREATE INDEX IF NOT EXISTS idx_tracks_year           ON tracks (year);

-- Albums: middle pane population filtered by left pane selection
CREATE INDEX IF NOT EXISTS idx_albums_artist_id      ON albums (artist_id);
CREATE INDEX IF NOT EXISTS idx_albums_label          ON albums (label);
CREATE INDEX IF NOT EXISTS idx_albums_genre          ON albums (genre);
CREATE INDEX IF NOT EXISTS idx_albums_grouping       ON albums (grouping);
CREATE INDEX IF NOT EXISTS idx_albums_year           ON albums (year);

-- Artists: name_sort for ordered left pane display
CREATE INDEX IF NOT EXISTS idx_artists_name_sort     ON artists (name_sort);

-- Play history: recent plays query, per-track history
CREATE INDEX IF NOT EXISTS idx_play_history_track_id  ON play_history (track_id);
CREATE INDEX IF NOT EXISTS idx_play_history_played_at ON play_history (played_at);

-- Playlists (post-MVP)
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_playlist_id ON playlist_tracks (playlist_id, position);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_track_id    ON playlist_tracks (track_id);
```

---

## 8. Migration Strategy

The migration system is intentionally minimal. A single `schema_version` table (§3) holds the current version. The application applies migrations in order at startup and halts with a blocking error dialog if any migration fails.

### Implementation

`internal/db` exposes a single exported function:

```go
// Migrate applies all outstanding migrations up to the current schema version.
// It is called once at application startup, before any other database access.
// It is idempotent: running it on an already-current database is a no-op.
func Migrate(db *sql.DB) error
```

Internally, migrations are a slice of structs defined in code — not SQL files on disk. This keeps the binary self-contained with no external file dependency:

```go
type migration struct {
    version int
    apply   func(tx *sql.Tx) error
}

var migrations = []migration{
    {
        version: 1,
        apply:   applyV1,  // creates all tables and indexes in this document
    },
    // future: { version: 2, apply: applyV2 }
}
```

Each migration runs in a transaction. If it fails, the transaction rolls back and the application halts. The `schema_version` row is updated inside the same transaction as the schema change, so a crash mid-migration leaves the version unchanged and the migration will be retried on next launch.

### Version increment rules

| Change type                          | Action                                                |
| ------------------------------------ | ----------------------------------------------------- |
| New table                            | New migration version                                 |
| New column, nullable or with DEFAULT | New migration version                                 |
| New column, NOT NULL without DEFAULT | New migration version + backfill                      |
| New index                            | New migration version                                 |
| Rename column or table               | New migration version (SQLite requires table rebuild) |
| Removing a column                    | New migration version (SQLite requires table rebuild) |
| Change to post-MVP tables only       | Can be bundled with other changes or standalone       |

---

## 9. Query Patterns

Reference queries for the three-pane browser, parameterised on browse mode. These are the queries the `internal/db` package must expose; the exact function signatures belong in that package, not here.

### Left pane — artist mode

```sql
SELECT id, name, name_sort FROM artists ORDER BY name_sort;
```

### Left pane — label mode

```sql
SELECT DISTINCT label FROM tracks WHERE label IS NOT NULL ORDER BY label;
```

_(Same pattern for `genre`, `grouping`, `year`.)_

### Middle pane — artist mode (albums by selected artist)

```sql
SELECT id, title, year, track_count
FROM   albums
WHERE  artist_id = ?
ORDER  BY year, title;
```

### Middle pane — label mode (albums with selected label)

```sql
SELECT DISTINCT a.id, a.title, a.year, a.track_count
FROM   albums a
JOIN   tracks t ON t.album_id = a.id
WHERE  t.label = ?
ORDER  BY a.year, a.title;
```

_(Same join pattern for `genre`, `grouping`, `year`.)_

### Tracks pane (tracks on selected album)

```sql
SELECT id, title, artist, track_number, disc_number,
       duration_ms, format, sample_rate, bit_depth, bitrate,
       label, grouping, genre, year, file_path
FROM   tracks
WHERE  album_id = ?
ORDER  BY disc_number, track_number;
```

The `ORDER BY` clause is the default sort. When the user applies a non-default sort, the `ORDER BY` is replaced at query construction time by the active `SortField` and `SortDirection`. Sort is applied in the database, not in Go, to keep the result set consistent with what is displayed.

### Incremental rescan — file identity check

```sql
SELECT id, last_modified FROM tracks WHERE file_path = ?;
```

If the row does not exist, insert. If `last_modified` differs from `os.FileInfo.ModTime().Unix()`, update. Otherwise skip.

### Play history — completion write

```sql
INSERT INTO play_history (track_id, played_at, percent_complete)
VALUES (?, unixepoch(), ?);
```
