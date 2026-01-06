package service

import (
	"database/sql"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/database"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/version"
)

// SystemService handles system-related operations
type SystemService struct {
	db *sql.DB
}

// NewSystemService creates a new SystemService
func NewSystemService(db *sql.DB) *SystemService {
	return &SystemService{
		db: db,
	}
}

// CheckHealth checks the health of the system
func (s *SystemService) CheckHealth() error {
	return database.HealthCheck(s.db)
}

func (s *SystemService) CheckVersion() string {
	return version.Version
}
