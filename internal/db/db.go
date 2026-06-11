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

func EnsureMigrated(db *sql.DB) error {
	var version int
	err := db.QueryRow(`SELECT version FROM schema_version ORDER BY version DESC LIMIT 1`).Scan(&version)
	if err == sql.ErrNoRows {
		return applyAllMigrations(db)
	}
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if version < len(migrations) {
		return applyPendingMigrations(db, version)
	}
	return nil
}

func applyAllMigrations(db *sql.DB) error {
	for _, m := range migrations {
		if err := m.apply(db); err != nil {
			return fmt.Errorf("migration v%d: %w", m.version, err)
		}
	}
	return nil
}

func applyPendingMigrations(db *sql.DB, currentVersion int) error {
	for _, m := range migrations {
		if m.version > currentVersion {
			if err := m.apply(db); err != nil {
				return fmt.Errorf("migration v%d: %w", m.version, err)
			}
		}
	}
	return nil
}
