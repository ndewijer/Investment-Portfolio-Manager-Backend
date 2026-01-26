package service

import (
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// DividendService handles dividend-related business logic operations.
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

// GetAllDividends retrieves all dividend records from the database.
// Returns raw dividend data without fund enrichment.
func (s *DividendService) GetAllDividends() ([]model.Dividend, error) {
	return s.dividendRepo.GetDividend()
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
