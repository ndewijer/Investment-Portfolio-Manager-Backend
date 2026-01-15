package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

// FundRepository provides data access methods for fund and fund_price tables.
// It handles retrieving fund metadata and historical price data.
type FundRepository struct {
	db *sql.DB
}

// NewFundRepository creates a new FundRepository with the provided database connection.
func NewFundRepository(db *sql.DB) *FundRepository {
	return &FundRepository{db: db}
}

// GetFunds retrieves all funds from the database.
// Returns an empty slice if no funds are found.
func (s *FundRepository) GetFunds() ([]model.Fund, error) {
	query := `
          SELECT id, name, isin, symbol, currency, exchange, investment_type, dividend_type
      	  FROM fund
      `

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query fund table: %w", err)
	}
	defer rows.Close()

	funds := []model.Fund{}

	for rows.Next() {
		var f model.Fund

		err := rows.Scan(

			&f.ID,
			&f.Name,
			&f.Isin,
			&f.Symbol,
			&f.Currency,
			&f.Exchange,
			&f.InvestmentType,
			&f.DividendType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fund table results: %w", err)
		}
		funds = append(funds, f)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fund table: %w", err)
	}

	return funds, nil
}

// GetFund retrieves fund records for the given fund IDs.
// Returns a slice of Fund objects containing metadata like name, ISIN, symbol, currency, etc.
func (s *FundRepository) GetFund(fundIDs []string) ([]model.Fund, error) {
	fundPlaceholders := make([]string, len(fundIDs))
	for i := range fundPlaceholders {
		fundPlaceholders[i] = "?"
	}

	// Retrieve all funds based on returned portfolio_fund IDs
	fundQuery := `
      SELECT id, name, isin, symbol, currency, exchange, investment_type, dividend_type
      FROM fund
      WHERE id IN (` + strings.Join(fundPlaceholders, ",") + `)
  `

	fundArgs := make([]any, len(fundIDs))
	for i, id := range fundIDs {
		fundArgs[i] = id
	}

	rows, err := s.db.Query(fundQuery, fundArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query fund table: %w", err)
	}
	defer rows.Close()

	var fundsByPortfolio []model.Fund

	for rows.Next() {
		var f model.Fund

		err := rows.Scan(

			&f.ID,
			&f.Name,
			&f.Isin,
			&f.Symbol,
			&f.Currency,
			&f.Exchange,
			&f.InvestmentType,
			&f.DividendType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fund table results: %w", err)
		}
		fundsByPortfolio = append(fundsByPortfolio, f)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fund table: %w", err)
	}

	return fundsByPortfolio, nil
}

// GetFundPrice retrieves historical price data for the given fund IDs within the specified date range.
//
// Parameters:
//   - fundIDs: slice of fund IDs to query
//   - startDate: inclusive start date for the query
//   - endDate: inclusive end date for the query
//   - sortOrder: "ASC" or "DESC" for date ordering (defaults to "DESC" if invalid)
//
// The sortOrder parameter controls how prices are sorted by date within each fund group:
//   - "ASC": oldest first - efficient for date-aware lookups (GetPriceForDate)
//   - "DESC": newest first - efficient for latest-price lookups
//
// Returns a map of fundID -> []FundPrice, grouped by fund and sorted by date according to sortOrder.
func (s *FundRepository) GetFundPrice(fundIDs []string, startDate, endDate time.Time, sortOrder string) (map[string][]model.FundPrice, error) {

	fundPricePlaceholders := make([]string, len(fundIDs))
	for i := range fundPricePlaceholders {
		fundPricePlaceholders[i] = "?"
	}

	// Validate and sanitize sortOrder (can't be parameterized)
	if strings.ToUpper(sortOrder) != "ASC" && strings.ToUpper(sortOrder) != "DESC" {
		sortOrder = "DESC"
	}

	// Build query with sortOrder directly in the string
	fundPriceQuery := `
    SELECT id, fund_id, date, price
    FROM fund_price
    WHERE fund_id IN (` + strings.Join(fundPricePlaceholders, ",") + `)
    AND date >= ?
    AND date <= ?
    ORDER BY fund_id ASC, date ` + sortOrder + `
`

	// Build args: fundIDs first, then startDate, then endDate
	fundPriceArgs := make([]any, 0, len(fundIDs)+2)
	for _, id := range fundIDs {
		fundPriceArgs = append(fundPriceArgs, id)
	}
	fundPriceArgs = append(fundPriceArgs, startDate.Format("2006-01-02"))
	fundPriceArgs = append(fundPriceArgs, endDate.Format("2006-01-02"))

	rows, err := s.db.Query(fundPriceQuery, fundPriceArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query fund_price table: %w", err)
	}
	defer rows.Close()

	fundPriceByFund := make(map[string][]model.FundPrice)

	for rows.Next() {
		var dateStr string
		var fp model.FundPrice

		err := rows.Scan(

			&fp.ID,
			&fp.FundID,
			&dateStr,
			&fp.Price,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fund table results: %w", err)
		}

		fp.Date, err = ParseTime(dateStr)
		if err != nil || fp.Date.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		fundPriceByFund[fp.FundID] = append(fundPriceByFund[fp.FundID], fp)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fund table: %w", err)
	}

	return fundPriceByFund, nil
}

// GetFund retrieves fund records for the given fund IDs.
// Returns a slice of Fund objects containing metadata like name, ISIN, symbol, currency, etc.
func (s *FundRepository) GetPortfolioFunds(PortfolioID string) ([]model.PortfolioFund, error) {

	// Retrieve all funds based on returned portfolio_fund IDs
	fundQuery := `
		SELECT
		portfolio_fund.id,
		fund.id, fund.name, fund.investment_type, fund.dividend_type
		FROM portfolio_fund
		JOIN fund ON fund.id = portfolio_fund.fund_id
		WHERE 1=1
	`

	var fundArgs []any

	if PortfolioID != "" {
		fundQuery += " AND portfolio_fund.portfolio_id = ?"
		fundArgs = append(fundArgs, PortfolioID)
	}

	rows, err := s.db.Query(fundQuery, fundArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve portfolio funds via portfolio_fund JOIN (portfolio_id=%s): %w", PortfolioID, err)
	}
	defer rows.Close()

	var portfolioFunds []model.PortfolioFund

	for rows.Next() {
		var f model.PortfolioFund

		err := rows.Scan(
			&f.ID,
			&f.FundId,
			&f.FundName,
			&f.InvestmentType,
			&f.DividendType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fund or portfolio_fund table results: %w", err)
		}

		portfolioFunds = append(portfolioFunds, f)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating portfolio_fund JOIN results: %w", err)
	}

	return portfolioFunds, nil
}
