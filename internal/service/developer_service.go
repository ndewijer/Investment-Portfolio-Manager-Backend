package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
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
	db              *sql.DB
	developerRepo   *repository.DeveloperRepository
	fundRepo        *repository.FundRepository
	transactionRepo *repository.TransactionRepository
	pfRepo          *repository.PortfolioFundRepository
}

// NewDeveloperService creates a new DeveloperService with the provided repository dependencies.
func NewDeveloperService(
	db *sql.DB,
	developerRepo *repository.DeveloperRepository,
	fundRepo *repository.FundRepository,
	transactionRepo *repository.TransactionRepository,
	pfRepo *repository.PortfolioFundRepository,
) *DeveloperService {
	return &DeveloperService{
		db:              db,
		developerRepo:   developerRepo,
		fundRepo:        fundRepo,
		transactionRepo: transactionRepo,
		pfRepo:          pfRepo,
	}
}

// GetLogs retrieves system logs with the specified filters and pagination.
// Returns a paginated response with cursor for fetching subsequent pages.
// The context parameter is currently unused but reserved for future cancellation support.
func (s *DeveloperService) GetLogs(_ context.Context, filters *model.LogFilters) (*model.LogResponse, error) {
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

// SetLoggingConfig updates the logging configuration in system settings.
// Upserts LOGGING_ENABLED and LOGGING_LEVEL settings within a single transaction.
// Only updates fields that are present in the request.
func (s *DeveloperService) SetLoggingConfig(ctx context.Context, req request.SetLoggingConfig) (model.LoggingSetting, error) {

	var logSetting model.LoggingSetting
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.LoggingSetting{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.
	time := time.Now().UTC()
	if req.Enabled != nil {
		settingEnabled := model.SystemSetting{
			ID:        uuid.New().String(),
			Key:       "LOGGING_ENABLED",
			Value:     req.Enabled,
			UpdatedAt: &time,
		}

		if err := s.developerRepo.WithTx(tx).SetLoggingConfig(ctx, settingEnabled); err != nil {
			return model.LoggingSetting{}, fmt.Errorf("failed to update LOGGING_ENABLED: %w", err)
		}

		logSetting.Enabled = *req.Enabled
	}
	if req.Level != "" {
		settingLevel := model.SystemSetting{
			ID:        uuid.New().String(),
			Key:       "LOGGING_LEVEL",
			Value:     req.Level,
			UpdatedAt: &time,
		}
		if err := s.developerRepo.WithTx(tx).SetLoggingConfig(ctx, settingLevel); err != nil {
			return model.LoggingSetting{}, fmt.Errorf("failed to update LOGGING_LEVEL: %w", err)
		}

		logSetting.Level = req.Level
	}

	if err = tx.Commit(); err != nil {
		return model.LoggingSetting{}, fmt.Errorf("commit transaction: %w", err)
	}

	return logSetting, nil
}

// GetExchangeRate retrieves the exchange rate for a specific currency pair and date.
// Returns ErrExchangeRateNotFound if no rate exists for the given parameters.
func (s *DeveloperService) GetExchangeRate(fromCurrency, toCurrency string, dateTime time.Time) (*model.ExchangeRate, error) {

	return s.developerRepo.GetExchangeRate(fromCurrency, toCurrency, dateTime)
}

// UpdateExchangeRate creates or updates an exchange rate for a given currency pair and date.
// Parses the rate from the request string and upserts the record within a transaction.
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

// UpdateFundPrice creates or updates a fund price for a given fund and date.
// Parses the price from the request string and upserts the record within a transaction.
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

// DeleteLogs clears all log entries and records a single "logs cleared" audit entry.
// Both the deletion and the new audit log are performed within a single transaction.
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

// ImportFundPrices parses a CSV file and upserts fund prices for the given fund.
// Validates that the fund exists, the file is valid CSV with required headers,
// and each row has a parseable date and positive price.
func (s *DeveloperService) ImportFundPrices(ctx context.Context, fundID string, content []byte) (int, error) {
	// Validate fund exists
	if _, err := s.fundRepo.GetFund(fundID); err != nil {
		return 0, apperrors.ErrFundNotFound
	}

	headers, rows, err := parseCSV(content)
	if err != nil {
		return 0, err
	}
	if err := validateCSVHeaders(headers, []string{"date", "price"}); err != nil {
		return 0, err
	}

	// Build column index map
	colIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		colIdx[h] = i
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck

	count := 0
	for i, row := range rows {
		rowNum := i + 2 // 1-indexed, row 1 is headers

		date, err := time.Parse("2006-01-02", strings.TrimSpace(row[colIdx["date"]]))
		if err != nil {
			return 0, fmt.Errorf("row %d: invalid date %q: %w", rowNum, row[colIdx["date"]], err)
		}

		price, err := strconv.ParseFloat(strings.TrimSpace(row[colIdx["price"]]), 64)
		if err != nil || price <= 0 {
			return 0, fmt.Errorf("row %d: price must be a positive number, got %q", rowNum, row[colIdx["price"]])
		}

		fp := model.FundPrice{
			ID:     uuid.NewString(),
			FundID: fundID,
			Date:   date,
			Price:  price,
		}
		if err := s.fundRepo.WithTx(tx).UpdateFundPrice(ctx, fp); err != nil {
			return 0, fmt.Errorf("row %d: failed to upsert fund price: %w", rowNum, err)
		}
		count++
	}

	if count == 0 {
		return 0, fmt.Errorf("no data rows found in CSV")
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	return count, nil
}

// transactionRow holds validated fields parsed from a single CSV transaction row.
type transactionRow struct {
	date         time.Time
	txType       string
	shares       float64
	costPerShare float64
}

// validateTransactionRow parses and validates a single CSV data row for transaction imports.
// Returns a transactionRow with the parsed values, or an error describing the invalid field.
func validateTransactionRow(row []string, colIdx map[string]int, rowNum int) (transactionRow, error) {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(row[colIdx["date"]]))
	if err != nil {
		return transactionRow{}, fmt.Errorf("row %d: invalid date %q: %w", rowNum, row[colIdx["date"]], err)
	}

	txType := strings.TrimSpace(strings.ToLower(row[colIdx["type"]]))
	if txType == "" {
		return transactionRow{}, fmt.Errorf("row %d: type is required", rowNum)
	}
	if !model.ValidTransactionTypes[model.TransactionType(txType)] {
		return transactionRow{}, fmt.Errorf("row %d: type must be one of buy, sell, dividend, fee; got %q", rowNum, txType)
	}

	shares, err := strconv.ParseFloat(strings.TrimSpace(row[colIdx["shares"]]), 64)
	if err != nil || shares <= 0 {
		return transactionRow{}, fmt.Errorf("row %d: shares must be a positive number, got %q", rowNum, row[colIdx["shares"]])
	}

	costPerShare, err := strconv.ParseFloat(strings.TrimSpace(row[colIdx["cost_per_share"]]), 64)
	if err != nil || costPerShare < 0 {
		return transactionRow{}, fmt.Errorf("row %d: cost_per_share must be a non-negative number, got %q", rowNum, row[colIdx["cost_per_share"]])
	}

	return transactionRow{
		date:         date,
		txType:       txType,
		shares:       shares,
		costPerShare: costPerShare,
	}, nil
}

// ImportTransactions parses a CSV file and inserts transactions for the given portfolio-fund.
// Validates that the portfolio-fund relationship exists, the file is valid CSV with required
// headers, and each row has valid values.
func (s *DeveloperService) ImportTransactions(ctx context.Context, portfolioFundID string, content []byte) (int, error) {
	// Validate portfolio-fund exists
	if _, err := s.pfRepo.GetPortfolioFund(portfolioFundID); err != nil {
		return 0, apperrors.ErrPortfolioFundNotFound
	}

	headers, rows, err := parseCSV(content)
	if err != nil {
		return 0, err
	}
	if err := validateCSVHeaders(headers, []string{"date", "type", "shares", "cost_per_share"}); err != nil {
		return 0, err
	}

	colIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		colIdx[h] = i
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck

	count := 0
	for i, row := range rows {
		rowNum := i + 2

		parsed, err := validateTransactionRow(row, colIdx, rowNum)
		if err != nil {
			return 0, err
		}

		t := &model.Transaction{
			ID:              uuid.NewString(),
			PortfolioFundID: portfolioFundID,
			Date:            parsed.date,
			Type:            parsed.txType,
			Shares:          parsed.shares,
			CostPerShare:    parsed.costPerShare,
			CreatedAt:       time.Now().UTC(),
		}
		if err := s.transactionRepo.WithTx(tx).InsertTransaction(ctx, t); err != nil {
			return 0, fmt.Errorf("row %d: failed to insert transaction: %w", rowNum, err)
		}
		count++
	}

	if count == 0 {
		return 0, fmt.Errorf("no data rows found in CSV")
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	return count, nil
}

// parseCSV strips a UTF-8 BOM if present and returns parsed CSV records.
// The first returned slice is the header row; the rest are data rows.
func parseCSV(content []byte) ([]string, [][]string, error) {
	// Strip UTF-8 BOM (EF BB BF) to avoid header corruption
	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})

	r := csv.NewReader(bytes.NewReader(content))
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("invalid CSV format: %w", err)
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("CSV file is empty")
	}

	headers := make([]string, len(records[0]))
	for i, h := range records[0] {
		headers[i] = strings.TrimSpace(strings.ToLower(h))
	}

	return headers, records[1:], nil
}

// validateCSVHeaders checks that all required headers are present.
func validateCSVHeaders(headers []string, required []string) error {
	headerSet := make(map[string]bool, len(headers))
	for _, h := range headers {
		headerSet[h] = true
	}
	var missing []string
	for _, r := range required {
		if !headerSet[r] {
			missing = append(missing, r)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required CSV columns: %s", strings.Join(missing, ", "))
	}
	return nil
}
