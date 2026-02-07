package model

import "time"

// LogResponse represents a paginated response containing log entries.
// Includes cursor-based pagination information for retrieving subsequent pages.
type LogResponse struct {
	Logs       []Log  `json:"logs"`       // Array of log entries for the current page
	NextCursor string `json:"nextCursor"` // Cursor for fetching the next page (empty if no more pages)
	HasMore    bool   `json:"hasMore"`    // Indicates if more results are available
	Count      int    `json:"count"`      // Number of logs in the current response
}

// Log represents a single system log entry with metadata and optional contextual information.
type Log struct {
	ID         string    `json:"id"`                   // Unique identifier for the log entry
	Timestamp  time.Time `json:"timestamp"`            // When the log was created
	Level      string    `json:"level"`                // Log level (DEBUG, INFO, WARNING, ERROR, CRITICAL)
	Category   string    `json:"category"`             // Category of the log (portfolio, fund, transaction, etc.)
	Message    string    `json:"message"`              // Primary log message
	Details    string    `json:"details,omitempty"`    // Additional details or context (optional)
	Source     string    `json:"source"`               // Source component that generated the log
	RequestID  string    `json:"requestId,omitempty"`  // Request ID for tracing (optional)
	HTTPStatus string    `json:"httpStatus,omitempty"` // HTTP status code if applicable (optional)
	IPAddress  string    `json:"ipAddress,omitempty"`  // IP address of the request (optional)
	UserAgent  string    `json:"userAgent,omitempty"`  // User agent string (optional)
}

// SystemSetting represents a key-value configuration setting stored in the database.
type SystemSetting struct {
	ID        string     // Unique identifier for the setting
	Key       string     // Setting key name
	Value     any        // Setting value (can be any type)
	UpdatedAt *time.Time // When the setting was last updated (optional)
}

// LoggingSetting represents the current logging configuration.
type LoggingSetting struct {
	Enabled bool   `json:"enabled"` // Whether logging is enabled
	Level   string `json:"level"`   // Current log level (debug, info, warning, error, critical)
}

// ExchangeRateWrapper wraps exchange rate query results.
// The Rate field will be nil if no exchange rate exists for the given parameters.
type ExchangeRateWrapper struct {
	FromCurrency string        `json:"fromCurrency"` // Source currency code
	ToCurrency   string        `json:"toCurrency"`   // Target currency code
	Rate         *ExchangeRate `json:"rate"`         // Exchange rate data (null if not found)
	Date         string        `json:"date"`         // Date for the query in YYYY-MM-DD format
}

// ExchangeRate represents a currency exchange rate for a specific date.
type ExchangeRate struct {
	FromCurrency string    `json:"fromCurrency"` // Source currency code
	ToCurrency   string    `json:"toCurrency"`   // Target currency code
	Rate         float64   `json:"rate"`         // Exchange rate value
	Date         time.Time `json:"date"`         // Date the rate applies to
}

// TemplateModel represents a CSV import template structure.
// Provides the expected headers, an example row, and format description.
type TemplateModel struct {
	Headers     []string          `json:"headers"`     // CSV column headers
	Example     map[string]string `json:"example"`     // Example values for each header
	Description string            `json:"description"` // Human-readable format description
}
