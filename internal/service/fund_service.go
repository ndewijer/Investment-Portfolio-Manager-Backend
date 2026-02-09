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
)

// FundService handles fund-related business logic operations.
type FundService struct {
	fundRepo                *repository.FundRepository
	transactionService      *TransactionService
	dividendService         *DividendService
	realizedGainLossService *RealizedGainLossService
	dataLoaderService       *DataLoaderService
	portfolioService        *PortfolioService
}

// NewFundService creates a new FundService with the provided repository dependencies.
func NewFundService(
	fundRepo *repository.FundRepository,
	transactionService *TransactionService,
	dividendService *DividendService,
	realizedGainLossService *RealizedGainLossService,
	dataLoaderService *DataLoaderService,
	portfolioService *PortfolioService,
) *FundService {
	return &FundService{
		fundRepo:                fundRepo,
		transactionService:      transactionService,
		dividendService:         dividendService,
		realizedGainLossService: realizedGainLossService,
		dataLoaderService:       dataLoaderService,
		portfolioService:        portfolioService,
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

	// Check if fund is in use by any portfolios
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
