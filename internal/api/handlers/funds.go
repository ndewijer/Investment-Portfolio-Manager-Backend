package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/validation"
)

// FundHandler handles HTTP requests for fund endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the fundService.
type FundHandler struct {
	fundService         *service.FundService
	materializedService *service.MaterializedService
}

// NewFundHandler creates a new FundHandler with the provided service dependency.
func NewFundHandler(fundService *service.FundService, materializedService *service.MaterializedService) *FundHandler {
	return &FundHandler{
		fundService:         fundService,
		materializedService: materializedService,
	}
}

// Funds handles GET requests to retrieve all funds.
// Returns a list of all available funds that can be held in portfolios.
//
// Endpoint: GET /api/fund
// Response: 200 OK with array of Fund
// Error: 500 Internal Server Error if retrieval fails
func (h *FundHandler) Funds(w http.ResponseWriter, r *http.Request) {

	funds, err := h.fundService.GetAllFunds()
	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to retrieve funds",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, funds)
}

// GetFundHistory handles GET requests to retrieve historical fund data for a portfolio.
// Returns time-series data showing individual fund values within a portfolio over time.
//
// Endpoint: GET /api/fund/history/{portfolioId}
// Query Parameters:
//   - start_date (optional): Start date (YYYY-MM-DD), defaults to 1970-01-01
//   - end_date (optional): End date (YYYY-MM-DD), defaults to current date
//
// Response: 200 OK with array of FundHistoryResponse
// Error: 400 Bad Request if portfolio ID is invalid or date parsing fails
// Error: 500 Internal Server Error if retrieval fails
func (h *FundHandler) GetFundHistory(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioId")

	if portfolioID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "portfolio ID is required",
		})
		return
	}

	if err := validation.ValidateUUID(portfolioID); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":  "invalid portfolio ID format",
			"detail": err.Error(),
		})
		return
	}

	// Parse date parameters
	startDate, endDate, err := parseDateParams(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":  "Invalid date parameters",
			"detail": err.Error(),
		})
		return
	}

	fundHistory, err := h.materializedService.GetFundHistoryWithFallback(portfolioID, startDate, endDate)
	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to retrieve fund history",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, fundHistory)
}
