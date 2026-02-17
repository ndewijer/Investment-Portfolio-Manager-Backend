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

// DividendService handles dividend-related business logic operations.
type DividendService struct {
	dividendRepo    *repository.DividendRepository
	fundRepo        *repository.FundRepository
	transactionRepo *repository.TransactionRepository
}

// NewDividendService creates a new DividendService with the provided repository dependencies.
func NewDividendService(
	dividendRepo *repository.DividendRepository,
	fundRepo *repository.FundRepository,
	transactionRepo *repository.TransactionRepository,
) *DividendService {
	return &DividendService{
		dividendRepo:    dividendRepo,
		fundRepo:        fundRepo,
		transactionRepo: transactionRepo,
	}
}

// GetAllDividends retrieves all dividend records from the database.
// Returns raw dividend data without fund enrichment.
func (s *DividendService) GetAllDividend() ([]model.Dividend, error) {
	return s.dividendRepo.GetAllDividend()
}

func (s *DividendService) GetDividend(DividendID string) (model.Dividend, error) {
	return s.dividendRepo.GetDividend(DividendID)
}

// GetDividendFund retrieves all dividend records for a specific portfolio with enriched fund information.
// This is the public service method that returns complete dividend details including fund names,
// dividend types, and reinvestment status for all funds held in the portfolio.
//
// Parameters:
//   - portfolioID: The portfolio ID to retrieve dividends for
//
// Returns a slice of DividendFund containing all historical dividend payments.
func (s *DividendService) GetDividendFund(portfolioID string) ([]model.DividendFund, error) {
	dividendFund, err := s.dividendRepo.GetDividendPerPortfolioFund(portfolioID)
	if err != nil {
		return nil, err
	}
	return dividendFund, nil
}

// loadDividend retrieves dividends for the given portfolio_fund IDs within the specified date range.
// Results are grouped by portfolio_fund ID, allowing callers to decide how to aggregate.
func (s *DividendService) loadDividendPerPF(pfIDs []string, startDate, endDate time.Time) (map[string][]model.Dividend, error) {
	return s.dividendRepo.GetDividendPerPF(pfIDs, startDate, endDate)
}

// ProcessDividendSharesForDate calculates shares acquired through dividend reinvestment as of the specified date.
// Only dividends with ex-dividend dates on or before the target date are included.
// Returns a map of portfolio_fund ID to total reinvested shares.
func (s *DividendService) processDividendSharesForDate(dividendMap map[string][]model.Dividend, transactions []model.Transaction, date time.Time) (map[string]float64, error) {
	totalDividendMap := make(map[string]float64)

	for pfID, dividend := range dividendMap {
		var dividendShares float64

		for _, div := range dividend {
			if div.ExDividendDate.Before(date) || div.ExDividendDate.Equal(date) {
				if div.ReinvestmentTransactionID != "" {
					// Find the transaction with this ID
					for _, transaction := range transactions {
						if transaction.ID == div.ReinvestmentTransactionID {
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

// ProcessDividendAmountForDate calculates the cumulative dividend amount as of the specified date.
// Only dividends with ex-dividend dates on or before the target date are included.
func (s *DividendService) processDividendAmountForDate(dividend []model.Dividend, date time.Time) (float64, error) {
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

// CreateDividend creates a new dividend with the provided details.
// Generates a new UUID for the dividend and inserts it into the database.
//
// Note: Once a dividend is used in a portfolio (has transactions), it becomes permanent
// and cannot be deleted. This preserves portfolio history and dividend price data.
// Only delete unused dividends (e.g., created by mistake).
//
// Parameters:
//   - ctx: Context for the operation
//   - req: CreateDividendRequest containing all required dividend fields
//
// Returns the created dividend with its generated ID, or an error if creation fails.
func (s *DividendService) CreateDividend(ctx context.Context, req request.CreateDividendRequest) (*model.Dividend, error) {

	pfs, err := s.fundRepo.GetAllPortfolioFundListings()
	if err != nil {
		return &model.Dividend{}, err
	}

	var portfolioFund model.PortfolioFundListing
	for _, v := range pfs {
		if v.ID == req.PortfolioFundID {
			portfolioFund = v
		}
	}
	if portfolioFund.ID == "" {
		return &model.Dividend{}, apperrors.ErrFailedToRetrievePortfolioFunds
	}

	if portfolioFund.DividendType == "None" {
		return &model.Dividend{}, fmt.Errorf("this fund does not pay out dividends")
	}

	recordDate, err := time.Parse("2006-01-02", req.RecordDate)
	if err != nil {
		return nil, err
	}

	exDividendDate, err := time.Parse("2006-01-02", req.ExDividendDate)
	if err != nil {
		return nil, err
	}

	// so, transactions and dividends are closely linked. if we have buyorder info, we need to make a trnsaction next to the dividend.
	// if not, just the dividend.

	var buyOrderDate time.Time
	var reinvestmentStatus, reinvestmentTransactionID string
	// need to figure out the logic.
	// var reinvestmentTransactionID float64
	if req.BuyOrderDate != "" {
		buyOrderDate, err = time.Parse("2006-01-02", req.BuyOrderDate)
		if err != nil {
			return nil, err
		}
		// so we have a buyorder date. Do we trust the validation for the rest to be filled on frontend? validation will let this pass so, no.
		if portfolioFund.DividendType == "STOCK" && req.ReinvestmentPrice > 0.0 && req.ReinvestmentShares > 0.0 {
			// lets make a transaction.

			// depending on full price or not, we set
			// reinvestmentStatus = "COMPLETED" or "PARTIAL"
			// reinvestmentTransactionID = whateverIDItIs.
		} else if portfolioFund.DividendType != "STOCK" && req.ReinvestmentPrice > 0.0 && req.ReinvestmentShares > 0.0 {
			// Info set, but the stock is cash, so no transaction required.
			//  reinvestmentStatus = "COMPLETED"
			//  reinvestmentTransactionID = ""
		} else {
			// not enough info to make a transaction. We don't hve to bomb, but we'll have to set status to pending.
			// enum will be ["PENDING", "COMPLETED", "PARTIAL"]. So PENDING.
			// reinvestmentStatus = "PENDING"
			// reinvestmentTransactionID = "" -- null
		}
	}

	shares, err := s.transactionRepo.GetSharesOnDate(req.PortfolioFundID, exDividendDate)
	totalAmount := shares * req.DividendPerShare

	// we also need to calculate SharesOwned and TotalAmount. What's smart, here? just query the PortfolioHistory/MaterializedView here?
	// those are based on portfolio so something will need to be modified.

	dividend := &model.Dividend{
		ID:                        uuid.New().String(),
		FundID:                    portfolioFund.FundID,
		PortfolioFundID:           req.PortfolioFundID,
		RecordDate:                recordDate,
		ExDividendDate:            exDividendDate,
		DividendPerShare:          req.DividendPerShare,
		SharesOwned:               shares,
		TotalAmount:               totalAmount,
		ReinvestmentStatus:        reinvestmentStatus,
		BuyOrderDate:              buyOrderDate,
		ReinvestmentTransactionID: reinvestmentTransactionID,
		CreatedAt:                 time.Now().UTC(),
	}

	if err := s.dividendRepo.InsertDividend(ctx, dividend); err != nil {
		return nil, fmt.Errorf("failed to create dividend: %w", err)
	}

	return dividend, nil
}
