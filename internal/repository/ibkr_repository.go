package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

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
		&ic.FlexQueryId,
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
		ic.TokenExpiresAt, err = ParseTime(tokenExpiresStr.String)
		if err != nil || ic.TokenExpiresAt.IsZero() {
			return &model.IbkrConfig{}, fmt.Errorf("failed to parse date on TokenExpiresAt: %w", err)
		}
	}

	if lastImportStr.Valid {
		ic.LastImportDate, err = ParseTime(lastImportStr.String)
		if err != nil || ic.LastImportDate.IsZero() {
			return &model.IbkrConfig{}, fmt.Errorf("failed to parse date on LastImportDate: %w", err)
		}
	}

	if defaultAllocationStr.Valid {
		var defaultAllocation []model.Allocation
		if err := json.Unmarshal([]byte(defaultAllocationStr.String), &defaultAllocation); err != nil {
			//log.Printf("warning: failed to parse allocation model: %v", err)
			// Continue without allocations
		} else {
			ic.DefaultAllocations = defaultAllocation
		}
	}

	return &ic, err
}

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
		if err == sql.ErrNoRows {
			return []model.PendingDividend{}, nil
		}
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
