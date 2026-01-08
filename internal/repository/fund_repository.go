package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

type FundRepository struct {
	db *sql.DB
}

func NewFundRepository(db *sql.DB) *FundRepository {
	return &FundRepository{db: db}
}

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

	fundArgs := make([]interface{}, len(fundIDs))
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

func (s *FundRepository) GetFundPrice(fundIDs []string) (map[string][]model.FundPrice, error) {

	fundPricePlaceholders := make([]string, len(fundIDs))
	for i := range fundPricePlaceholders {
		fundPricePlaceholders[i] = "?"
	}

	// Retrieve all funds based on returned portfolio_fund IDs
	fundPriceQuery := `
		SELECT id, fund_id, date, price
		FROM fund_price
		WHERE fund_id IN (` + strings.Join(fundPricePlaceholders, ",") + `)
		ORDER BY fund_id ASC,date DESC
	`

	fundPriceArgs := make([]interface{}, len(fundIDs))
	for i, id := range fundIDs {
		fundPriceArgs[i] = id
	}

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
