package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

type RealizedGainLossRepository struct {
	db *sql.DB
}

func NewRealizedGainLossRepository(db *sql.DB) *RealizedGainLossRepository {
	return &RealizedGainLossRepository{db: db}
}

func (s *RealizedGainLossRepository) GetRealizedGainLossByPortfolio(portfolio []model.Portfolio, startDate, endDate time.Time) (map[string][]model.RealizedGainLoss, error) {
	if len(portfolio) == 0 {
		return make(map[string][]model.RealizedGainLoss), nil
	}

	realizedGainLossPlaceholders := make([]string, len(portfolio))
	for i := range realizedGainLossPlaceholders {
		realizedGainLossPlaceholders[i] = "?"
	}

	// Retrieve all realizedGainLosss based on returned portfolio_fund IDs
	realizedGainLossQuery := `
		SELECT id, portfolio_id, fund_id, transaction_id, transaction_date, shares_sold, cost_basis,
		sale_proceeds, realized_gain_loss, created_at
		FROM realized_gain_loss
		WHERE portfolio_id IN (` + strings.Join(realizedGainLossPlaceholders, ",") + `)
		AND transaction_date >= '` + startDate.Format("2006-01-02") + `' and transaction_date <= '` + endDate.Format("2006-01-02") + `'
		ORDER BY created_at ASC
	`

	realizedGainLossdArgs := make([]any, len(portfolio))
	for i, p := range portfolio {
		realizedGainLossdArgs[i] = p.ID
	}

	rows, err := s.db.Query(realizedGainLossQuery, realizedGainLossdArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query realizedGainLoss table: %w", err)
	}
	defer rows.Close()

	realizedGainLosssByPortfolio := make(map[string][]model.RealizedGainLoss)

	for rows.Next() {
		var transactionDateStr, createdAtStr string
		var r model.RealizedGainLoss

		err := rows.Scan(
			&r.ID,
			&r.PortfolioID,
			&r.FundID,
			&r.TransactionID,
			&transactionDateStr,
			&r.SharesSold,
			&r.CostBasis,
			&r.SaleProceeds,
			&r.RealizedGainLoss,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan realizedGainLoss table results: %w", err)
		}

		r.TransactionDate, err = ParseTime(transactionDateStr)
		if err != nil || r.TransactionDate.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		r.CreatedAt, err = ParseTime(createdAtStr)
		if err != nil || r.CreatedAt.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		realizedGainLosssByPortfolio[r.PortfolioID] = append(realizedGainLosssByPortfolio[r.PortfolioID], r)

	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating realizedGainLoss table: %w", err)
	}

	return realizedGainLosssByPortfolio, nil
}
