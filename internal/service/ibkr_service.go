package service

import (
	"fmt"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// IbkrService handles IBKR (Interactive Brokers) integration business logic operations.
type IbkrService struct {
	ibkrRepo      *repository.IbkrRepository
	portfolioRepo *repository.PortfolioRepository
}

// NewIbkrService creates a new IbkrService with the provided repository dependencies.
func NewIbkrService(
	ibkrRepo *repository.IbkrRepository, portfolioRepo *repository.PortfolioRepository,
) *IbkrService {
	return &IbkrService{
		ibkrRepo:      ibkrRepo,
		portfolioRepo: portfolioRepo,
	}
}

// GetIbkrConfig retrieves the IBKR integration configuration.
// Adds a token expiration warning if the token expires within 30 days.
func (s *IbkrService) GetIbkrConfig() (*model.IbkrConfig, error) {
	config, err := s.ibkrRepo.GetIbkrConfig()

	if err != nil {
		return config, err // Return whatever we got
	}
	if config == nil {
		return nil, fmt.Errorf("unexpected nil config")
	}

	if !config.TokenExpiresAt.IsZero() {
		diff := time.Until(*config.TokenExpiresAt)
		if diff.Hours() <= 720.0 {
			config.TokenWarning = fmt.Sprintf("Token expires in %d days",
				int64(diff.Hours()/24))
		}
	}

	return config, err
}

// GetActivePortfolios retrieves all active portfolios that can be used for IBKR import allocation.
// Returns portfolios that are not archived and not excluded from tracking.
func (s *IbkrService) GetActivePortfolios() ([]model.Portfolio, error) {
	return s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: false,
		IncludeExcluded: false,
	})
}

// GetPendingDividends retrieves dividend records with PENDING reinvestment status.
// These dividends can be matched to incoming IBKR dividend transactions.
// Optionally filters by fund symbol or ISIN.
func (s *IbkrService) GetPendingDividends(symbol, isin string) ([]model.PendingDividend, error) {
	return s.ibkrRepo.GetPendingDividends(symbol, isin)
}

// GetInbox retrieves IBKR imported transactions from the inbox.
// Returns transactions filtered by status (defaults to "pending") and optionally by transaction type.
// Used to display imported IBKR transactions that need to be allocated to portfolios.
func (s *IbkrService) GetInbox(status, transactionType string) ([]model.IBKRTransaction, error) {
	return s.ibkrRepo.GetInbox(status, transactionType)
}
