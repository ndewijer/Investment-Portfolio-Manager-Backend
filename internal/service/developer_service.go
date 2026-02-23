package service

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// DeveloperService handles Developer-related business logic operations.
type DeveloperService struct {
	db            *sql.DB
	developerRepo *repository.DeveloperRepository
	fundRepo      *repository.FundRepository
}

// NewDeveloperService creates a new DeveloperService with the provided repository dependencies.
func NewDeveloperService(
	db *sql.DB,
	developerRepo *repository.DeveloperRepository,
	fundRepo *repository.FundRepository,
) *DeveloperService {
	return &DeveloperService{
		db:            db,
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

func (s *DeveloperService) UpdateExchangeRate(
	ctx context.Context,
	req request.SetExchangeRateRequest,
) (model.ExchangeRate, error) {

	var exRate model.ExchangeRate
	var err error

	exRate.Date, err = time.Parse("2006-01-02", req.Date)
	if err != nil {
		return model.ExchangeRate{}, err
	}
	exRate.ID = uuid.New().String()
	exRate.ToCurrency = req.ToCurrency
	exRate.FromCurrency = req.FromCurrency

	rateFloat, err := strconv.ParseFloat(strings.TrimSpace(req.Rate), 64)
	if err != nil {
		return model.ExchangeRate{}, err
	}
	exRate.Rate = rateFloat

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.ExchangeRate{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	if err := s.developerRepo.WithTx(tx).UpdateExchangeRate(ctx, exRate); err != nil {
		return model.ExchangeRate{}, fmt.Errorf("failed to update exchange rate: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return model.ExchangeRate{}, fmt.Errorf("commit transaction: %w", err)
	}

	return exRate, nil
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

func (s *DeveloperService) UpdateFundPrice(
	ctx context.Context,
	req request.SetFundPriceRequest,
) (model.FundPrice, error) {

	var fp model.FundPrice
	var err error

	fp.Date, err = time.Parse("2006-01-02", req.Date)
	if err != nil {
		return model.FundPrice{}, err
	}
	priceFloat, err := strconv.ParseFloat(strings.TrimSpace(req.Price), 64)
	if err != nil {
		return model.FundPrice{}, err
	}
	fp.Price = priceFloat
	fp.FundID = req.FundID

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.FundPrice{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	fp.ID = uuid.New().String()

	if err := s.fundRepo.WithTx(tx).UpdateFundPrice(ctx, fp); err != nil {
		return model.FundPrice{}, fmt.Errorf("failed to update fund price: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return model.FundPrice{}, fmt.Errorf("commit transaction: %w", err)
	}

	return fp, nil
}

func (s *DeveloperService) DeleteLogs(ctx context.Context, ipAddress any, userAgent string) error {

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	if err := s.developerRepo.WithTx(tx).DeleteLogs(ctx); err != nil {
		return fmt.Errorf("failed to delete logs: %w", err)
	}
	resetLog := model.Log{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Level:     "INFO",
		Category:  "DEVELOPER",
		Message:   "All logs cleared by user",
		Source:    "DeleteLogs",
		UserAgent: userAgent,
	}

	if ip, ok := ipAddress.(string); ok {
		resetLog.IPAddress = ip
	}
	if err := s.developerRepo.WithTx(tx).AddLog(ctx, resetLog); err != nil {
		return fmt.Errorf("failed to add log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
