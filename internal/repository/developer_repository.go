package repository

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

// DeveloperRepository provides data access methods for the Developer table.
// It handles retrieving Developer records and reinvestment information.
type DeveloperRepository struct {
	db *sql.DB
}

// NewDeveloperRepository creates a new DeveloperRepository with the provided database connection.
func NewDeveloperRepository(db *sql.DB) *DeveloperRepository {
	return &DeveloperRepository{db: db}
}

// GetLogs retrieves log entries from the database with dynamic filtering and cursor-based pagination.
// Builds a WHERE clause dynamically based on provided filters and supports multiple filter types:
//   - Level filtering: Filters by one or more log levels (debug, info, warning, error, critical)
//   - Category filtering: Filters by one or more categories (portfolio, fund, transaction, etc.)
//   - Date range filtering: Filters by start and/or end timestamps
//   - Source filtering: Partial match on source field using LIKE
//   - Message filtering: Partial match on message content using LIKE
//   - Cursor pagination: Supports efficient pagination using timestamp+id cursor
//
// The pagination uses cursor-based approach for efficiency with large result sets.
// Returns one extra record beyond perPage to determine if more results exist.
//
//nolint:gocyclo,funlen // Complex filtering logic with dynamic WHERE clause requires length
func (r *DeveloperRepository) GetLogs(filters *request.LogFilters) (*model.LogResponse, error) {
	// Build dynamic WHERE clause
	var whereClauses []string
	var args []any

	// 1. Level filtering (OR logic)
	if len(filters.Levels) > 0 {
		placeholders := make([]string, len(filters.Levels))
		for i, level := range filters.Levels {
			placeholders[i] = "?"
			// Convert to uppercase to match database storage
			args = append(args, strings.ToUpper(level))
		}
		whereClauses = append(whereClauses, "level IN ("+strings.Join(placeholders, ",")+")")
	}

	// 2. Category filtering (OR logic)
	if len(filters.Categories) > 0 {
		placeholders := make([]string, len(filters.Categories))
		for i, category := range filters.Categories {
			placeholders[i] = "?"
			// Convert to uppercase to match database storage
			args = append(args, strings.ToUpper(category))
		}
		whereClauses = append(whereClauses, "category IN ("+strings.Join(placeholders, ",")+")")
	}

	// 3. Date range filtering
	if filters.StartDate != nil {
		whereClauses = append(whereClauses, "timestamp >= ?")
		args = append(args, filters.StartDate.Format(time.RFC3339))
	}

	if filters.EndDate != nil {
		whereClauses = append(whereClauses, "timestamp <= ?")
		args = append(args, filters.EndDate.Format(time.RFC3339))
	}

	// 4. Source filtering (partial match)
	if filters.Source != "" {
		whereClauses = append(whereClauses, "source LIKE ?")
		args = append(args, "%"+filters.Source+"%")
	}

	// 5. Message filtering (partial match)
	if filters.Message != "" {
		whereClauses = append(whereClauses, "message LIKE ?")
		args = append(args, "%"+filters.Message+"%")
	}

	// 6. Cursor pagination
	if filters.Cursor != "" {
		parts := strings.Split(filters.Cursor, "_")
		if len(parts) == 2 {
			timestamp, err := time.Parse(time.RFC3339, parts[0])
			if err == nil {
				id := parts[1]

				if filters.SortDir == "desc" {
					// For descending: (timestamp, id) < (cursor_timestamp, cursor_id)
					whereClauses = append(whereClauses,
						"(timestamp < ? OR (timestamp = ? AND id < ?))")
					args = append(args, timestamp.Format(time.RFC3339),
						timestamp.Format(time.RFC3339), id)
				} else {
					// For ascending: (timestamp, id) > (cursor_timestamp, cursor_id)
					whereClauses = append(whereClauses,
						"(timestamp > ? OR (timestamp = ? AND id > ?))")
					args = append(args, timestamp.Format(time.RFC3339),
						timestamp.Format(time.RFC3339), id)
				}
			}
		}
	}

	// Build WHERE clause
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Build ORDER BY clause based on sort_dir
	orderSQL := "ORDER BY timestamp DESC, id DESC"
	if filters.SortDir == "asc" {
		orderSQL = "ORDER BY timestamp ASC, id ASC"
	}

	// Build complete query
	//nolint:gosec // G202: SQL concatenation is safe - whereSQL and orderSQL contain no user input, all user values are parameterized
	query := `
		SELECT id, timestamp, level, category, message, details, source,
		       user_id, request_id, stack_trace, http_status, ip_address, user_agent
		FROM log
		` + whereSQL + `
		` + orderSQL + `
		LIMIT ?
	`
	args = append(args, filters.PerPage+1)

	// Execute query
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query log table: %w", err)
	}
	defer rows.Close()

	// Scan results
	logs := []model.Log{}

	for rows.Next() {
		var timestampStr string
		var detailsStr, userIDStr, RequestIDStr, stackTraceStr, HTTPStatusStr, IPAddressStr, UserAgentStr sql.NullString
		var l model.Log

		err := rows.Scan(
			&l.ID,
			&timestampStr,
			&l.Level,
			&l.Category,
			&l.Message,
			&detailsStr,
			&l.Source,
			&userIDStr,
			&RequestIDStr,
			&stackTraceStr,
			&HTTPStatusStr,
			&IPAddressStr,
			&UserAgentStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log results: %w", err)
		}

		l.Timestamp, err = ParseTime(timestampStr)
		if err != nil || l.Timestamp.IsZero() {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
		}

		// Handle nullable fields
		if detailsStr.Valid {
			l.Details = detailsStr.String
		}
		if RequestIDStr.Valid {
			l.RequestID = RequestIDStr.String
		}
		if HTTPStatusStr.Valid {
			l.HTTPStatus = HTTPStatusStr.String
		}
		if IPAddressStr.Valid {
			l.IPAddress = IPAddressStr.String
		}
		if UserAgentStr.Valid {
			l.UserAgent = UserAgentStr.String
		}

		logs = append(logs, l)
	}

	// Generate next cursor
	hasMore := len(logs) > filters.PerPage
	var nextCursor string
	if hasMore {
		last := logs[filters.PerPage-1]
		nextCursor = fmt.Sprintf("%s_%s",
			last.Timestamp.Format(time.RFC3339),
			last.ID)
		logs = logs[:filters.PerPage]
	}

	return &model.LogResponse{
		Logs:       logs,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Count:      len(logs),
	}, nil
}

// GetLoggingConfig retrieves the current logging configuration from system_setting table.
// Returns the LOGGING_ENABLED and LOGGING_LEVEL settings.
// If settings are not configured, returns default values: enabled=true, level="info".
// Logs a warning message when default values are used.
func (r *DeveloperRepository) GetLoggingConfig() (model.LoggingSetting, error) {

	queryEnabled := `
        SELECT value
		FROM system_setting
		WHERE key = 'LOGGING_ENABLED'
      `
	queryLevel := `
        SELECT value
		FROM system_setting
		WHERE key = 'LOGGING_LEVEL'
      `
	// Setting default logging mode if settings are not configured.
	conf := model.LoggingSetting{
		Enabled: true,
		Level:   "info",
	}
	err := r.db.QueryRow(queryEnabled).Scan(
		&conf.Enabled,
	)
	if err == sql.ErrNoRows {
		log.Println("Logging not set, defaulting to enabled")
	} else if err != nil {
		return conf, err
	}

	err = r.db.QueryRow(queryLevel).Scan(
		&conf.Level,
	)
	if err == sql.ErrNoRows {
		log.Println("Level not set, defaulting to INFO")
	} else if err != nil {
		return conf, err
	}

	return conf, nil
}

// GetExchangeRate retrieves an exchange rate for a specific currency pair and date.
// Queries the exchange_rate table for an exact match on from_currency, to_currency, and date.
// Returns ErrExchangeRateNotFound if no matching rate exists.
// The date parameter should be in the format YYYY-MM-DD.
func (r *DeveloperRepository) GetExchangeRate(fromCurrency, toCurrency string, dateTime time.Time) (*model.ExchangeRate, error) {

	query := `
	SELECT from_currency, to_currency, rate, date
	FROM exchange_rate
	WHERE from_currency = ?
	AND to_currency = ?
	AND date = ?
`
	rate := model.ExchangeRate{}
	var dateStr string
	err := r.db.QueryRow(query, fromCurrency, toCurrency, dateTime.Format("2006-01-02")).Scan(
		&rate.FromCurrency,
		&rate.ToCurrency,
		&rate.Rate,
		&dateStr,
	)

	if err == sql.ErrNoRows {
		return nil, apperrors.ErrExchangeRateNotFound
	}

	rate.Date, err = ParseTime(dateStr)
	if err != nil || rate.Date.IsZero() {
		return nil, fmt.Errorf("failed to parse date: %w", err)
	}

	return &rate, nil
}
