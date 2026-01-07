package service

import (
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

	fundsByPortfolio, portfolioFundToPortfolio, pfIDs, fundIDs, err := s.loadPortfolioFunds(portfolios)
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

	// TODO: Calculate metrics and return summaries
	// For now, just return empty summaries
	_ = fundsByPortfolio
	_ = transactionsByPortfolio
	_ = dividendByPortfolio
	_ = fundPriceByFund
	_ = realizedGainLossByPortfolio
	return []PortfolioSummary{}, nil
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
func (s *PortfolioService) loadPortfolioFunds(portfolios []model.Portfolio) (map[string][]model.Fund, map[string]string, []string, []string, error) {
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
