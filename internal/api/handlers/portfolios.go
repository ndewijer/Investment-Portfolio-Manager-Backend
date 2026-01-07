package handlers

import (
	"net/http"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
)

// PortfolioHandler handles portfolio-related HTTP requests
type PortfolioHandler struct {
	portfolioService *service.PortfolioService
}

// NewPortfolioHandler creates a new PortfolioHandler
func NewPortfolioHandler(portfolioService *service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{
		portfolioService: portfolioService,
	}
}

// PortfoliosResponse represents the Portfolios get response
type PortfoliosResponse struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	IsArchived          bool   `json:"is_archived"`
	ExcludeFromOverview bool   `json:"exclude_from_overview"`
}

// Gets basic Portfolios information
func (h *PortfolioHandler) Portfolios(w http.ResponseWriter, r *http.Request) {

	portfolios, err := h.portfolioService.GetAllPortfolios()
	if err != nil {
		errorResponse := map[string]string{
			"error": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	response := make([]PortfoliosResponse, len(portfolios))
	for i, p := range portfolios {
		response[i] = PortfoliosResponse{
			ID:                  p.ID,
			Name:                p.Name,
			Description:         p.Description,
			IsArchived:          p.IsArchived,
			ExcludeFromOverview: p.ExcludeFromOverview,
		}
	}

	respondJSON(w, http.StatusOK, response)
}

type PortfolioSummaryResponse struct {
	ID                      string  `json:"id"`
	Name                    string  `json:"name"`
	TotalValue              float64 `json:"totalValue"`
	TotalCost               float64 `json:"totalCost"`
	TotalDividends          float64 `json:"totalDividends"`
	TotalUnrealizedGainLoss float64 `json:"totalUnrealizedGainLoss"`
	TotalRealizedGainLoss   float64 `json:"totalRealizedGainLoss"`
	TotalSaleProceeds       float64 `json:"totalSaleProceeds"`
	TotalOriginalCost       float64 `json:"totalOriginalCost"`
	TotalGainLoss           float64 `json:"totalGainLoss"`
	IsArchived              bool    `json:"is_archived"`
}

func (h *PortfolioHandler) PortfolioSummary(w http.ResponseWriter, r *http.Request) {
	portfolioSummary, err := h.portfolioService.GetPortfolioSummary()
	if err != nil {
		errorResponse := map[string]string{
			"error": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	response := make([]PortfolioSummaryResponse, len(portfolioSummary))
	for i, p := range portfolioSummary {
		response[i] = PortfolioSummaryResponse{
			ID:                      p.ID,
			Name:                    p.Name,
			TotalValue:              p.TotalValue,
			TotalCost:               p.TotalCost,
			TotalDividends:          p.TotalDividends,
			TotalUnrealizedGainLoss: p.TotalUnrealizedGainLoss,
			TotalRealizedGainLoss:   p.TotalRealizedGainLoss,
			TotalSaleProceeds:       p.TotalSaleProceeds,
			TotalOriginalCost:       p.TotalOriginalCost,
			TotalGainLoss:           p.TotalGainLoss,
			IsArchived:              p.IsArchived,
		}
	}

	respondJSON(w, http.StatusOK, response)
}
