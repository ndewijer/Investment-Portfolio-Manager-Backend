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
	fundRepository      *repository.FundRepository
	developerRepository *repository.DeveloperRepository
	ibkrClient          ibkr.Client
	pfRepo              *repository.PortfolioFundRepository
	transactionRepo     *repository.TransactionRepository
	dividendRepo        *repository.DividendRepository
}

// IbkrServiceOption is a functional option for configuring an IbkrService.
type IbkrServiceOption func(*IbkrService)

// IbkrWithIbkrRepo injects the IbkrRepository dependency.
func IbkrWithIbkrRepo(r *repository.IbkrRepository) IbkrServiceOption {
	return func(s *IbkrService) { s.ibkrRepo = r }
}

// IbkrWithPortfolioRepo injects the PortfolioRepository dependency.
func IbkrWithPortfolioRepo(r *repository.PortfolioRepository) IbkrServiceOption {
	return func(s *IbkrService) { s.portfolioRepo = r }
}

// IbkrWithFundRepo injects the FundRepository dependency.
func IbkrWithFundRepo(r *repository.FundRepository) IbkrServiceOption {
	return func(s *IbkrService) { s.fundRepository = r }
}

// IbkrWithDeveloperRepo injects the DeveloperRepository dependency.
func IbkrWithDeveloperRepo(r *repository.DeveloperRepository) IbkrServiceOption {
	return func(s *IbkrService) { s.developerRepository = r }
}

// IbkrWithClient injects the IBKR API client dependency.
func IbkrWithClient(c ibkr.Client) IbkrServiceOption {
	return func(s *IbkrService) { s.ibkrClient = c }
}

// IbkrWithPortfolioFundRepo injects the PortfolioFundRepository dependency.
func IbkrWithPortfolioFundRepo(r *repository.PortfolioFundRepository) IbkrServiceOption {
	return func(s *IbkrService) { s.pfRepo = r }
}

// IbkrWithTransactionRepo injects the TransactionRepository dependency.
func IbkrWithTransactionRepo(r *repository.TransactionRepository) IbkrServiceOption {
	return func(s *IbkrService) { s.transactionRepo = r }
}

// IbkrWithDividendRepo injects the DividendRepository dependency.
func IbkrWithDividendRepo(r *repository.DividendRepository) IbkrServiceOption {
	return func(s *IbkrService) { s.dividendRepo = r }
}

// NewIbkrService creates a new IbkrService. Pass IbkrWith* options to inject dependencies.
// Only the options relevant to the calling context need to be provided; unset fields remain
// nil and will panic if the corresponding method is called — a clear wiring error.
func NewIbkrService(db *sql.DB, opts ...IbkrServiceOption) *IbkrService {
	s := &IbkrService{db: db}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// GetIbkrConfig retrieves the IBKR integration configuration.
// Adds a token expiration warning if the token expires within 30 days.
func (s *IbkrService) GetIbkrConfig() (*model.IbkrConfig, error) {
	config, err := s.ibkrRepo.GetIbkrConfig()

	if err != nil {
		if errors.Is(err, apperrors.ErrIbkrConfigNotFound) {
			return &model.IbkrConfig{Configured: false}, nil
		}
		return &model.IbkrConfig{}, err
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

	fund, err := s.findFundByISINOrSymbol(transaction.ISIN, transaction.Symbol)
	if errors.Is(err, apperrors.ErrIBKRFundNotMatched) {
		return model.IBKREligiblePortfolioResponse{
			MatchInfo: model.FundMatchInfo{
				Found:     false,
				MatchedBy: "",
			},
			Portfolios: []model.Portfolio{},
			Warning:    fmt.Sprintf("No fund found matching this transaction (Symbol: %s, ISIN: %s). Please add the fund to the system first.", transaction.Symbol, transaction.ISIN),
		}, nil
	}
	if err != nil {
		return model.IBKREligiblePortfolioResponse{}, err
	}

	// Determine how the fund was matched
	matchedBy := "symbol"
	if fund.Isin != "" && fund.Isin == transaction.ISIN {
		matchedBy = "isin"
	}

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
//
//nolint:gocyclo // Primary Flex Report Import orchestrator. Mostly filled with error handling.
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

		req, body, err = s.ibkrClient.RetreiveIbkrFlexReport(ctx, token, config.FlexQueryID)
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
		return 0, 0, fmt.Errorf("ImportFlexReport: failed to update last_import_date: %w", err)
	}

	return len(missingTransactions), len(report) - len(missingTransactions), nil
}

// decryptToken decrypts a fernet-encrypted IBKR flex token using the key in IBKR_ENCRYPTION_KEY.
// Returns an error if the env var is unset, the key is malformed, or decryption fails.
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

// encryptToken encrypts a plaintext IBKR flex token using fernet with the key in IBKR_ENCRYPTION_KEY.
// Returns an error if the env var is unset, the key is malformed, or encryption fails.
func (s *IbkrService) encryptToken(token string) (string, error) {

	enckey := os.Getenv("IBKR_ENCRYPTION_KEY")
	if enckey == "" {
		return "", fmt.Errorf("IBKR_ENCRYPTION_KEY not set")
	}
	key, err := fernet.DecodeKey(enckey)
	if err != nil {
		return "", fmt.Errorf("invalid encryption key: %w", err)
	}
	encryptedToken, err := fernet.EncryptAndSign([]byte(token), key)
	if err != nil {
		return "", err
	}
	if encryptedToken == nil {
		return "", fmt.Errorf("encryption failed")
	}

	return strings.TrimSpace(string(encryptedToken)), nil
}

// writeImportCache persists a Flex report cache entry to the database within a transaction.
// The caller is responsible for populating all fields on importCache before calling.
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

// addExchangeRates upserts a slice of exchange rates within a single transaction.
// Iterates all rates and calls UpdateExchangeRate for each; rolls back on the first failure.
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

// parseIBKRFlexReport converts a raw IBKR Flex report into slices of IBKRTransaction and ExchangeRate models.
// Dates are parsed from IBKR's "20060102" format. Each trade's quantity, net cash, and commission
// are normalised to absolute values. Returns an error if any date or JSON marshal step fails.
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
// Only non-nil fields in the request are applied; omitted fields retain their current values.
// FlexToken is an exception: passing an empty string also means "no change" — only a non-empty
// value overwrites the existing encrypted token.
//
// The table holds exactly one row, so the existing config is fetched and the request is merged
// onto it before persisting. overwriteConfig is set when flex_query_id changes while enabled is
// true; in that case the repository deletes the old row before inserting the new one to avoid a
// stale primary key. Once multiple IBKR accounts are supported this check will need to use the
// config ID directly rather than comparing query IDs.
//
//nolint:gocyclo,funlen // if req.X != nil pattern is intrinsic to patch-style updates in Go; no meaningful split possible
func (s *IbkrService) UpdateIbkrConfig(
	ctx context.Context,
	req request.UpdateIbkrConfigRequest,
) (*model.IbkrConfig, error) {

	var config *model.IbkrConfig
	var err error
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() //nolint:errcheck // Rollback is a no-op after Commit; error is intentionally ignored.

	config, err = s.ibkrRepo.WithTx(tx).GetIbkrConfig()
	if err != nil && !errors.Is(err, apperrors.ErrIbkrConfigNotFound) {
		return nil, err
	} else if errors.Is(err, apperrors.ErrIbkrConfigNotFound) {
		config.ID = uuid.New().String()
		config.CreatedAt = time.Now().UTC()
	}
	var overwriteConfig bool

	if (req.FlexQueryID != nil && *req.FlexQueryID != config.FlexQueryID) && (req.Enabled != nil && *req.Enabled) {
		overwriteConfig = true
	}

	if req.FlexQueryID != nil {
		config.FlexQueryID = *req.FlexQueryID
	}

	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	config.UpdatedAt = time.Now().UTC()

	if !config.Enabled {
		config.AutoImportEnabled = false

		if err := s.ibkrRepo.WithTx(tx).UpdateIbkrConfig(ctx, overwriteConfig, config); err != nil {
			return nil, fmt.Errorf("failed to update IBKR config: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit transaction: %w", err)
		}

		config.Configured = true

		return config, nil
	}

	if req.FlexToken != nil && *req.FlexToken != "" {
		encToken, err := s.encryptToken(*req.FlexToken)
		if err != nil {
			return nil, err
		}
		config.FlexToken = encToken
	}

	if config.Enabled && config.FlexToken == "" {
		return nil, fmt.Errorf("flexToken is required when enabled is true")
	}

	if req.TokenExpiresAt != nil {
		time, err := repository.ParseTime(*req.TokenExpiresAt)
		if err != nil {
			return nil, err
		}
		config.TokenExpiresAt = &time
	}
	if req.AutoImportEnabled != nil {
		config.AutoImportEnabled = *req.AutoImportEnabled
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

	if err := s.ibkrRepo.WithTx(tx).UpdateIbkrConfig(ctx, overwriteConfig, config); err != nil {
		return nil, fmt.Errorf("failed to update IBKR config: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	config.Configured = true

	return config, nil
}

// DeleteIbkrConfig removes the IBKR configuration from the database.
// Returns ErrIbkrConfigNotFound (propagated from the repository) if no config exists.
func (s *IbkrService) DeleteIbkrConfig(ctx context.Context) error {

	err := s.ibkrRepo.DeleteIbkrConfig(ctx)
	if err != nil {
		return err
	}

	return nil
}

// TestIbkrConnection verifies that the provided credentials are accepted by IBKR.
// Unlike other token operations, the token here comes directly from the caller rather
// than from the encrypted config — this is intentional for a pre-save credential check.
// Returns true if IBKR accepts the credentials, or an error if the call fails.
func (s *IbkrService) TestIbkrConnection(ctx context.Context, req request.TestIbkrConnectionRequest) (bool, error) {

	return s.ibkrClient.TestIbkrConnection(ctx, req.FlexToken, req.FlexQueryID)
}

// GetIbkrTransactionDetail retrieves a single IBKR transaction with its allocation details.
// If the transaction is processed, allocations are included in the response.
func (s *IbkrService) GetIbkrTransactionDetail(transactionID string) (model.IBKRTransactionDetail, error) {
	tx, err := s.ibkrRepo.GetIbkrTransaction(transactionID)
	if err != nil {
		return model.IBKRTransactionDetail{}, err
	}

	detail := model.IBKRTransactionDetail{IBKRTransaction: tx}

	if tx.Status == "processed" {
		alloc, err := s.GetTransactionAllocations(transactionID)
		if err != nil {
			return model.IBKRTransactionDetail{}, err
		}
		detail.Allocations = alloc.Allocations
	}

	return detail, nil
}

// DeleteIbkrTransaction removes a pending IBKR transaction.
// Returns ErrIBKRTransactionAlreadyProcessed if the transaction is not pending.
func (s *IbkrService) DeleteIbkrTransaction(ctx context.Context, transactionID string) error {
	tx, err := s.ibkrRepo.GetIbkrTransaction(transactionID)
	if err != nil {
		return err
	}

	if tx.Status != "pending" {
		return apperrors.ErrIBKRTransactionAlreadyProcessed
	}

	return s.ibkrRepo.DeleteIbkrTransaction(ctx, transactionID)
}

// IgnoreIbkrTransaction marks a pending IBKR transaction as ignored.
// Returns ErrIBKRTransactionAlreadyProcessed if the transaction is not pending.
func (s *IbkrService) IgnoreIbkrTransaction(ctx context.Context, transactionID string) error {
	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }() //nolint:errcheck

	ibkrTx, err := s.ibkrRepo.WithTx(dbTx).GetIbkrTransaction(transactionID)
	if err != nil {
		return err
	}

	if ibkrTx.Status != "pending" {
		return apperrors.ErrIBKRTransactionAlreadyProcessed
	}

	if err := s.ibkrRepo.WithTx(dbTx).UpdateIbkrTransactionStatus(ctx, transactionID, "ignored", nil); err != nil {
		return err
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// findFundByISINOrSymbol looks up a fund by ISIN first, then by symbol.
// Returns ErrIBKRFundNotMatched if neither matches.
func (s *IbkrService) findFundByISINOrSymbol(isin, symbol string) (model.Fund, error) {
	fund, err := s.fundRepository.GetFundBySymbolOrIsin("", isin)
	if err != nil && !errors.Is(err, apperrors.ErrFundNotFound) {
		return model.Fund{}, err
	}
	if fund.ID != "" {
		return fund, nil
	}

	fund, err = s.fundRepository.GetFundBySymbolOrIsin(symbol, "")
	if err != nil && !errors.Is(err, apperrors.ErrFundNotFound) {
		return model.Fund{}, err
	}
	if fund.ID != "" {
		return fund, nil
	}

	return model.Fund{}, apperrors.ErrIBKRFundNotMatched
}

// AllocateIbkrTransaction allocates a pending IBKR transaction to portfolios.
// If allocations is empty and default allocation is enabled in config, uses default allocations.
// Creates Transaction and IBKRTransactionAllocation records for each portfolio, including
// separate fee transactions when fees > 0.
//
//nolint:gocyclo // Allocation orchestrator with per-portfolio loop, fee handling, and auto-allocate fallback.
func (s *IbkrService) AllocateIbkrTransaction(ctx context.Context, transactionID string, allocations []request.AllocationEntry) error {
	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }() //nolint:errcheck

	ibkrTx, err := s.ibkrRepo.WithTx(dbTx).GetIbkrTransaction(transactionID)
	if err != nil {
		return err
	}

	if ibkrTx.Status != "pending" {
		return apperrors.ErrIBKRTransactionAlreadyProcessed
	}

	// Auto-allocate fallback
	if len(allocations) == 0 {
		config, err := s.ibkrRepo.WithTx(dbTx).GetIbkrConfig()
		if err != nil && !errors.Is(err, apperrors.ErrIbkrConfigNotFound) {
			return err
		}
		if config != nil && config.DefaultAllocationEnabled && len(config.DefaultAllocations) > 0 {
			allocations = make([]request.AllocationEntry, len(config.DefaultAllocations))
			for i, a := range config.DefaultAllocations {
				allocations[i] = request.AllocationEntry{
					PortfolioID: a.PortfolioID,
					Percentage:  a.Percentage,
				}
			}
		}
	}

	if len(allocations) == 0 {
		return apperrors.ErrIBKRInvalidAllocations
	}

	// Find the fund
	fund, err := s.findFundByISINOrSymbol(ibkrTx.ISIN, ibkrTx.Symbol)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	for _, alloc := range allocations {
		pf, err := s.pfRepo.WithTx(dbTx).GetPortfolioFundByPortfolioAndFund(alloc.PortfolioID, fund.ID)
		if errors.Is(err, apperrors.ErrPortfolioFundNotFound) {
			if err := s.pfRepo.WithTx(dbTx).InsertPortfolioFund(ctx, alloc.PortfolioID, fund.ID); err != nil {
				return fmt.Errorf("failed to create portfolio_fund: %w", err)
			}
			pf, err = s.pfRepo.WithTx(dbTx).GetPortfolioFundByPortfolioAndFund(alloc.PortfolioID, fund.ID)
			if err != nil {
				return fmt.Errorf("failed to retrieve created portfolio_fund: %w", err)
			}
		} else if err != nil {
			return err
		}

		pct := alloc.Percentage / 100.0
		allocatedAmount := round(ibkrTx.TotalAmount * pct)
		allocatedShares := round(ibkrTx.Quantity * pct)
		allocatedFees := round(ibkrTx.Fees * pct)

		txn := &model.Transaction{
			ID:              uuid.New().String(),
			PortfolioFundID: pf.ID,
			Date:            ibkrTx.TransactionDate,
			Type:            ibkrTx.TransactionType,
			Shares:          allocatedShares,
			CostPerShare:    ibkrTx.Price,
			CreatedAt:       now,
		}

		if err := s.transactionRepo.WithTx(dbTx).InsertTransaction(ctx, txn); err != nil {
			return fmt.Errorf("failed to insert transaction: %w", err)
		}

		tradeAlloc := model.IBKRTransactionAllocation{
			ID:                   uuid.New().String(),
			IBKRTransactionID:    transactionID,
			PortfolioID:          alloc.PortfolioID,
			AllocationPercentage: alloc.Percentage,
			AllocatedAmount:      allocatedAmount,
			AllocatedShares:      allocatedShares,
			TransactionID:        txn.ID,
			CreatedAt:            now,
		}
		if err := s.ibkrRepo.WithTx(dbTx).InsertIbkrTransactionAllocation(ctx, tradeAlloc); err != nil {
			return fmt.Errorf("failed to insert trade allocation: %w", err)
		}

		if allocatedFees > 0 {
			feeTxn := &model.Transaction{
				ID:              uuid.New().String(),
				PortfolioFundID: pf.ID,
				Date:            ibkrTx.TransactionDate,
				Type:            "fee",
				Shares:          0,
				CostPerShare:    allocatedFees,
				CreatedAt:       now,
			}
			if err := s.transactionRepo.WithTx(dbTx).InsertTransaction(ctx, feeTxn); err != nil {
				return fmt.Errorf("failed to insert fee transaction: %w", err)
			}

			feeAlloc := model.IBKRTransactionAllocation{
				ID:                   uuid.New().String(),
				IBKRTransactionID:    transactionID,
				PortfolioID:          alloc.PortfolioID,
				AllocationPercentage: alloc.Percentage,
				AllocatedAmount:      allocatedFees,
				AllocatedShares:      0,
				TransactionID:        feeTxn.ID,
				CreatedAt:            now,
			}
			if err := s.ibkrRepo.WithTx(dbTx).InsertIbkrTransactionAllocation(ctx, feeAlloc); err != nil {
				return fmt.Errorf("failed to insert fee allocation: %w", err)
			}
		}
	}

	if err := s.ibkrRepo.WithTx(dbTx).UpdateIbkrTransactionStatus(ctx, transactionID, "processed", &now); err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// BulkAllocateIbkrTransactions allocates multiple IBKR transactions using the same allocation split.
// Each transaction is allocated in its own DB transaction so partial success is possible.
func (s *IbkrService) BulkAllocateIbkrTransactions(ctx context.Context, req request.BulkAllocateRequest) model.BulkAllocateResponse {
	resp := model.BulkAllocateResponse{Errors: []string{}}

	for _, txID := range req.TransactionIds {
		if err := s.AllocateIbkrTransaction(ctx, txID, req.Allocations); err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, fmt.Sprintf("%s: %s", txID, err.Error()))
		} else {
			resp.Success++
		}
	}

	return resp
}

// unallocateIbkrTransactionTx performs unallocation within an existing DB transaction.
// Deletes linked Transaction records and IBKRTransactionAllocation records, then resets
// the IBKR transaction status to "pending".
func (s *IbkrService) unallocateIbkrTransactionTx(ctx context.Context, dbTx *sql.Tx, transactionID string) error {
	ibkrTx, err := s.ibkrRepo.WithTx(dbTx).GetIbkrTransaction(transactionID)
	if err != nil {
		return err
	}

	if ibkrTx.Status != "processed" {
		return apperrors.ErrIBKRTransactionAlreadyProcessed
	}

	// Get linked transaction IDs before deleting allocations
	txIDs, err := s.ibkrRepo.WithTx(dbTx).GetTransactionIDsByIbkrTransaction(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("failed to get linked transaction IDs: %w", err)
	}

	// Delete allocations first (they FK to transactions)
	if err := s.ibkrRepo.WithTx(dbTx).DeleteIbkrTransactionAllocations(ctx, transactionID); err != nil {
		return fmt.Errorf("failed to delete allocations: %w", err)
	}

	// Delete the linked transactions
	for _, txID := range txIDs {
		if err := s.transactionRepo.WithTx(dbTx).DeleteTransaction(ctx, txID); err != nil {
			return fmt.Errorf("failed to delete transaction %s: %w", txID, err)
		}
	}

	if err := s.ibkrRepo.WithTx(dbTx).UpdateIbkrTransactionStatus(ctx, transactionID, "pending", nil); err != nil {
		return fmt.Errorf("failed to reset transaction status: %w", err)
	}

	return nil
}

// UnallocateIbkrTransaction reverses the allocation of a processed IBKR transaction.
// Deletes all linked Transaction and IBKRTransactionAllocation records, then resets
// the status to "pending".
func (s *IbkrService) UnallocateIbkrTransaction(ctx context.Context, transactionID string) error {
	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }() //nolint:errcheck

	if err := s.unallocateIbkrTransactionTx(ctx, dbTx, transactionID); err != nil {
		return err
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// ModifyAllocations atomically unallocates and reallocates an IBKR transaction with new allocation percentages.
// Both operations run in a single DB transaction for atomicity.
func (s *IbkrService) ModifyAllocations(ctx context.Context, transactionID string, allocations []request.AllocationEntry) error {
	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }() //nolint:errcheck

	// Unallocate within this tx
	if err := s.unallocateIbkrTransactionTx(ctx, dbTx, transactionID); err != nil {
		return err
	}

	// Now reallocate within the same tx — we need to inline the allocate logic
	// because AllocateIbkrTransaction opens its own tx.
	if err := s.allocateIbkrTransactionTx(ctx, dbTx, transactionID, allocations); err != nil {
		return err
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// allocateIbkrTransactionTx performs allocation within an existing DB transaction.
// Used by ModifyAllocations to share a tx with unallocate.
//
//nolint:gocyclo // Mirrors AllocateIbkrTransaction but within a provided tx.
func (s *IbkrService) allocateIbkrTransactionTx(ctx context.Context, dbTx *sql.Tx, transactionID string, allocations []request.AllocationEntry) error {
	ibkrTx, err := s.ibkrRepo.WithTx(dbTx).GetIbkrTransaction(transactionID)
	if err != nil {
		return err
	}

	// Auto-allocate fallback
	if len(allocations) == 0 {
		config, err := s.ibkrRepo.WithTx(dbTx).GetIbkrConfig()
		if err != nil && !errors.Is(err, apperrors.ErrIbkrConfigNotFound) {
			return err
		}
		if config != nil && config.DefaultAllocationEnabled && len(config.DefaultAllocations) > 0 {
			allocations = make([]request.AllocationEntry, len(config.DefaultAllocations))
			for i, a := range config.DefaultAllocations {
				allocations[i] = request.AllocationEntry{
					PortfolioID: a.PortfolioID,
					Percentage:  a.Percentage,
				}
			}
		}
	}

	if len(allocations) == 0 {
		return apperrors.ErrIBKRInvalidAllocations
	}

	fund, err := s.findFundByISINOrSymbol(ibkrTx.ISIN, ibkrTx.Symbol)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	for _, alloc := range allocations {
		pf, err := s.pfRepo.WithTx(dbTx).GetPortfolioFundByPortfolioAndFund(alloc.PortfolioID, fund.ID)
		if errors.Is(err, apperrors.ErrPortfolioFundNotFound) {
			if err := s.pfRepo.WithTx(dbTx).InsertPortfolioFund(ctx, alloc.PortfolioID, fund.ID); err != nil {
				return fmt.Errorf("failed to create portfolio_fund: %w", err)
			}
			pf, err = s.pfRepo.WithTx(dbTx).GetPortfolioFundByPortfolioAndFund(alloc.PortfolioID, fund.ID)
			if err != nil {
				return fmt.Errorf("failed to retrieve created portfolio_fund: %w", err)
			}
		} else if err != nil {
			return err
		}

		pct := alloc.Percentage / 100.0
		allocatedAmount := round(ibkrTx.TotalAmount * pct)
		allocatedShares := round(ibkrTx.Quantity * pct)
		allocatedFees := round(ibkrTx.Fees * pct)

		txn := &model.Transaction{
			ID:              uuid.New().String(),
			PortfolioFundID: pf.ID,
			Date:            ibkrTx.TransactionDate,
			Type:            ibkrTx.TransactionType,
			Shares:          allocatedShares,
			CostPerShare:    ibkrTx.Price,
			CreatedAt:       now,
		}
		if err := s.transactionRepo.WithTx(dbTx).InsertTransaction(ctx, txn); err != nil {
			return fmt.Errorf("failed to insert transaction: %w", err)
		}

		tradeAlloc := model.IBKRTransactionAllocation{
			ID:                   uuid.New().String(),
			IBKRTransactionID:    transactionID,
			PortfolioID:          alloc.PortfolioID,
			AllocationPercentage: alloc.Percentage,
			AllocatedAmount:      allocatedAmount,
			AllocatedShares:      allocatedShares,
			TransactionID:        txn.ID,
			CreatedAt:            now,
		}
		if err := s.ibkrRepo.WithTx(dbTx).InsertIbkrTransactionAllocation(ctx, tradeAlloc); err != nil {
			return fmt.Errorf("failed to insert trade allocation: %w", err)
		}

		if allocatedFees > 0 {
			feeTxn := &model.Transaction{
				ID:              uuid.New().String(),
				PortfolioFundID: pf.ID,
				Date:            ibkrTx.TransactionDate,
				Type:            "fee",
				Shares:          0,
				CostPerShare:    allocatedFees,
				CreatedAt:       now,
			}
			if err := s.transactionRepo.WithTx(dbTx).InsertTransaction(ctx, feeTxn); err != nil {
				return fmt.Errorf("failed to insert fee transaction: %w", err)
			}

			feeAlloc := model.IBKRTransactionAllocation{
				ID:                   uuid.New().String(),
				IBKRTransactionID:    transactionID,
				PortfolioID:          alloc.PortfolioID,
				AllocationPercentage: alloc.Percentage,
				AllocatedAmount:      allocatedFees,
				AllocatedShares:      0,
				TransactionID:        feeTxn.ID,
				CreatedAt:            now,
			}
			if err := s.ibkrRepo.WithTx(dbTx).InsertIbkrTransactionAllocation(ctx, feeAlloc); err != nil {
				return fmt.Errorf("failed to insert fee allocation: %w", err)
			}
		}
	}

	if err := s.ibkrRepo.WithTx(dbTx).UpdateIbkrTransactionStatus(ctx, transactionID, "processed", &now); err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	return nil
}

// MatchDividend links a processed IBKR transaction (DRIP) to pending dividend records.
// The IBKR transaction must be allocated first (status "processed") so that Transaction records
// exist. Each dividend's reinvestment_transaction_id is set to the allocation's transaction_id.
//
//nolint:gocyclo // Dividend matching with per-dividend portfolio lookup and validation.
func (s *IbkrService) MatchDividend(ctx context.Context, transactionID string, dividendIds []string) error {
	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }() //nolint:errcheck

	ibkrTx, err := s.ibkrRepo.WithTx(dbTx).GetIbkrTransaction(transactionID)
	if err != nil {
		return err
	}

	if ibkrTx.Status != "processed" {
		return fmt.Errorf("%w: transaction must be allocated before matching dividends", apperrors.ErrIBKRTransactionAlreadyProcessed)
	}

	if !strings.Contains(ibkrTx.Notes, "R") {
		log.Printf("warning: IBKR transaction %s notes field does not contain 'R' (DRIP code), notes: %q", transactionID, ibkrTx.Notes)
	}

	allocationDetails, err := s.ibkrRepo.WithTx(dbTx).GetIbkrTransactionAllocations(transactionID)
	if err != nil {
		return fmt.Errorf("failed to get allocations: %w", err)
	}

	portfolioToTxID := make(map[string]string)
	for _, a := range allocationDetails {
		if a.Type != "fee" && a.TransactionID != "" {
			portfolioToTxID[a.PortfolioID] = a.TransactionID
		}
	}

	for _, dividendID := range dividendIds {
		dividend, err := s.dividendRepo.WithTx(dbTx).GetDividend(dividendID)
		if err != nil {
			return fmt.Errorf("failed to get dividend %s: %w", dividendID, err)
		}

		if dividend.ReinvestmentTransactionID != "" {
			return fmt.Errorf("dividend %s already matched to transaction %s", dividendID, dividend.ReinvestmentTransactionID)
		}

		pf, err := s.pfRepo.WithTx(dbTx).GetPortfolioFund(dividend.PortfolioFundID)
		if err != nil {
			return fmt.Errorf("failed to get portfolio_fund for dividend %s: %w", dividendID, err)
		}

		allocTxID, ok := portfolioToTxID[pf.PortfolioID]
		if !ok {
			return fmt.Errorf("no allocation found for portfolio %s (dividend %s)", pf.PortfolioID, dividendID)
		}

		dividend.ReinvestmentTransactionID = allocTxID
		dividend.BuyOrderDate = ibkrTx.TransactionDate
		dividend.ReinvestmentStatus = "COMPLETED"

		if err := s.dividendRepo.WithTx(dbTx).UpdateDividend(ctx, &dividend); err != nil {
			return fmt.Errorf("failed to update dividend %s: %w", dividendID, err)
		}
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
