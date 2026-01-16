package service

import (
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// DividendService handles fund-related business logic operations.
type DividendService struct {
	dividendRepo *repository.DividendRepository
}

// NewDividendService creates a new DividendService with the provided repository dependencies.
func NewDividendService(
	dividendRepo *repository.DividendRepository,
) *DividendService {
	return &DividendService{
		dividendRepo: dividendRepo,
	}
}

func (s *DividendService) GetDividendFund(portfolioID string) ([]model.DividendFund, error) {
	dividendFund, err := s.loadDividendPerPortfolioFund(portfolioID)
	if err != nil {
		return nil, err
	}
	return dividendFund, nil
}

// LoadDividend retrieves dividends for the given portfolio_fund IDs within the specified date range.
// Results are grouped by portfolio_fund ID, allowing callers to decide how to aggregate.
func (s *DividendService) loadDividend(pfIDs []string, startDate, endDate time.Time) (map[string][]model.Dividend, error) {
	return s.dividendRepo.GetDividend(pfIDs, startDate, endDate)
}

// LoadDividend retrieves dividends for the given portfolio_fund IDs within the specified date range.
// Results are grouped by portfolio_fund ID, allowing callers to decide how to aggregate.
func (s *DividendService) loadDividendPerPortfolioFund(portfolioID string) ([]model.DividendFund, error) {
	return s.dividendRepo.GetDividendPerPortfolioFund(portfolioID)
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
