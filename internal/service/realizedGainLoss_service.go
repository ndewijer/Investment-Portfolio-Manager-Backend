package service

import (
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/repository"
)

// RealizedGainLossService handles fund-related business logic operations.
type RealizedGainLossService struct {
	realizedgainlossRepo *repository.RealizedGainLossRepository
}

// NewRealizedGainLossService creates a new RealizedGainLossService with the provided repository dependencies.
func NewRealizedGainLossService(
	realizedgainlossRepo *repository.RealizedGainLossRepository,
) *RealizedGainLossService {
	return &RealizedGainLossService{
		realizedgainlossRepo: realizedgainlossRepo,
	}
}

// loadRealizedGainLoss retrieves realized gain/loss records for the given portfolios within the specified date range.
// Results are grouped by portfolio ID.
func (s *RealizedGainLossService) loadRealizedGainLoss(portfolio []string, startDate, endDate time.Time) (map[string][]model.RealizedGainLoss, error) {
	return s.realizedgainlossRepo.GetRealizedGainLossByPortfolio(portfolio, startDate, endDate)
}

// ProcessRealizedGainLossForDate calculates cumulative realized gains/losses as of the specified date.
// Only realized gains from sales that occurred on or before the target date are included.
// Returns (totalRealizedGainLoss, totalSaleProceeds, totalCostBasis, error).
func (s *RealizedGainLossService) processRealizedGainLossForDate(realizedGainLoss []model.RealizedGainLoss, date time.Time) (float64, float64, float64, error) {
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
