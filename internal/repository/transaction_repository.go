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

	//#nosec G202 -- Safe: placeholders are generated programmatically, not from user input
	transactionQuery := `
		SELECT id, portfolio_fund_id, date, type, shares, cost_per_share, created_at
		FROM "transaction"
		WHERE portfolio_fund_id IN (` + strings.Join(transactionPlaceholders, ",") + `)
		AND date >= ?
		AND date <= ?
		ORDER BY date ASC
	`

	transactionArgs := make([]any, 0, len(pfIDs)+2)
	for _, id := range pfIDs {
		transactionArgs = append(transactionArgs, id)
	}
	transactionArgs = append(transactionArgs, startDate.Format("2006-01-02"))
	transactionArgs = append(transactionArgs, endDate.Format("2006-01-02"))

	rows, err := s.db.Query(transactionQuery, transactionArgs...)
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
	if err == sql.ErrNoRows {
		return transactionsByPortfolioFund, nil
	}

	if err = rows.Err(); err != nil {
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
	var oldestDateStr sql.NullString

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
	if err != nil || !oldestDateStr.Valid {
		return time.Time{}
	}
	oldestDate, err := time.Parse("2006-01-02", oldestDateStr.String)
	if err != nil {
		return time.Time{}
	}

	return oldestDate
}

// GetTransactionsPerPortfolio retrieves all transactions for a specific portfolio or all transactions if portfolioId is empty.
// Returns enriched transaction data including fund names and IBKR linkage status.
// Transactions are sorted by date in ascending order.
func (s *TransactionRepository) GetTransactionsPerPortfolio(portfolioID string) ([]model.TransactionResponse, error) {

	transactionQuery := `
		SELECT 
			t.id, 
			t.portfolio_fund_id, 
			f.name, 
			t.date, 
			t.type, 
			t.shares, 
			t.cost_per_share, 
			ita.ibkr_transaction_id,
			CASE 
				WHEN ita.ibkr_transaction_id IS NOT NULL THEN 1 
				ELSE 0 
			END AS ibkr_linked
		FROM "transaction" t
		JOIN portfolio_fund pf ON t.portfolio_fund_id = pf.id
		JOIN portfolio p ON pf.portfolio_id = p.id
		JOIN fund f ON pf.fund_id = f.id
		LEFT JOIN ibkr_transaction_allocation ita ON t.id = ita.transaction_id
	`

	var args []any

	if portfolioID == "" {
		transactionQuery += `
		ORDER BY t.date ASC
		`
	} else {
		transactionQuery += `
		WHERE pf.portfolio_id = ?
		ORDER BY t.date ASC
		`
		args = append(args, portfolioID)
	}

	rows, err := s.db.Query(transactionQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction table: %w", err)
	}
	defer rows.Close()

	transactionResponse := []model.TransactionResponse{}

	for rows.Next() {

		var dateStr string
		var ibkrTransactionIDStr sql.NullString
		var t model.TransactionResponse

		err := rows.Scan(
			&t.ID,
			&t.PortfolioFundID,
			&t.FundName,
			&dateStr,
			&t.Type,
			&t.Shares,
			&t.CostPerShare,
			&ibkrTransactionIDStr,
			&t.IbkrLinked,
		)
		if err == sql.ErrNoRows {
			return []model.TransactionResponse{}, nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction table results: %w", err)
		}
		t.Date, err = ParseTime(dateStr)
		if err != nil || t.Date.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		// IbkrTransactionId is nullable
		if ibkrTransactionIDStr.Valid {
			t.IbkrTransactionID = ibkrTransactionIDStr.String
		}

		transactionResponse = append(transactionResponse, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction table: %w", err)
	}

	return transactionResponse, nil
}

// GetTransaction retrieves a single transaction by its ID.
// Returns enriched transaction data including fund name and IBKR linkage status.
// Returns an empty TransactionResponse if transactionID is empty or not found.
func (s *TransactionRepository) GetTransaction(transactionID string) (model.TransactionResponse, error) {
	if transactionID == "" {
		return model.TransactionResponse{}, nil
	}

	transactionQuery := `
		SELECT 
			t.id, 
			t.portfolio_fund_id, 
			f.name, 
			t.date, 
			t.type, 
			t.shares, 
			t.cost_per_share, 
			ita.ibkr_transaction_id,
			CASE 
				WHEN ita.ibkr_transaction_id IS NOT NULL THEN 1 
				ELSE 0 
			END AS ibkr_linked
		FROM "transaction" t
		JOIN portfolio_fund pf ON t.portfolio_fund_id = pf.id
		JOIN portfolio p ON pf.portfolio_id = p.id
		JOIN fund f ON pf.fund_id = f.id
		LEFT JOIN ibkr_transaction_allocation ita ON t.id = ita.transaction_id
		WHERE t.id = ?
	`
	var t model.TransactionResponse
	var dateStr string
	var ibkrTransactionIDStr sql.NullString
	err := s.db.QueryRow(transactionQuery, transactionID).Scan(
		&t.ID,
		&t.PortfolioFundID,
		&t.FundName,
		&dateStr,
		&t.Type,
		&t.Shares,
		&t.CostPerShare,
		&ibkrTransactionIDStr,
		&t.IbkrLinked,
	)
	if err == sql.ErrNoRows {
		return model.TransactionResponse{}, nil
	}

	if err != nil {
		return t, fmt.Errorf("failed to scan transaction table results: %w", err)
	}
	t.Date, err = ParseTime(dateStr)
	if err != nil || t.Date.IsZero() {
		return t, fmt.Errorf("failed to parse date: %w", err)
	}

	if ibkrTransactionIDStr.Valid {
		t.IbkrTransactionID = ibkrTransactionIDStr.String
	}

	return t, nil
}
