package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/fernet/fernet-go"
	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/ibkr"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// IbkrService handles IBKR (Interactive Brokers) integration business logic operations.
type IbkrService struct {
	db                  *sql.DB
	ibkrRepo            *repository.IbkrRepository
	portfolioRepo       *repository.PortfolioRepository
	transactionService  *TransactionService
	fundRepository      *repository.FundRepository
	developerRepository *repository.DeveloperRepository
	ibkrClient          ibkr.Client
}

// NewIbkrService creates a new IbkrService with the provided repository dependencies.
func NewIbkrService(
	db *sql.DB, ibkrRepo *repository.IbkrRepository, portfolioRepo *repository.PortfolioRepository, transactionService *TransactionService, fundRepository *repository.FundRepository, developerRepository *repository.DeveloperRepository, ibkrClient ibkr.Client,
) *IbkrService {
	return &IbkrService{
		db:                  db,
		ibkrRepo:            ibkrRepo,
		portfolioRepo:       portfolioRepo,
		transactionService:  transactionService,
		fundRepository:      fundRepository,
		developerRepository: developerRepository,
		ibkrClient:          ibkrClient,
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

	if config.TokenExpiresAt != nil && !config.TokenExpiresAt.IsZero() {
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

// GetInboxCount retrieves the count of IBKR imported transactions with status "pending".
// Returns only the count without fetching full transaction records for efficiency.
func (s *IbkrService) GetInboxCount() (model.IBKRInboxCount, error) {
	return s.ibkrRepo.GetIbkrInboxCount()
}

// GetTransactionAllocations retrieves the allocation details for an IBKR transaction.
// Fetches the transaction and its allocations, then processes and aggregates the data:
//   - Separates fee allocations from trade allocations
//   - Aggregates fees by portfolio ID and includes them in AllocatedCommission
//   - Rounds monetary values to standard precision
//   - Filters out fee transactions from the final response
//
// Parameters:
//   - transactionID: The UUID of the IBKR transaction
//
// Returns the transaction allocation summary with portfolio-level details,
// or an error if the transaction is not found or a database error occurs.
func (s *IbkrService) GetTransactionAllocations(transactionID string) (model.IBKRAllocation, error) {

	ibkrTransaction, err := s.ibkrRepo.GetIbkrTransaction(transactionID)
	if err != nil {
		return model.IBKRAllocation{}, err
	}

	allocationDetails, err := s.ibkrRepo.GetIbkrTransactionAllocations(transactionID)
	if err != nil {
		return model.IBKRAllocation{}, err
	}

	feesByID := make(map[string]float64)
	for _, allocation := range allocationDetails {
		if allocation.Type == "fee" {
			feesByID[allocation.PortfolioID] += allocation.AllocatedAmount
		}
	}

	allocationDetailsResponse := make([]model.IBKRTransactionAllocationResponse, 0, len(allocationDetails))

	for _, allocation := range allocationDetails {
		if allocation.Type == "fee" {
			continue
		}

		allocationDetailsResponse = append(allocationDetailsResponse, model.IBKRTransactionAllocationResponse{
			PortfolioID:          allocation.PortfolioID,
			PortfolioName:        allocation.PortfolioName,
			AllocationPercentage: allocation.AllocationPercentage,
			AllocatedAmount:      round(allocation.AllocatedAmount),
			AllocatedShares:      round(allocation.AllocatedShares),
			AllocatedCommission:  round(feesByID[allocation.PortfolioID]),
		})
	}

	allocationReturn := model.IBKRAllocation{
		IBKRTransactionID: ibkrTransaction.ID,
		Status:            ibkrTransaction.Status,
		Allocations:       allocationDetailsResponse,
	}

	return allocationReturn, nil
}

// GetEligiblePortfolios finds portfolios eligible for allocating an IBKR transaction.
// Matches the transaction's fund using a two-step process:
//  1. First attempts to match by ISIN (most reliable identifier)
//  2. If ISIN match fails, attempts to match by symbol
//
// Once the fund is found, retrieves all portfolios that hold this fund.
// If the fund exists but is not assigned to any portfolios, a warning is included.
//
//   - Returns 200 OK with found=false if no fund match (not an error)
//   - Uses nested match_info structure for compatibility
//
// Parameters:
//   - transactionID: The UUID of the IBKR transaction
//
// Returns:
//   - Response with matchInfo, portfolios, and optional warning
//   - Only returns error for database failures, not for "fund not found"
//   - ErrIBKRTransactionNotFound if the transaction doesn't exist
func (s *IbkrService) GetEligiblePortfolios(transactionID string) (model.IBKREligiblePortfolioResponse, error) {
	transaction, err := s.ibkrRepo.GetIbkrTransaction(transactionID)
	if err != nil {
		return model.IBKREligiblePortfolioResponse{}, err
	}

	// First we try to find the fund on ISIN as it's most reliable
	fund, err := s.fundRepository.GetFundBySymbolOrIsin("", transaction.ISIN)
	if err != nil && !errors.Is(err, apperrors.ErrFundNotFound) {
		return model.IBKREligiblePortfolioResponse{}, err
	}

	matchedBy := ""
	if fund.ID == "" {
		// Second, we try on Symbol.
		fund, err = s.fundRepository.GetFundBySymbolOrIsin(transaction.Symbol, "")
		if err != nil && !errors.Is(err, apperrors.ErrFundNotFound) {
			return model.IBKREligiblePortfolioResponse{}, err
		}
		if fund.ID == "" {
			// No fund found - return success response with found=false
			return model.IBKREligiblePortfolioResponse{
				MatchInfo: model.FundMatchInfo{
					Found:     false,
					MatchedBy: "",
				},
				Portfolios: []model.Portfolio{},
				Warning:    fmt.Sprintf("No fund found matching this transaction (Symbol: %s, ISIN: %s). Please add the fund to the system first.", transaction.Symbol, transaction.ISIN),
			}, nil
		}
		matchedBy = "symbol"
	} else {
		matchedBy = "isin"
	}

	// Fund was found - get portfolios
	portfolios, err := s.portfolioRepo.GetPortfoliosByFundID(fund.ID)
	if err != nil {
		return model.IBKREligiblePortfolioResponse{}, err
	}

	warning := ""
	if len(portfolios) == 0 {
		warning = fmt.Sprintf("Fund '%s' (%s) exists but is not assigned to any portfolio. Please add this fund to a portfolio first.", fund.Name, fund.Symbol)
	}

	return model.IBKREligiblePortfolioResponse{
		MatchInfo: model.FundMatchInfo{
			Found:      true,
			MatchedBy:  matchedBy,
			FundID:     fund.ID,
			FundName:   fund.Name,
			FundSymbol: fund.Symbol,
			FundISIN:   fund.Isin,
		},
		Portfolios: portfolios,
		Warning:    warning,
	}, nil
}

// ImportFlexReport fetches and processes an IBKR Flex statement.
// Checks the local cache first and only calls the IBKR API if the cache is missing or expired.
// New transactions are compared against existing records and only new ones are inserted.
// Updates the last import date on the config after a successful run.
// Returns the number of imported and skipped transactions, or an error if the import fails.
func (s *IbkrService) ImportFlexReport(ctx context.Context) (int, int, error) {

	config, err := s.GetIbkrConfig()
	if err != nil {
		return 0, 0, err
	}

	cache, err := s.ibkrRepo.GetIbkrImportCache()
	if err != nil {
		// No cache is fine, error on the rest.
		if !errors.Is(err, apperrors.ErrIbkrImportCacheNotFound) {
			return 0, 0, err
		}
	}
	var req ibkr.FlexQueryResponse
	var body []byte
	var cacheSet bool

	if cache.ExpiresAt.IsZero() || cache.ExpiresAt.Before(time.Now().UTC()) {
		token, err := s.decryptToken(config.FlexToken)
		if err != nil {
			return 0, 0, err
		}

		req, body, err = s.ibkrClient.RequestIBKRFlexReport(ctx, token, config.FlexQueryID)
		if err != nil {
			return 0, 0, err
		}
	} else {
		cacheSet = true
		err := xml.Unmarshal(cache.Data, &req)
		if err != nil {
			return 0, 0, err
		}
	}

	now := time.Now().UTC()

	if len(body) > 0 && !cacheSet {
		cacheKey := fmt.Sprintf("ibkr_flex_%d_%s", req.QueryID, now.Truncate(24*time.Hour).Format("2006-01-02"))
		importCache := model.IbkrImportCache{
			ID:        uuid.New().String(),
			CacheKey:  cacheKey,
			Data:      body,
			CreatedAt: now,
			ExpiresAt: now.Add(time.Hour),
		}
		if err := s.writeImportCache(ctx, importCache); err != nil {
			return 0, 0, err
		}
	}

	report, rates, err := s.parseIBKRFlexReport(req)
	if err != nil {
		return 0, 0, err
	}

	missingTransactions := []model.IBKRTransaction{}

	for _, v := range report {
		if !s.ibkrRepo.CompareIbkrTransaction(v) {
			missingTransactions = append(missingTransactions, v)
		}
	}

	if len(missingTransactions) > 0 {
		if err := s.AddIbkrTransactions(ctx, missingTransactions); err != nil {
			return 0, 0, err
		}
	}

	if len(rates) > 0 {
		if err := s.addExchangeRates(ctx, rates); err != nil {
			return 0, 0, err
		}
	}

	if err := s.ibkrRepo.UpdateLastImportDate(ctx, config.FlexQueryID, now); err != nil {
		log.Printf("ImportFlexReport: failed to update last_import_date: %v", err)
	}

	return len(missingTransactions), len(report) - len(missingTransactions), nil
}

func (s *IbkrService) decryptToken(token string) (string, error) {

	enckey := os.Getenv("IBKR_ENCRYPTION_KEY")
	if enckey == "" {
		return "", fmt.Errorf("IBKR_ENCRYPTION_KEY not set")
	}
	key, err := fernet.DecodeKey(enckey)
	if err != nil {
		return "", fmt.Errorf("invalid encryption key: %w", err)
	}
	decryptedToken := fernet.VerifyAndDecrypt([]byte(token), 0, []*fernet.Key{key})
	if decryptedToken == nil {
		return "", fmt.Errorf("decryption failed")
	}

	return strings.TrimSpace(string(decryptedToken)), nil
}

func (s *IbkrService) writeImportCache(ctx context.Context, importCache model.IbkrImportCache) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck

	if err := s.ibkrRepo.WithTx(tx).WriteImportCache(ctx, importCache); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (s *IbkrService) addExchangeRates(ctx context.Context, rates []model.ExchangeRate) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck

	for _, v := range rates {

		if err = s.developerRepository.WithTx(tx).UpdateExchangeRate(ctx, v); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// AddIbkrTransactions persists a slice of IBKR transactions to the database within a single transaction.
// Returns an error if the transaction cannot be started or if any insert fails.
func (s *IbkrService) AddIbkrTransactions(ctx context.Context, transactions []model.IBKRTransaction) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck

	if err = s.ibkrRepo.WithTx(tx).AddIbkrTransactions(ctx, transactions); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (s *IbkrService) parseIBKRFlexReport(report ibkr.FlexQueryResponse) ([]model.IBKRTransaction, []model.ExchangeRate, error) {

	ibkrTransactions := make([]model.IBKRTransaction, len(report.FlexStatements.FlexStatement.Trades.Trade))
	ibkrExchangeRate := make([]model.ExchangeRate, len(report.FlexStatements.FlexStatement.ConversionRates.ConversionRate))
	for i, v := range report.FlexStatements.FlexStatement.Trades.Trade {
		transactionDate, err := time.Parse("20060102", v.TradeDate)
		if err != nil {
			return nil, nil, err
		}

		reportDate, err := time.Parse("20060102", v.ReportDate)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse reportDate for transaction %d: %w", v.TransactionID, err)
		}

		rawBytes, err := json.Marshal(v)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal raw transaction data: %w", err)
		}

		currency := v.CurrencyPrimary
		if currency == "" {
			currency = v.Currency
		}

		t := model.IBKRTransaction{
			ID:                uuid.New().String(),
			IBKRTransactionID: fmt.Sprintf("%d_%d", v.TransactionID, v.IbOrderID),
			TransactionDate:   transactionDate,
			Symbol:            v.Symbol,
			ISIN:              v.Isin,
			Description:       v.Description,
			TransactionType:   strings.ToLower(v.BuySell),
			Quantity:          math.Abs(v.Quantity),
			Price:             v.TradePrice,
			TotalAmount:       math.Abs(v.NetCash),
			Currency:          currency,
			Fees:              math.Abs(v.IbCommission),
			Status:            "pending",
			ImportedAt:        report.ImportedAt,
			RawData:           rawBytes,
			Notes:             v.Notes,
			ReportDate:        reportDate,
		}

		ibkrTransactions[i] = t
	}

	for i, v := range report.FlexStatements.FlexStatement.ConversionRates.ConversionRate {
		reportDate, err := time.Parse("20060102", v.ReportDate)
		if err != nil {
			return nil, nil, err
		}

		rate := model.ExchangeRate{
			ID:           uuid.New().String(),
			Date:         reportDate,
			FromCurrency: v.FromCurrency,
			ToCurrency:   v.ToCurrency,
			Rate:         v.Rate,
		}
		ibkrExchangeRate[i] = rate
	}

	return ibkrTransactions, ibkrExchangeRate, nil
}

// UpdateIbkrConfig applies a partial update to the IBKR configuration.
// Only fields present in the request (non-nil pointers) are applied; unset fields are ignored.
// The flexToken parameter is the flex_query_id used to identify the config row to update.
//
//nolint:gocyclo // if req.X != nil pattern is intrinsic to patch-style updates in Go
func (s *IbkrService) UpdateIbkrConfig(
	ctx context.Context,
	flexToken int,
	req request.UpdateIbkrConfigRequest,
) (*model.IbkrConfig, error) {

	config := model.IbkrConfig{}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	// portfolio, err := s.portfolioRepo.WithTx(tx).GetPortfolioOnID(id)
	// if err != nil {
	// 	return nil, err
	// }

	if req.Configured != nil {
		config.Configured = *req.Configured
	}
	if req.FlexToken != nil {
		config.FlexToken = *req.FlexToken
	}
	if req.FlexQueryID != nil {
		config.FlexQueryID = *req.FlexQueryID
	}
	if req.TokenExpiresAt != nil {
		config.TokenExpiresAt = req.TokenExpiresAt
	}
	if req.LastImportDate != nil {
		config.LastImportDate = req.LastImportDate
	}
	if req.AutoImportEnabled != nil {
		config.AutoImportEnabled = *req.AutoImportEnabled
	}
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.DefaultAllocationEnabled != nil {
		config.DefaultAllocationEnabled = *req.DefaultAllocationEnabled
	}
	if req.DefaultAllocations != nil {
		all := make([]model.Allocation, len(req.DefaultAllocations))
		for i, v := range req.DefaultAllocations {
			all[i].Percentage = *v.Percentage
			all[i].PortfolioID = *v.PortfolioID
		}
		config.DefaultAllocations = all
	}
	if req.CreatedAt != nil {
		config.CreatedAt = *req.CreatedAt
	}
	if req.UpdatedAt != nil {
		config.UpdatedAt = *req.UpdatedAt
	}

	if err := s.ibkrRepo.WithTx(tx).UpdateIbkrConfig(ctx, flexToken, &config); err != nil {
		return nil, fmt.Errorf("failed to update IBKR config: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &config, nil
}
