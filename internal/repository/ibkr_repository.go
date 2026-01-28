package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

// IbkrRepository provides data access methods for the ibkr table.
// It handles retrieving ibkr information.
type IbkrRepository struct {
	db *sql.DB
}

// NewIbkrRepository creates a new IbkrRepository with the provided database connection.
func NewIbkrRepository(db *sql.DB) *IbkrRepository {
	return &IbkrRepository{db: db}
}

// GetIbkrConfig retrieves the IBKR integration configuration from the database.
// Returns a config with Configured=false if no configuration exists.
// Parses nullable fields (token expiration, last import date, default allocations) safely.
func (r *IbkrRepository) GetIbkrConfig() (*model.IbkrConfig, error) {

	query := `
        SELECT flex_query_id, token_expires_at, last_import_date, auto_import_enabled, created_at, updated_at, enabled, default_allocation_enabled, default_allocations
		FROM ibkr_config 
      `

	var ic model.IbkrConfig
	var tokenExpiresStr, lastImportStr, defaultAllocationStr sql.NullString
	err := r.db.QueryRow(query).Scan(
		&ic.FlexQueryID,
		&tokenExpiresStr,
		&lastImportStr,
		&ic.AutoImportEnabled,
		&ic.CreatedAt,
		&ic.UpdatedAt,
		&ic.Enabled,
		&ic.DefaultAllocationEnabled,
		&defaultAllocationStr,
	)
	if err == sql.ErrNoRows {
		return &model.IbkrConfig{Configured: false}, nil
	}
	if err != nil {
		return nil, err
	}

	// Config exists in database
	ic.Configured = true

	if tokenExpiresStr.Valid {
		t, err := ParseTime(tokenExpiresStr.String)
		if err != nil || t.IsZero() {
			return &model.IbkrConfig{}, fmt.Errorf("failed to parse date on TokenExpiresAt: %w", err)
		}
		ic.TokenExpiresAt = &t
	}

	if lastImportStr.Valid {
		l, err := ParseTime(lastImportStr.String)
		if err != nil || l.IsZero() {
			return &model.IbkrConfig{}, fmt.Errorf("failed to parse date on LastImportDate: %w", err)
		}
		ic.LastImportDate = &l
	}

	if defaultAllocationStr.Valid {
		var defaultAllocation []model.Allocation
		if err := json.Unmarshal([]byte(defaultAllocationStr.String), &defaultAllocation); err != nil {
			log.Printf("warning: failed to parse allocation model: %v", err)
			// Continue without allocations
		} else {
			ic.DefaultAllocations = defaultAllocation
		}
	}

	return &ic, err
}

// GetPendingDividends retrieves dividend records with reinvestment_status = 'PENDING'.
// Optionally filters by fund symbol or ISIN by joining with the fund table.
// Returns an empty slice if no pending dividends match the criteria.
func (r *IbkrRepository) GetPendingDividends(symbol, isin string) ([]model.PendingDividend, error) {

	var query string
	var args []any
	if symbol != "" || isin != "" {
		query = `
			SELECT d.id, d.fund_id, d.portfolio_fund_id, d.record_date, d.ex_dividend_date, d.shares_owned,
				d.dividend_per_share, d.total_amount
			FROM dividend d
			INNER JOIN fund f
			ON f.id = d.fund_id
			WHERE reinvestment_status = 'PENDING'
		`

		if symbol != "" && isin != "" {
			query += " AND (f.symbol = ? OR f.isin = ?)"
			args = append(args, symbol, isin)
		} else if symbol != "" {
			query += " AND f.symbol = ?"
			args = append(args, symbol)
		} else if isin != "" {
			query += " AND f.isin = ?"
			args = append(args, isin)
		}

		query += `
			ORDER BY ex_dividend_date ASC
		`

	} else {
		query = `
			SELECT id, fund_id, portfolio_fund_id, record_date, ex_dividend_date, shares_owned,
			dividend_per_share, total_amount
			FROM dividend
			WHERE reinvestment_status = 'PENDING'
			ORDER BY ex_dividend_date ASC
		`
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query dividend table: %w", err)
	}
	defer rows.Close()

	dividend := []model.PendingDividend{}

	for rows.Next() {
		var recordDateStr, exDividendStr string
		var t model.PendingDividend

		err := rows.Scan(
			&t.ID,
			&t.FundID,
			&t.PortfolioFundID,
			&recordDateStr,
			&exDividendStr,
			&t.SharesOwned,
			&t.DividendPerShare,
			&t.TotalAmount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dividend table results: %w", err)
		}

		t.RecordDate, err = ParseTime(recordDateStr)
		if err != nil || t.RecordDate.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		t.ExDividendDate, err = ParseTime(exDividendStr)
		if err != nil || t.ExDividendDate.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		dividend = append(dividend, t)

	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dividend table: %w", err)
	}

	return dividend, nil

}

// GetInbox retrieves IBKR imported transactions from the ibkr_transaction table.
// Filters by status (defaults to "pending" if not provided) and optionally by transaction_type.
// Returns transactions ordered by transaction_date descending.
// Returns an empty slice if no transactions match the criteria.
func (r *IbkrRepository) GetInbox(status, transactionType string) ([]model.IBKRTransaction, error) {
	var query string
	var args []any

	query = `
	SELECT id, ibkr_transaction_id, transaction_date, symbol, isin, description,
         transaction_type, quantity, price, total_amount, currency, fees,
         status, imported_at
	FROM ibkr_transaction
	WHERE status = ?
  `
	if status == "" {
		args = append(args, "pending")
	} else {
		args = append(args, status)
	}
	if transactionType != "" {
		query += `
			AND transaction_type = ? 
		`
		args = append(args, transactionType)
	}

	query += `
		ORDER BY transaction_date DESC
	`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query IBKR Transactions table: %w", err)
	}
	defer rows.Close()

	ibkrTransactions := []model.IBKRTransaction{}

	for rows.Next() {
		var transactionDateStr, importedAtStr string
		t := model.IBKRTransaction{}
		err := rows.Scan(
			&t.ID,
			&t.IBKRTransactionID,
			&transactionDateStr,
			&t.Symbol,
			&t.ISIN,
			&t.Description,
			&t.TransactionType,
			&t.Quantity,
			&t.Price,
			&t.TotalAmount,
			&t.Currency,
			&t.Fees,
			&t.Status,
			&importedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan IBKR Transactions table results: %w", err)
		}

		t.TransactionDate, err = ParseTime(transactionDateStr)
		if err != nil || t.TransactionDate.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		t.ImportedAt, err = ParseTime(importedAtStr)
		if err != nil || t.ImportedAt.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		ibkrTransactions = append(ibkrTransactions, t)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating IBKR Transactions table: %w", err)
	}

	return ibkrTransactions, nil
}

// GetIbkrInboxCount retrieves the count of IBKR imported transactions.
// Uses a COUNT(*) query for efficiency rather than fetching all records.
// Returns 0 if no transactions exist.
func (r *IbkrRepository) GetIbkrInboxCount() (model.IBKRInboxCount, error) {

	query := `
        SELECT count(*)
		FROM ibkr_transaction
		WHERE status = 'pending'
      `

	count := model.IBKRInboxCount{}
	err := r.db.QueryRow(query).Scan(&count.Count)
	if err == sql.ErrNoRows {
		return model.IBKRInboxCount{
			Count: 0,
		}, nil
	}
	if err != nil {
		return model.IBKRInboxCount{}, err
	}

	return count, err
}
