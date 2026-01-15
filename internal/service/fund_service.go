package service

import (
	"errors"
	"fmt"
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

type FundMetrics struct {
	PortfolioFundID string
	FundID          string
	Shares          float64
	Cost            float64
	LatestPrice     float64
	Dividend        float64
	Value           float64
	UnrealizedGain  float64
	Fees            float64
}

// GetAllPortfolios retrieves all portfolios from the database with no filters applied.
// This includes both archived and excluded portfolios.
func (s *FundService) GetAlFunds() ([]model.Fund, error) {
	return s.fundRepo.GetFunds()
}

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

	dividendsByPF, err := s.dividendService.loadDividend(pfIDs, oldestTransactionDate, today)
	if err != nil {
		return nil, err
	}

	fundPriceByFund, err := s.loadFundPrices(fundIDs, oldestTransactionDate, today, "ASC")
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

func (s *FundService) GetPortfolioFundMetrics() {

}

// LoadFundPrices retrieves fund prices for the given fund IDs within the specified date range.
// Prices are sorted flexibility based on need. (ASC or DESC)
// Results are grouped by fund ID.
func (s *FundService) loadFundPrices(fundIDs []string, startDate, endDate time.Time, sortOrder string) (map[string][]model.FundPrice, error) {
	return s.fundRepo.GetFundPrice(fundIDs, startDate, endDate, sortOrder)
}

func (s *FundService) calculateFundMetrics(
	pfID string,
	fundID string,
	date time.Time,
	transactions []model.Transaction,
	dividendShares float64,
	fundPrices []model.FundPrice,
	useLatestPrice bool,
) (FundMetrics, error) {

	var totalValue, shares, cost, dividends, value, fees float64
	shares = dividendShares

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
				return FundMetrics{}, fmt.Errorf(": %w", err)
			}
		} else {
			break
		}
	}
	latestPrice := 0.0
	if len(fundPrices) > 0 {
		if useLatestPrice {
			latestPrice = s.getLatestPrice(fundPrices)
		} else {
			latestPrice = s.getPriceForDate(fundPrices, date)
		}
		if latestPrice > 0 {
			value = shares * latestPrice
			totalValue += value
		}
	}

	return FundMetrics{
		PortfolioFundID: pfID,
		FundID:          fundID,
		Shares:          shares,
		Cost:            cost,
		LatestPrice:     latestPrice,
		Dividend:        dividends,
		Value:           value,
		UnrealizedGain:  value - cost,
		Fees:            fees,
	}, nil
}

// GetPriceForDate finds the most recent fund price on or before the target date.
// Assumes prices are sorted in ASC order (oldest first).
// Returns 0 if no price is found on or before the target date.
func (s *FundService) getPriceForDate(prices []model.FundPrice, targetDate time.Time) float64 {
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

// GetLatestPrice returns the most recent price available regardless of date.
// Assumes prices are sorted in ASC order (oldest first).
// Returns 0 if the prices slice is empty.
func (s *FundService) getLatestPrice(prices []model.FundPrice) float64 {
	if len(prices) == 0 {
		return 0
	}
	return prices[len(prices)-1].Price
}
