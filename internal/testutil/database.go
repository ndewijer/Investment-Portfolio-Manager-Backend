package testutil

import (
	"database/sql"
	"testing"

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

// createTestSchema creates all database tables for testing.
// Schema is synchronized with the production database schema.
//
//nolint:funlen // Database schema DDL
func createTestSchema(db *sql.DB) error {
	schema := `
		-- Portfolio table
		CREATE TABLE IF NOT EXISTS portfolio (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			description TEXT,
			is_archived BOOLEAN,
			exclude_from_overview BOOLEAN DEFAULT FALSE NOT NULL
		);

		-- Fund table
		CREATE TABLE fund (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			isin VARCHAR(12) NOT NULL UNIQUE,
			symbol VARCHAR(10),
			currency VARCHAR(3) NOT NULL,
			exchange VARCHAR(50) NOT NULL,
			investment_type VARCHAR(5) NOT NULL,
			dividend_type VARCHAR(5) NOT NULL
		);

		-- Portfolio-Fund junction table
		CREATE TABLE IF NOT EXISTS portfolio_fund (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			portfolio_id VARCHAR(36) NOT NULL,
			fund_id VARCHAR(36) NOT NULL,
			FOREIGN KEY(portfolio_id) REFERENCES portfolio(id) ON DELETE CASCADE,
			FOREIGN KEY(fund_id) REFERENCES fund(id) ON DELETE CASCADE,
			CONSTRAINT unique_portfolio_fund UNIQUE (portfolio_id, fund_id)
		);

		-- Transaction table (quoted because transaction is a reserved keyword)
		CREATE TABLE IF NOT EXISTS "transaction" (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			portfolio_fund_id VARCHAR(36) NOT NULL,
			date DATE NOT NULL,
			type VARCHAR(10) NOT NULL,
			shares FLOAT NOT NULL,
			cost_per_share FLOAT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(portfolio_fund_id) REFERENCES portfolio_fund(id) ON DELETE CASCADE
		);

		-- Dividend table
		CREATE TABLE IF NOT EXISTS dividend (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			fund_id VARCHAR(36) NOT NULL,
			portfolio_fund_id VARCHAR(36) NOT NULL,
			record_date DATE NOT NULL,
			ex_dividend_date DATE NOT NULL,
			shares_owned FLOAT NOT NULL,
			dividend_per_share FLOAT NOT NULL,
			total_amount FLOAT NOT NULL,
			reinvestment_status VARCHAR(9) NOT NULL,
			buy_order_date DATE,
			reinvestment_transaction_id VARCHAR(36),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(fund_id) REFERENCES fund(id),
			FOREIGN KEY(portfolio_fund_id) REFERENCES portfolio_fund(id) ON DELETE CASCADE,
			FOREIGN KEY(reinvestment_transaction_id) REFERENCES "transaction"(id) ON DELETE CASCADE
		);

		-- Fund price table
		CREATE TABLE fund_price (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			fund_id VARCHAR(36) NOT NULL,
			date DATE NOT NULL,
			price FLOAT NOT NULL,
			FOREIGN KEY(fund_id) REFERENCES fund(id)
		);

		-- Realized Gain/Loss table
		CREATE TABLE realized_gain_loss (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			portfolio_id VARCHAR(36) NOT NULL,
			fund_id VARCHAR(36) NOT NULL,
			transaction_id VARCHAR(36) NOT NULL,
			transaction_date DATE NOT NULL,
			shares_sold FLOAT NOT NULL,
			cost_basis FLOAT NOT NULL,
			sale_proceeds FLOAT NOT NULL,
			realized_gain_loss FLOAT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(portfolio_id) REFERENCES portfolio(id) ON DELETE CASCADE,
			FOREIGN KEY(fund_id) REFERENCES fund(id),
			FOREIGN KEY(transaction_id) REFERENCES "transaction"(id) ON DELETE CASCADE
		);

		-- Exchange rate table
		CREATE TABLE exchange_rate (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			from_currency VARCHAR(3) NOT NULL,
			to_currency VARCHAR(3) NOT NULL,
			rate FLOAT NOT NULL,
			date DATE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT unique_exchange_rate UNIQUE (from_currency, to_currency, date)
		);

		-- System setting table
		CREATE TABLE system_setting (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			"key" VARCHAR(15) NOT NULL UNIQUE,
			value VARCHAR(255) NOT NULL,
			updated_at DATETIME
		);

		-- Symbol info table
		CREATE TABLE symbol_info (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			symbol VARCHAR(10) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			exchange VARCHAR(50),
			currency VARCHAR(3),
			isin VARCHAR(12) UNIQUE,
			last_updated DATETIME,
			data_source VARCHAR(50),
			is_valid BOOLEAN
		);

		-- IBKR configuration table
		CREATE TABLE IF NOT EXISTS ibkr_config (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			flex_token VARCHAR(500) NOT NULL,
			flex_query_id VARCHAR(100) NOT NULL,
			token_expires_at DATETIME,
			last_import_date DATETIME,
			auto_import_enabled BOOLEAN NOT NULL,
			created_at DATETIME DEFAULT (CURRENT_TIMESTAMP),
			updated_at DATETIME DEFAULT (CURRENT_TIMESTAMP),
			enabled BOOLEAN NOT NULL,
			default_allocation_enabled BOOLEAN NOT NULL,
			default_allocations TEXT
		);

		-- IBKR transaction table
		CREATE TABLE ibkr_transaction (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			ibkr_transaction_id VARCHAR(100) NOT NULL UNIQUE,
			transaction_date DATE NOT NULL,
			symbol VARCHAR(10),
			isin VARCHAR(12),
			description TEXT,
			transaction_type VARCHAR(20) NOT NULL,
			quantity FLOAT,
			price FLOAT,
			total_amount FLOAT NOT NULL,
			currency VARCHAR(3) NOT NULL,
			fees FLOAT NOT NULL,
			status VARCHAR(20) NOT NULL,
			imported_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			processed_at DATETIME,
			raw_data TEXT
		);

		-- IBKR transaction allocation table
		CREATE TABLE ibkr_transaction_allocation (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			ibkr_transaction_id VARCHAR(36) NOT NULL,
			portfolio_id VARCHAR(36) NOT NULL,
			allocation_percentage FLOAT NOT NULL,
			allocated_amount FLOAT NOT NULL,
			allocated_shares FLOAT NOT NULL,
			transaction_id VARCHAR(36),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(ibkr_transaction_id) REFERENCES ibkr_transaction(id) ON DELETE CASCADE,
			FOREIGN KEY(portfolio_id) REFERENCES portfolio(id) ON DELETE CASCADE,
			FOREIGN KEY(transaction_id) REFERENCES "transaction"(id) ON DELETE CASCADE
		);

		-- IBKR import cache table
		CREATE TABLE ibkr_import_cache (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			cache_key VARCHAR(255) NOT NULL UNIQUE,
			data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		);

		-- Fund history materialized table
		CREATE TABLE fund_history_materialized (
			id VARCHAR(36) NOT NULL PRIMARY KEY,
			portfolio_fund_id VARCHAR(36) NOT NULL,
			fund_id VARCHAR(36) NOT NULL,
			date VARCHAR(10) NOT NULL,
			shares FLOAT NOT NULL,
			price FLOAT NOT NULL,
			value FLOAT NOT NULL,
			cost FLOAT NOT NULL,
			realized_gain FLOAT NOT NULL,
			unrealized_gain FLOAT NOT NULL,
			total_gain_loss FLOAT NOT NULL,
			dividends FLOAT NOT NULL,
			fees FLOAT NOT NULL,
			calculated_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
			FOREIGN KEY(portfolio_fund_id) REFERENCES portfolio_fund(id) ON DELETE CASCADE,
			CONSTRAINT uq_portfolio_fund_date UNIQUE (portfolio_fund_id, date)
		);

		-- Indexes for performance
		CREATE INDEX IF NOT EXISTS ix_realized_gain_loss_portfolio_id ON realized_gain_loss(portfolio_id);
		CREATE INDEX IF NOT EXISTS ix_realized_gain_loss_fund_id ON realized_gain_loss(fund_id);
		CREATE INDEX IF NOT EXISTS ix_realized_gain_loss_transaction_date ON realized_gain_loss(transaction_date);
		CREATE INDEX IF NOT EXISTS ix_realized_gain_loss_transaction_id ON realized_gain_loss(transaction_id);
		CREATE INDEX IF NOT EXISTS ix_exchange_rate_date ON exchange_rate(date);
		CREATE INDEX IF NOT EXISTS ix_fund_price_date ON fund_price(date);
		CREATE INDEX IF NOT EXISTS ix_fund_price_fund_id ON fund_price(fund_id);
		CREATE INDEX IF NOT EXISTS ix_fund_price_fund_id_date ON fund_price(fund_id, date);
		CREATE INDEX IF NOT EXISTS ix_transaction_date ON "transaction"(date);
		CREATE INDEX IF NOT EXISTS ix_transaction_portfolio_fund_id ON "transaction"(portfolio_fund_id);
		CREATE INDEX IF NOT EXISTS ix_transaction_portfolio_fund_id_date ON "transaction"(portfolio_fund_id, date);
		CREATE INDEX IF NOT EXISTS ix_ibkr_transaction_status ON ibkr_transaction(status);
		CREATE INDEX IF NOT EXISTS ix_ibkr_transaction_date ON ibkr_transaction(transaction_date);
		CREATE INDEX IF NOT EXISTS ix_ibkr_transaction_ibkr_id ON ibkr_transaction(ibkr_transaction_id);
		CREATE INDEX IF NOT EXISTS ix_ibkr_allocation_ibkr_transaction_id ON ibkr_transaction_allocation(ibkr_transaction_id);
		CREATE INDEX IF NOT EXISTS ix_ibkr_allocation_portfolio_id ON ibkr_transaction_allocation(portfolio_id);
		CREATE INDEX IF NOT EXISTS ix_ibkr_allocation_transaction_id ON ibkr_transaction_allocation(transaction_id);
		CREATE INDEX IF NOT EXISTS ix_ibkr_cache_expires_at ON ibkr_import_cache(expires_at);
		CREATE INDEX IF NOT EXISTS idx_fund_history_pf_date ON fund_history_materialized(portfolio_fund_id, date);
		CREATE INDEX IF NOT EXISTS idx_fund_history_date ON fund_history_materialized(date);
		CREATE INDEX IF NOT EXISTS idx_fund_history_fund_id ON fund_history_materialized(fund_id);
		CREATE INDEX IF NOT EXISTS ix_dividend_fund_id ON dividend(fund_id);
		CREATE INDEX IF NOT EXISTS ix_dividend_portfolio_fund_id ON dividend(portfolio_fund_id);
		CREATE INDEX IF NOT EXISTS ix_dividend_record_date ON dividend(record_date);
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
	}

	for _, table := range tables {
		//nolint:gosec // G202: Table names are from hardcoded slice, no SQL injection risk
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
