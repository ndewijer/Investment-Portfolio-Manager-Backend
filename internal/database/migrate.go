package database

import (
	"database/sql"
	"embed"

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
