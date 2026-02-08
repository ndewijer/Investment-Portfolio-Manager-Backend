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

// GetLogs retrieves system logs with the specified filters and pagination.
// Returns a paginated response with cursor for fetching subsequent pages.
// The context parameter is currently unused but reserved for future cancellation support.
func (s *DeveloperService) GetLogs(_ context.Context, filters *request.LogFilters) (*model.LogResponse, error) {
	// Add any business logic validation here if needed

	// Pass filters to repository
	return s.developerRepo.GetLogs(filters)
}

// GetLoggingConfig retrieves the current logging configuration settings.
// Returns the enabled status and logging level from system settings.
// Returns default values (enabled=true, level="info") if settings are not configured.
func (s *DeveloperService) GetLoggingConfig() (model.LoggingSetting, error) {

	return s.developerRepo.GetLoggingConfig()
}

// GetExchangeRate retrieves the exchange rate for a specific currency pair and date.
// Returns ErrExchangeRateNotFound if no rate exists for the given parameters.
func (s *DeveloperService) GetExchangeRate(fromCurrency, toCurrency string, dateTime time.Time) (*model.ExchangeRate, error) {

	return s.developerRepo.GetExchangeRate(fromCurrency, toCurrency, dateTime)
}

// GetFundPrice retrieves the price for a specific fund on a specific date.
// Uses the FundRepository's GetFundPrice method to fetch the price.
// Returns ErrFundPriceNotFound if no price exists for the given fund and date.
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
