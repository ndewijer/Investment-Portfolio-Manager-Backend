package repository

import (
	"fmt"
	"time"
)

// ParseTime parses a date string in "2006-01-02", "2006-01-02 15:04:05", or RFC3339 format.
// Note: mirrors validation.ParseTime — both are intentionally kept local to avoid cross-layer imports.
func ParseTime(str string) (time.Time, error) {
	for _, layout := range []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, str); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("failed to parse date: %q matches no known format", str)
}
