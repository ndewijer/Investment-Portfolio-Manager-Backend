package repository

import (
	"fmt"
	"time"
)

// ParseTime parses a date string in "2006-01-02" or RFC3339 format.
// Note: mirrors validation.ParseTime — both are intentionally kept local to avoid cross-layer imports.
func ParseTime(str string) (time.Time, error) {
	returnTime, err := time.Parse("2006-01-02", str)
	if err != nil {
		returnTime, err = time.Parse(time.RFC3339, str)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse date: %w", err)
		}
	}
	return returnTime.UTC(), nil
}
