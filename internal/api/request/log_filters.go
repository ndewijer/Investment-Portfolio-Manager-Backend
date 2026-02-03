package request

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// LogFilters represents parsed and validated log filter parameters
type LogFilters struct {
	Levels     []string
	Categories []string
	StartDate  *time.Time
	EndDate    *time.Time
	Source     string
	Message    string
	SortDir    string
	Cursor     string
	PerPage    int
}

// ValidLogLevels are the accepted log level values
var ValidLogLevels = map[string]bool{
	"debug": true, "info": true, "warning": true, "error": true, "critical": true,
}

// ValidCategories are the accepted category values
var ValidCategories = map[string]bool{
	"portfolio": true, "fund": true, "transaction": true, "dividend": true,
	"system": true, "database": true, "security": true, "ibkr": true, "developer": true,
}

// ParseLogFilters extracts and validates log filters from query parameters
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
			if !ValidLogLevels[level] {
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
			if !ValidCategories[category] {
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

// ParseTime parses date strings in multiple formats
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
