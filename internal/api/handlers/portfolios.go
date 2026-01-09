package handlers

import (
	"net/http"
	"time"

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
			"error":  "Failed to retreive portfolios",
			"detail": err.Error(),
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
			"error":  "Failed to get portfolio summary",
			"detail": err.Error(),
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

type PortfolioHistoryResponse struct {
	Date       string                             `json:"date"`
	Portfolios []service.PorfolioHistoryPortfolio `json:"portfolios"`
	// Portfolios []PortfolioHistoryPortfolioResponse `json:"portfolios"`
}

// type PortfolioHistoryPortfolioResponse struct {
// 	ID             string  `json:"id"`
// 	Name           string  `json:"name"`
// 	Value          float64 `json:"value"`
// 	Cost           float64 `json:"cost"`
// 	RealizedGain   float64 `json:"realized_gain"`
// 	UnrealizedGain float64 `json:"unrealized_gain"`
// }

func (h *PortfolioHandler) PortfolioHistory(w http.ResponseWriter, r *http.Request) {
	var startDate, endDate time.Time
	var err error

	if r.URL.Query().Get("start_date") == "" && r.URL.Query().Get("end_date") == "" {

		respondJSON(w, http.StatusBadRequest, "start_date and/or end_date are required")
		return
	}

	if r.URL.Query().Get("start_date") == "" {
		startDate, err = time.Parse("2006-01-02", "1970-01-01")
		if err != nil {
			errorResponse := map[string]string{
				"error":  "Could not set empty start_date to 1970-01-01",
				"detail": err.Error(),
			}
			respondJSON(w, http.StatusBadRequest, errorResponse)
			return
		}
	} else {
		startDate, err = time.Parse("2006-01-02", r.URL.Query().Get("start_date"))
		if err != nil {
			startDate, err = time.Parse(time.RFC3339, r.URL.Query().Get("start_date"))
			if err != nil {
				errorResponse := map[string]string{
					"error":  "Failed to parse start_date into time.Time",
					"detail": err.Error(),
				}
				respondJSON(w, http.StatusBadRequest, errorResponse)
				return
			}
		}
	}

	if r.URL.Query().Get("end_date") == "" {
		endDate = time.Now()
	} else {
		endDate, err = time.Parse("2006-01-02", r.URL.Query().Get("end_date"))
		if err != nil {
			endDate, err = time.Parse(time.RFC3339, r.URL.Query().Get("end_date"))
			if err != nil {
				errorResponse := map[string]string{
					"error":  "Failed to parse end_date into time.Time",
					"detail": err.Error(),
				}
				respondJSON(w, http.StatusBadRequest, errorResponse)
				return
			}
		}
	}

	portfolioHistory, err := h.portfolioService.GetPortfolioHistory(startDate, endDate)
	if err != nil {
		errorResponse := map[string]string{
			"error":  "Failed to get portfolio history",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	response := make([]PortfolioHistoryResponse, len(portfolioHistory))
	for i, p := range portfolioHistory {
		response[i] = PortfolioHistoryResponse{
			Date:       p.Date,
			Portfolios: p.Portfolios,
		}
	}

	respondJSON(w, http.StatusOK, response)
}
