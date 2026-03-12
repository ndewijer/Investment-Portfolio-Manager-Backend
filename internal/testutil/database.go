package testutil

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/database"
	_ "modernc.org/sqlite" // Test Package
)

// SetupTestDB creates an in-memory SQLite database for testing.
// The database is automatically cleaned up when the test completes.
//
// Example usage:
//
//	func TestSomething(t *testing.T) {
//	    db := testutil.SetupTestDB(t)
//	    // db is ready to use with schema created
//	}
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Named shared in-memory database: all connections to the same name share
	// the same data, so transactions see schema and test data without needing
	// to pin to a single connection. Each test gets a unique name so they
	// remain fully isolated from one another.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_texttotime=1", uuid.New().String())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Configure SQLite for testing
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA timezone = 'UTC'",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			t.Fatalf("Failed to set pragma: %v", err)
		}
	}

	// Create schema
	if err := createTestSchema(db); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	// Cleanup when test ends
	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// createTestSchema creates all database tables for testing.
// Schema is synchronized with the production database schema.
//
//nolint:funlen // Database schema DDL
func createTestSchema(db *sql.DB) error {
	return database.Migrate(db)
}

// CleanDatabase truncates all tables in dependency order.
// Useful for reusing the same database across multiple tests.
//
// Example usage:
//
//	func TestMultipleThings(t *testing.T) {
//	    db := testutil.SetupTestDB(t)
//
//	    t.Run("First test", func(t *testing.T) {
//	        // Create data
//	        testutil.CleanDatabase(t, db)  // Clean after
//	    })
//
//	    t.Run("Second test", func(t *testing.T) {
//	        // Fresh clean database
//	    })
//	}
func CleanDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	// Order matters: delete children before parents due to foreign keys
	tables := []string{
		"fund_history_materialized",
		"ibkr_transaction_allocation",
		"ibkr_transaction",
		"ibkr_import_cache",
		"ibkr_config",
		"realized_gain_loss",
		"dividend",
		"transaction",
		"fund_price",
		"portfolio_fund",
		"fund",
		"portfolio",
		"exchange_rate",
		"system_setting",
		"symbol_info",
		"log",
	}

	for _, table := range tables {
		//nolint:gosec // G201: Table names are from hardcoded slice, no SQL injection risk. Sprintf to handle "transaction" table
		query := fmt.Sprintf("DELETE FROM \"%s\"", table)
		if _, err := db.Exec(query); err != nil {
			t.Fatalf("Failed to clean table %s: %v", table, err)
		}
	}
}

// CountRows returns the number of rows in a table.
// Useful for assertions in tests.
//
// Example usage:
//
//	count := testutil.CountRows(t, db, "portfolio")
//	assert.Equal(t, 2, count)
func CountRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()

	var count int
	//nolint:gosec // G201: Table names are from hardcoded slice, no SQL injection risk. Sprintf to handle "transaction" table
	query := fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", table)
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows in %s: %v", table, err)
	}

	return count
}

// AssertRowCount asserts that a table has the expected number of rows.
//
// Example usage:
//
//	testutil.AssertRowCount(t, db, "portfolio", 2)
func AssertRowCount(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()

	actual := CountRows(t, db, table)
	if actual != expected {
		t.Errorf("Expected %d rows in %s, got %d", expected, table, actual)
	}
}
