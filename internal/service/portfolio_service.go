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

// PortfolioService handles Portfolio-related operations
type PortfolioService struct {
	portfolioRepo        *repository.PortfolioRepository
	transactionRepo      *repository.TransactionRepository
	fundRepo             *repository.FundRepository
	dividendRepo         *repository.DividendRepository
	realizedGainLossRepo *repository.RealizedGainLossRepository
}

func NewPortfolioService(
	portfolioRepo *repository.PortfolioRepository,
	transactionRepo *repository.TransactionRepository,
	fundRepo *repository.FundRepository,
	dividendRepo *repository.DividendRepository,
	realizedGainLossRepo *repository.RealizedGainLossRepository,
) *PortfolioService {
	return &PortfolioService{
		portfolioRepo:        portfolioRepo,
		transactionRepo:      transactionRepo,
		fundRepo:             fundRepo,
		dividendRepo:         dividendRepo,
		realizedGainLossRepo: realizedGainLossRepo,
	}
}

// GetAllPortfolios retrieves all portfolios from the database (no filters)
func (s *PortfolioService) GetAllPortfolios() ([]model.Portfolio, error) {
	return s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: true,
		IncludeExcluded: true,
	})
}

// Struct for returning TransactionMetrics out off processTransactionsForDate()
type TransactionMetrics struct {
	TotalShares    float64
	TotalCost      float64
	TotalValue     float64
	TotalDividends float64
	TotalFees      float64
}

// The struc that will be returned by GetPortfolioSummary
type PortfolioSummary struct {
	ID                      string
	Name                    string
	TotalValue              float64
	TotalCost               float64
	TotalDividends          float64
	TotalUnrealizedGainLoss float64
	TotalRealizedGainLoss   float64
	TotalSaleProceeds       float64
	TotalOriginalCost       float64
	TotalGainLoss           float64
	IsArchived              bool
}

// Gets portfolio summary from database
func (s *PortfolioService) GetPortfolioSummary() ([]PortfolioSummary, error) {
	// Load data
	portfolios, err := s.loadActivePortfolios()
	if err != nil {
		return nil, err
	}

	_, portfolioFundToPortfolio, portfolioFundToFund, pfIDs, fundIDs, err := s.loadAllPortfolioFunds(portfolios)
	if err != nil {
		return nil, err
	}

	transactionsByPortfolio, err := s.loadAllTransactions(pfIDs, portfolioFundToPortfolio)
	if err != nil {
		return nil, err
	}

	dividendByPortfolio, err := s.loadAllDividend(pfIDs, portfolioFundToPortfolio)
	if err != nil {
		return nil, err
	}

	fundPriceByFund, err := s.loadAllFundPrices(fundIDs)
	if err != nil {
		return nil, err
	}

	realizedGainLossByPortfolio, err := s.loadAllRealizedGainLoss(portfolios)
	if err != nil {
		return nil, err
	}

	portfolioSummary := []PortfolioSummary{}

	for _, portfolio := range portfolios {

		dividendsByPF := make(map[string][]model.Dividend)
		for _, div := range dividendByPortfolio[portfolio.ID] {
			dividendsByPF[div.PortfolioFundID] = append(dividendsByPF[div.PortfolioFundID], div)
		}

		transactionsByPF := make(map[string][]model.Transaction)
		for _, tx := range transactionsByPortfolio[portfolio.ID] {
			transactionsByPF[tx.PortfolioFundID] = append(transactionsByPF[tx.PortfolioFundID], tx)
		}

		totalDividendSharesPerPF, err := s.processDividendSharesForDate(dividendsByPF, transactionsByPortfolio[portfolio.ID], time.Now())
		if err != nil {
			return nil, err
		}

		transactionMetrics, err := s.processTransactionsForDate(transactionsByPF, totalDividendSharesPerPF, portfolioFundToFund, fundPriceByFund, time.Now())
		if err != nil {
			return nil, err
		}
		totalDividendAmount, err := s.processDividendAmountForDate(dividendByPortfolio[portfolio.ID], time.Now())
		if err != nil {
			return nil, err
		}
		totalRealizedGainLoss, totalSaleProceeds, totalCostBasis, err := s.processRealizedGainLossForDate(realizedGainLossByPortfolio[portfolio.ID], time.Now())
		if err != nil {
			return nil, err
		}

		p := PortfolioSummary{
			ID:                      portfolio.ID,
			Name:                    portfolio.Name,
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

		portfolioSummary = append(portfolioSummary, p)
	}

	return portfolioSummary, nil
}

type PortfolioHistory struct {
	Date       string
	Portfolios []PortfolioHistoryPortfolio
}

type PortfolioHistoryPortfolio struct {
	ID             string
	Name           string
	Value          float64
	Cost           float64
	RealizedGain   float64
	UnrealizedGain float64
}

func (s *PortfolioService) GetPortfolioHistory(startDate, endDate time.Time) ([]PortfolioHistory, error) {

	portfolios, err := s.loadActivePortfolios()
	if err != nil {
		return nil, err
	}

	_, portfolioFundToPortfolio, portfolioFundToFund, pfIDs, fundIDs, err := s.loadAllPortfolioFunds(portfolios)
	if err != nil {
		return nil, err
	}

	oldestTransactionDate := s.getOldestTransaction(pfIDs)

	if oldestTransactionDate.After(startDate) {
		startDate = oldestTransactionDate
	}
	if endDate.After(time.Now()) {
		endDate = time.Now()
	}

	transactionsByPortfolio, err := s.loadTransactions(pfIDs, portfolioFundToPortfolio, startDate, endDate)
	if err != nil {
		return nil, err
	}

	dividendByPortfolio, err := s.loadDividend(pfIDs, portfolioFundToPortfolio, startDate, endDate)
	if err != nil {
		return nil, err
	}

	fundPriceByFund, err := s.loadFundPrices(fundIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}

	realizedGainLossByPortfolio, err := s.loadRealizedGainLoss(portfolios, startDate, endDate)
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

	portfolioHistory := []PortfolioHistory{}
	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {

		portfolioHistoryPortfolio := []PortfolioHistoryPortfolio{}

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

			totalRealizedGainLoss, _, _, err := s.processRealizedGainLossForDate(realizedGainLossByPortfolio[portfolio.ID], date)
			if err != nil {
				return nil, err
			}

			php := PortfolioHistoryPortfolio{

				ID:             portfolio.ID,
				Name:           portfolio.Name,
				Value:          math.Round(transactionMetrics.TotalValue*RoundingPrecision) / RoundingPrecision,
				Cost:           math.Round(transactionMetrics.TotalCost*RoundingPrecision) / RoundingPrecision,
				RealizedGain:   math.Round(totalRealizedGainLoss*RoundingPrecision) / RoundingPrecision,
				UnrealizedGain: math.Round((transactionMetrics.TotalValue-transactionMetrics.TotalCost)*RoundingPrecision) / RoundingPrecision,
			}

			portfolioHistoryPortfolio = append(portfolioHistoryPortfolio, php)

		}
		ph := PortfolioHistory{
			Date:       date.Format("2006-01-02"),
			Portfolios: portfolioHistoryPortfolio,
		}
		portfolioHistory = append(portfolioHistory, ph)
	}

	return portfolioHistory, nil
}

//
// SUPPORTING FUNCTIONS
//

// loadActivePortfolios retrieves only active, non-excluded portfolios
func (s *PortfolioService) loadActivePortfolios() ([]model.Portfolio, error) {
	return s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: false,
		IncludeExcluded: false,
	})
}

func (s *PortfolioService) getOldestTransaction(pfIDs []string) time.Time {
	return s.transactionRepo.GetOldestTransaction(pfIDs)
}

// loadPortfolioFunds retrieves all funds for the given portfolios
// Returns: fundsByPortfolio map, portfolioFundToPortfolio map, pfIDs slice, error
func (s *PortfolioService) loadAllPortfolioFunds(portfolios []model.Portfolio) (map[string][]model.Fund, map[string]string, map[string]string, []string, []string, error) {
	return s.portfolioRepo.GetPortfolioFundsOnPortfolioID(portfolios)
}

// loadTransactions retrieves all transactions for the given portfolio_fund IDs
func (s *PortfolioService) loadAllTransactions(pfIDs []string, portfolioFundToPortfolio map[string]string) (map[string][]model.Transaction, error) {
	startDate, _ := time.Parse("2006-01-02", "1970-01-01")
	endDate := time.Now()
	return s.transactionRepo.GetTransactions(pfIDs, portfolioFundToPortfolio, startDate, endDate)
}

func (s *PortfolioService) loadTransactions(pfIDs []string, portfolioFundToPortfolio map[string]string, startDate, endDate time.Time) (map[string][]model.Transaction, error) {
	return s.transactionRepo.GetTransactions(pfIDs, portfolioFundToPortfolio, startDate, endDate)
}

// loadDividend retrieves all dividend for the given portfolio_fund IDs
func (s *PortfolioService) loadAllDividend(pfIDs []string, portfolioFundToPortfolio map[string]string) (map[string][]model.Dividend, error) {
	startDate, _ := time.Parse("2006-01-02", "1970-01-01")
	endDate := time.Now()
	return s.dividendRepo.GetDividend(pfIDs, portfolioFundToPortfolio, startDate, endDate)
}
func (s *PortfolioService) loadDividend(pfIDs []string, portfolioFundToPortfolio map[string]string, startDate, endDate time.Time) (map[string][]model.Dividend, error) {
	return s.dividendRepo.GetDividend(pfIDs, portfolioFundToPortfolio, startDate, endDate)
}

// loadDividend retrieves all dividend for the given portfolio_fund IDs
func (s *PortfolioService) loadAllFundPrices(fundIDs []string) (map[string][]model.FundPrice, error) {
	startDate, _ := time.Parse("2006-01-02", "1970-01-01")
	endDate := time.Now()
	return s.fundRepo.GetFundPrice(fundIDs, startDate, endDate, "desc")
}

func (s *PortfolioService) loadFundPrices(fundIDs []string, startDate, endDate time.Time) (map[string][]model.FundPrice, error) {
	return s.fundRepo.GetFundPrice(fundIDs, startDate, endDate, "asc")
}

func (s *PortfolioService) loadAllRealizedGainLoss(portfolio []model.Portfolio) (map[string][]model.RealizedGainLoss, error) {
	startDate, _ := time.Parse("2006-01-02", "1970-01-01")
	endDate := time.Now()
	return s.realizedGainLossRepo.GetRealizedGainLossByPortfolio(portfolio, startDate, endDate)
}

func (s *PortfolioService) loadRealizedGainLoss(portfolio []model.Portfolio, startDate, endDate time.Time) (map[string][]model.RealizedGainLoss, error) {
	return s.realizedGainLossRepo.GetRealizedGainLossByPortfolio(portfolio, startDate, endDate)
}

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
