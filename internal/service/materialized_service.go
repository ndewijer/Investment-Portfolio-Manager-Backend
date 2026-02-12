package service

import (
	"sort"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// TransactionMetrics aggregates calculated metrics from processing transactions for a specific date.
// It is used internally by processTransactionsForDate to return all computed values in a single struct.
type TransactionMetrics struct {
	TotalShares    float64 // Total number of shares held
	TotalCost      float64 // Total cost basis
	TotalValue     float64 // Total market value
	TotalDividends float64 // Total dividend amounts
	TotalFees      float64 // Total fees paid
}

// MaterializedService handles history-related business logic operations.
// It coordinates between materialized views and on-demand calculations to provide
// portfolio and fund history data with fallback capabilities.
type MaterializedService struct {
	materializedRepo        *repository.MaterializedRepository
	portfolioRepo           *repository.PortfolioRepository
	fundRepo                *repository.FundRepository
	transactionService      *TransactionService
	fundService             *FundService
	dividendService         *DividendService
	realizedGainLossService *RealizedGainLossService
	dataLoaderService       *DataLoaderService
	portfolioService        *PortfolioService
}

// NewMaterializedService creates a new MaterializedService with the provided dependencies.
func NewMaterializedService(
	materializedRepo *repository.MaterializedRepository,
	portfolioRepo *repository.PortfolioRepository,
	fundRepo *repository.FundRepository,
	transactionService *TransactionService,
	fundService *FundService,
	dividendService *DividendService,
	realizedGainLossService *RealizedGainLossService,
	dataLoaderService *DataLoaderService,
	portfolioService *PortfolioService,
) *MaterializedService {
	return &MaterializedService{
		materializedRepo:        materializedRepo,
		portfolioRepo:           portfolioRepo,
		fundRepo:                fundRepo,
		transactionService:      transactionService,
		fundService:             fundService,
		dividendService:         dividendService,
		realizedGainLossService: realizedGainLossService,
		dataLoaderService:       dataLoaderService,
		portfolioService:        portfolioService,
	}
}

// =============================================================================
// PORTFOLIO HISTORY METHODS
// =============================================================================

// GetPortfolioHistoryMaterialized retrieves daily portfolio valuations from the materialized view table.
//
// This method provides significantly faster performance compared to GetPortfolioHistory() by querying
// pre-calculated daily snapshots instead of recomputing values from raw transactions, dividends, and prices.
//
// The method performs the following:
//  1. Resolves portfolio(s) from the ID parameter using GetPortfoliosForRequest
//  2. Queries the materialized view (delegates to materializedRepo.GetMaterializedHistory)
//  3. Groups results by date using a callback pattern
//  4. Transforms grouped records into PortfolioSummary structs with portfolio metadata
//
// Performance Characteristics:
//   - 1 year (365 days): ~3-5ms (compared to ~50ms for on-demand calculation)
//   - 5 years (1825 days): ~8-10ms (compared to ~50ms for on-demand calculation)
//   - 6-16x faster than on-demand calculation
//
// Parameters:
//   - requestedStartDate: First date to include in returned results
//   - requestedEndDate: Last date to include in returned results
//   - portfolioID: Optional portfolio ID. If empty, returns all active portfolios.
//
// Returns:
// A slice of PortfolioHistory structs, one per date, each containing portfolio summaries for that date.
func (s *MaterializedService) GetPortfolioHistoryMaterialized(requestedStartDate, requestedEndDate time.Time, portfolioID string) ([]model.PortfolioHistory, error) {

	portfolios, err := s.portfolioService.GetPortfoliosForRequest(portfolioID)
	if err != nil {
		return nil, err
	}

	portfolioIDs := make([]string, len(portfolios))
	portfolioNames := make(map[string]string, len(portfolios))
	portfolioDescription := make(map[string]string, len(portfolios))
	for i, p := range portfolios {
		portfolioIDs[i] = p.ID
		portfolioNames[p.ID] = p.Name
		portfolioDescription[p.ID] = p.Description
	}
	historyMap := make(map[string][]model.PortfolioHistoryMaterialized)

	err = s.materializedRepo.GetMaterializedHistory(
		portfolioIDs,
		requestedStartDate,
		requestedEndDate,
		func(record model.PortfolioHistoryMaterialized) error {
			historyKey := record.Date.Format("2006-01-02")
			historyMap[historyKey] = append(historyMap[historyKey], record)
			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	result := []model.PortfolioHistory{}
	for date := requestedStartDate; !date.After(requestedEndDate); date = date.AddDate(0, 0, 1) {
		historyKey := date.Format("2006-01-02")
		records, exists := historyMap[historyKey]
		if !exists {
			continue
		}

		summaries := make([]model.PortfolioSummary, len(records))
		for i, record := range records {
			summaries[i] = model.PortfolioSummary{
				ID:                      record.PortfolioID,
				Name:                    portfolioNames[record.PortfolioID],
				Description:             portfolioDescription[record.PortfolioID],
				TotalValue:              record.Value,
				TotalCost:               record.Cost,
				TotalDividends:          record.TotalDividends,
				TotalUnrealizedGainLoss: record.UnrealizedGain,
				TotalRealizedGainLoss:   record.RealizedGain,
				TotalSaleProceeds:       record.TotalSaleProceeds,
				TotalOriginalCost:       record.TotalOriginalCost,
				TotalGainLoss:           record.TotalGainLoss,
				IsArchived:              record.IsArchived,
			}
		}

		result = append(result, model.PortfolioHistory{
			Date:       historyKey,
			Portfolios: summaries,
		})
	}
	return result, nil
}

// GetPortfolioHistory retrieves daily portfolio valuations for the requested date range.
// This method orchestrates the calculation of portfolio metrics for each day, providing
// time-series data showing how portfolio value evolved over the period.
//
// Calculation Pipeline:
//  1. Resolves portfolio(s) from the ID parameter
//  2. Batch-loads all required data (transactions, dividends, prices, realized gains)
//  3. Adjusts display date range to fit within available data boundaries
//  4. Groups transactions and dividends by portfolio for efficient calculations
//  5. Calculates daily portfolio summaries across the date range
//
// Data Loading Strategy:
// The method loads COMPLETE transaction history from the oldest transaction to present,
// regardless of the requested date range. This is necessary because share counts and
// cost basis depend on all prior transactions. However, only dates within the requested
// range are included in the returned results.
//
// The actual daily calculations are delegated to helper functions in materialized_helpers.go.
// This method focuses on orchestration and data preparation.
//
// Parameters:
//   - requestedStartDate: First date to include in returned results
//   - requestedEndDate: Last date to include in returned results
//   - portfolioID: Optional portfolio ID. If empty, returns all active portfolios.
//
// Returns a slice of PortfolioHistory, one entry per date, each containing portfolio
// summaries with metrics like TotalValue, TotalCost, TotalGainLoss, etc.
func (s *MaterializedService) GetPortfolioHistory(requestedStartDate, requestedEndDate time.Time, portfolioID string) ([]model.PortfolioHistory, error) {

	portfolios, err := s.portfolioService.GetPortfoliosForRequest(portfolioID)
	if err != nil {
		return nil, err
	}

	data, err := s.dataLoaderService.LoadForPortfolios(portfolios, requestedStartDate, requestedEndDate)
	if err != nil {
		return nil, err
	}

	displayStart, displayEnd := s.calculateDisplayDateRange(
		requestedStartDate,
		requestedEndDate,
		data.OldestTransactionDate,
	)

	txByPortfolio, txByPFByPortfolio := s.groupTransactionsByPortfolio(
		data.TransactionsByPF,
		data.PortfolioFundToPortfolio,
	)

	divByPortfolio, divByPFByPortfolio := s.groupDividendsByPortfolio(
		data.DividendsByPF,
		data.PortfolioFundToPortfolio,
	)

	return s.calculateDailyPortfolioHistory(
		portfolios,
		data,
		txByPortfolio,
		txByPFByPortfolio,
		divByPortfolio,
		divByPFByPortfolio,
		displayStart,
		displayEnd,
	)
}

// GetPortfolioHistoryWithFallback tries to retrieve history from the materialized view,
// falling back to on-demand calculation if the materialized data is incomplete or empty.
//
// Fallback Logic:
// The method verifies that materialized data covers the FULL requested date range by checking
// if the last date in the results is >= endDate. If the materialized view is stale (e.g., has
// data through Jan 18 but today is Jan 28), it falls back to on-demand calculation to ensure
// completeness.
//
// This provides the best of both worlds:
//   - Fast materialized view when available and complete (~3-10ms)
//   - Reliable on-demand calculation as fallback when data is stale or missing (~50ms)
//
// Parameters:
//   - startDate: First date to include in results
//   - endDate: Last date to include in results (typically today)
//   - portfolioID: Optional portfolio ID. Empty string returns all active portfolios.
//
// Returns complete portfolio history from startDate to endDate, using the fastest available method.
func (s *MaterializedService) GetPortfolioHistoryWithFallback(
	startDate, endDate time.Time,
	portfolioID string,
) ([]model.PortfolioHistory, error) {

	// Step 1: Try materialized view first (fast path)
	materialized, err := s.GetPortfolioHistoryMaterialized(startDate, endDate, portfolioID)

	// Step 2: Check if materialized data is complete
	if err == nil && len(materialized) > 0 {
		// Verify the data covers the full requested range
		lastDate, parseErr := time.Parse("2006-01-02", materialized[len(materialized)-1].Date)
		if parseErr == nil && !lastDate.UTC().Before(endDate) {
			// Materialized view covers the requested range
			return materialized, nil
		}
		// Data is incomplete (stale or partial) - fall through to on-demand
	}

	// Step 3: Fallback to on-demand calculation
	// (Materialized view is empty, stale, or query failed)
	return s.GetPortfolioHistory(startDate, endDate, portfolioID)
}

// =============================================================================
// FUND HISTORY METHODS
// =============================================================================

// GetFundHistoryMaterialized retrieves historical fund data from the materialized view.
// Returns time-series data showing individual fund values within a portfolio over time.
//
// The method performs the following:
//  1. Queries the materialized view (delegates to materializedRepo.GetFundHistoryMaterialized)
//  2. Groups results by date using a callback pattern
//  3. Formats the grouped data into chronological response (delegates to formatFundHistoryFromMaterialized)
//
// Parameters:
//   - portfolioID: The portfolio ID to retrieve fund history for
//   - startDate: Inclusive start date for the query
//   - endDate: Inclusive end date for the query
//
// Returns a slice of FundHistoryResponse, one entry per date with all funds for that date.
func (s *MaterializedService) GetFundHistoryMaterialized(portfolioID string, startDate, endDate time.Time) ([]model.FundHistoryResponse, error) {
	fundHistoryByDate := make(map[string][]model.FundHistoryEntry)

	err := s.materializedRepo.GetFundHistoryMaterialized(
		portfolioID,
		startDate,
		endDate,
		func(entry model.FundHistoryEntry) error {
			dateKey := entry.Date.Format("2006-01-02")
			fundHistoryByDate[dateKey] = append(fundHistoryByDate[dateKey], entry)
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	if len(fundHistoryByDate) > 0 {
		return s.formatFundHistoryFromMaterialized(fundHistoryByDate), nil
	}

	return []model.FundHistoryResponse{}, nil
}

// formatFundHistoryFromMaterialized converts the date-keyed map structure to chronological response format.
// This helper transforms the callback-accumulated map (date string -> fund entries) into a properly
// ordered slice of FundHistoryResponse structs.
//
// The method:
//  1. Extracts all date strings from the map keys
//  2. Sorts them chronologically (string sort works for YYYY-MM-DD format)
//  3. Builds response entries in date order
//
// Parameters:
//   - fundHistoryByDate: Map of date strings to fund entries for that date
//
// Returns a slice of FundHistoryResponse sorted by date, with each entry containing all
// fund metrics for that date. Empty dates (no funds) are excluded from results.
func (s *MaterializedService) formatFundHistoryFromMaterialized(fundHistoryByDate map[string][]model.FundHistoryEntry) []model.FundHistoryResponse {

	dates := make([]string, 0, len(fundHistoryByDate))
	for date := range fundHistoryByDate {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	var response []model.FundHistoryResponse
	for _, dateStr := range dates {
		funds := fundHistoryByDate[dateStr]
		if len(funds) > 0 {
			response = append(response, model.FundHistoryResponse{
				Date:  funds[0].Date, // All entries have same date
				Funds: funds,
			})
		}
	}

	return response
}

// calculateFundHistoryOnFly computes fund-level history on-demand when materialized data is unavailable.
// This is the fallback calculation path that processes raw transactions, dividends, and prices to
// produce per-fund metrics for each day in the requested range.
//
// Calculation Pipeline:
//  1. Resolves portfolio from the ID parameter
//  2. Batch-loads all required data (transactions, dividends, prices, realized gains)
//  3. Maps realized gains from portfolio level to fund level
//  4. Calculates daily fund metrics across the date range
//
// This method delegates the actual calculations to calculateFundHistoryByDate and its helpers
// in materialized_helpers.go. It focuses on orchestration and data preparation.
//
// Performance Note: This method is slower than the materialized view (~50ms vs ~3-10ms) but
// ensures complete and current data when the materialized view is stale or unavailable.
//
// Parameters:
//   - portfolioID: The portfolio to calculate fund history for
//   - startDate: First date to include in results
//   - endDate: Last date to include in results
//
// Returns a slice of FundHistoryResponse, one per date, with per-fund metrics for that date.
func (s *MaterializedService) calculateFundHistoryOnFly(portfolioID string, startDate, endDate time.Time) ([]model.FundHistoryResponse, error) {

	portfolio, err := s.portfolioService.GetPortfoliosForRequest(portfolioID)
	if err != nil {
		return nil, err
	}

	data, err := s.dataLoaderService.LoadForPortfolios(portfolio, startDate, endDate)
	if err != nil {
		return nil, err
	}

	if len(data.PFIDs) == 0 {
		return []model.FundHistoryResponse{}, nil
	}

	realizedGainsByPF := data.MapRealizedGainsByPF(portfolioID)

	return s.calculateFundHistoryByDate(
		data,
		realizedGainsByPF,
		startDate,
		endDate,
	)
}

// GetFundHistoryWithFallback tries to retrieve history from the materialized view,
// falling back to on-demand calculation if the materialized data is incomplete or empty.
//
// Fallback Logic:
// The method verifies that materialized data covers the FULL requested date range by checking
// if the last date in the results is >= endDate. If the materialized view is stale, it falls
// back to on-demand calculation to ensure completeness.
//
// This provides the best of both worlds:
//   - Fast materialized view when available and complete (~3-10ms)
//   - Reliable on-demand calculation as fallback when data is stale or missing (~50ms)
//
// Parameters:
//   - portfolioID: The portfolio ID to retrieve fund history for
//   - startDate: First date to include in results
//   - endDate: Last date to include in results (typically today)
//
// Returns complete fund-level history from startDate to endDate, using the fastest available method.
func (s *MaterializedService) GetFundHistoryWithFallback(
	portfolioID string,
	startDate, endDate time.Time,
) ([]model.FundHistoryResponse, error) {

	// Step 1: Try materialized view first (fast path)
	materialized, err := s.GetFundHistoryMaterialized(portfolioID, startDate, endDate)

	// Step 2: Check if materialized data is complete
	if err == nil && len(materialized) > 0 {
		// Verify the data covers the full requested range
		lastDate := materialized[len(materialized)-1].Date
		if !lastDate.Before(endDate) {
			// Materialized view covers the requested range
			return materialized, nil
		}
		// Data is incomplete (stale or partial) - fall through to on-demand
	}

	// Step 3: Fallback to on-demand calculation
	// (Materialized view is empty, stale, or query failed)
	return s.calculateFundHistoryOnFly(portfolioID, startDate, endDate)
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// DividendService returns the DividendService for use in handlers that need dividend-specific operations.
func (s *MaterializedService) DividendService() *DividendService {
	return s.dividendService
}

// processTransactionsForDate calculates portfolio metrics as of the specified date.
// This is a local helper that delegates to FundService for per-fund calculations.
func (s *MaterializedService) processTransactionsForDate(transactionsMap map[string][]model.Transaction, dividendShares map[string]float64, fundMapping map[string]string, fundPriceByFund map[string][]model.FundPrice, date time.Time) (TransactionMetrics, error) {
	if len(transactionsMap) == 0 {
		return TransactionMetrics{}, nil
	}
	var totalShares, totalCost, totalValue, totalDividends, totalFees float64
	for pfID, transactions := range transactionsMap {
		fundID := fundMapping[pfID]
		prices := fundPriceByFund[fundID]

		fundMetrics, err := s.fundService.calculateFundMetrics(
			pfID, fundID, date, transactions, dividendShares[pfID], prices, false)

		if err != nil {
			return TransactionMetrics{}, err
		}

		totalValue += fundMetrics.Value
		totalShares += fundMetrics.Shares
		totalCost += fundMetrics.Cost
		totalDividends += fundMetrics.Dividend
		totalFees += fundMetrics.Fees
	}

	totalShares = max(0, totalShares)
	totalCost = max(0, totalCost)
	totalValue = max(0, totalValue)
	totalDividends = max(0, totalDividends)
	totalFees = max(0, totalFees)

	return TransactionMetrics{
		TotalShares:    totalShares,
		TotalCost:      totalCost,
		TotalValue:     totalValue,
		TotalDividends: totalDividends,
		TotalFees:      totalFees,
	}, nil
}
