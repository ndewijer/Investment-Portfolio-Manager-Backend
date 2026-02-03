package repository

import "database/sql"

// DeveloperRepository provides data access methods for the Developer table.
// It handles retrieving Developer records and reinvestment information.
type DeveloperRepository struct {
	db *sql.DB
}

// NewDeveloperRepository creates a new DeveloperRepository with the provided database connection.
func NewDeveloperRepository(db *sql.DB) *DeveloperRepository {
	return &DeveloperRepository{db: db}
}
