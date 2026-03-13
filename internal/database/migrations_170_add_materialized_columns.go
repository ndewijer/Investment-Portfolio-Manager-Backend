package database

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddNamedMigrationNoTxContext("170_add_materialized_columns.go", up170, down170)
	registerGoMigrationVersion(170)
}

func up170(_ context.Context, db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(fund_history_materialized)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasSaleProceeds := false
	hasOriginalCost := false
	for rows.Next() {
		var cid, notNull, pk int
		var name, colType string
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == "sale_proceeds" {
			hasSaleProceeds = true
		}
		if name == "original_cost" {
			hasOriginalCost = true
		}
	}

	if !hasSaleProceeds {
		if _, err := db.Exec("ALTER TABLE fund_history_materialized ADD COLUMN sale_proceeds FLOAT NOT NULL DEFAULT 0"); err != nil {
			return err
		}
	}
	if !hasOriginalCost {
		if _, err := db.Exec("ALTER TABLE fund_history_materialized ADD COLUMN original_cost FLOAT NOT NULL DEFAULT 0"); err != nil {
			return err
		}
	}
	return nil
}

func down170(_ context.Context, db *sql.DB) error {
	if _, err := db.Exec("ALTER TABLE fund_history_materialized DROP COLUMN sale_proceeds"); err != nil {
		return err
	}
	if _, err := db.Exec("ALTER TABLE fund_history_materialized DROP COLUMN original_cost"); err != nil {
		return err
	}
	return nil
}
