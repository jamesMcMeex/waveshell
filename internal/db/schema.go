package db

import (
	"database/sql"
	"fmt"
)

type migration struct {
	version int
	apply   func(db *sql.DB) error
}

var migrations = []migration{
	{version: 1, apply: applyV1},
}

func applyV1(d *sql.DB) error {
	return execTx(d, []string{
		`CREATE TABLE IF NOT EXISTS schema_version (
			version    INTEGER NOT NULL UNIQUE,
			applied_at INTEGER NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS artists (
			id        INTEGER PRIMARY KEY,
			name      TEXT    NOT NULL UNIQUE,
			name_sort TEXT    NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS albums (
			id            INTEGER PRIMARY KEY,
			title         TEXT    NOT NULL,
			artist_id     INTEGER REFERENCES artists(id),
			album_artist  TEXT,
			year          INTEGER,
			genre         TEXT,
			label         TEXT,
			grouping      TEXT,
			track_count   INTEGER NOT NULL DEFAULT 0,
			disc_count    INTEGER NOT NULL DEFAULT 1,
			rg_album_gain REAL,
			rg_album_peak REAL,
			r128_album_gain REAL,
			UNIQUE (title, artist_id)
		)`,

		`CREATE TABLE IF NOT EXISTS tracks (
			id               INTEGER PRIMARY KEY,
			file_path        TEXT    NOT NULL UNIQUE,
			file_size_bytes  INTEGER NOT NULL,
			last_modified    INTEGER NOT NULL,
			date_added       INTEGER NOT NULL,
			album_id         INTEGER REFERENCES albums(id),
			artist_id        INTEGER REFERENCES artists(id),
			title            TEXT    NOT NULL,
			artist           TEXT    NOT NULL,
			album_artist     TEXT,
			album            TEXT,
			track_number     INTEGER,
			disc_number      INTEGER,
			year             INTEGER,
			genre            TEXT,
			grouping         TEXT,
			label            TEXT,
			duration_ms      INTEGER NOT NULL,
			format           TEXT    NOT NULL,
			codec            TEXT    NOT NULL,
			container        TEXT    NOT NULL,
			sample_rate      INTEGER NOT NULL,
			bit_depth        INTEGER,
			bitrate          INTEGER NOT NULL,
			rg_track_gain    REAL,
			rg_track_peak    REAL,
			rg_album_gain    REAL,
			rg_album_peak    REAL,
			r128_track_gain  REAL,
			r128_album_gain  REAL,
			has_artwork      INTEGER NOT NULL DEFAULT 0,
			artwork_width    INTEGER,
			artwork_height   INTEGER,
			artwork_format   TEXT,
			artwork_size_bytes INTEGER
		)`,

		`CREATE INDEX IF NOT EXISTS idx_tracks_file_path     ON tracks (file_path)`,
		`CREATE INDEX IF NOT EXISTS idx_tracks_last_modified ON tracks (last_modified)`,
		`CREATE INDEX IF NOT EXISTS idx_tracks_artist_id     ON tracks (artist_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tracks_album_id      ON tracks (album_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tracks_label         ON tracks (label)`,
		`CREATE INDEX IF NOT EXISTS idx_tracks_genre         ON tracks (genre)`,
		`CREATE INDEX IF NOT EXISTS idx_tracks_grouping      ON tracks (grouping)`,
		`CREATE INDEX IF NOT EXISTS idx_tracks_year          ON tracks (year)`,

		`CREATE INDEX IF NOT EXISTS idx_albums_artist_id  ON albums (artist_id)`,
		`CREATE INDEX IF NOT EXISTS idx_albums_label      ON albums (label)`,
		`CREATE INDEX IF NOT EXISTS idx_albums_genre      ON albums (genre)`,
		`CREATE INDEX IF NOT EXISTS idx_albums_grouping   ON albums (grouping)`,
		`CREATE INDEX IF NOT EXISTS idx_albums_year       ON albums (year)`,

		`CREATE INDEX IF NOT EXISTS idx_artists_name_sort ON artists (name_sort)`,

		`INSERT OR IGNORE INTO schema_version (version, applied_at) VALUES (1, unixepoch())`,
	})
}

func execTx(d *sql.DB, stmts []string) (err error) {
	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:60], err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	tx = nil
	return nil
}
