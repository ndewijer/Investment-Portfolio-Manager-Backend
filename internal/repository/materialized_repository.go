package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

// MaterializedRepository provides data access methods for the portfolio_history_materialized table.
type MaterializedRepository struct {
	db *sql.DB
}

// NewMaterializedRepository creates a new repository instance.
func NewMaterializedRepository(db *sql.DB) *MaterializedRepository {
	return &MaterializedRepository{db: db}
}

// GetMaterializedHistory retrieves pre-calculated portfolio history records from the materialized view table.
// This method streams results using a callback pattern to minimize memory usage.
//
// The materialized view contains daily snapshots of portfolio valuations that have been
// pre-calculated and stored, eliminating the need to recompute historical data on each request.
//
// Parameters:
//   - portfolioIDs: Slice of portfolio IDs to retrieve history for
//   - startDate: First date to include in results (inclusive)
//   - endDate: Last date to include in results (inclusive)
//   - callback: Function called for each record found, receives the record and should return error if processing fails
//
// The callback pattern allows the caller to process records one at a time without loading
// the entire result set into memory, which is efficient for large date ranges.
//
// Returns an error if the query fails or if the callback returns an error during processing.
func (r *MaterializedRepository) GetMaterializedHistory(
	portfolioIDs []string,
	startDate, endDate time.Time,
	callback func(record model.PortfolioHistoryMaterialized) error,
) error {

	if len(portfolioIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(portfolioIDs))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := `
		SELECT id, portfolio_id, date, value, cost, realized_gain, unrealized_gain,
		       total_dividends, total_sale_proceeds, total_original_cost,
		       total_gain_loss, is_archived, calculated_at
		FROM portfolio_history_materialized
		WHERE portfolio_id IN (` + strings.Join(placeholders, ",") + `)
		AND date >= ?
		AND date <= ?
		ORDER BY date ASC
	`

	args := make([]any, 0, len(portfolioIDs)+2)
	for _, id := range portfolioIDs {
		args = append(args, id)
	}
	args = append(args, startDate.Format("2006-01-02"))
	args = append(args, endDate.Format("2006-01-02"))

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to query portfolio_history_materialized: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var record model.PortfolioHistoryMaterialized
		var dateStr, calculatedAtStr string

		err := rows.Scan(
			&record.ID,
			&record.PortfolioID,
			&dateStr,
			&record.Value,
			&record.Cost,
			&record.RealizedGain,
			&record.UnrealizedGain,
			&record.TotalDividends,
			&record.TotalSaleProceeds,
			&record.TotalOriginalCost,
			&record.TotalGainLoss,
			&record.IsArchived,
			&calculatedAtStr,
		)
		if err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		record.Date, err = ParseTime(dateStr)
		if err != nil {
			return fmt.Errorf("failed to parse date: %w", err)
		}

		record.CalculatedAt, err = ParseTime(calculatedAtStr)
		if err != nil {
			return fmt.Errorf("failed to parse calculated_at: %w", err)
		}

		if err := callback(record); err != nil {
			return err
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	return nil
}
