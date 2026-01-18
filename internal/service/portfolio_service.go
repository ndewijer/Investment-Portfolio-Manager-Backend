package service

import (
	"math"
	"slices"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// PortfolioService handles portfolio-related business logic operations.
// It coordinates between multiple repositories to compute portfolio summaries,
// historical valuations, and aggregate metrics across transactions, dividends,
// fund prices, and realized gains/losses.
type PortfolioService struct {
	portfolioRepo           *repository.PortfolioRepository
	materializedRepo        *repository.MaterializedRepository
	transactionService      *TransactionService
	fundService             *FundService
	dividendService         *DividendService
	realizedGainLossService *RealizedGainLossService
}

// NewPortfolioService creates a new PortfolioService with the provided repository dependencies.
// All repository parameters are required for proper portfolio calculations.
func NewPortfolioService(
	portfolioRepo *repository.PortfolioRepository,
	dividendService *DividendService,
	materializedRepo *repository.MaterializedRepository,
	transactionService *TransactionService,
	fundService *FundService,
	realizedGainLossService *RealizedGainLossService,
) *PortfolioService {
	return &PortfolioService{
		portfolioRepo:           portfolioRepo,
		dividendService:         dividendService,
		materializedRepo:        materializedRepo,
		transactionService:      transactionService,
		fundService:             fundService,
		realizedGainLossService: realizedGainLossService,
	}
}

// GetAllPortfolios retrieves all portfolios from the database with no filters applied.
// This includes both archived and excluded portfolios.
func (s *PortfolioService) GetAllPortfolios() ([]model.Portfolio, error) {
	return s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: true,
		IncludeExcluded: true,
	})
}

// TransactionMetrics aggregates calculated metrics from processing transactions for a specific date.
// It is used internally by processTransactionsForDate to return all computed values in a single struct.
type TransactionMetrics struct {
	TotalShares    float64 // Total number of shares held
	TotalCost      float64 // Total cost basis
	TotalValue     float64 // Total market value
	TotalDividends float64 // Total dividend amounts
	TotalFees      float64 // Total fees paid
}

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
// Data Source:
// The materialized view table (portfolio_history_materialized) contains pre-calculated snapshots
// that are updated whenever portfolio data changes. This eliminates the need to:
//   - Load all transactions, dividends, and prices
//   - Iterate through every date in the range
//   - Recalculate share counts and valuations
//
// Parameters:
//   - requestedStartDate: First date to include in returned results
//   - requestedEndDate: Last date to include in returned results
//   - portfolioID: Optional portfolio ID. If empty, returns all active portfolios.
//     If specified, returns only the specified portfolio.
//
// Returns:
// A slice of PortfolioHistory structs, one per date, each containing portfolio summaries for that date.
// Only dates with existing data in the materialized view are included in the result.
//
// Note:
// The materialized view must be kept up-to-date by the data ingestion process. If the view is stale,
// results may not reflect the most recent transactions. Use GetPortfolioHistory() for guaranteed
// real-time accuracy at the cost of performance.
func (s *PortfolioService) GetPortfolioHistoryMaterialized(requestedStartDate, requestedEndDate time.Time, portfolioID string) ([]model.PortfolioHistory, error) {
	var portfolios []model.Portfolio
	var err error

	if portfolioID != "" {
		// Load single portfolio
		portfolio, err := s.GetPortfolio(portfolioID)
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
// The requested date range is used only to filter which calculated results are returned,
// not which data is loaded for calculations.
//
// Parameters:
//   - requestedStartDate: First date to include in returned results
//   - requestedEndDate: Last date to include in returned results
//   - portfolioID: Optional portfolio ID. If empty, returns all active portfolios.
//     If specified, returns only the specified portfolio.
//
// The actual returned range will be clamped to:
//   - Start: max(requestedStartDate, oldestTransactionDate)
//   - End: min(requestedEndDate, today)
func (s *PortfolioService) GetPortfolioHistory(requestedStartDate, requestedEndDate time.Time, portfolioID string) ([]model.PortfolioHistory, error) {

	var portfolios []model.Portfolio
	var err error

	if portfolioID != "" {
		// Load single portfolio
		portfolio, err := s.GetPortfolio(portfolioID)
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

	_, portfolioFundToPortfolio, portfolioFundToFund, pfIDs, fundIDs, err := s.loadAllPortfolioFunds(portfolios)
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
//
// Since write operations trigger materialized view regeneration, the only check needed
// is whether the result set is empty (table being rebuilt or never populated).
func (s *PortfolioService) GetPortfolioHistoryWithFallback(
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

// GetPortfolio retrieves a single portfolio by its ID.
// Returns the portfolio metadata including name, description, and archive status.
// This is a simple wrapper around the repository layer for portfolio lookup.
func (s *PortfolioService) GetPortfolio(portfolioID string) (model.Portfolio, error) {
	result, err := s.portfolioRepo.GetPortfolioOnID(portfolioID)
	if err != nil {
		return model.Portfolio{}, err
	}
	return result, nil
}

//
// DATA LOADING FUNCTIONS
//
// These functions load data from repositories for portfolio calculations.
// Functions prefixed with "loadAll" retrieve complete historical datasets,
// while others accept date range parameters.
//

// loadActivePortfolios retrieves only active, non-excluded portfolios.
// Archived and excluded portfolios are filtered out.
func (s *PortfolioService) loadActivePortfolios() ([]model.Portfolio, error) {
	return s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: false,
		IncludeExcluded: false,
	})
}

// loadAllPortfolioFunds retrieves all funds associated with the given portfolios.
// Returns:
//   - fundsByPortfolio: map[portfolioID][]Fund
//   - portfolioFundToPortfolio: map[portfolioFundID]portfolioID
//   - portfolioFundToFund: map[portfolioFundID]fundID
//   - pfIDs: slice of all portfolio_fund IDs
//   - fundIDs: slice of all unique fund IDs
//   - error: any error encountered
func (s *PortfolioService) loadAllPortfolioFunds(portfolios []model.Portfolio) (map[string][]model.Fund, map[string]string, map[string]string, []string, []string, error) {
	return s.portfolioRepo.GetPortfolioFundsOnPortfolioID(portfolios)
}

//
// CALCULATION FUNCTIONS
//
// These functions perform time-aware calculations on portfolio data.
// All functions that accept a date parameter compute values AS OF that date,
// including only records that occurred on or before the specified date.
//

// processTransactionsForDate calculates portfolio metrics as of the specified date.
// It processes all transactions that occurred on or before the target date to compute:
//   - Total shares held (including buy, sell, and dividend reinvestment transactions)
//   - Total cost basis
//   - Total market value (using the most recent price on or before the date)
//   - Total dividends
//   - Total fees
//
// Transaction Processing Logic:
//   - "buy": Increases shares and cost
//   - "sell": Decreases shares and adjusts cost basis proportionally
//   - "dividend": Adds to dividend total
//   - "fee": Adds to both cost and fees
//
// The function ensures all totals are non-negative before returning.
func (s *PortfolioService) processTransactionsForDate(transactionsMap map[string][]model.Transaction, dividendShares map[string]float64, fundMapping map[string]string, fundPriceByFund map[string][]model.FundPrice, date time.Time) (TransactionMetrics, error) {
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

	transactionMetrics := TransactionMetrics{
		TotalShares:    totalShares,
		TotalCost:      totalCost,
		TotalValue:     totalValue,
		TotalDividends: totalDividends,
		TotalFees:      totalFees,
	}

	return transactionMetrics, nil
}
