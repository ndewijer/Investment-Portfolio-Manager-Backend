package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/response"
	apperrors "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/errors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
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
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveDividends.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, dividends)
}

// DividendPerPortfolio handles GET requests to retrieve all dividends for a specific portfolio.
// Returns dividend details including fund information, amounts, dates, and reinvestment status
// for all funds held in the specified portfolio.
//
// Endpoint: GET /api/dividend/portfolio/{uuid}
// Response: 200 OK with array of DividendFund
// Error: 400 Bad Request if portfolio ID is invalid (validated by middleware)
// Error: 500 Internal Server Error if retrieval fails
func (h *DividendHandler) DividendPerPortfolio(w http.ResponseWriter, r *http.Request) {

	portfolioID := chi.URLParam(r, "uuid")

	dividends, err := h.dividendService.GetDividendFund(portfolioID)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrPortfolioNotFound.Error(), err.Error())
			return
		}
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveDividends.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, dividends)
}
