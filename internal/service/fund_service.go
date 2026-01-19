package service

import (
	"math"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// FundService handles fund-related business logic operations.
type FundService struct {
	fundRepo                *repository.FundRepository
	transactionService      *TransactionService
	dividendService         *DividendService
	realizedGainLossService *RealizedGainLossService
}

// NewFundService creates a new FundService with the provided repository dependencies.
func NewFundService(
	fundRepo *repository.FundRepository,
	transactionService *TransactionService,
	dividendService *DividendService,
	realizedGainLossService *RealizedGainLossService,
) *FundService {
	return &FundService{
		fundRepo:                fundRepo,
		transactionService:      transactionService,
		dividendService:         dividendService,
		realizedGainLossService: realizedGainLossService,
	}
}

// GetAllFunds retrieves all funds from the database with no filters applied.
// Returns a complete list of all available funds that can be held in portfolios.
func (s *FundService) GetAllFunds() ([]model.Fund, error) {
	return s.fundRepo.GetFunds()
}

// GetAllPortfolioFundListings retrieves all portfolio-fund relationships with basic metadata.
// Returns a listing of funds across all portfolios (non-archived) with portfolio and fund names.
// Used for the GET /api/portfolio/funds endpoint.
func (s *FundService) GetAllPortfolioFundListings() ([]model.PortfolioFundListing, error) {
	return s.fundRepo.GetAllPortfolioFundListings()
}

// GetPortfolioFunds retrieves detailed fund metrics for all funds in a portfolio.
// Returns per-fund breakdowns including shares, cost, value, gains/losses, dividends, and fees.
//
// This method calculates current valuations by:
//   - Loading all historical transactions and dividends from inception to present
//   - Processing dividend reinvestments
//   - Calculating share counts, cost basis, and market value using latest available prices
//   - Computing realized gains from sale transactions
//   - Aggregating dividend payments
//
// Parameters:
//   - portfolioID: The portfolio ID to retrieve funds for. If empty, returns all portfolio funds.
//
// Returns:
// A slice of PortfolioFund structs with populated metrics including totalShares, currentValue,
// unrealizedGainLoss, realizedGainLoss, totalDividends, and totalFees.
// All monetary values are rounded to two decimal places.
func (s *FundService) GetPortfolioFunds(portfolioID string) ([]model.PortfolioFund, error) {

	portfolioFunds, err := s.fundRepo.GetPortfolioFunds(portfolioID)
	if err != nil {
		return nil, err
	}

	if len(portfolioFunds) == 0 {
		return portfolioFunds, nil
	}
	var pfIDs, fundIDs []string
	for _, fund := range portfolioFunds {
		pfIDs = append(pfIDs, fund.ID)
		fundIDs = append(fundIDs, fund.FundId)
	}
	oldestTransactionDate := s.transactionService.getOldestTransaction(pfIDs)
	today := time.Now()

	transactionsByPF, err := s.transactionService.loadTransactions(pfIDs, oldestTransactionDate, today)
	if err != nil {
		return nil, err
	}

	dividendsByPF, err := s.dividendService.loadDividendPerPF(pfIDs, oldestTransactionDate, today)
	if err != nil {
		return nil, err
	}

	fundPriceByFund, err := s.loadFundPrices(fundIDs, oldestTransactionDate, today, true) //ASC
	if err != nil {
		return nil, err
	}

	realizedGainLossByPortfolio, err := s.realizedGainLossService.loadRealizedGainLoss([]string{portfolioID}, oldestTransactionDate, today)
	if err != nil {
		return nil, err
	}

	for i := range portfolioFunds {
		fund := &portfolioFunds[i]

		realizedGainsByPF := make(map[string][]model.RealizedGainLoss)
		for _, entry := range realizedGainLossByPortfolio[portfolioID] {
			if entry.FundID == fund.FundId {
				realizedGainsByPF[fund.ID] = append(realizedGainsByPF[fund.ID], entry)
			}
		}

		totalDividendSharesPerPF, err := s.dividendService.processDividendSharesForDate(dividendsByPF, transactionsByPF[fund.ID], today)
		if err != nil {
			return nil, err
		}

		fundMetrics, err := s.calculateFundMetrics(
			fund.ID, fund.FundId, today, transactionsByPF[fund.ID], totalDividendSharesPerPF[fund.ID], fundPriceByFund[fund.FundId], true)
		if err != nil {
			return nil, err
		}

		totalDividendAmount, err := s.dividendService.processDividendAmountForDate(
			dividendsByPF[fund.ID],
			today,
		)
		if err != nil {
			return nil, err
		}

		// Calculate realized gains for this fund
		totalRealizedGainLoss, _, _, err := s.realizedGainLossService.processRealizedGainLossForDate(
			realizedGainsByPF[fund.ID],
			today,
		)
		if err != nil {
			return nil, err
		}

		roundedShares := math.Round(fundMetrics.Shares*RoundingPrecision) / RoundingPrecision
		averageCost := 0.0
		if roundedShares > 0 {
			averageCost = fundMetrics.Cost / roundedShares
		}

		fund.TotalShares = roundedShares
		fund.LatestPrice = math.Round(fundMetrics.LatestPrice*RoundingPrecision) / RoundingPrecision
		fund.AverageCost = math.Round(averageCost*RoundingPrecision) / RoundingPrecision
		fund.TotalCost = math.Round(fundMetrics.Cost*RoundingPrecision) / RoundingPrecision
		fund.CurrentValue = math.Round(fundMetrics.Value*RoundingPrecision) / RoundingPrecision
		fund.UnrealizedGainLoss = math.Round(fundMetrics.UnrealizedGain*RoundingPrecision) / RoundingPrecision
		fund.RealizedGainLoss = math.Round(totalRealizedGainLoss*RoundingPrecision) / RoundingPrecision
		fund.TotalGainLoss = math.Round((fundMetrics.UnrealizedGain+totalRealizedGainLoss)*RoundingPrecision) / RoundingPrecision
		fund.TotalDividends = math.Round(totalDividendAmount*RoundingPrecision) / RoundingPrecision
		fund.TotalFees = math.Round(fundMetrics.Fees*RoundingPrecision) / RoundingPrecision

	}
	return portfolioFunds, nil
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
func (s *FundService) loadFundPrices(fundIDs []string, startDate, endDate time.Time, ascending bool) (map[string][]model.FundPrice, error) {
	return s.fundRepo.GetFundPrice(fundIDs, startDate, endDate, ascending)
}
