package service

import (
	"context"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// DeveloperService handles Developer-related business logic operations.
type DeveloperService struct {
	developerRepo *repository.DeveloperRepository
	fundRepo      *repository.FundRepository
}

// NewDeveloperService creates a new DeveloperService with the provided repository dependencies.
func NewDeveloperService(
	developerRepo *repository.DeveloperRepository,
	fundRepo *repository.FundRepository,
) *DeveloperService {
	return &DeveloperService{
		developerRepo: developerRepo,
		fundRepo:      fundRepo,
	}
}

func (s *DeveloperService) GetLogs(_ context.Context, filters *request.LogFilters) (*model.LogResponse, error) {
	// Add any business logic validation here if needed

	// Pass filters to repository
	return s.developerRepo.GetLogs(filters)
}

func (s *DeveloperService) GetLoggingConfig() (model.LoggingSetting, error) {

	return s.developerRepo.GetLoggingConfig()
}

func (s *DeveloperService) GetExchangeRate(fromCurrency, toCurrency string, dateTime time.Time) (*model.ExchangeRate, error) {

	return s.developerRepo.GetExchangeRate(fromCurrency, toCurrency, dateTime)
}

func (s *DeveloperService) GetFundPrice(fundID string, dateTime time.Time) (model.FundPrice, error) {

	fundIDs := []string{fundID}
	fundPrices, err := s.fundRepo.GetFundPrice(fundIDs, dateTime, dateTime, true)
	if err != nil {
		return model.FundPrice{}, err
	}

	prices, exists := fundPrices[fundID]
	if !exists || len(prices) == 0 {
		return model.FundPrice{}, apperrors.ErrFundPriceNotFound
	}

	return fundPrices[fundID][0], nil
}
