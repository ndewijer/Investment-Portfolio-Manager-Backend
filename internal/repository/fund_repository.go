package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/errors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

// FundRepository provides data access methods for fund and fund_price tables.
// It handles retrieving fund metadata and historical price data.
type FundRepository struct {
	db *sql.DB
	tx *sql.Tx
}

// NewFundRepository creates a new FundRepository with the provided database connection.
func NewFundRepository(db *sql.DB) *FundRepository {
	return &FundRepository{db: db}
}

func (r *FundRepository) WithTx(tx *sql.Tx) *FundRepository {
	return &FundRepository{
		db: r.db,
		tx: tx,
	}
}

func (r *FundRepository) getQuerier() interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// GetFund retrieves all funds from the database.
// Returns an empty slice if no funds are found.
func (r *FundRepository) GetFund(fundID string) ([]model.Fund, error) {
	query := `
        SELECT f.id, f.name, f.isin, f.symbol, f.currency, f.exchange, f.investment_type, f.dividend_type, fp.price
		FROM fund f
		INNER JOIN fund_price fp ON f.id = fp.fund_id
		INNER JOIN (
			SELECT fund_id, MAX(date) as latest_date
			FROM fund_price
			GROUP BY fund_id
		) latest ON fp.fund_id = latest.fund_id AND fp.date = latest.latest_date
      `

	var args []any

	if fundID != "" {
		query += ` WHERE f.id = ?`
		args = append(args, fundID)
	}

	rows, err := r.db.Query(query, args...)

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
			&f.LatestPrice,
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

// GetFunds retrieves fund records for the given fund IDs.
// Returns a slice of Fund objects containing metadata like name, ISIN, symbol, currency, etc.
func (r *FundRepository) GetFunds(fundIDs []string) ([]model.Fund, error) {
	fundPlaceholders := make([]string, len(fundIDs))
	for i := range fundPlaceholders {
		fundPlaceholders[i] = "?"
	}

	//#nosec G202 -- Safe: placeholders are generated programmatically, not from user input
	fundQuery := `
      SELECT id, name, isin, symbol, currency, exchange, investment_type, dividend_type
      FROM fund
      WHERE id IN (` + strings.Join(fundPlaceholders, ",") + `)
  `

	fundArgs := make([]any, len(fundIDs))
	for i, id := range fundIDs {
		fundArgs[i] = id
	}

	rows, err := r.db.Query(fundQuery, fundArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query fund table: %w", err)
	}
	defer rows.Close()

	fundsByPortfolio := []model.Fund{}

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
//   - ascending: if true, sort by date ASC (oldest first); if false, DESC (newest first)
//
// The sortOrder parameter controls how prices are sorted by date within each fund group:
//   - "ASC": oldest first - efficient for date-aware lookups (GetPriceForDate)
//   - "DESC": newest first - efficient for latest-price lookups
//
// Returns a map of fundID -> []FundPrice, grouped by fund and sorted by date according to sortOrder.
func (r *FundRepository) GetFundPrice(fundIDs []string, startDate, endDate time.Time, ascending bool) (map[string][]model.FundPrice, error) {

	if startDate.After(endDate) {
		return nil, fmt.Errorf("startDate (%s) must be before or equal to endDate (%s)",
			startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	}

	fundPricePlaceholders := make([]string, len(fundIDs))
	for i := range fundPricePlaceholders {
		fundPricePlaceholders[i] = "?"
	}

	var sortOrder string
	if ascending {
		sortOrder = "ASC"
	} else {
		sortOrder = "DESC"
	}

	//#nosec G202 -- Safe: placeholders are generated programmatically, not from user input
	fundPriceQuery := `
    SELECT id, fund_id, date, price
    FROM fund_price
    WHERE fund_id IN (` + strings.Join(fundPricePlaceholders, ",") + `)
    AND date >= ?
    AND date <= ?
    ORDER BY fund_id ASC, date ` + sortOrder + `
`

	fundPriceArgs := make([]any, 0, len(fundIDs)+2)
	for _, id := range fundIDs {
		fundPriceArgs = append(fundPriceArgs, id)
	}
	fundPriceArgs = append(fundPriceArgs, startDate.Format("2006-01-02"))
	fundPriceArgs = append(fundPriceArgs, endDate.Format("2006-01-02"))

	rows, err := r.db.Query(fundPriceQuery, fundPriceArgs...)
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

// GetPortfolioFunds retrieves all funds associated with a portfolio.
// If PortfolioID is empty, returns funds across all portfolios.
// Returns basic fund metadata from the portfolio_fund and fund tables.
func (r *FundRepository) GetPortfolioFunds(PortfolioID string) ([]model.PortfolioFundResponse, error) {

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

	rows, err := r.db.Query(fundQuery, fundArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve portfolio funds via portfolio_fund JOIN (portfolio_id=%s): %w", PortfolioID, err)
	}
	defer rows.Close()

	portfolioFunds := []model.PortfolioFundResponse{}

	for rows.Next() {
		var f model.PortfolioFundResponse

		err := rows.Scan(
			&f.ID,
			&f.FundID,
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

// GetAllPortfolioFundListings retrieves all portfolio-fund relationships with metadata.
// Returns a listing of all funds across all portfolios with basic information.
// Used for the GET /api/portfolio/funds endpoint.
func (r *FundRepository) GetAllPortfolioFundListings() ([]model.PortfolioFundListing, error) {
	query := `
		SELECT
			pf.id,
			pf.portfolio_id,
			f.id as fund_id,
			p.name as portfolio_name,
			f.name as fund_name,
			f.dividend_type
		FROM portfolio_fund pf
		JOIN portfolio p ON pf.portfolio_id = p.id
		JOIN fund f ON pf.fund_id = f.id
		WHERE p.is_archived = 0
		ORDER BY p.name ASC, f.name ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query portfolio_fund listings: %w", err)
	}
	defer rows.Close()

	listings := []model.PortfolioFundListing{}

	for rows.Next() {
		var l model.PortfolioFundListing
		err := rows.Scan(
			&l.ID,
			&l.PortfolioID,
			&l.FundID,
			&l.PortfolioName,
			&l.FundName,
			&l.DividendType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan portfolio_fund listing: %w", err)
		}
		listings = append(listings, l)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating portfolio_fund listings: %w", err)
	}

	return listings, nil
}

// GetSymbol retrieves symbol information by ticker symbol from the symbol_info table.
// Returns nil, nil if the symbol is not found.
// Returns nil, error if a database error occurs.
func (r *FundRepository) GetSymbol(symbol string) (*model.Symbol, error) {
	if symbol == "" {
		return nil, nil
	}

	query := `
        SELECT s.id, s.symbol, s.name, s.exchange, s.currency, s.isin, s.last_updated, s.data_source, s.is_valid
		FROM symbol_info s
		WHERE s.symbol = ?
      `

	var sb model.Symbol
	var exchangeStr, currencyStr, isinStr, dataSource, lastUpdatedStr sql.NullString
	err := r.db.QueryRow(query, symbol).Scan(
		&sb.ID,
		&sb.Symbol,
		&sb.Name,
		&exchangeStr,
		&currencyStr,
		&isinStr,
		&lastUpdatedStr,
		&dataSource,
		&sb.IsValid,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if lastUpdatedStr.Valid {
		sb.LastUpdated, err = ParseTime(lastUpdatedStr.String)
		if err != nil || sb.LastUpdated.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}
	}

	if exchangeStr.Valid {
		sb.Exchange = exchangeStr.String
	}
	if currencyStr.Valid {
		sb.Currency = currencyStr.String
	}
	if isinStr.Valid {
		sb.Isin = isinStr.String
	}
	if dataSource.Valid {
		sb.DataSource = dataSource.String
	}

	return &sb, err
}

// GetFundBySymbolOrIsin retrieves a fund by matching either its symbol or ISIN.
// At least one of symbol or isin must be provided.
//
// Symbol matching uses a special comparison that strips the exchange suffix:
//   - Database symbol "AAPL.NASDAQ" will match input symbol "AAPL"
//   - This is done using: substr(symbol, 1, instr(symbol || '.', '.') - 1)
//   - Allows matching IBKR symbols (without exchange) to database symbols (with exchange)
//
// Parameters:
//   - symbol: The fund symbol to match (exchange suffix will be stripped from database values)
//   - isin: The fund ISIN to match (exact match)
//
// If both are provided, the query matches funds where EITHER the symbol OR isin matches.
// Returns ErrFundNotFound if no matching fund is found.
func (r *FundRepository) GetFundBySymbolOrIsin(symbol, isin string) (model.Fund, error) {

	var query string
	var args []any
	if symbol == "" && isin == "" {
		return model.Fund{}, fmt.Errorf("symbol or isin required")
	}

	query = `
		SELECT f.id, f.name, f.isin, f.symbol, f.currency, f.exchange, f.investment_type, f.dividend_type
		FROM fund f
		WHERE 1=1
		`

	if symbol != "" && isin != "" {
		query += " AND (substr(symbol, 1, instr(symbol || '.', '.') - 1) = ? OR f.isin = ?)"
		args = append(args, symbol, isin)
	} else if symbol != "" {
		query += " AND substr(symbol, 1, instr(symbol || '.', '.') - 1) = ?"
		args = append(args, symbol)
	} else if isin != "" {
		query += " AND f.isin = ?"
		args = append(args, isin)
	}

	var f model.Fund

	err := r.db.QueryRow(query, args...).Scan(
		&f.ID,
		&f.Name,
		&f.Isin,
		&f.Symbol,
		&f.Currency,
		&f.Exchange,
		&f.InvestmentType,
		&f.DividendType,
	)
	if err == sql.ErrNoRows {
		return model.Fund{}, errors.ErrFundNotFound
	}
	if err != nil {
		return model.Fund{}, err
	}

	return f, nil

}

func (r *FundRepository) GetPortfolioFund(pfID string) (model.PortfolioFund, error) {
	if pfID == "" {
		return model.PortfolioFund{}, fmt.Errorf("portfolio_fund ID required")
	}

	query := `
		SELECT id, portfolio_id, fund_id
		FROM portfolio_fund
		WHERE id = ?
	`

	var pf model.PortfolioFund

	err := r.db.QueryRow(query, pfID).Scan(
		&pf.ID,
		&pf.PortfolioID,
		&pf.FundID,
	)
	if err == sql.ErrNoRows {
		return model.PortfolioFund{}, errors.ErrPortfolioFundNotFound
	}
	if err != nil {
		return model.PortfolioFund{}, err
	}

	return pf, nil
}

func (r *FundRepository) InsertPortfolioFund(ctx context.Context, p, f string) error {
	query := `
        INSERT INTO portfolio_fund (id, portfolio_id, fund_id)
        VALUES (?, ?, ?)
    `

	_, err := r.getQuerier().ExecContext(ctx, query,
		uuid.New().String(),
		p,
		f,
	)

	if err != nil {
		return fmt.Errorf("failed to insert portfolio_fund: %w", err)
	}

	return nil
}

func (r *FundRepository) DeletePortfolioFund(ctx context.Context, portfolioFundID string) error {
	query := `DELETE FROM portfolio_fund WHERE id = ?`

	result, err := r.getQuerier().ExecContext(ctx, query, portfolioFundID)
	if err != nil {
		return fmt.Errorf("failed to delete portfolio_fund: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.ErrPortfolioFundNotFound
	}

	return nil
}
