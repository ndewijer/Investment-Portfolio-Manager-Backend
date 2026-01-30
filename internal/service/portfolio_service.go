package service

import (
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// PortfolioService handles portfolio-related business logic operations.
// It coordinates between multiple repositories to compute portfolio summaries
// and aggregate metrics.
type PortfolioService struct {
	portfolioRepo *repository.PortfolioRepository
}

// NewPortfolioService creates a new PortfolioService with the provided repository dependencies.
func NewPortfolioService(
	portfolioRepo *repository.PortfolioRepository,
) *PortfolioService {
	return &PortfolioService{
		portfolioRepo: portfolioRepo,
	}
}

// GetAllPortfolios retrieves all portfolios from the database with no filters applied.
// This includes both archived and excluded portfolios.
func (s *PortfolioService) GetAllPortfolios() ([]model.Portfolio, error) {
	return s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: true,
		IncludeExcluded: true,
	})
}

// GetPortfolio retrieves a single portfolio by its ID.
// Returns the portfolio metadata including name, description, and archive status.
// This is a simple wrapper around the repository layer for portfolio lookup.
func (s *PortfolioService) GetPortfolio(portfolioID string) (model.Portfolio, error) {
	result, err := s.portfolioRepo.GetPortfolioOnID(portfolioID)
	if err != nil {
		return model.Portfolio{}, err
	}
	return result, nil
}

// LoadActivePortfolios retrieves only active, non-excluded portfolios.
// Archived and excluded portfolios are filtered out.
func (s *PortfolioService) LoadActivePortfolios() ([]model.Portfolio, error) {
	return s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: false,
		IncludeExcluded: false,
	})
}

// LoadAllPortfolioFunds retrieves all funds associated with the given portfolios.
// Returns:
//   - fundsByPortfolio: map[portfolioID][]Fund
//   - portfolioFundToPortfolio: map[portfolioFundID]portfolioID
//   - portfolioFundToFund: map[portfolioFundID]fundID
//   - pfIDs: slice of all portfolio_fund IDs
//   - fundIDs: slice of all unique fund IDs
//   - error: any error encountered
func (s *PortfolioService) LoadAllPortfolioFunds(portfolios []model.Portfolio) (map[string][]model.Fund, map[string]string, map[string]string, []string, []string, error) {
	return s.portfolioRepo.GetPortfolioFundsOnPortfolioID(portfolios)
}

// GetPortfoliosForRequest resolves a portfolio ID parameter into a slice of portfolios.
// This is a helper method that handles the common pattern where API endpoints accept
// an optional portfolio ID - if provided, return that one portfolio; if empty, return
// all active portfolios.
//
// This centralizes the "specific portfolio vs all portfolios" logic that was duplicated
// across multiple service methods.
//
// Parameters:
//   - portfolioID: Optional portfolio ID. Empty string means "all portfolios"
//
// Returns:
//   - If portfolioID is provided: a slice containing just that portfolio
//   - If portfolioID is empty: all active, non-excluded portfolios
//   - Error if the specific portfolio ID is not found or database query fails
//
// Usage in other services:
//
//	portfolios, err := portfolioService.GetPortfoliosForRequest(portfolioID)
//	// portfolios is always a slice, simplifying downstream code
func (s *PortfolioService) GetPortfoliosForRequest(portfolioID string) ([]model.Portfolio, error) {
	if portfolioID != "" {
		portfolio, err := s.portfolioRepo.GetPortfolioOnID(portfolioID)
		if err != nil {
			return []model.Portfolio{}, err
		}

		return []model.Portfolio{portfolio}, nil
	}

	// Load all active portfolios
	return s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: false,
		IncludeExcluded: false,
	})
}
