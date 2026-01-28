package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

// MaterializedRepository provides data access methods for the fund_history_materialized table.
type MaterializedRepository struct {
	db *sql.DB
}

// NewMaterializedRepository creates a new repository instance.
func NewMaterializedRepository(db *sql.DB) *MaterializedRepository {
	return &MaterializedRepository{db: db}
}

// GetMaterializedHistory retrieves aggregated portfolio history by querying fund_history_materialized.
// This method streams results using a callback pattern to minimize memory usage.
//
// The method performs the following:
//  1. Builds the SQL query and parameter list (delegates to buildMaterializedQuery)
//  2. Executes the query against the fund_history_materialized table
//  3. Scans each row into a PortfolioHistoryMaterialized struct
//  4. Parses date fields (date, calculated_at)
//  5. Invokes the callback function for each record
//
// The query aggregates fund-level data from fund_history_materialized using GROUP BY,
// and uses correlated subqueries to fetch cumulative values:
//   - realized_gain, total_sale_proceeds, total_original_cost from realized_gain_loss table
//   - total_dividends from dividend table
//   - is_archived from portfolio table
//
// Parameters:
//   - portfolioIDs: Slice of portfolio IDs to retrieve history for
//   - startDate: First date to include in results (inclusive)
//   - endDate: Last date to include in results (inclusive)
//   - callback: Function called for each aggregated record, receives the record and should return error if processing fails
//
// The callback pattern allows the caller to process records one at a time without loading
// the entire result set into memory, which is efficient for large date ranges.
//
// Returns an error if the query fails, date parsing fails, or if the callback returns an error during processing.
func (r *MaterializedRepository) GetMaterializedHistory(
	portfolioIDs []string,
	startDate, endDate time.Time,
	callback func(record model.PortfolioHistoryMaterialized) error,
) error {

	if len(portfolioIDs) == 0 {
		return nil
	}

	query, args := r.buildMaterializedQuery(portfolioIDs, startDate, endDate)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to query fund_history_materialized: %w", err)
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

// buildMaterializedQuery constructs the SQL query and argument list for fetching portfolio history
// from the materialized view. This method was extracted from GetMaterializedHistory to reduce
// function complexity and improve readability.
//
// The query performs portfolio-level aggregation by:
//   - Joining fund_history_materialized with portfolio_fund and portfolio tables
//   - Grouping by date and portfolio_id to sum fund-level metrics
//   - Calculating cumulative realized gains, dividends, and sale proceeds via subqueries
//   - Filtering by portfolio IDs and date range
//
// Parameters:
//   - portfolioIDs: Slice of portfolio UUIDs to include in the query
//   - startDate: Inclusive start date for the history range
//   - endDate: Inclusive end date for the history range
//
// Returns:
//   - SQL query string with placeholders for portfolioIDs and dates
//   - Argument slice containing portfolio IDs followed by formatted start and end dates
//
// Security Note: The #nosec G202 directive is used because placeholder concatenation
// is safe here - we're building "?" placeholders programmatically, not concatenating user input.
func (r *MaterializedRepository) buildMaterializedQuery(portfolioIDs []string, startDate, endDate time.Time) (string, []any) {

	placeholders := make([]string, len(portfolioIDs))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	//#nosec G202 -- Safe: placeholders are generated programmatically, not from user input
	query := `
	SELECT
		'' as id,
		pf.portfolio_id,
		fh.date,
		SUM(fh.value) as value,
		SUM(fh.cost) as cost,
		COALESCE((
			SELECT SUM(realized_gain_loss)
			FROM realized_gain_loss rgl
			WHERE rgl.portfolio_id = pf.portfolio_id
			AND date(rgl.transaction_date) <= fh.date
		), 0) as realized_gain,
		SUM(fh.unrealized_gain) as unrealized_gain,
		COALESCE((
			SELECT SUM(d.total_amount)
			FROM dividend d
			JOIN portfolio_fund pf2 ON d.portfolio_fund_id = pf2.id
			WHERE pf2.portfolio_id = pf.portfolio_id
			AND date(d.ex_dividend_date) <= fh.date
		), 0) as total_dividends,
		COALESCE((
			SELECT SUM(sale_proceeds)
			FROM realized_gain_loss rgl
			WHERE rgl.portfolio_id = pf.portfolio_id
			AND date(rgl.transaction_date) <= fh.date
		), 0) as total_sale_proceeds,
		COALESCE((
			SELECT SUM(cost_basis)
			FROM realized_gain_loss rgl
			WHERE rgl.portfolio_id = pf.portfolio_id
			AND date(rgl.transaction_date) <= fh.date
		), 0) as total_original_cost,
		SUM(fh.total_gain_loss) as total_gain_loss,
		p.is_archived,
		strftime('%Y-%m-%dT%H:%M:%SZ', 'now') as calculated_at
	FROM fund_history_materialized fh
	JOIN portfolio_fund pf ON fh.portfolio_fund_id = pf.id
	JOIN portfolio p ON pf.portfolio_id = p.id
	WHERE pf.portfolio_id IN (` + strings.Join(placeholders, ",") + `)
	AND fh.date >= ?
	AND fh.date <= ?
	GROUP BY fh.date, pf.portfolio_id, p.is_archived
	ORDER BY fh.date ASC
`

	args := make([]any, 0, len(portfolioIDs)+2)
	for _, id := range portfolioIDs {
		args = append(args, id)
	}
	args = append(args, startDate.Format("2006-01-02"))
	args = append(args, endDate.Format("2006-01-02"))

	return query, args
}

// GetFundHistoryMaterialized retrieves historical fund data from the materialized view.
// This method streams results using a callback pattern to minimize memory usage.
//
// The callback pattern allows the caller to process records one at a time without loading
// the entire result set into memory, which is efficient for large date ranges.
//
// Parameters:
//   - portfolioID: The portfolio ID to retrieve fund history for
//   - startDate: Inclusive start date for the query
//   - endDate: Inclusive end date for the query
//   - callback: Function called for each record found, receives the record and should return error if processing fails
//
// Returns an error if the query fails or if the callback returns an error during processing.
func (r *MaterializedRepository) GetFundHistoryMaterialized(
	portfolioID string,
	startDate, endDate time.Time,
	callback func(entry model.FundHistoryEntry) error,
) error {
	query := `
		SELECT
			fh.id,
			fh.portfolio_fund_id,
			fh.fund_id,
			f.name as fund_name,
			fh.date,
			fh.shares,
			fh.price,
			fh.value,
			fh.cost,
			fh.realized_gain,
			fh.unrealized_gain,
			fh.total_gain_loss,
			fh.dividends,
			fh.fees
		FROM fund_history_materialized fh
		JOIN portfolio_fund pf ON fh.portfolio_fund_id = pf.id
		JOIN fund f ON fh.fund_id = f.id
		WHERE pf.portfolio_id = ?
		  AND fh.date >= ?
		  AND fh.date <= ?
		ORDER BY fh.date ASC, f.name ASC
	`

	rows, err := r.db.Query(query, portfolioID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to query fund_history_materialized: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var entry model.FundHistoryEntry
		var dateStr string

		err := rows.Scan(
			&entry.ID,
			&entry.PortfolioFundID,
			&entry.FundID,
			&entry.FundName,
			&dateStr,
			&entry.Shares,
			&entry.Price,
			&entry.Value,
			&entry.Cost,
			&entry.RealizedGain,
			&entry.UnrealizedGain,
			&entry.TotalGainLoss,
			&entry.Dividends,
			&entry.Fees,
		)
		if err != nil {
			return fmt.Errorf("failed to scan fund_history_materialized results: %w", err)
		}

		entry.Date, err = ParseTime(dateStr)
		if err != nil || entry.Date.IsZero() {
			return fmt.Errorf("failed to parse date: %w", err)
		}

		if err := callback(entry); err != nil {
			return err
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating fund_history_materialized: %w", err)
	}

	return nil
}
