package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

// RealizedGainLossRepository provides data access methods for the realized_gain_loss table.
// It handles retrieving records of gains and losses from sold positions.
type RealizedGainLossRepository struct {
	db *sql.DB
}

// NewRealizedGainLossRepository creates a new RealizedGainLossRepository with the provided database connection.
func NewRealizedGainLossRepository(db *sql.DB) *RealizedGainLossRepository {
	return &RealizedGainLossRepository{db: db}
}

// GetRealizedGainLossByPortfolio retrieves all realized gain/loss records for the given portfolios within the specified date range.
// Records are filtered by transaction_date and sorted in ascending order by created_at.
//
// Parameters:
//   - portfolio: slice of portfolios to query
//   - startDate: inclusive start date for the query (compared against transaction_date)
//   - endDate: inclusive end date for the query (compared against transaction_date)
//
// Returns a map of portfolioID -> []RealizedGainLoss. If portfolio is empty, returns an empty map.
// Each record contains details about a sell transaction including shares sold, cost basis,
// sale proceeds, and the calculated realized gain or loss.
func (s *RealizedGainLossRepository) GetRealizedGainLossByPortfolio(portfolio []string, startDate, endDate time.Time) (map[string][]model.RealizedGainLoss, error) {
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
		AND transaction_date >= ?
		AND transaction_date <= ?
		ORDER BY created_at ASC
	`

	// Build args: portfolio first, then startDate, then endDate
	realizedGainLossdArgs := make([]any, 0, len(portfolio)+2)
	for _, id := range portfolio {
		realizedGainLossdArgs = append(realizedGainLossdArgs, id)
	}
	realizedGainLossdArgs = append(realizedGainLossdArgs, startDate.Format("2006-01-02"))
	realizedGainLossdArgs = append(realizedGainLossdArgs, endDate.Format("2006-01-02"))

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
