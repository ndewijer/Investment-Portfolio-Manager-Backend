package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/apperrors"
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
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// GetAllFunds retrieves all funds from the database with their latest prices.
// Returns an empty slice if no funds are found.
func (r *FundRepository) GetAllFunds() ([]model.Fund, error) {
	query := `
        SELECT f.id, f.name, f.isin, f.symbol, f.currency, f.exchange, f.investment_type, f.dividend_type, fp.price
		FROM fund f
		LEFT JOIN (
			SELECT fp.fund_id, fp.price, fp.date
			FROM fund_price fp
			INNER JOIN (
				SELECT fund_id, MAX(date) as latest_date
				FROM fund_price
				GROUP BY fund_id
			) latest ON fp.fund_id = latest.fund_id AND fp.date = latest.latest_date
		)  fp ON f.id = fp.fund_id
      `

	rows, err := r.getQuerier().Query(query)

	if err != nil {
		return nil, fmt.Errorf("failed to query fund table: %w", err)
	}
	defer rows.Close()

	funds := []model.Fund{}

	for rows.Next() {
		var f model.Fund
		var priceStr sql.NullFloat64

		err := rows.Scan(

			&f.ID,
			&f.Name,
			&f.Isin,
			&f.Symbol,
			&f.Currency,
			&f.Exchange,
			&f.InvestmentType,
			&f.DividendType,
			&priceStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fund table results: %w", err)
		}

		if priceStr.Valid {
			f.LatestPrice = priceStr.Float64
		}

		funds = append(funds, f)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fund table: %w", err)
	}

	return funds, nil
}

// GetFund retrieves a single fund by ID with its latest price.
func (r *FundRepository) GetFund(fundID string) (model.Fund, error) {
	query := `
        SELECT f.id, f.name, f.isin, f.symbol, f.currency, f.exchange, f.investment_type, f.dividend_type, fp.price
		FROM fund f
		LEFT JOIN (
			SELECT fp.fund_id, fp.price, fp.date
			FROM fund_price fp
			INNER JOIN (
				SELECT fund_id, MAX(date) as latest_date
				FROM fund_price
				GROUP BY fund_id
			) latest ON fp.fund_id = latest.fund_id AND fp.date = latest.latest_date
		)  fp ON f.id = fp.fund_id
 		WHERE f.id = ?
		`

	var f model.Fund
	var priceStr sql.NullFloat64
	err := r.getQuerier().QueryRow(query, fundID).Scan(
		&f.ID,
		&f.Name,
		&f.Isin,
		&f.Symbol,
		&f.Currency,
		&f.Exchange,
		&f.InvestmentType,
		&f.DividendType,
		&priceStr,
	)
	if err == sql.ErrNoRows {
		return model.Fund{}, apperrors.ErrFundNotFound
	}
	if err != nil {
		return model.Fund{}, err
	}
	if priceStr.Valid {
		f.LatestPrice = priceStr.Float64
	}

	return f, nil
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
		SELECT f.id, f.name, f.isin, f.symbol, f.currency, f.exchange, f.investment_type, f.dividend_type, fp.price
		FROM fund f
		LEFT JOIN (
			SELECT fp.fund_id, fp.price, fp.date
			FROM fund_price fp
			INNER JOIN (
				SELECT fund_id, MAX(date) as latest_date
				FROM fund_price
				GROUP BY fund_id
			) latest ON fp.fund_id = latest.fund_id AND fp.date = latest.latest_date
		)  fp ON f.id = fp.fund_id
      WHERE f.id IN (` + strings.Join(fundPlaceholders, ",") + `)
  `

	fundArgs := make([]any, len(fundIDs))
	for i, id := range fundIDs {
		fundArgs[i] = id
	}

	rows, err := r.getQuerier().Query(fundQuery, fundArgs...)
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

	rows, err := r.getQuerier().Query(fundPriceQuery, fundPriceArgs...)
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
	var isValidstr sql.NullBool
	err := r.getQuerier().QueryRow(query, symbol).Scan(
		&sb.ID,
		&sb.Symbol,
		&sb.Name,
		&exchangeStr,
		&currencyStr,
		&isinStr,
		&lastUpdatedStr,
		&dataSource,
		&isValidstr,
	)
	if err == sql.ErrNoRows {
		return nil, apperrors.ErrSymbolNotFound
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
	if isValidstr.Valid {
		sb.IsValid = isValidstr.Bool
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

	err := r.getQuerier().QueryRow(query, args...).Scan(
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
		return model.Fund{}, apperrors.ErrFundNotFound
	}
	if err != nil {
		return model.Fund{}, err
	}

	return f, nil

}

// InsertFund inserts a new fund into the database.
// The fund struct should have all required fields populated including a generated ID.
// Returns an error if the insertion fails (e.g., due to constraint violations).
func (r *FundRepository) InsertFund(ctx context.Context, f *model.Fund) error {
	query := `
        INSERT INTO fund (id, name, isin, symbol, exchange, currency, investment_type, dividend_type)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `

	_, err := r.getQuerier().ExecContext(ctx, query,
		f.ID,
		f.Name,
		f.Isin,
		f.Symbol,
		f.Exchange,
		f.Currency,
		f.InvestmentType,
		f.DividendType,
	)

	if err != nil {
		return fmt.Errorf("failed to insert fund: %w", err)
	}

	return nil
}

// UpdateFund updates an existing fund in the database.
// Updates all fund fields based on the provided fund struct.
// Returns ErrFundNotFound if no fund with the given ID exists.
// Returns an error if the update fails.
func (r *FundRepository) UpdateFund(ctx context.Context, f *model.Fund) error {
	query := `
        UPDATE fund
        SET name = ?, isin = ?, symbol = ?, exchange = ?, currency = ?, investment_type = ?, dividend_type = ?
        WHERE id = ?
    `

	result, err := r.getQuerier().ExecContext(ctx, query,
		f.Name,
		f.Isin,
		f.Symbol,
		f.Exchange,
		f.Currency,
		f.InvestmentType,
		f.DividendType,
		f.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update fund: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return apperrors.ErrFundNotFound
	}

	return nil
}

// DeleteFund removes a fund from the database.
// Returns ErrFundNotFound if no fund with the given ID exists.
// Returns an error if the deletion fails.
func (r *FundRepository) DeleteFund(ctx context.Context, fundID string) error {
	query := `DELETE FROM fund WHERE id = ?`

	result, err := r.getQuerier().ExecContext(ctx, query, fundID)
	if err != nil {
		return fmt.Errorf("failed to delete fund: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return apperrors.ErrFundNotFound
	}

	return nil
}

// InsertFundPrice inserts a single fund price record into the database.
// This method is used for single price updates, such as adding the latest daily price.
//
// The insertion respects transaction context through getQuerier(), allowing this method
// to participate in larger transactions when called via WithTx().
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - fp: FundPrice record containing ID, FundID, Date, and Price
//
// Returns:
//   - error: If the insertion fails, wrapped with context
//
// Note: This method does not check for duplicate dates. Callers should verify
// that the price doesn't already exist before calling this method.
func (r *FundRepository) InsertFundPrice(ctx context.Context, fp model.FundPrice) error {
	query := `
        INSERT INTO fund_price (id, fund_id, date, price)
        VALUES (?, ?, ?, ?)
    `

	_, err := r.getQuerier().ExecContext(ctx, query,
		fp.ID,
		fp.FundID,
		fp.Date.Format("2006-01-02"),
		fp.Price,
	)

	if err != nil {
		return fmt.Errorf("failed to insert fund price: %w", err)
	}

	return nil
}

// InsertFundPrices performs a batch insert of multiple fund price records.
// This method is optimized for inserting large numbers of prices at once, such as
// during historical data backfilling operations.
//
// The method uses a prepared statement via getQuerier() to efficiently insert
// multiple records. Atomicity depends on the caller: this method participates in
// an external transaction when called via WithTx(tx), and has no atomicity guarantee
// when called without one. The sole caller (UpdateHistoricalFundPrice in FundService)
// always wraps this call in a service-level transaction.
//
// Implementation Details:
//   - Participates in the caller's transaction via WithTx; no internal transaction is created
//   - Prepares a single INSERT statement reused for all records
//   - Formats dates as "2006-01-02" for database compatibility
//   - Rollback/commit is the responsibility of the calling service
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - fundPrices: Slice of FundPrice records to insert
//
// Returns:
//   - error: If statement preparation or any insertion fails. All errors are wrapped with context.
//   - nil: If fundPrices is empty (no-op) or all insertions succeed
func (r *FundRepository) InsertFundPrices(ctx context.Context, fundPrices []model.FundPrice) error {
	if len(fundPrices) == 0 {
		return nil
	}

	stmt, err := r.getQuerier().PrepareContext(ctx, `
        INSERT INTO fund_price (id, fund_id, date, price)
        VALUES (?, ?, ?, ?)
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, fp := range fundPrices {
		_, err := stmt.ExecContext(ctx, fp.ID, fp.FundID, fp.Date.Format("2006-01-02"), fp.Price)
		if err != nil {
			return fmt.Errorf("failed to insert fund price for %s on %s: %w", fp.FundID, fp.Date.Format("2006-01-02"), err)
		}
	}

	return nil
}

func (r *FundRepository) UpdateFundPrice(ctx context.Context, fp model.FundPrice) error {

	query := `
		INSERT INTO fund_price (id, fund_id, date, price)
		VALUES (?, ?, ?, ?)
        ON CONFLICT(fund_id, date) DO UPDATE SET
            price = ?
    `

	_, err := r.getQuerier().ExecContext(ctx, query,
		fp.ID,
		fp.FundID,
		fp.Date.Format("2006-01-02"),
		fp.Price,
		fp.Price,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert fund price: %w", err)
	}

	return nil
}
