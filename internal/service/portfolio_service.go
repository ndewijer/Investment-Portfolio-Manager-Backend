package service

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// PortfolioService handles Portfolio-related operations
type PortfolioService struct {
	db *sql.DB
}

// NewPortfolioService creates a new PortfolioService
func NewPortfolioService(db *sql.DB) *PortfolioService {
	return &PortfolioService{
		db: db,
	}
}

// Portfolio represents a portfolio from the database
type Portfolio struct {
	ID                  string
	Name                string
	Description         string
	IsArchived          bool
	ExcludeFromOverview bool
}

// Fund represents a fund from the database
type Fund struct {
	Id             string
	Name           string
	Isin           string
	Symbol         string
	Currency       string
	Exchange       string
	InvestmentType string
	DividendType   string
}

type Transaction struct {
	ID              string
	PortfolioFundID string
	Date            time.Time
	Type            string
	Shares          float64
	CostPerShare    float64
	CreatedAt       time.Time
}

// GetAllPortfolios retrieves all portfolios from the database
func (s *PortfolioService) GetAllPortfolios() ([]Portfolio, error) {
	query := `
          SELECT id, name, description, is_archived, exclude_from_overview
          FROM portfolio
      `
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query portfolios table: %w", err)
	}
	defer rows.Close()

	portfolios := []Portfolio{}

	for rows.Next() {
		var p Portfolio

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

// The struc that will be returned by GetPortfolioSummary
type PortfolioSummary struct {
	ID                      string
	Name                    string
	TotalValue              float64
	TotalCost               float64
	TotalDividends          float64
	TotalUnrealizedGainLoss float64
	TotalRealizedGainLoss   float64
	TotalSaleProceeds       float64
	TotalOriginalCost       float64
	TotalGainLoss           float64
	IsArchived              bool
}

// Holds all funds per portfolio
type PortfolioWithFunds struct {
	Portfolio Portfolio
	Funds     []Fund
}

// Gets portfolio summary from database
func (s *PortfolioService) GetPortfolioSummary() ([]PortfolioSummary, error) {
	// Define Variables

	var portfolios []Portfolio
	var portfolioIDs, pfIDs []string

	portfolioFundToPortfolio := make(map[string]string)

	fundsByPortfolio := make(map[string][]Fund)
	transactionsByPortfolio := make(map[string][]Transaction)

	// Retrieve all portfolios
	portfolioQuery := `
          SELECT id, name, description, is_archived, exclude_from_overview
          FROM portfolio
		  WHERE is_archived = 0
		  AND exclude_from_overview = 0
       `
	rows, err := s.db.Query(portfolioQuery)

	if err != nil {
		return nil, fmt.Errorf("failed to query portfolios table: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p Portfolio

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
		portfolioIDs = append(portfolioIDs, p.ID)

		if err = rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating portfolios table: %w", err)
		}

		if len(portfolios) == 0 {
			//return []PortfolioWithFunds{}, nil
		}
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

	rows, err = s.db.Query(fundQuery, fundArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query portfolio_fund or funds table: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pfID string
		var portfolioID string
		var f Fund

		err := rows.Scan(
			&pfID,
			&portfolioID,
			&f.Id,
			&f.Name,
			&f.Isin,
			&f.Symbol,
			&f.Currency,
			&f.Exchange,
			&f.InvestmentType,
			&f.DividendType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan funds table results: %w", err)
		}

		fundsByPortfolio[portfolioID] = append(fundsByPortfolio[portfolioID], f)
		portfolioFundToPortfolio[pfID] = portfolioID
		pfIDs = append(pfIDs, pfID)

		if err = rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating funds table: %w", err)
		}
	}

	// Combine portfolios with their funds
	results := make([]PortfolioWithFunds, len(portfolios))
	for i, p := range portfolios {
		results[i] = PortfolioWithFunds{
			Portfolio: p,
			Funds:     fundsByPortfolio[p.ID], // Will be nil/empty if no funds
		}
	}

	transactionPlaceholders := make([]string, len(pfIDs))
	for i := range transactionPlaceholders {
		transactionPlaceholders[i] = "?"
	}
	// Retrieve all transactions based on returned portfolio_fund IDs
	transactionQuery := `
		SELECT id, portfolio_fund_id, date, type, shares, cost_per_share, created_at
		FROM 'transaction'
		WHERE portfolio_fund_id IN (` + strings.Join(transactionPlaceholders, ",") + `)
	`

	transactiondArgs := make([]interface{}, len(pfIDs))
	for i, id := range pfIDs {
		transactiondArgs[i] = id
	}

	rows, err = s.db.Query(transactionQuery, transactiondArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction table: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dateStr, createdAtStr string
		var t Transaction

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

		t.Date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			t.Date, err = time.Parse(time.RFC3339, dateStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse date: %w", err)
			}
		}

		t.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err != nil {
			// Try ISO format if the first format fails
			t.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse created_at: %w", err)
			}
		}
		portfolioID := portfolioFundToPortfolio[t.PortfolioFundID]
		transactionsByPortfolio[portfolioID] = append(transactionsByPortfolio[portfolioID], t)

		if err = rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating transaction table: %w", err)
		}
	}

	return []PortfolioSummary{}, nil
}
