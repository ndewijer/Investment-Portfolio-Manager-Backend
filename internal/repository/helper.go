package repository

import (
	"fmt"
	"time"
)

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
