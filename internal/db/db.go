// Package db provides all SQLite query and write operations as tea.Cmd
// functions that accept a *sql.DB handle. No DB type or service-object
// abstraction is exported — the handle lives in Model and is threaded through.
package db

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA foreign_keys = OFF",
		"PRAGMA busy_timeout = 5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("set pragma %q: %w", p, err)
		}
	}

	slog.Debug("database opened", "path", path, "journal_mode", "WAL")
	return db, nil
}

func Migrate(db *sql.DB) error {
	for _, m := range migrations {
		slog.Debug("running migration", "version", m.version)
		if err := m.apply(db); err != nil {
			return fmt.Errorf("migration v%d: %w", m.version, err)
		}
		slog.Debug("migration complete", "version", m.version)
	}
	return nil
}
