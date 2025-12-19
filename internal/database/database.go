package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // SQLite driver
)

// Open opens a connection to the SQLite database
func Open(dbPath string) (*sql.DB, error) {
	// Open database connection
	db, err := sql.Open("sqlite", dbPath)
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

	return db, nil
}

// HealthCheck performs a simple health check on the database
func HealthCheck(db *sql.DB) error {
	return db.Ping()
}
