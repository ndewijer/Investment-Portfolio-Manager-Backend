package request

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
)

// LogFilters represents parsed and validated log filter parameters for querying system logs.
// All filter fields are optional and can be combined for more specific queries.
type LogFilters struct {
	Levels     []string   // Log levels to filter by (debug, info, warning, error, critical)
	Categories []string   // Categories to filter by (portfolio, fund, transaction, etc.)
	StartDate  *time.Time // Filter logs from this timestamp onwards (inclusive)
	EndDate    *time.Time // Filter logs up to this timestamp (inclusive)
	Source     string     // Filter by source field using partial match
	Message    string     // Filter by message content using partial match
	SortDir    string     // Sort direction: "asc" or "desc" (default: "desc")
	Cursor     string     // Pagination cursor from previous response (format: "timestamp_id")
	PerPage    int        // Number of results per page (1-100, default: 50)
}

// ParseLogFilters extracts and validates log filters from query parameters.
// Converts raw query string parameters into a validated LogFilters struct.
//
// Parameters are expected as comma-separated strings (for levels and categories)
// or single values (for other fields). All parameters are optional.
//
// Validation rules:
//   - levels: Must be valid log levels (debug, info, warning, error, critical)
//   - categories: Must be valid categories (portfolio, fund, transaction, etc.)
//   - startDate/endDate: Must be valid date/datetime strings (YYYY-MM-DD or RFC3339)
//   - sortDir: Must be "asc" or "desc" (defaults to "desc")
//   - perPage: Must be between 1 and 100 (defaults to 50)
//
// Returns an error if any parameter fails validation.
//
//nolint:gocyclo // Complex validation logic is intentional and clear
func ParseLogFilters(
	levelsParam, categoriesParam, startDateParam, endDateParam,
	sourceParam, messageParam, sortDirParam, cursorParam, perPageParam string,
) (*LogFilters, error) {
	filters := &LogFilters{
		Cursor:  cursorParam,
		Source:  sourceParam,
		Message: messageParam,
	}

	// Parse levels (comma-separated)
	if levelsParam != "" {
		levels := strings.Split(levelsParam, ",")
		for _, level := range levels {
			level = strings.TrimSpace(strings.ToLower(level))
			if !model.ValidLogLevels[model.LogLevel(level)] {
				return nil, fmt.Errorf("invalid log level: %s", level)
			}
			filters.Levels = append(filters.Levels, level)
		}
	}

	// Parse categories (comma-separated)
	if categoriesParam != "" {
		categories := strings.Split(categoriesParam, ",")
		for _, category := range categories {
			category = strings.TrimSpace(strings.ToLower(category))
			if !model.ValidLogCategories[model.LogCategory(category)] {
				return nil, fmt.Errorf("invalid category: %s", category)
			}
			filters.Categories = append(filters.Categories, category)
		}
	}

	// Parse start_date
	if startDateParam != "" {
		startTime, err := ParseTime(startDateParam)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date format: %w", err)
		}
		filters.StartDate = &startTime
	}

	// Parse end_date
	if endDateParam != "" {
		endTime, err := ParseTime(endDateParam)
		if err != nil {
			return nil, fmt.Errorf("invalid end_date format: %w", err)
		}
		filters.EndDate = &endTime
	}

	// Validate sort_dir
	if sortDirParam != "" {
		sortDir := strings.ToLower(sortDirParam)
		if sortDir != "asc" && sortDir != "desc" {
			return nil, fmt.Errorf("invalid sort_dir: must be 'asc' or 'desc'")
		}
		filters.SortDir = sortDir
	} else {
		filters.SortDir = "desc" // Default
	}

	// Parse and validate per_page
	if perPageParam != "" {
		perPage, err := strconv.Atoi(perPageParam)
		if err != nil {
			return nil, fmt.Errorf("invalid per_page: must be a number")
		}
		if perPage < 1 || perPage > 100 {
			return nil, fmt.Errorf("invalid per_page: must be between 1 and 100")
		}
		filters.PerPage = perPage
	} else {
		filters.PerPage = 50 // Default
	}

	return filters, nil
}

// ParseTime parses date strings in multiple formats and returns a time.Time value.
// Attempts to parse the input string using the following formats in order:
//  1. Date-only format: "2006-01-02" (YYYY-MM-DD)
//  2. RFC3339 format: "2006-01-02T15:04:05Z07:00" (ISO 8601 with timezone)
//  3. RFC3339 with milliseconds: "2006-01-02T15:04:05.000Z07:00"
//
// Returns the parsed time on success, or an error if none of the formats match.
func ParseTime(str string) (time.Time, error) {
	// Try date-only format first
	t, err := time.Parse("2006-01-02", str)
	if err == nil {
		return t, nil
	}

	// Try RFC3339 (ISO 8601 with timezone)
	t, err = time.Parse(time.RFC3339, str)
	if err == nil {
		return t, nil
	}

	// Try RFC3339 with milliseconds
	t, err = time.Parse("2006-01-02T15:04:05.000Z07:00", str)
	return t, err
}
