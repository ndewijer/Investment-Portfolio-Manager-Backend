package service

import (
	"errors"
	"fmt"
	"math"
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

	fundsByPortfolio, portfolioFundToPortfolio, portfolioFundToFund, pfIDs, fundIDs, err := s.loadPortfolioFunds(portfolios)
	if err != nil {
		return nil, err
	}

	transactionsByPortfolio, err := s.loadTransactions(pfIDs, portfolioFundToPortfolio)
	if err != nil {
		return nil, err
	}

	dividendByPortfolio, err := s.loadDividend(pfIDs, portfolioFundToPortfolio)
	if err != nil {
		return nil, err
	}

	fundPriceByFund, err := s.loadFundPrices(fundIDs)
	if err != nil {
		return nil, err
	}

	realizedGainLossByPortfolio, err := s.loadRealizedGainLoss(portfolios)
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

		// fundPricebyPF := make(map[string][]model.FundPrice)
		// for f, fp := range fundPriceByFund {
		// 	fundPricebyPF[fundtoPortfolioFund[f]] = append(fundPricebyPF[fundtoPortfolioFund[f]], fp)
		// }

		totalDividendSharesPerPF, err := s.processDividendSharesForDate(dividendsByPF, transactionsByPortfolio[portfolio.ID])
		if err != nil {
			return nil, err
		}

		_, totalCost, totalValue, _, _, err := s.processTransactionsForDate(transactionsByPF, totalDividendSharesPerPF, portfolioFundToFund, fundPriceByFund, time.Now())
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
			TotalValue:              math.Round(totalValue*1e6) / 1e6,
			TotalCost:               math.Round(totalCost*1e6) / 1e6,
			TotalDividends:          math.Round(totalDividendAmount*1e6) / 1e6,
			TotalUnrealizedGainLoss: math.Round((totalValue-totalCost)*1e6) / 1e6,
			TotalRealizedGainLoss:   math.Round(totalRealizedGainLoss*1e6) / 1e6,
			TotalSaleProceeds:       math.Round(totalSaleProceeds*1e6) / 1e6,
			TotalOriginalCost:       math.Round(totalCostBasis*1e6) / 1e6,
			TotalGainLoss:           math.Round((totalRealizedGainLoss+(totalValue-totalCost))*1e6) / 1e6,
			IsArchived:              portfolio.IsArchived,
		}

		portfolioSummary = append(portfolioSummary, p)
	}

	// TODO: Calculate metrics and return summaries
	// For now, just return empty summaries
	_ = fundsByPortfolio
	_ = fundPriceByFund

	return portfolioSummary, nil
}

// loadActivePortfolios retrieves only active, non-excluded portfolios
func (s *PortfolioService) loadActivePortfolios() ([]model.Portfolio, error) {
	return s.portfolioRepo.GetPortfolios(model.PortfolioFilter{
		IncludeArchived: false,
		IncludeExcluded: false,
	})
}

// loadPortfolioFunds retrieves all funds for the given portfolios
// Returns: fundsByPortfolio map, portfolioFundToPortfolio map, pfIDs slice, error
func (s *PortfolioService) loadPortfolioFunds(portfolios []model.Portfolio) (map[string][]model.Fund, map[string]string, map[string]string, []string, []string, error) {
	return s.portfolioRepo.GetPortfolioFundsOnPortfolioID(portfolios)
}

// loadTransactions retrieves all transactions for the given portfolio_fund IDs
func (s *PortfolioService) loadTransactions(pfIDs []string, portfolioFundToPortfolio map[string]string) (map[string][]model.Transaction, error) {
	return s.transactionRepo.GetTransactions(pfIDs, portfolioFundToPortfolio)
}

// loadDividend retrieves all dividend for the given portfolio_fund IDs
func (s *PortfolioService) loadDividend(pfIDs []string, portfolioFundToPortfolio map[string]string) (map[string][]model.Dividend, error) {
	return s.dividendRepo.GetDividend(pfIDs, portfolioFundToPortfolio)
}

// loadDividend retrieves all dividend for the given portfolio_fund IDs
func (s *PortfolioService) loadFundPrices(fundIDs []string) (map[string][]model.FundPrice, error) {
	return s.fundRepo.GetFundPrice(fundIDs)
}

func (s *PortfolioService) loadRealizedGainLoss(portfolio []model.Portfolio) (map[string][]model.RealizedGainLoss, error) {
	return s.realizedGainLossRepo.GetRealizedGainLossByPortfolio(portfolio)
}

func (s *PortfolioService) processRealizedGainLossForDate(realizedGainLoss []model.RealizedGainLoss, date time.Time) (float64, float64, float64, error) {
	if len(realizedGainLoss) == 0 {
		err := errors.New("RealizedGainLoss is empty")
		return 0.0, 0.0, 0.0, fmt.Errorf(": %w", err)
	}
	var totalRealizedGainLoss, totalSaleProceeds, totalCostBasis float64

	for _, r := range realizedGainLoss {
		totalRealizedGainLoss += r.RealizedGainLoss
		totalSaleProceeds += r.SaleProceeds
		totalCostBasis += r.CostBasis
	}

	return totalRealizedGainLoss, totalSaleProceeds, totalCostBasis, nil
}

func (s *PortfolioService) processDividendSharesForDate(dividendMap map[string][]model.Dividend, transactions []model.Transaction) (map[string]float64, error) {
	if len(dividendMap) == 0 {
		err := errors.New("Dividend is empty")
		return nil, fmt.Errorf(": %w", err)
	}
	totalDividendMap := make(map[string]float64)
	for pfID, dividend := range dividendMap {
		var dividendShares float64
		for _, div := range dividend {
			if div.ReinvestmentTransactionId != "" {
				// Find the transaction with this ID
				for _, transaction := range transactions {
					if transaction.ID == div.ReinvestmentTransactionId {
						dividendShares += transaction.Shares
						break
					}
				}
			}
		}
		totalDividendMap[pfID] = dividendShares
	}

	return totalDividendMap, nil
}

func (s *PortfolioService) processDividendAmountForDate(dividend []model.Dividend, date time.Time) (float64, error) {
	if len(dividend) == 0 {
		err := errors.New("Dividend is empty")
		return 0.0, fmt.Errorf(": %w", err)
	}
	var totalDividend float64

	for _, d := range dividend {
		totalDividend += d.TotalAmount
	}

	return totalDividend, nil
}

func (s *PortfolioService) processTransactionsForDate(transactionsMap map[string][]model.Transaction, dividendShares map[string]float64, fundMapping map[string]string, fundPriceByFund map[string][]model.FundPrice, date time.Time) (float64, float64, float64, float64, float64, error) {
	if len(transactionsMap) == 0 {
		err := errors.New("transactions is empty")
		return 0.0, 0.0, 0.0, 0.0, 0.0, fmt.Errorf(": %w", err)
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
					return 0.0, 0.0, 0.0, 0.0, 0.0, fmt.Errorf(": %w", err)
				}
			}
		}
		fundID := fundMapping[pfID]
		prices := fundPriceByFund[fundID]

		if len(prices) > 0 {
			latestPrice := prices[0].Price
			value = shares * latestPrice
			totalValue += value
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

	return totalShares, totalCost, totalValue, totalDividends, totalFees, nil
}
