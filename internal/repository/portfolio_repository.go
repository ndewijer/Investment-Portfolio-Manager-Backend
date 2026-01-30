package repository

import (
	"database/sql"
	"fmt"
	"strings"

	apperrors "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/errors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

// PortfolioRepository provides data access methods for portfolio and portfolio_fund tables.
// It handles retrieving portfolio metadata and their associated fund relationships.
type PortfolioRepository struct {
	db *sql.DB
}

// NewPortfolioRepository creates a new PortfolioRepository with the provided database connection.
func NewPortfolioRepository(db *sql.DB) *PortfolioRepository {
	return &PortfolioRepository{db: db}
}

// GetPortfolios retrieves portfolios from the database based on filter criteria.
// The filter allows control over whether archived and overview-excluded portfolios are included.
// Returns an empty slice if no portfolios match the filter criteria.
func (s *PortfolioRepository) GetPortfolios(filter model.PortfolioFilter) ([]model.Portfolio, error) {
	query := `
          SELECT id, name, description, is_archived, exclude_from_overview
          FROM portfolio
          WHERE 1=1
      `
	var args []any

	if !filter.IncludeArchived {
		query += " AND is_archived = ?"
		args = append(args, 0)
	}

	if !filter.IncludeExcluded {
		query += " AND exclude_from_overview = ?"
		args = append(args, 0)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query portfolios table: %w", err)
	}
	defer rows.Close()

	portfolios := []model.Portfolio{}

	for rows.Next() {
		var p model.Portfolio

		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.IsArchived,
			&p.ExcludeFromOverview,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan portfolio table results: %w", err)
		}

		portfolios = append(portfolios, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating portfolios table: %w", err)
	}

	return portfolios, nil
}

func (s *PortfolioRepository) GetPortfolioOnID(portfolioID string) (model.Portfolio, error) {
	query := `
          SELECT id, name, description, is_archived, exclude_from_overview
          FROM portfolio
          WHERE id = ?
      `
	var p model.Portfolio

	err := s.db.QueryRow(query, portfolioID).Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.IsArchived,
		&p.ExcludeFromOverview,
	)
	if err == sql.ErrNoRows {
		return model.Portfolio{}, apperrors.ErrPortfolioNotFound
	}
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("failed to query portfolio: %w", err)
	}

	return p, nil
}

// GetPortfolioFundsOnPortfolioID retrieves all funds associated with the given portfolios.
// It performs a JOIN between portfolio_fund and fund tables to get complete fund information.
//
// Returns:
//   - fundsByPortfolio: map[portfolioID][]Fund - funds grouped by portfolio
//   - portfolioFundToPortfolio: map[portfolioFundID]portfolioID - lookup table
//   - portfolioFundToFund: map[portfolioFundID]fundID - lookup table
//   - pfIDs: slice of all portfolio_fund IDs
//   - fundIDs: slice of all unique fund IDs (may contain duplicates)
//   - error: any error encountered during the query
//
// If the input portfolios slice is empty, returns all nil values.
func (s *PortfolioRepository) GetPortfolioFundsOnPortfolioID(portfolios []model.Portfolio) (map[string][]model.Fund, map[string]string, map[string]string, []string, []string, error) {
	if len(portfolios) == 0 {
		return nil, nil, nil, nil, nil, nil
	}

	portfolioPlaceholders := make([]string, len(portfolios))
	for i := range portfolioPlaceholders {
		portfolioPlaceholders[i] = "?"
	}

	//#nosec G202 -- Safe: placeholders are generated programmatically, not from user input
	fundQuery := `
		SELECT
		portfolio_fund.id, portfolio_fund.portfolio_id,
		fund.id, fund.name, fund.isin, fund.symbol, fund.currency, fund.exchange, fund.investment_type, fund.dividend_type
		FROM portfolio_fund
		JOIN fund ON fund.id = portfolio_fund.fund_id
		WHERE portfolio_fund.portfolio_id IN (` + strings.Join(portfolioPlaceholders, ",") + `)
	`

	fundArgs := make([]any, len(portfolios))
	for i, p := range portfolios {
		fundArgs[i] = p.ID
	}

	rows, err := s.db.Query(fundQuery, fundArgs...)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to query portfolio_fund or funds table: %w", err)
	}
	defer rows.Close()

	fundsByPortfolio := make(map[string][]model.Fund)
	portfolioFundToPortfolio := make(map[string]string)
	portfolioFundToFund := make(map[string]string)
	var fundIDs, pfIDs []string

	for rows.Next() {
		var pfID string
		var portfolioID string
		var f model.Fund

		err := rows.Scan(
			&pfID,
			&portfolioID,
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
			return nil, nil, nil, nil, nil, fmt.Errorf("failed to scan funds table results: %w", err)
		}

		fundsByPortfolio[portfolioID] = append(fundsByPortfolio[portfolioID], f)
		portfolioFundToPortfolio[pfID] = portfolioID
		portfolioFundToFund[pfID] = f.ID
		pfIDs = append(pfIDs, pfID)
		fundIDs = append(fundIDs, f.ID)

	}
	if err = rows.Err(); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("error iterating funds table: %w", err)
	}

	return fundsByPortfolio, portfolioFundToPortfolio, portfolioFundToFund, pfIDs, fundIDs, nil
}

// GetPortfoliosByFundID retrieves all portfolios that hold a specific fund.
// Joins the portfolio and portfolio_fund tables to find portfolios where the fund is assigned.
// Returns an empty slice if the fund is not assigned to any portfolios (not an error).
//
// Parameters:
//   - fundID: The UUID of the fund
//
// Returns a slice of portfolios that hold this fund, or an error if the database query fails.
func (s *PortfolioRepository) GetPortfoliosByFundID(fundID string) ([]model.Portfolio, error) {

	fundQuery := `
		SELECT p.id, p.name, p.description, p.is_archived, p.exclude_from_overview
        FROM portfolio p
		INNER JOIN portfolio_fund pf
		ON pf.portfolio_id = p.id
		WHERE pf.fund_id = ?
	`

	rows, err := s.db.Query(fundQuery, fundID)
	if err != nil {
		return nil, fmt.Errorf("failed to query portfolio_fund or portfolio table: %w", err)
	}
	defer rows.Close()

	portfolios := []model.Portfolio{}

	for rows.Next() {
		var p model.Portfolio

		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.IsArchived,
			&p.ExcludeFromOverview,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan portfolio_fund or portfolio table results: %w", err)
		}

		portfolios = append(portfolios, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating portfolio_fund or portfolio table: %w", err)
	}

	return portfolios, nil
}
