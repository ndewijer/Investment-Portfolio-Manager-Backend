package service

import (
	"fmt"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// IbkrService handles fund-related business logic operations.
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

func (s *IbkrService) GetIbkrConfig() (*model.IbkrConfig, error) {
	config, err := s.ibkrRepo.GetIbkrConfig()

	if err != nil {
		return config, err // Return whatever we got
	}
	if config == nil {
		return nil, fmt.Errorf("unexpected nil config")
	}

	if !config.TokenExpiresAt.IsZero() {
		diff := time.Until(config.TokenExpiresAt)
		if diff.Hours() <= 720.0 {
			config.TokenWarning = fmt.Sprintf("Token expires in %d days",
				int64(diff.Hours()/24))
		}
	}

	return config, err
}

func (s *IbkrService) GetActivePortfolios() ([]model.Portfolio, error) {
	return s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: false,
		IncludeExcluded: false,
	})
}
