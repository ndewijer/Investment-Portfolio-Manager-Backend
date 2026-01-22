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

func (s *TransactionRepository) GetTransactionsPerPortfolio(portfolioId string) ([]model.TransactionResponse, error) {

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

	if portfolioId == "" {
		transactionQuery += `
		ORDER BY t.date ASC
		`
	} else {
		transactionQuery += `
		WHERE pf.portfolio_id = ?
		ORDER BY t.date ASC
		`
		args = append(args, portfolioId)
	}

	rows, err := s.db.Query(transactionQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction table: %w", err)
	}
	defer rows.Close()

	transactionResponse := []model.TransactionResponse{}

	for rows.Next() {

		var dateStr string
		var ibkrTransactionIdStr sql.NullString
		var t model.TransactionResponse

		err := rows.Scan(
			&t.Id,
			&t.PortfolioFundId,
			&t.FundName,
			&dateStr,
			&t.Type,
			&t.Shares,
			&t.CostPerShare,
			&ibkrTransactionIdStr,
			&t.IbkrLinked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction table results: %w", err)
		}
		t.Date, err = ParseTime(dateStr)
		if err != nil || t.Date.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		// IbkrTransactionId is nullable
		if ibkrTransactionIdStr.Valid {
			t.IbkrTransactionId = ibkrTransactionIdStr.String
		}

		transactionResponse = append(transactionResponse, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction table: %w", err)
	}

	return transactionResponse, nil
}

func (s *TransactionRepository) GetTransaction(transactionId string) (model.TransactionResponse, error) {
	if transactionId == "" {
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
	var ibkrTransactionIdStr sql.NullString
	err := s.db.QueryRow(transactionQuery, transactionId).Scan(
		&t.Id,
		&t.PortfolioFundId,
		&t.FundName,
		&dateStr,
		&t.Type,
		&t.Shares,
		&t.CostPerShare,
		&ibkrTransactionIdStr,
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

	if ibkrTransactionIdStr.Valid {
		t.IbkrTransactionId = ibkrTransactionIdStr.String
	}

	return t, nil
}
