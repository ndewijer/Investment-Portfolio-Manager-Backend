package database

import (
	"database/sql"
	"embed"
	"io/fs"
	"strconv"
	"strings"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate runs all pending migrations.
// On a fresh DB this creates the full schema.
// On an existing DB it applies only new migrations.
func Migrate(db *sql.DB) error {
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite"); err != nil {
		return err
	}
	return goose.Up(db, "migrations")
}

// HasPendingMigrations reports whether any migration files in the embedded FS
// have not yet been applied to the database. It compares the highest migration
// file version against the current DB version — mirroring the Python Alembic
// head_rev vs current_rev check. This avoids false positives when the app
// version advances without a corresponding new migration file.
func HasPendingMigrations(db *sql.DB) (bool, error) {
	var dbVersion int64
	err := db.QueryRow(
		"SELECT version_id FROM goose_db_version WHERE is_applied = 1 ORDER BY id DESC LIMIT 1",
	).Scan(&dbVersion)
	if err != nil {
		// goose_db_version table doesn't exist yet — DB needs initialising
		return true, nil
	}

	headVersion, err := headMigrationVersion()
	if err != nil {
		return false, err
	}

	return dbVersion < headVersion, nil
}

// headMigrationVersion returns the highest version number present in the embedded migrations FS.
// Migration filenames follow the goose convention: {version}_{name}.sql
func headMigrationVersion() (int64, error) {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return 0, err
	}

	var head int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Extract the numeric prefix before the first underscore
		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 2 {
			continue
		}
		v, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}
		if v > head {
			head = v
		}
	}
	return head, nil
}
