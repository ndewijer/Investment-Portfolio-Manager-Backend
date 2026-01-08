package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

type PortfolioRepository struct {
	db *sql.DB
}

func NewPortfolioRepository(db *sql.DB) *PortfolioRepository {
	return &PortfolioRepository{db: db}
}

// GetPortfolios retrieves portfolios from the database based on filter criteria
func (s *PortfolioRepository) GetPortfolios(filter model.PortfolioFilter) ([]model.Portfolio, error) {
	query := `
          SELECT id, name, description, is_archived, exclude_from_overview
          FROM portfolio
          WHERE 1=1
      `
	var args []interface{}

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

func (s *PortfolioRepository) GetPortfolioFundsOnPortfolioID(portfolios []model.Portfolio) (map[string][]model.Fund, map[string]string, map[string]string, []string, []string, error) {
	if len(portfolios) == 0 {
		return nil, nil, nil, nil, nil, nil
	}

	// Set placeholder for the lazy loading
	portfolioPlaceholders := make([]string, len(portfolios))
	for i := range portfolioPlaceholders {
		portfolioPlaceholders[i] = "?"
	}

	// Retrieve all funds based on returned portfolio IDs
	fundQuery := `
		SELECT
		portfolio_fund.id, portfolio_fund.portfolio_id,
		fund.id, fund.name, fund.isin, fund.symbol, fund.currency, fund.exchange, fund.investment_type, fund.dividend_type
		FROM portfolio_fund
		JOIN fund ON fund.id = portfolio_fund.fund_id
		WHERE portfolio_fund.portfolio_id IN (` + strings.Join(portfolioPlaceholders, ",") + `)
	`

	// Extract portfolio IDs
	fundArgs := make([]interface{}, len(portfolios))
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
