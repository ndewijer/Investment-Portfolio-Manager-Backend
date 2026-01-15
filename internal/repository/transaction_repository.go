package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

// TransactionRepository provides data access methods for the transaction table.
// It handles retrieving and querying portfolio transactions within specified date ranges.
type TransactionRepository struct {
	db *sql.DB
}

// NewTransactionRepository creates a new TransactionRepository with the provided database connection.
func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// GetTransactions retrieves all transactions for the given portfolio_fund IDs within the specified date range.
// Transactions are sorted by date in ascending order and grouped by portfolio_fund ID.
//
// Parameters:
//   - pfIDs: slice of portfolio_fund IDs to query
//   - startDate: inclusive start date for the query
//   - endDate: inclusive end date for the query
//
// Returns a map of portfolioFundID -> []Transaction. If pfIDs is empty, returns an empty map.
// This grouping allows callers to decide how to aggregate (by portfolio, by fund, etc.) after retrieval.
func (s *TransactionRepository) GetTransactions(pfIDs []string, startDate, endDate time.Time) (map[string][]model.Transaction, error) {
	if len(pfIDs) == 0 {
		return make(map[string][]model.Transaction), nil
	}

	transactionPlaceholders := make([]string, len(pfIDs))
	for i := range transactionPlaceholders {
		transactionPlaceholders[i] = "?"
	}

	// Retrieve all transactions based on returned portfolio_fund IDs
	transactionQuery := `
		SELECT id, portfolio_fund_id, date, type, shares, cost_per_share, created_at
		FROM "transaction"
		WHERE portfolio_fund_id IN (` + strings.Join(transactionPlaceholders, ",") + `)
		AND date >= ?
		AND date <= ?
		ORDER BY date ASC
	`

	// Build args: pfIDs first, then startDate, then endDate
	transactiondArgs := make([]any, 0, len(pfIDs)+2)
	for _, id := range pfIDs {
		transactiondArgs = append(transactiondArgs, id)
	}
	transactiondArgs = append(transactiondArgs, startDate.Format("2006-01-02"))
	transactiondArgs = append(transactiondArgs, endDate.Format("2006-01-02"))

	rows, err := s.db.Query(transactionQuery, transactiondArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction table: %w", err)
	}
	defer rows.Close()

	transactionsByPortfolioFund := make(map[string][]model.Transaction)

	for rows.Next() {

		var dateStr, createdAtStr string
		var t model.Transaction

		err := rows.Scan(
			&t.ID,
			&t.PortfolioFundID,
			&dateStr,
			&t.Type,
			&t.Shares,
			&t.CostPerShare,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction table results: %w", err)
		}
		t.Date, err = ParseTime(dateStr)
		if err != nil || t.Date.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		t.CreatedAt, err = ParseTime(createdAtStr)
		if err != nil || t.CreatedAt.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		transactionsByPortfolioFund[t.PortfolioFundID] = append(transactionsByPortfolioFund[t.PortfolioFundID], t)
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("ERROR during iteration: %v\n", err)
		return nil, fmt.Errorf("error iterating transaction table: %w", err)
	}

	return transactionsByPortfolioFund, nil
}

// GetOldestTransaction finds and returns the date of the earliest transaction across the given portfolio_fund IDs.
// This is used to determine the starting point for historical portfolio calculations.
//
// Returns time.Time{} (zero value) if:
//   - pfIDs is empty
//   - no transactions are found
//   - database query fails
//   - date parsing fails
func (s *TransactionRepository) GetOldestTransaction(pfIDs []string) time.Time {
	if len(pfIDs) == 0 {
		return time.Time{}
	}
	var oldestDateStr string

	oldestTransactionPlaceholders := make([]string, len(pfIDs))
	for i := range oldestTransactionPlaceholders {
		oldestTransactionPlaceholders[i] = "?"
	}

	oldestTransactionQuery := `
		SELECT MIN(date) 
		FROM "transaction"
		WHERE portfolio_fund_id IN (` + strings.Join(oldestTransactionPlaceholders, ",") + `)
		`

	oldestTransactionArgs := make([]any, len(pfIDs))
	for i, id := range pfIDs {
		oldestTransactionArgs[i] = id
	}

	err := s.db.QueryRow(oldestTransactionQuery, oldestTransactionArgs...).Scan(&oldestDateStr)
	if err != nil {
		return time.Time{}
	}

	oldestDate, err := time.Parse("2006-01-02", oldestDateStr)
	if err != nil {
		return time.Time{}
	}

	return oldestDate
}
