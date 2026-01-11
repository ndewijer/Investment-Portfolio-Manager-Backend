package handlers

import (
	"net/http"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
)

// PortfolioHandler handles HTTP requests for portfolio endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the PortfolioService.
type PortfolioHandler struct {
	portfolioService *service.PortfolioService
}

// NewPortfolioHandler creates a new PortfolioHandler with the provided service dependency.
func NewPortfolioHandler(portfolioService *service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{
		portfolioService: portfolioService,
	}
}

// PortfoliosResponse represents the JSON response structure for the portfolios list endpoint.
// It includes all basic portfolio metadata including archive and overview exclusion flags.
type PortfoliosResponse struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	IsArchived          bool   `json:"is_archived"`
	ExcludeFromOverview bool   `json:"exclude_from_overview"`
}

// Portfolios handles GET requests to retrieve all portfolios.
// This endpoint returns all portfolios including archived and excluded ones.
//
// Endpoint: GET /api/portfolios
// Response: 200 OK with array of PortfoliosResponse
// Error: 500 Internal Server Error if retrieval fails
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

// PortfolioSummaryResponse represents the JSON response structure for portfolio summary data.
// It contains comprehensive portfolio metrics including valuations, costs, and gains/losses.
// All monetary values are rounded to two decimal places by the service layer.
type PortfolioSummaryResponse struct {
	ID                      string  `json:"id"`
	Name                    string  `json:"name"`
	TotalValue              float64 `json:"totalValue"`              // Current market value
	TotalCost               float64 `json:"totalCost"`               // Current cost basis
	TotalDividends          float64 `json:"totalDividends"`          // Cumulative dividends
	TotalUnrealizedGainLoss float64 `json:"totalUnrealizedGainLoss"` // Unrealized gain/loss
	TotalRealizedGainLoss   float64 `json:"totalRealizedGainLoss"`   // Realized gain/loss from sales
	TotalSaleProceeds       float64 `json:"totalSaleProceeds"`       // Total proceeds from sales
	TotalOriginalCost       float64 `json:"totalOriginalCost"`       // Original cost of sold positions
	TotalGainLoss           float64 `json:"totalGainLoss"`           // Combined realized + unrealized
	IsArchived              bool    `json:"is_archived"`
}

// PortfolioSummary handles GET requests to retrieve current portfolio summaries.
// Returns comprehensive metrics for all active (non-archived, non-excluded) portfolios
// as of the current time.
//
// Endpoint: GET /api/portfolio/summary
// Response: 200 OK with array of PortfolioSummaryResponse
// Error: 500 Internal Server Error if calculation fails
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

// PortfolioHistoryResponse represents the JSON response structure for a single date's portfolio data.
// Each response contains the date and an array of portfolio states for that date.
type PortfolioHistoryResponse struct {
	Date       string                              `json:"date"` // Date in YYYY-MM-DD format
	Portfolios []PortfolioHistoryPortfolioResponse `json:"portfolios"`
}

// PortfolioHistoryPortfolioResponse represents a single portfolio's state on a specific historical date.
// It includes valuation, cost basis, and gain/loss information as of that date.
type PortfolioHistoryPortfolioResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Value          float64 `json:"value"`           // Market value on this date
	Cost           float64 `json:"cost"`            // Cost basis on this date
	RealizedGain   float64 `json:"realized_gain"`   // Realized gains/losses as of this date
	UnrealizedGain float64 `json:"unrealized_gain"` // Unrealized gains/losses on this date
}

// PortfolioHistory handles GET requests to retrieve historical portfolio valuations.
//
// Query Parameters:
//   - start_date (optional): First date to include (YYYY-MM-DD or RFC3339 format)
//     Defaults to 1970-01-01 if not provided
//   - end_date (optional): Last date to include (YYYY-MM-DD or RFC3339 format)
//     Defaults to current date if not provided
//
// The endpoint returns daily portfolio valuations for the requested date range.
// Only active portfolios are included, and the actual returned range may be
// narrowed to the range where transaction data exists.
//
// Endpoint: GET /api/portfolio/history?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD
// Response: 200 OK with array of PortfolioHistoryResponse (one per day)
// Error: 400 Bad Request if date parsing fails
// Error: 500 Internal Server Error if calculation fails
func (h *PortfolioHandler) PortfolioHistory(w http.ResponseWriter, r *http.Request) {
	var startDate, endDate time.Time
	var err error

	if r.URL.Query().Get("start_date") == "" {
		startDate, _ = time.Parse("2006-01-02", "1970-01-01")
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
		subResponse := make([]PortfolioHistoryPortfolioResponse, len(p.Portfolios))
		for j, q := range p.Portfolios {
			subResponse[j] = PortfolioHistoryPortfolioResponse{
				ID:             q.ID,
				Name:           q.Name,
				Value:          q.TotalValue,
				Cost:           q.TotalCost,
				RealizedGain:   q.TotalRealizedGainLoss,
				UnrealizedGain: q.TotalUnrealizedGainLoss,
			}
		}
		response[i] = PortfolioHistoryResponse{
			Date:       p.Date,
			Portfolios: subResponse,
		}
	}

	respondJSON(w, http.StatusOK, response)
}
