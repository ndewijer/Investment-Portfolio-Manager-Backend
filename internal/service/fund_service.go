package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/yahoo"
)

// FundService handles fund-related business logic operations.
type FundService struct {
	fundRepo                *repository.FundRepository
	transactionService      *TransactionService
	dividendService         *DividendService
	realizedGainLossService *RealizedGainLossService
	dataLoaderService       *DataLoaderService
	portfolioService        *PortfolioService
	yahooClient             yahoo.Client
}

// NewFundService creates a new FundService with the provided repository dependencies.
func NewFundService(
	fundRepo *repository.FundRepository,
	transactionService *TransactionService,
	dividendService *DividendService,
	realizedGainLossService *RealizedGainLossService,
	dataLoaderService *DataLoaderService,
	portfolioService *PortfolioService,
	yahooClient yahoo.Client,
) *FundService {
	return &FundService{
		fundRepo:                fundRepo,
		transactionService:      transactionService,
		dividendService:         dividendService,
		realizedGainLossService: realizedGainLossService,
		dataLoaderService:       dataLoaderService,
		portfolioService:        portfolioService,
		yahooClient:             yahooClient,
	}
}

// GetFund retrieves fund from the database.
// Returns fund metadata including latest prices.
func (s *FundService) GetFund(fundID string) (model.Fund, error) {
	return s.fundRepo.GetFund(fundID)
}

// GetAllFunds retrieves all funds from the database.
// Returns fund metadata including latest prices for all funds in the system.
func (s *FundService) GetAllFunds() ([]model.Fund, error) {
	return s.fundRepo.GetAllFunds()
}

// GetSymbol retrieves symbol information by ticker symbol.
// Returns symbol metadata including name, exchange, currency, and ISIN.
func (s *FundService) GetSymbol(symbol string) (*model.Symbol, error) {
	return s.fundRepo.GetSymbol(symbol)
}

// GetAllPortfolioFundListings retrieves all portfolio-fund relationships with basic metadata.
// Returns a listing of funds across all portfolios (non-archived) with portfolio and fund names.
// Used for the GET /api/portfolio/funds endpoint.
func (s *FundService) GetAllPortfolioFundListings() ([]model.PortfolioFundListing, error) {
	return s.fundRepo.GetAllPortfolioFundListings()
}

// GetPortfolioFunds retrieves detailed fund metrics for all funds in a portfolio.
// This method orchestrates the complete calculation pipeline to produce enriched fund data
// with current valuations, gains/losses, dividends, and fees.
//
// Calculation Pipeline:
//  1. Resolves portfolio(s) from the ID parameter (specific portfolio or all active)
//  2. Batch-loads all required data (transactions, dividends, prices, realized gains)
//  3. Maps realized gains from portfolio level to fund level
//  4. Enriches each fund with calculated metrics using historical data
//
// The actual calculations (share counts, cost basis, valuations) are delegated to
// enrichPortfolioFundsWithMetrics and its helpers. This method focuses on orchestration
// and data loading.
//
// Parameters:
//   - portfolioID: The portfolio ID to retrieve funds for. If empty, returns all active portfolios.
//
// Returns a slice of PortfolioFund structs with all metric fields populated:
// TotalShares, LatestPrice, AverageCost, TotalCost, CurrentValue, UnrealizedGainLoss,
// RealizedGainLoss, TotalGainLoss, TotalDividends, and TotalFees.
// All monetary values are rounded to two decimal places.
func (s *FundService) GetPortfolioFunds(portfolioID string) ([]model.PortfolioFundResponse, error) {

	portfolio, err := s.portfolioService.GetPortfoliosForRequest(portfolioID)
	if err != nil {
		return nil, err
	}

	data, err := s.dataLoaderService.LoadForPortfolios(portfolio, time.Time{}, time.Now())
	if err != nil {
		return nil, err
	}

	if len(data.PFIDs) == 0 {
		return []model.PortfolioFundResponse{}, nil
	}

	realizedGainsByPF := data.MapRealizedGainsByPF(portfolioID)

	return s.enrichPortfolioFundsWithMetrics(data, realizedGainsByPF)
}

// loadFundPrices retrieves fund prices for the given fund IDs within the specified date range.
// Prices are sorted by date based on the ascending parameter (true=ASC, false=DESC).
// Results are grouped by fund ID, allowing per-fund price lookups.
//
// Parameters:
//   - fundIDs: Slice of fund IDs to retrieve prices for
//   - startDate: Inclusive start date for the price range
//   - endDate: Inclusive end date for the price range
//   - ascending: If true, sort prices oldest-first (ASC); if false, newest-first (DESC)
//
// Returns a map of fundID -> []FundPrice, where prices are sorted according to the ascending parameter.
// ASC order is typically used for date-aware price lookups (getPriceForDate),
// while DESC order is efficient for latest-price queries.
func (s *FundService) LoadFundPrices(fundIDs []string, startDate, endDate time.Time, ascending bool) (map[string][]model.FundPrice, error) {
	return s.fundRepo.GetFundPrice(fundIDs, startDate, endDate, ascending)
}

// CheckUsage checks if a fund is currently in use by any portfolios.
// A fund is considered "in use" if it has portfolio_fund relationships with transactions.
// This check is critical for data integrity - funds with usage history should not be deleted
// to preserve portfolio history and fund price data for historical calculations.
//
// Returns a FundUsage object:
//   - InUsage: true if the fund has been used in any portfolio (has transactions)
//   - Portfolios: list of portfolios using the fund with transaction counts
//   - Empty Portfolios slice means the fund can be safely deleted
//
// Use case: Call before deletion to prevent losing historical data.
func (s *FundService) CheckUsage(fundID string) (model.FundUsage, error) {
	checkUsage, err := s.fundRepo.CheckUsage(fundID)
	if err != nil {
		return model.FundUsage{}, err
	}
	var fundUsage model.FundUsage
	if len(checkUsage) == 0 {
		fundUsage.InUsage = false
	} else {
		fundUsage.InUsage = true
		fundUsage.Portfolios = checkUsage
	}

	return fundUsage, nil
}

// CreatePortfolioFund creates a relationship between a portfolio and a fund.
// Validates that both the portfolio and fund exist before creating the relationship.
// This allows a fund to be tracked within a specific portfolio.
func (s *FundService) CreatePortfolioFund(ctx context.Context, req request.CreatePortfolioFundRequest) error {
	_, err := s.portfolioService.GetPortfolio(req.PortfolioID)
	if err != nil {
		return err
	}

	_, err = s.GetFund(req.FundID)
	if err != nil {
		return err
	}

	if err := s.fundRepo.InsertPortfolioFund(ctx, req.PortfolioID, req.FundID); err != nil {
		return fmt.Errorf("failed to create portfolio_fund: %w", err)
	}

	return nil
}

// DeletePortfolioFund removes the relationship between a portfolio and a fund.
// Validates that the portfolio-fund relationship exists before deletion.
// This does not delete the fund itself, only removes it from the portfolio.
func (s *FundService) DeletePortfolioFund(ctx context.Context, pfID string) error {

	_, err := s.fundRepo.GetPortfolioFund(pfID)
	if err != nil {
		return err
	}

	err = s.fundRepo.DeletePortfolioFund(ctx, pfID)
	if err != nil {
		return err
	}

	return nil
}

// CreateFund creates a new fund with the provided details.
// Generates a new UUID for the fund and inserts it into the database.
//
// Note: Once a fund is used in a portfolio (has transactions), it becomes permanent
// and cannot be deleted. This preserves portfolio history and fund price data.
// Only delete unused funds (e.g., created by mistake).
//
// Parameters:
//   - ctx: Context for the operation
//   - req: CreateFundRequest containing all required fund fields
//
// Returns the created fund with its generated ID, or an error if creation fails.
func (s *FundService) CreateFund(ctx context.Context, req request.CreateFundRequest) (*model.Fund, error) {
	fund := &model.Fund{
		ID:             uuid.New().String(),
		Name:           req.Name,
		Isin:           req.Isin,
		Symbol:         req.Symbol,
		Exchange:       req.Exchange,
		Currency:       req.Currency,
		InvestmentType: req.InvestmentType,
		DividendType:   req.DividendType,
	}

	if err := s.fundRepo.InsertFund(ctx, fund); err != nil {
		return nil, fmt.Errorf("failed to create fund: %w", err)
	}

	return fund, nil
}

// UpdateFund updates an existing fund with the provided fields.
// Only provided fields in the request are updated; omitted fields remain unchanged.
// Validates that the fund exists before updating.
//
// Parameters:
//   - ctx: Context for the operation
//   - id: The fund ID to update
//   - req: UpdateFundRequest containing the fields to update
//
// Returns the updated fund or an error if the fund doesn't exist or update fails.
func (s *FundService) UpdateFund(
	ctx context.Context,
	id string,
	req request.UpdateFundRequest,
) (*model.Fund, error) {
	fund, err := s.fundRepo.GetFund(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		fund.Name = *req.Name
	}
	if req.Isin != nil {
		fund.Isin = *req.Isin
	}
	if req.Symbol != nil {
		fund.Symbol = *req.Symbol
	}
	if req.Currency != nil {
		fund.Currency = *req.Currency
	}
	if req.Exchange != nil {
		fund.Exchange = *req.Exchange
	}
	if req.InvestmentType != nil {
		fund.InvestmentType = *req.InvestmentType
	}
	if req.DividendType != nil {
		fund.DividendType = *req.DividendType
	}

	if err := s.fundRepo.UpdateFund(ctx, &fund); err != nil {
		return nil, fmt.Errorf("failed to update fund: %w", err)
	}

	return &fund, nil
}

// DeleteFund removes a fund from the database.
// Validates that the fund exists and is not in use before deletion.
// A fund is considered "in use" if it has been associated with any portfolios
// (has transactions). This preserves portfolio history and fund price data.
//
// Parameters:
//   - ctx: Context for the operation
//   - id: The fund ID to delete
//
// Returns:
//   - apperrors.ErrFundNotFound if the fund doesn't exist
//   - apperrors.ErrFundInUse if the fund is being used by portfolios
//   - error if deletion fails
func (s *FundService) DeleteFund(ctx context.Context, id string) error {

	_, err := s.fundRepo.GetFund(id)
	if err != nil {
		return err
	}

	usage, err := s.CheckUsage(id)
	if err != nil {
		return fmt.Errorf("failed to check fund usage: %w", err)
	}

	if usage.InUsage {
		return apperrors.ErrFundInUse
	}

	err = s.fundRepo.DeleteFund(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete fund: %w", err)
	}

	return nil
}

// UpdateCurrentFundPrice fetches and stores the latest available price for a fund.
// This method retrieves yesterday's closing price from Yahoo Finance, as stock markets
// provide the previous day's close as the most recent complete data point.
//
// The method follows this workflow:
//  1. Validates that the fund exists and has a ticker symbol
//  2. Checks if yesterday's price already exists in the database (early return if found)
//  3. Fetches the last 5 days of price data from Yahoo Finance
//  4. Attempts to extract yesterday's price from the data
//  5. Falls back to the most recent available price if yesterday is not found
//  6. Inserts the price into the database if it doesn't already exist
//
// Duplicate Prevention:
// The method includes multiple safeguards against duplicate insertions:
//   - Early return if yesterday's price already exists
//   - Fallback path checks if the fallback date already exists
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - fundID: The unique identifier of the fund to update
//
// Returns:
//   - FundPrice: The inserted or existing price record
//   - bool: true if a new price was inserted, false if price already existed
//   - error: If the fund doesn't exist, has no symbol, or Yahoo Finance query fails
//
// Note: This method does not invalidate materialized views. See issue #35 for planned
// materialized view invalidation support.
func (s *FundService) UpdateCurrentFundPrice(ctx context.Context, fundID string) (model.FundPrice, bool, error) {

	fund, err := s.GetFund(fundID)
	if err != nil {
		return model.FundPrice{}, false, err
	}

	if fund.Symbol == "" {
		return model.FundPrice{}, false, apperrors.ErrInvalidSymbol
	}

	now := time.Now().UTC()
	yesterdayDate := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC)

	fundPrices, err := s.fundRepo.GetFundPrice([]string{fundID}, yesterdayDate, yesterdayDate, true)
	if err != nil {
		return model.FundPrice{}, false, err
	}

	yesterdayPrice, exists := fundPrices[fundID]
	if exists && len(yesterdayPrice) > 0 {
		return yesterdayPrice[0], false, nil
	}

	raw, err := s.yahooClient.QueryYahooFiveDaySymbol(fund.Symbol)
	if err != nil {
		return model.FundPrice{}, false, err
	}
	chart, err := s.yahooClient.ParseChart(raw)
	if err != nil {
		return model.FundPrice{}, false, err
	}

	indicator, ok := chart.GetIndicatorForDate(yesterdayDate)
	var fundPrice model.FundPrice
	if ok {
		fundPrice = model.FundPrice{
			ID:     uuid.New().String(),
			FundID: fund.ID,
			Date:   yesterdayDate,
			Price:  indicator.PriceClose,
		}
	} else {
		if len(chart.Indicators) == 0 {
			return model.FundPrice{}, false, fmt.Errorf("no price indicators available")
		}
		fallbackDate := chart.Indicators[len(chart.Indicators)-1].Date.Truncate(24 * time.Hour)
		fallbackPrices, err := s.fundRepo.GetFundPrice([]string{fundID}, fallbackDate, fallbackDate, true)
		if err != nil {
			return model.FundPrice{}, false, err
		}
		if prices, exists := fallbackPrices[fundID]; exists && len(prices) > 0 {
			return prices[0], false, nil
		}
		fundPrice = model.FundPrice{
			ID:     uuid.New().String(),
			FundID: fund.ID,
			Date:   fallbackDate,
			Price:  chart.Indicators[len(chart.Indicators)-1].PriceClose,
		}
	}

	if err = s.fundRepo.InsertFundPrice(ctx, fundPrice); err != nil {
		return model.FundPrice{}, false, err
	}

	return fundPrice, true, nil
}

// buildMissingDatesMap creates a map of date strings that are missing from the existing prices.
// This helper function reduces cyclomatic complexity in UpdateHistoricalFundPrice.
func (s *FundService) buildMissingDatesMap(existingPrices []model.FundPrice, startDate, endDate time.Time) map[string]bool {
	existingDates := make(map[string]bool)
	for _, fp := range existingPrices {
		existingDates[fp.Date.UTC().Truncate(24*time.Hour).Format("2006-01-02")] = true
	}

	missingDates := make(map[string]bool)
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		key := d.UTC().Truncate(24 * time.Hour).Format("2006-01-02")
		if !existingDates[key] {
			missingDates[key] = true
		}
	}

	return missingDates
}

// filterMissingPrices filters Yahoo Finance indicators to only include dates that are missing.
// This helper function reduces cyclomatic complexity in UpdateHistoricalFundPrice.
func (s *FundService) filterMissingPrices(indicators []yahoo.Indicators, missingDates map[string]bool, fundID string) []model.FundPrice {
	missingFundPrices := make([]model.FundPrice, 0, len(missingDates))
	for _, v := range indicators {
		sanitizedDate := v.Date.Truncate(24 * time.Hour).Format("2006-01-02")
		if missingDates[sanitizedDate] {
			missingFundPrices = append(missingFundPrices, model.FundPrice{
				ID:     uuid.New().String(),
				FundID: fundID,
				Price:  v.PriceClose,
				Date:   v.Date.Truncate(24 * time.Hour),
			})
		}
	}
	return missingFundPrices
}

// UpdateHistoricalFundPrice backfills missing historical prices for a fund.
// This method identifies all missing price dates from the fund's earliest transaction
// to yesterday, fetches the data from Yahoo Finance, and performs a batch insert of
// all missing prices.
//
// The method follows this workflow:
//  1. Validates that the fund exists and has a ticker symbol
//  2. Retrieves all portfolio_fund relationships for this fund
//  3. Finds the earliest transaction date across all portfolios using this fund
//  4. Queries existing prices in the database for the date range
//  5. Identifies missing dates by comparing all dates in range with existing prices
//  6. Returns early if no missing dates are found (no-op)
//  7. Fetches historical data from Yahoo Finance for the entire date range
//  8. Filters Yahoo data to only include missing dates
//  9. Performs a single batch insert of all missing prices
//
// Efficiency Notes:
// This implementation is more efficient than the Python equivalent as it:
//   - Uses map-based lookups for O(1) date existence checks
//   - Builds all prices in memory before insertion
//   - Performs a single batch insert instead of individual inserts
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - fundID: The unique identifier of the fund to update
//
// Returns:
//   - int: The number of new prices added to the database
//   - error: If the fund doesn't exist, has no symbol, has no transactions,
//     or Yahoo Finance query fails
//
// Note: This method does not invalidate materialized views. See issue #35 for planned
// materialized view invalidation support.
func (s *FundService) UpdateHistoricalFundPrice(ctx context.Context, fundID string) (int, error) {
	fund, err := s.GetFund(fundID)
	if err != nil {
		return 0, err
	}

	if fund.Symbol == "" {
		return 0, apperrors.ErrInvalidSymbol
	}

	portfolioFunds, err := s.fundRepo.GetPortfolioFundsbyFundID(fundID)
	if err != nil {
		return 0, err
	}
	if len(portfolioFunds) == 0 {
		return 0, fmt.Errorf("no portfolio funds found for fund %s", fundID)
	}

	pfIDs := make([]string, len(portfolioFunds))
	for i, v := range portfolioFunds {
		pfIDs[i] = v.ID
	}

	oldestDate := s.transactionService.getOldestTransaction(pfIDs)
	if oldestDate.IsZero() {
		return 0, fmt.Errorf("no transactions found for fund %s", fundID)
	}
	now := time.Now()
	yesterdayDate := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC)

	existingPrices, err := s.fundRepo.GetFundPrice([]string{fundID}, oldestDate, yesterdayDate, true)
	if err != nil {
		return 0, err
	}

	missingDates := s.buildMissingDatesMap(existingPrices[fundID], oldestDate, yesterdayDate)
	if len(missingDates) == 0 {
		return 0, nil // nothing to do
	}

	raw, err := s.yahooClient.QueryYahooSymbolByDateRange(fund.Symbol, oldestDate, yesterdayDate)
	if err != nil {
		return 0, err
	}
	chart, err := s.yahooClient.ParseChart(raw)
	if err != nil {
		return 0, err
	}

	missingFundPrices := s.filterMissingPrices(chart.Indicators, missingDates, fundID)
	if len(missingFundPrices) == 0 {
		return 0, nil
	}

	err = s.fundRepo.InsertFundPrices(ctx, missingFundPrices)
	if err != nil {
		return 0, err
	}

	return len(missingFundPrices), nil
}
