package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

// DividendRepository provides data access methods for the dividend table.
// It handles retrieving dividend records and reinvestment information.
type DividendRepository struct {
	db *sql.DB
}

// NewDividendRepository creates a new DividendRepository with the provided database connection.
func NewDividendRepository(db *sql.DB) *DividendRepository {
	return &DividendRepository{db: db}
}

// GetDividend retrieves all dividends for the given portfolio_fund IDs within the specified date range.
// Dividends are filtered by ex-dividend date and sorted in ascending order by that date.
//
// Parameters:
//   - pfIDs: slice of portfolio_fund IDs to query
//   - startDate: inclusive start date for the query (compared against ex_dividend_date)
//   - endDate: inclusive end date for the query (compared against ex_dividend_date)
//
// Returns a map of portfolioFundID -> []Dividend. If pfIDs is empty, returns an empty map.
// Handles nullable fields like buy_order_date and reinvestment_transaction_id appropriately.
// This grouping allows callers to decide how to aggregate (by portfolio, by fund, etc.) after retrieval.
func (s *DividendRepository) GetDividend(pfIDs []string, startDate, endDate time.Time) (map[string][]model.Dividend, error) {
	if len(pfIDs) == 0 {
		return make(map[string][]model.Dividend), nil
	}

	dividendPlaceholders := make([]string, len(pfIDs))
	for i := range dividendPlaceholders {
		dividendPlaceholders[i] = "?"
	}

	// Retrieve all dividend based on returned portfolio_fund IDs
	dividendQuery := `
		SELECT id, fund_id, portfolio_fund_id, record_date, ex_dividend_date, shares_owned,
		dividend_per_share, total_amount, reinvestment_status, buy_order_date, reinvestment_transaction_id, created_at
		FROM dividend
		WHERE portfolio_fund_id IN (` + strings.Join(dividendPlaceholders, ",") + `)
		AND ex_dividend_date >= ?
		AND ex_dividend_date <= ?
		ORDER BY ex_dividend_date ASC
	`

	// Build args: pfIDs first, then startDate, then endDate
	dividendArgs := make([]any, 0, len(pfIDs)+2)
	for _, id := range pfIDs {
		dividendArgs = append(dividendArgs, id)
	}
	dividendArgs = append(dividendArgs, startDate.Format("2006-01-02"))
	dividendArgs = append(dividendArgs, endDate.Format("2006-01-02"))

	rows, err := s.db.Query(dividendQuery, dividendArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query dividend table: %w", err)
	}
	defer rows.Close()

	dividendByPortfolioFund := make(map[string][]model.Dividend)

	for rows.Next() {
		var recordDateStr, exDividendStr, createdAtStr string
		var buyOrderStr, reinvestmentTxID sql.NullString
		var t model.Dividend

		err := rows.Scan(
			&t.ID,
			&t.FundID,
			&t.PortfolioFundID,
			&recordDateStr,
			&exDividendStr,
			&t.SharesOwned,
			&t.DividendPerShare,
			&t.TotalAmount,
			&t.ReinvestmentStatus,
			&buyOrderStr,
			&reinvestmentTxID,
			&createdAtStr,
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

		// BuyOrderDate is nullable
		if buyOrderStr.Valid {
			t.BuyOrderDate, err = ParseTime(buyOrderStr.String)
			if err != nil || t.BuyOrderDate.IsZero() {
				return nil, fmt.Errorf("failed to parse buy_order_date: %w", err)
			}
		}

		// ReinvestmentTransactionId is nullable
		if reinvestmentTxID.Valid {
			t.ReinvestmentTransactionId = reinvestmentTxID.String
		}

		t.CreatedAt, err = ParseTime(createdAtStr)
		if err != nil || t.CreatedAt.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		dividendByPortfolioFund[t.PortfolioFundID] = append(dividendByPortfolioFund[t.PortfolioFundID], t)

	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dividend table: %w", err)
	}

	return dividendByPortfolioFund, nil
}

// GetDividendPerPortfolioFund retrieves all dividend records for funds in a specific portfolio.
// This method performs a JOIN across dividend, portfolio_fund, and fund tables to return
// enriched dividend data including fund names and dividend types.
//
// Parameters:
//   - portfolioID: The portfolio ID to retrieve dividends for. Returns empty slice if empty string.
//
// Returns a slice of DividendFund containing all historical dividend payments for the portfolio.
// The results include both reinvested and distributed dividends across all funds held in the portfolio.
func (s *DividendRepository) GetDividendPerPortfolioFund(portfolioID string) ([]model.DividendFund, error) {
	if portfolioID == "" {
		return []model.DividendFund{}, nil
	}

	query := `
	SELECT
		d.id, d.fund_id, f.name, d.portfolio_fund_id, d.record_date, d.ex_dividend_date,
		d.shares_owned, d.dividend_per_share, d.total_amount, d.reinvestment_status,
		d.buy_order_date, d.reinvestment_transaction_id, f.dividend_type
	FROM dividend d
	INNER JOIN portfolio_fund pf ON d.portfolio_fund_id = pf.id
	INNER JOIN fund f ON pf.fund_id = f.id
	WHERE pf.portfolio_id = ?
	`

	rows, err := s.db.Query(query, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to query dividend table: %w", err)
	}
	defer rows.Close()

	var dividendFund []model.DividendFund

	for rows.Next() {
		var recordDateStr, exDividendStr string
		var buyOrderStr, reinvestmentTxID sql.NullString
		var t model.DividendFund

		err := rows.Scan(
			&t.ID,
			&t.FundID,
			&t.FundName,
			&t.PortfolioFundID,
			&recordDateStr,
			&exDividendStr,
			&t.SharesOwned,
			&t.DividendPerShare,
			&t.TotalAmount,
			&t.ReinvestmentStatus,
			&buyOrderStr,
			&reinvestmentTxID,
			&t.DividendType,
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

		// BuyOrderDate is nullable
		if buyOrderStr.Valid {
			buyDate, err := ParseTime(buyOrderStr.String)
			if err != nil || buyDate.IsZero() {
				return nil, fmt.Errorf("failed to parse buy_order_date: %w", err)
			}
			t.BuyOrderDate = &buyDate // Assign pointer for nullable field
		}

		// ReinvestmentTransactionId is nullable
		if reinvestmentTxID.Valid {
			t.ReinvestmentTransactionId = reinvestmentTxID.String
		}

		dividendFund = append(dividendFund, t)

	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dividend table: %w", err)
	}

	return dividendFund, nil
}
