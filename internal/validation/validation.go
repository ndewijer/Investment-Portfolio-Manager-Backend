package validation

import (
	"fmt"

	"github.com/google/uuid"
)

// Common validation errors
var (
	ErrInvalidUUID      = fmt.Errorf("invalid UUID format")
	ErrInvalidDateRange = fmt.Errorf("invalid date range")
	ErrEmptySlice       = fmt.Errorf("slice cannot be empty")
)

// ValidateUUID checks if a string is a valid UUID
func ValidateUUID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidUUID, id)
	}
	return nil
}

// ValidateUUIDs validates a slice of UUIDs
func ValidateUUIDs(ids []string) error {
	if len(ids) == 0 {
		return ErrEmptySlice
	}
	for _, id := range ids {
		if err := ValidateUUID(id); err != nil {
			return err
		}
	}
	return nil
}
