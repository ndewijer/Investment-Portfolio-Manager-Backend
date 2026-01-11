package testutil

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
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

	// In-memory database (destroyed when connection closes)
	db, err := sql.Open("sqlite", ":memory:")
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
		"PRAGMA journal_mode = MEMORY", // Faster for tests
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

// createTestSchema creates all database tables for testing
func createTestSchema(db *sql.DB) error {
	schema := `
		-- Portfolio table
		CREATE TABLE IF NOT EXISTS portfolio (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			is_archived BOOLEAN DEFAULT 0,
			exclude_from_overview BOOLEAN DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		-- Fund table
		CREATE TABLE IF NOT EXISTS fund (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			isin TEXT UNIQUE NOT NULL,
			symbol TEXT,
			currency TEXT NOT NULL,
			exchange TEXT NOT NULL,
			investment_type TEXT DEFAULT 'fund',
			dividend_type TEXT DEFAULT 'none',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		-- Portfolio-Fund junction table
		CREATE TABLE IF NOT EXISTS portfolio_fund (
			id TEXT PRIMARY KEY,
			portfolio_id TEXT NOT NULL,
			fund_id TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (portfolio_id) REFERENCES portfolio(id) ON DELETE CASCADE,
			FOREIGN KEY (fund_id) REFERENCES fund(id) ON DELETE CASCADE,
			UNIQUE(portfolio_id, fund_id)
		);

		-- Transaction table (quoted because transaction is a reserved keyword)
		CREATE TABLE IF NOT EXISTS "transaction" (
			id TEXT PRIMARY KEY,
			portfolio_fund_id TEXT NOT NULL,
			date DATE NOT NULL,
			type TEXT NOT NULL,
			shares REAL NOT NULL,
			cost_per_share REAL NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (portfolio_fund_id) REFERENCES portfolio_fund(id) ON DELETE CASCADE
		);

		-- Dividend table
		CREATE TABLE IF NOT EXISTS dividend (
			id TEXT PRIMARY KEY,
			fund_id TEXT NOT NULL,
			portfolio_fund_id TEXT NOT NULL,
			record_date DATE NOT NULL,
			ex_dividend_date DATE NOT NULL,
			shares_owned REAL NOT NULL,
			dividend_per_share REAL NOT NULL,
			total_amount REAL NOT NULL,
			reinvestment_status TEXT DEFAULT 'pending',
			buy_order_date DATE,
			reinvestment_transaction_id TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (fund_id) REFERENCES fund(id) ON DELETE CASCADE,
			FOREIGN KEY (portfolio_fund_id) REFERENCES portfolio_fund(id) ON DELETE CASCADE,
			FOREIGN KEY (reinvestment_transaction_id) REFERENCES "transaction"(id) ON DELETE SET NULL
		);

		-- Fund price table
		CREATE TABLE IF NOT EXISTS fund_price (
			id TEXT PRIMARY KEY,
			fund_id TEXT NOT NULL,
			date DATE NOT NULL,
			price REAL NOT NULL,
			FOREIGN KEY (fund_id) REFERENCES fund(id) ON DELETE CASCADE,
			UNIQUE(fund_id, date)
		);

		-- Realized Gain/Loss table
		CREATE TABLE IF NOT EXISTS realized_gain_loss (
			id TEXT PRIMARY KEY,
			portfolio_id TEXT NOT NULL,
			fund_id TEXT NOT NULL,
			transaction_id TEXT NOT NULL,
			transaction_date DATE NOT NULL,
			shares_sold REAL NOT NULL,
			cost_basis REAL NOT NULL,
			sale_proceeds REAL NOT NULL,
			realized_gain_loss REAL NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (portfolio_id) REFERENCES portfolio(id) ON DELETE CASCADE,
			FOREIGN KEY (fund_id) REFERENCES fund(id) ON DELETE CASCADE,
			FOREIGN KEY (transaction_id) REFERENCES "transaction"(id) ON DELETE CASCADE
		);
	`

	_, err := db.Exec(schema)
	return err
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
		"realized_gain_loss",
		"dividend",
		"transaction",
		"fund_price",
		"portfolio_fund",
		"fund",
		"portfolio",
	}

	for _, table := range tables {
		query := "DELETE FROM " + table
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
	query := "SELECT COUNT(*) FROM " + table
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
