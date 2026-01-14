package service

import (
	"errors"
	"fmt"
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
	portfolioRepo        *repository.PortfolioRepository
	transactionRepo      *repository.TransactionRepository
	fundRepo             *repository.FundRepository
	dividendRepo         *repository.DividendRepository
	realizedGainLossRepo *repository.RealizedGainLossRepository
	materializedRepo     *repository.MaterializedRepository
}

// NewPortfolioService creates a new PortfolioService with the provided repository dependencies.
// All repository parameters are required for proper portfolio calculations.
func NewPortfolioService(
	portfolioRepo *repository.PortfolioRepository,
	transactionRepo *repository.TransactionRepository,
	fundRepo *repository.FundRepository,
	dividendRepo *repository.DividendRepository,
	realizedGainLossRepo *repository.RealizedGainLossRepository,
	materializedRepo *repository.MaterializedRepository,
) *PortfolioService {
	return &PortfolioService{
		portfolioRepo:        portfolioRepo,
		transactionRepo:      transactionRepo,
		fundRepo:             fundRepo,
		dividendRepo:         dividendRepo,
		realizedGainLossRepo: realizedGainLossRepo,
		materializedRepo:     materializedRepo,
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

// GetPortfolioSummary retrieves the current summary for all active portfolios.
// This is implemented as a wrapper around GetPortfolioHistory for a single day (today),
// ensuring consistency between summary and history calculations.
//
// Parameters:
//   - portfolioID: Optional portfolio ID. If empty, returns all active portfolios.
//     If specified, returns only the specified portfolio.
func (s *PortfolioService) GetPortfolioSummary(portfolioID string) ([]model.PortfolioSummary, error) {

	today := time.Now()

	history, err := s.GetPortfolioHistory(today, today, portfolioID)
	if err != nil {
		return nil, err
	}

	if len(history) != 1 {
		return []model.PortfolioSummary{}, nil
	}

	return history[0].Portfolios, nil
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

	_, portfolioFundToPortfolio, portfolioFundToFund, pfIDs, fundIDs, err := s.loadAllPortfolioFunds(portfolios)
	if err != nil {
		return nil, err
	}

	oldestTransactionDate := s.getOldestTransaction(pfIDs)

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

	transactionsByPortfolio, err := s.loadTransactions(pfIDs, portfolioFundToPortfolio, dataStartDate, dataEndDate)
	if err != nil {
		return nil, err
	}

	dividendByPortfolio, err := s.loadDividend(pfIDs, portfolioFundToPortfolio, dataStartDate, dataEndDate)
	if err != nil {
		return nil, err
	}

	fundPriceByFund, err := s.loadFundPrices(fundIDs, dataStartDate, dataEndDate, "ASC")
	if err != nil {
		return nil, err
	}

	realizedGainLossByPortfolio, err := s.loadRealizedGainLoss(portfolios, dataStartDate, dataEndDate)
	if err != nil {
		return nil, err
	}

	dividendsByPFByPortfolio := make(map[string]map[string][]model.Dividend)
	transactionsByPFByPortfolio := make(map[string]map[string][]model.Transaction)
	for _, portfolio := range portfolios {
		dividendsByPF := make(map[string][]model.Dividend)
		for _, div := range dividendByPortfolio[portfolio.ID] {
			dividendsByPF[div.PortfolioFundID] = append(dividendsByPF[div.PortfolioFundID], div)
		}
		dividendsByPFByPortfolio[portfolio.ID] = dividendsByPF

		transactionsByPF := make(map[string][]model.Transaction)
		for _, tx := range transactionsByPortfolio[portfolio.ID] {
			transactionsByPF[tx.PortfolioFundID] = append(transactionsByPF[tx.PortfolioFundID], tx)
		}
		transactionsByPFByPortfolio[portfolio.ID] = transactionsByPF
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

			totalDividendSharesPerPF, err := s.processDividendSharesForDate(dividendsByPF, transactionsByPortfolio[portfolio.ID], date)
			if err != nil {
				return nil, err
			}

			transactionMetrics, err := s.processTransactionsForDate(transactionsByPF, totalDividendSharesPerPF, portfolioFundToFund, fundPriceByFund, date)
			if err != nil {
				return nil, err
			}

			totalDividendAmount, err := s.processDividendAmountForDate(dividendByPortfolio[portfolio.ID], date)
			if err != nil {
				return nil, err
			}
			totalRealizedGainLoss, totalSaleProceeds, totalCostBasis, err := s.processRealizedGainLossForDate(realizedGainLossByPortfolio[portfolio.ID], date)
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

// getOldestTransaction returns the date of the earliest transaction across the given portfolio_fund IDs.
// This is used to determine the earliest date for which portfolio calculations can be performed.
func (s *PortfolioService) getOldestTransaction(pfIDs []string) time.Time {
	return s.transactionRepo.GetOldestTransaction(pfIDs)
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

// loadTransactions retrieves transactions for the given portfolio_fund IDs within the specified date range.
// Results are grouped by portfolio ID.
func (s *PortfolioService) loadTransactions(pfIDs []string, portfolioFundToPortfolio map[string]string, startDate, endDate time.Time) (map[string][]model.Transaction, error) {
	return s.transactionRepo.GetTransactions(pfIDs, portfolioFundToPortfolio, startDate, endDate)
}

// loadDividend retrieves dividends for the given portfolio_fund IDs within the specified date range.
// Results are grouped by portfolio ID.
func (s *PortfolioService) loadDividend(pfIDs []string, portfolioFundToPortfolio map[string]string, startDate, endDate time.Time) (map[string][]model.Dividend, error) {
	return s.dividendRepo.GetDividend(pfIDs, portfolioFundToPortfolio, startDate, endDate)
}

// loadFundPrices retrieves fund prices for the given fund IDs within the specified date range.
// Prices are sorted flexibility based on need. (ASC or DESC)
// Results are grouped by fund ID.
func (s *PortfolioService) loadFundPrices(fundIDs []string, startDate, endDate time.Time, sortOrder string) (map[string][]model.FundPrice, error) {
	return s.fundRepo.GetFundPrice(fundIDs, startDate, endDate, sortOrder)
}

// loadRealizedGainLoss retrieves realized gain/loss records for the given portfolios within the specified date range.
// Results are grouped by portfolio ID.
func (s *PortfolioService) loadRealizedGainLoss(portfolio []model.Portfolio, startDate, endDate time.Time) (map[string][]model.RealizedGainLoss, error) {
	return s.realizedGainLossRepo.GetRealizedGainLossByPortfolio(portfolio, startDate, endDate)
}

//
// CALCULATION FUNCTIONS
//
// These functions perform time-aware calculations on portfolio data.
// All functions that accept a date parameter compute values AS OF that date,
// including only records that occurred on or before the specified date.
//

// processRealizedGainLossForDate calculates cumulative realized gains/losses as of the specified date.
// Only realized gains from sales that occurred on or before the target date are included.
// Returns (totalRealizedGainLoss, totalSaleProceeds, totalCostBasis, error).
func (s *PortfolioService) processRealizedGainLossForDate(realizedGainLoss []model.RealizedGainLoss, date time.Time) (float64, float64, float64, error) {
	if len(realizedGainLoss) == 0 {
		return 0.0, 0.0, 0.0, nil
	}
	var totalRealizedGainLoss, totalSaleProceeds, totalCostBasis float64

	for _, r := range realizedGainLoss {
		if r.TransactionDate.Before(date) || r.TransactionDate.Equal(date) {
			totalRealizedGainLoss += r.RealizedGainLoss
			totalSaleProceeds += r.SaleProceeds
			totalCostBasis += r.CostBasis
		} else {
			break
		}
	}

	return totalRealizedGainLoss, totalSaleProceeds, totalCostBasis, nil
}

// processDividendSharesForDate calculates shares acquired through dividend reinvestment as of the specified date.
// Only dividends with ex-dividend dates on or before the target date are included.
// Returns a map of portfolio_fund ID to total reinvested shares.
func (s *PortfolioService) processDividendSharesForDate(dividendMap map[string][]model.Dividend, transactions []model.Transaction, date time.Time) (map[string]float64, error) {
	totalDividendMap := make(map[string]float64)

	for pfID, dividend := range dividendMap {
		var dividendShares float64

		for _, div := range dividend {
			if div.ExDividendDate.Before(date) || div.ExDividendDate.Equal(date) {
				if div.ReinvestmentTransactionId != "" {
					// Find the transaction with this ID
					for _, transaction := range transactions {
						if transaction.ID == div.ReinvestmentTransactionId {
							dividendShares += transaction.Shares
							break
						}
					}
				}
			} else {
				break
			}
		}
		totalDividendMap[pfID] = dividendShares
	}

	return totalDividendMap, nil
}

// processDividendAmountForDate calculates the cumulative dividend amount as of the specified date.
// Only dividends with ex-dividend dates on or before the target date are included.
func (s *PortfolioService) processDividendAmountForDate(dividend []model.Dividend, date time.Time) (float64, error) {
	if len(dividend) == 0 {
		return 0.0, nil
	}
	var totalDividend float64

	for _, d := range dividend {

		if d.ExDividendDate.Before(date) || d.ExDividendDate.Equal(date) {
			totalDividend += d.TotalAmount
		} else {
			break
		}
	}

	return totalDividend, nil
}

// getPriceForDate finds the most recent fund price on or before the target date.
// Assumes prices are sorted in ASC order (oldest first).
// Returns 0 if no price is found on or before the target date.
func (s *PortfolioService) getPriceForDate(prices []model.FundPrice, targetDate time.Time) float64 {
	var latestPrice float64 = 0

	// Prices are sorted ASC, so iterate forward
	for _, price := range prices {
		if price.Date.Before(targetDate) || price.Date.Equal(targetDate) {
			latestPrice = price.Price // Keep updating with more recent prices
		} else {
			break // We've passed the target date, stop
		}
	}

	return latestPrice
}

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
		var shares, cost, dividends, value, fees float64
		shares = dividendShares[pfID]

		for _, transaction := range transactions {

			if transaction.Date.Before(date) || transaction.Date.Equal(date) {

				switch transaction.Type {
				case "buy":
					shares += transaction.Shares
					cost += transaction.Shares * transaction.CostPerShare
				case "dividend":
					dividends += transaction.Shares * transaction.CostPerShare
				case "sell":
					shares -= transaction.Shares
					if shares > 0.0 {
						cost = (cost / (shares + transaction.Shares)) * shares
					} else {
						cost = 0.0
					}
				case "fee":
					cost += transaction.CostPerShare
					fees += transaction.CostPerShare
				default:
					err := errors.New("Unknown transaction type.")
					return TransactionMetrics{}, fmt.Errorf(": %w", err)
				}
			} else {
				break
			}
		}
		fundID := fundMapping[pfID]
		prices := fundPriceByFund[fundID]

		if len(prices) > 0 {
			latestPrice := s.getPriceForDate(prices, date)
			if latestPrice > 0 {
				value = shares * latestPrice
				totalValue += value
			}
		}

		totalShares += shares
		totalCost += cost
		totalDividends += dividends
		totalFees += fees
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
