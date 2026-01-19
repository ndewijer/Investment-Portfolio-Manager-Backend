package service

import (
	"math"
	"slices"
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
) *MaterializedService {
	return &MaterializedService{
		materializedRepo:        materializedRepo,
		portfolioRepo:           portfolioRepo,
		fundRepo:                fundRepo,
		transactionService:      transactionService,
		fundService:             fundService,
		dividendService:         dividendService,
		realizedGainLossService: realizedGainLossService,
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
	var portfolios []model.Portfolio
	var err error

	if portfolioID != "" {
		// Load single portfolio
		portfolio, err := s.portfolioRepo.GetPortfolioOnID(portfolioID)
		if err != nil {
			return nil, err
		}
		portfolios = []model.Portfolio{portfolio}
	} else {
		// Load all active portfolios
		portfolios, err = s.loadActivePortfolios()
		if err != nil {
			return nil, err
		}
	}
	portfolioIDs := make([]string, len(portfolios))
	portfolioNames := make(map[string]string)
	portfolioDescription := make(map[string]string)
	for i, p := range portfolios {
		portfolioIDs[i] = p.ID
		portfolioNames[p.ID] = p.Name
		portfolioDescription[p.ID] = p.Name
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
//
// Data Loading Strategy:
// To ensure accurate calculations, this method always loads the COMPLETE transaction history
// from the oldest transaction to the present, regardless of the requested date range.
// This is necessary because share counts and cost basis depend on all prior transactions.
//
// Parameters:
//   - requestedStartDate: First date to include in returned results
//   - requestedEndDate: Last date to include in returned results
//   - portfolioID: Optional portfolio ID. If empty, returns all active portfolios.
func (s *MaterializedService) GetPortfolioHistory(requestedStartDate, requestedEndDate time.Time, portfolioID string) ([]model.PortfolioHistory, error) {

	var portfolios []model.Portfolio
	var err error

	if portfolioID != "" {
		// Load single portfolio
		portfolio, err := s.portfolioRepo.GetPortfolioOnID(portfolioID)
		if err != nil {
			return nil, err
		}
		portfolios = []model.Portfolio{portfolio}
	} else {
		// Load all active portfolios
		portfolios, err = s.loadActivePortfolios()
		if err != nil {
			return nil, err
		}
	}

	var portfolioUuids []string
	for _, port := range portfolios {
		portfolioUuids = append(portfolioUuids, port.ID)
	}

	_, portfolioFundToPortfolio, portfolioFundToFund, pfIDs, fundIDs, err := s.portfolioRepo.GetPortfolioFundsOnPortfolioID(portfolios)
	if err != nil {
		return nil, err
	}

	oldestTransactionDate := s.transactionService.getOldestTransaction(pfIDs)

	dataStartDate := oldestTransactionDate
	dataEndDate := time.Now()

	displayStartDate := requestedStartDate
	displayEndDate := requestedEndDate
	if displayStartDate.Before(dataStartDate) {
		displayStartDate = dataStartDate
	}
	if displayEndDate.After(dataEndDate) {
		displayEndDate = dataEndDate
	}

	// Truncate to midnight in UTC to match loop dates (which come from DB as UTC)
	displayStartDate = time.Date(displayStartDate.Year(), displayStartDate.Month(), displayStartDate.Day(), 0, 0, 0, 0, time.UTC)
	displayEndDate = time.Date(displayEndDate.Year(), displayEndDate.Month(), displayEndDate.Day(), 0, 0, 0, 0, time.UTC)

	// Load transactions grouped by portfolio_fund
	transactionsByPF, err := s.transactionService.loadTransactions(pfIDs, dataStartDate, dataEndDate)
	if err != nil {
		return nil, err
	}

	// Load dividends grouped by portfolio_fund
	dividendsByPF, err := s.dividendService.loadDividendPerPF(pfIDs, dataStartDate, dataEndDate)
	if err != nil {
		return nil, err
	}

	fundPriceByFund, err := s.fundService.loadFundPrices(fundIDs, dataStartDate, dataEndDate, true) //ASC
	if err != nil {
		return nil, err
	}

	realizedGainLossByPortfolio, err := s.realizedGainLossService.loadRealizedGainLoss(portfolioUuids, dataStartDate, dataEndDate)
	if err != nil {
		return nil, err
	}

	// Build both flat portfolio-level and nested PF-within-portfolio structures in one pass
	transactionsByPortfolio := make(map[string][]model.Transaction)
	transactionsByPFByPortfolio := make(map[string]map[string][]model.Transaction)
	dividendByPortfolio := make(map[string][]model.Dividend)
	dividendsByPFByPortfolio := make(map[string]map[string][]model.Dividend)

	for pfID, transactions := range transactionsByPF {
		portfolioID := portfolioFundToPortfolio[pfID]

		// Used for per-fund calculations
		if transactionsByPFByPortfolio[portfolioID] == nil {
			transactionsByPFByPortfolio[portfolioID] = make(map[string][]model.Transaction)
		}
		transactionsByPFByPortfolio[portfolioID][pfID] = transactions

		// Used for oldest transaction check
		transactionsByPortfolio[portfolioID] = append(transactionsByPortfolio[portfolioID], transactions...)
	}

	for pfID, dividends := range dividendsByPF {
		portfolioID := portfolioFundToPortfolio[pfID]

		// Used for per-fund calculations
		if dividendsByPFByPortfolio[portfolioID] == nil {
			dividendsByPFByPortfolio[portfolioID] = make(map[string][]model.Dividend)
		}
		dividendsByPFByPortfolio[portfolioID][pfID] = dividends

		// Used for dividend amount calculations
		dividendByPortfolio[portfolioID] = append(dividendByPortfolio[portfolioID], dividends...)
	}

	portfolioHistory := []model.PortfolioHistory{}
	for date := dataStartDate; !date.After(dataEndDate); date = date.AddDate(0, 0, 1) {

		portfolioSummary := []model.PortfolioSummary{}

		for _, portfolio := range portfolios {

			if len(transactionsByPortfolio[portfolio.ID]) == 0 {
				continue
			}

			oldest := slices.MinFunc(transactionsByPortfolio[portfolio.ID], func(a, b model.Transaction) int {
				return a.Date.Compare(b.Date)
			})
			if oldest.Date.After(date) {
				continue
			}

			dividendsByPF := dividendsByPFByPortfolio[portfolio.ID]
			transactionsByPF := transactionsByPFByPortfolio[portfolio.ID]

			totalDividendSharesPerPF, err := s.dividendService.processDividendSharesForDate(dividendsByPF, transactionsByPortfolio[portfolio.ID], date)
			if err != nil {
				return nil, err
			}

			transactionMetrics, err := s.processTransactionsForDate(transactionsByPF, totalDividendSharesPerPF, portfolioFundToFund, fundPriceByFund, date)
			if err != nil {
				return nil, err
			}

			totalDividendAmount, err := s.dividendService.processDividendAmountForDate(dividendByPortfolio[portfolio.ID], date)
			if err != nil {
				return nil, err
			}
			totalRealizedGainLoss, totalSaleProceeds, totalCostBasis, err := s.realizedGainLossService.processRealizedGainLossForDate(realizedGainLossByPortfolio[portfolio.ID], date)
			if err != nil {
				return nil, err
			}

			ps := model.PortfolioSummary{
				ID:                      portfolio.ID,
				Name:                    portfolio.Name,
				Description:             portfolio.Description,
				TotalValue:              math.Round(transactionMetrics.TotalValue*RoundingPrecision) / RoundingPrecision,
				TotalCost:               math.Round(transactionMetrics.TotalCost*RoundingPrecision) / RoundingPrecision,
				TotalDividends:          math.Round(totalDividendAmount*RoundingPrecision) / RoundingPrecision,
				TotalUnrealizedGainLoss: math.Round((transactionMetrics.TotalValue-transactionMetrics.TotalCost)*RoundingPrecision) / RoundingPrecision,
				TotalRealizedGainLoss:   math.Round(totalRealizedGainLoss*RoundingPrecision) / RoundingPrecision,
				TotalSaleProceeds:       math.Round(totalSaleProceeds*RoundingPrecision) / RoundingPrecision,
				TotalOriginalCost:       math.Round(totalCostBasis*RoundingPrecision) / RoundingPrecision,
				TotalGainLoss:           math.Round((totalRealizedGainLoss+(transactionMetrics.TotalValue-transactionMetrics.TotalCost))*RoundingPrecision) / RoundingPrecision,
				IsArchived:              portfolio.IsArchived,
			}

			portfolioSummary = append(portfolioSummary, ps)

		}

		if (date.After(displayStartDate) || date.Equal(displayStartDate)) &&
			(date.Before(displayEndDate) || date.Equal(displayEndDate)) {
			ph := model.PortfolioHistory{
				Date:       date.Format("2006-01-02"),
				Portfolios: portfolioSummary,
			}
			portfolioHistory = append(portfolioHistory, ph)
		}
	}

	return portfolioHistory, nil
}

// GetPortfolioHistoryWithFallback tries to retrieve history from the materialized view,
// falling back to on-demand calculation if the materialized data is incomplete or empty.
//
// This provides the best of both worlds:
// - Fast materialized view when available (~3-10ms)
// - Reliable on-demand calculation as fallback (~50ms)
func (s *MaterializedService) GetPortfolioHistoryWithFallback(
	startDate, endDate time.Time,
	portfolioID string,
) ([]model.PortfolioHistory, error) {

	// Step 1: Try materialized view first (fast path)
	materialized, err := s.GetPortfolioHistoryMaterialized(startDate, endDate, portfolioID)
	// If query succeeded and we got data, use it
	if err == nil && len(materialized) > 0 {
		return materialized, nil
	}
	// Step 2: Fallback to on-demand calculation
	// (Materialized view is empty, being regenerated, or query failed)
	return s.GetPortfolioHistory(startDate, endDate, portfolioID)
}

// =============================================================================
// FUND HISTORY METHODS
// =============================================================================

// GetFundHistoryMaterialized retrieves historical fund data from the materialized view.
// Returns time-series data showing individual fund values within a portfolio over time.
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

	// If we got data from materialized view, format it
	if len(fundHistoryByDate) > 0 {
		return s.formatFundHistoryFromMaterialized(fundHistoryByDate), nil
	}

	// Return empty if no data
	return []model.FundHistoryResponse{}, nil
}

// formatFundHistoryFromMaterialized converts the map structure to response format.
func (s *MaterializedService) formatFundHistoryFromMaterialized(fundHistoryByDate map[string][]model.FundHistoryEntry) []model.FundHistoryResponse {
	// Get sorted date keys
	dates := make([]string, 0, len(fundHistoryByDate))
	for date := range fundHistoryByDate {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	// Build response
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

// calculateFundHistoryOnFly calculates fund history when materialized view is unavailable.
// This is the fallback mechanism that computes values from raw transactions and prices.
func (s *MaterializedService) calculateFundHistoryOnFly(portfolioID string, startDate, endDate time.Time) ([]model.FundHistoryResponse, error) {
	// Get portfolio funds
	portfolioFunds, err := s.fundRepo.GetPortfolioFunds(portfolioID)
	if err != nil {
		return nil, err
	}

	if len(portfolioFunds) == 0 {
		return []model.FundHistoryResponse{}, nil
	}

	// Collect IDs
	var pfIDs, fundIDs []string
	for _, fund := range portfolioFunds {
		pfIDs = append(pfIDs, fund.ID)
		fundIDs = append(fundIDs, fund.FundId)
	}

	// Get oldest transaction to determine data start
	oldestTxDate := s.transactionService.getOldestTransaction(pfIDs)
	if oldestTxDate.After(endDate) {
		return []model.FundHistoryResponse{}, nil
	}

	// Adjust start date if needed
	dataStartDate := startDate
	if dataStartDate.Before(oldestTxDate) {
		dataStartDate = oldestTxDate
	}

	// Load all data once (batch loading for efficiency)
	transactionsByPF, err := s.transactionService.loadTransactions(pfIDs, oldestTxDate, endDate)
	if err != nil {
		return nil, err
	}

	dividendsByPF, err := s.dividendService.loadDividendPerPF(pfIDs, oldestTxDate, endDate)
	if err != nil {
		return nil, err
	}

	fundPriceByFund, err := s.fundService.loadFundPrices(fundIDs, oldestTxDate, endDate, true)
	if err != nil {
		return nil, err
	}

	realizedGainsByPortfolio, err := s.realizedGainLossService.loadRealizedGainLoss([]string{portfolioID}, oldestTxDate, endDate)
	if err != nil {
		return nil, err
	}

	// Build map of realized gains by portfolio fund
	realizedGainsByPF := make(map[string][]model.RealizedGainLoss)
	for _, entry := range realizedGainsByPortfolio[portfolioID] {
		for _, pf := range portfolioFunds {
			if entry.FundID == pf.FundId {
				realizedGainsByPF[pf.ID] = append(realizedGainsByPF[pf.ID], entry)
			}
		}
	}

	// Iterate through each date
	var response []model.FundHistoryResponse

	for currentDate := dataStartDate; !currentDate.After(endDate); currentDate = currentDate.AddDate(0, 0, 1) {
		var fundsForDate []model.FundHistoryEntry

		for _, pf := range portfolioFunds {
			// Calculate dividend shares as of this date
			dividendSharesMap, err := s.dividendService.processDividendSharesForDate(dividendsByPF, transactionsByPF[pf.ID], currentDate)
			if err != nil {
				return nil, err
			}

			// Calculate metrics for this fund on this date
			fundMetrics, err := s.fundService.calculateFundMetrics(
				pf.ID,
				pf.FundId,
				currentDate,
				transactionsByPF[pf.ID],
				dividendSharesMap[pf.ID],
				fundPriceByFund[pf.FundId],
				false, // Use price as of date, not latest
			)
			if err != nil {
				return nil, err
			}

			// Calculate dividend amount
			dividendAmount, err := s.dividendService.processDividendAmountForDate(dividendsByPF[pf.ID], currentDate)
			if err != nil {
				return nil, err
			}

			// Calculate realized gains
			realizedGain, _, _, err := s.realizedGainLossService.processRealizedGainLossForDate(realizedGainsByPF[pf.ID], currentDate)
			if err != nil {
				return nil, err
			}

			// Build entry
			fundsForDate = append(fundsForDate, model.FundHistoryEntry{
				PortfolioFundID: pf.ID,
				FundID:          pf.FundId,
				FundName:        pf.FundName,
				Shares:          math.Round(fundMetrics.Shares*RoundingPrecision) / RoundingPrecision,
				Price:           math.Round(fundMetrics.LatestPrice*RoundingPrecision) / RoundingPrecision,
				Value:           math.Round(fundMetrics.Value*RoundingPrecision) / RoundingPrecision,
				Cost:            math.Round(fundMetrics.Cost*RoundingPrecision) / RoundingPrecision,
				RealizedGain:    math.Round(realizedGain*RoundingPrecision) / RoundingPrecision,
				UnrealizedGain:  math.Round(fundMetrics.UnrealizedGain*RoundingPrecision) / RoundingPrecision,
				TotalGainLoss:   math.Round((fundMetrics.UnrealizedGain+realizedGain)*RoundingPrecision) / RoundingPrecision,
				Dividends:       math.Round(dividendAmount*RoundingPrecision) / RoundingPrecision,
				Fees:            math.Round(fundMetrics.Fees*RoundingPrecision) / RoundingPrecision,
			})
		}

		if len(fundsForDate) > 0 {
			response = append(response, model.FundHistoryResponse{
				Date:  currentDate,
				Funds: fundsForDate,
			})
		}
	}

	return response, nil
}

// GetFundHistoryWithFallback tries to retrieve history from the materialized view,
// falling back to on-demand calculation if the materialized data is incomplete or empty.
func (s *MaterializedService) GetFundHistoryWithFallback(
	portfolioID string,
	startDate, endDate time.Time,
) ([]model.FundHistoryResponse, error) {

	// Step 1: Try materialized view first (fast path)
	materialized, err := s.GetFundHistoryMaterialized(portfolioID, startDate, endDate)

	// If query succeeded and we got data, use it
	if err == nil && len(materialized) > 0 {
		return materialized, nil
	}

	// Step 2: Fallback to on-demand calculation
	return s.calculateFundHistoryOnFly(portfolioID, startDate, endDate)
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// DividendService returns the DividendService for use in handlers that need dividend-specific operations.
func (s *MaterializedService) DividendService() *DividendService {
	return s.dividendService
}

// loadActivePortfolios retrieves only active, non-excluded portfolios.
func (s *MaterializedService) loadActivePortfolios() ([]model.Portfolio, error) {
	return s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: false,
		IncludeExcluded: false,
	})
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
