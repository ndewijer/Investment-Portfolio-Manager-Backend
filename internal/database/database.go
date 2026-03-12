package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // SQLite driver
)

func EnsureDir(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o750)
}

// Open opens a connection to the SQLite database
func Open(dbPath string) (*sql.DB, error) {
	// Build DSN with _texttotime=1 so the driver auto-parses DATE/DATETIME
	// TEXT columns into time.Time (modernc.org/sqlite v1.46.0+).
	dsn := dbPath
	if strings.Contains(dsn, "?") {
		dsn += "&_texttotime=1"
	} else {
		dsn += "?_texttotime=1"
	}

	// Open database connection
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Set timezone to UTC
	if _, err := db.Exec("PRAGMA timezone = 'UTC'"); err != nil {
		return nil, fmt.Errorf("failed to set timezone: %w", err)
	}

	// Enable WAL mode for concurrent read/write support. Without this,
	// background goroutines (e.g., materialized view regeneration) that
	// attempt writes while another transaction is open get SQLITE_BUSY.
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set busy timeout so concurrent writers queue instead of failing
	// immediately with SQLITE_BUSY. 5 seconds is generous for background jobs.
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	return db, nil
}

// HealthCheck performs a simple health check on the database
func HealthCheck(db *sql.DB) error {
	return db.Ping()
}
