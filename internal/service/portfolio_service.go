package service

import (
	"database/sql"
	"fmt"
)

// PortfolioService handles Portfolio-related operations
type PortfolioService struct {
	db *sql.DB
}

// NewPortfolioService creates a new PortfolioService
func NewPortfolioService(db *sql.DB) *PortfolioService {
	return &PortfolioService{
		db: db,
	}
}

// Portfolio represents a portfolio from the database
type Portfolio struct {
	ID                  string
	Name                string
	Description         string
	IsArchived          bool
	ExcludeFromOverview bool
}

// GetAllPortfolios retrieves all portfolios from the database
func (s *PortfolioService) GetAllPortfolios() ([]Portfolio, error) {
	query := `
          SELECT id, name, description, is_archived, exclude_from_overview
          FROM portfolio
      `
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query portfolios table: %w", err)
	}
	defer rows.Close() // IMPORTANT: Always close rows!

	portfolios := []Portfolio{}

	for rows.Next() {
		var p Portfolio

		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Description,
			&p.IsArchived,
			&p.ExcludeFromOverview,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan portfolio table results: %w", err)
		}

		portfolios = append(portfolios, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating portfolios table: %w", err)
	}

	return portfolios, nil
}
