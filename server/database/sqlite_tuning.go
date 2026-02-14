package database

import (
	"database/sql"
	"fmt"
	"log/slog"
)

// ConfigureSQLiteDB applies standard SQLite tuning pragmas and connection pool
// settings to a *sql.DB. This should be called immediately after sql.Open for
// any SQLite database used by the server.
func ConfigureSQLiteDB(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute %s: %w", pragma, err)
		}
	}

	// SQLite is single-writer; one connection avoids lock contention.
	// WAL mode allows concurrent reads from the same connection.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // never expire — SQLite file is local

	slog.Debug("SQLite tuning applied")
	return nil
}
