package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/validation"
)

// DividendHandler handles HTTP requests for dividend endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the dividendService.
type DividendHandler struct {
	dividendService *service.DividendService
}

// NewDividendHandler creates a new DividendHandler with the provided service dependency.
func NewDividendHandler(dividendService *service.DividendService) *DividendHandler {
	return &DividendHandler{
		dividendService: dividendService,
	}
}

// Dividends handles GET requests to retrieve all dividends.
// Returns a list of all available dividends that can be held in portfolios.
//
// Endpoint: GET /api/dividend
// Response: 200 OK with array of Dividend
// Error: 500 Internal Server Error if retrieval fails
func (h *DividendHandler) GetAllDividends(w http.ResponseWriter, _ *http.Request) {

	dividends, err := h.dividendService.GetAllDividends()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, errorResponse("failed to retrieve dividends", err.Error()))
		return
	}

	respondJSON(w, http.StatusOK, dividends)
}

// DividendPerPortfolio handles GET requests to retrieve all dividends for a specific portfolio.
// Returns dividend details including fund information, amounts, dates, and reinvestment status
// for all funds held in the specified portfolio.
//
// Endpoint: GET /api/dividend/portfolio/{portfolioId}
// Response: 200 OK with array of DividendFund
// Error: 500 Internal Server Error if retrieval fails
func (h *DividendHandler) DividendPerPortfolio(w http.ResponseWriter, r *http.Request) {

	portfolioID := chi.URLParam(r, "portfolioId")
	if portfolioID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "portfolio ID is required",
		})
		return
	}

	if err := validation.ValidateUUID(portfolioID); err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse("invalid portfolio ID format", err.Error()))
		return
	}

	dividends, err := h.dividendService.GetDividendFund(portfolioID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, errorResponse("failed to retrieve dividends", err.Error()))
		return
	}

	respondJSON(w, http.StatusOK, dividends)
}
