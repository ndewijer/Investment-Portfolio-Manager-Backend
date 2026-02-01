package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	apperrors "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/errors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/validation"
)

// PortfolioHandler handles HTTP requests for portfolio endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the PortfolioService.
type PortfolioHandler struct {
	portfolioService    *service.PortfolioService
	fundService         *service.FundService
	materializedService *service.MaterializedService
}

// NewPortfolioHandler creates a new PortfolioHandler with the provided service dependency.
func NewPortfolioHandler(portfolioService *service.PortfolioService, fundService *service.FundService, materializedService *service.MaterializedService) *PortfolioHandler {
	return &PortfolioHandler{
		portfolioService:    portfolioService,
		fundService:         fundService,
		materializedService: materializedService,
	}
}

// Portfolios handles GET requests to retrieve all portfolios.
// This endpoint returns all portfolios including archived and excluded ones.
//
// Endpoint: GET /api/portfolio
// Response: 200 OK with array of PortfoliosResponse
// Error: 500 Internal Server Error if retrieval fails
func (h *PortfolioHandler) Portfolios(w http.ResponseWriter, _ *http.Request) {

	portfolios, err := h.portfolioService.GetAllPortfolios()
	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to retrieve portfolios",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, portfolios)
}

// GetPortfolio handles GET requests to retrieve a single portfolio with its current summary.
// Returns the portfolio details along with current valuations (totalValue, totalCost, etc.).
//
// Endpoint: GET /api/portfolio/{portfolioId}
// Response: 200 OK with PortfolioSummary
// Error: 500 Internal Server Error if retrieval or calculation fails
func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {

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

	startDate, err := time.Parse("2006-01-02", "1970-01-01")
	if err != nil {
		panic("impossible: hardcoded date failed to parse: " + err.Error())
	}
	endDate := time.Now()
	history, err := h.materializedService.GetPortfolioHistoryWithFallback(startDate, endDate, portfolioID)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{
				"error": "portfolio does not exist",
			})
			return
		}

		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":  "failed to get portfolio summary",
			"detail": err.Error(),
		})
		return
	}

	// Should return exactly one portfolio
	if len(history) == 0 || len(history[0].Portfolios) == 0 {
		// Portfolio exists but has no transactions - return zero values
		//nolint:errcheck // Portfolio validation already performed by earlier steps
		portfolio, _ := h.portfolioService.GetPortfolio(portfolioID)
		respondJSON(w, http.StatusOK, model.PortfolioSummary{
			ID:          portfolio.ID,
			Name:        portfolio.Name,
			Description: portfolio.Description,
			IsArchived:  portfolio.IsArchived,
		})
		return
	}

	// Return the single portfolio summary
	respondJSON(w, http.StatusOK, history[len(history)-1].Portfolios[0])
}

// PortfolioSummary handles GET requests to retrieve current portfolio summaries.
// Returns comprehensive metrics for all active (non-archived, non-excluded) portfolios
// as of the current time.
//
// Endpoint: GET /api/portfolio/summary
// Response: 200 OK with array of PortfolioSummaryResponse
// Error: 500 Internal Server Error if calculation fails
func (h *PortfolioHandler) PortfolioSummary(w http.ResponseWriter, _ *http.Request) {

	startDate, err := time.Parse("2006-01-02", "1970-01-01")
	if err != nil {
		panic("impossible: hardcoded date failed to parse: " + err.Error())
	}
	endDate := time.Now()
	portfolioSummary, err := h.materializedService.GetPortfolioHistoryWithFallback(startDate, endDate, "")
	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to get portfolio summary",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}
	if len(portfolioSummary) == 0 {
		respondJSON(w, http.StatusOK, []model.PortfolioSummary{})
		return
	}

	respondJSON(w, http.StatusOK, portfolioSummary[len(portfolioSummary)-1].Portfolios)
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
	startDate, endDate, err := parseDateParams(r)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":  "Invalid date parameters",
			"detail": err.Error(),
		})
		return
	}

	portfolioHistory, err := h.materializedService.GetPortfolioHistoryWithFallback(startDate, endDate, "")
	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to get portfolio history",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, portfolioHistory)
}

// parseDateParams extracts and validates start_date and end_date from query parameters.
func parseDateParams(r *http.Request) (time.Time, time.Time, error) {
	var startDate, endDate time.Time
	var err error

	if r.URL.Query().Get("start_date") == "" {
		startDate, err = time.Parse("2006-01-02", "1970-01-01")
		if err != nil {
			panic("impossible: hardcoded date failed to parse: " + err.Error())
		}
	} else {
		startDate, err = time.Parse("2006-01-02", r.URL.Query().Get("start_date"))
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	if r.URL.Query().Get("end_date") == "" {
		endDate = time.Now()
	} else {
		endDate, err = time.Parse("2006-01-02", r.URL.Query().Get("end_date"))
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	return startDate, endDate, nil
}

// PortfolioFunds handles GET requests to retrieve all portfolio-fund relationships.
// Returns a listing of all funds across all portfolios with basic metadata.
//
// Endpoint: GET /api/portfolio/funds
// Response: 200 OK with array of PortfolioFundListing
// Error: 500 Internal Server Error if retrieval fails
func (h *PortfolioHandler) PortfolioFunds(w http.ResponseWriter, _ *http.Request) {
	listings, err := h.fundService.GetAllPortfolioFundListings()
	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to retrieve portfolio funds",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	if listings == nil {
		listings = []model.PortfolioFundListing{}
	}

	respondJSON(w, http.StatusOK, listings)
}

// GetPortfolioFunds handles GET requests to retrieve all funds for a specific portfolio.
// Returns detailed fund metrics including shares, cost, value, gains/losses, dividends, and fees
// for each fund held in the specified portfolio.
//
// Endpoint: GET /api/portfolio/funds/{portfolioId}
// Response: 200 OK with array of PortfolioFund
// Error: 500 Internal Server Error if retrieval or calculation fails
func (h *PortfolioHandler) GetPortfolioFunds(w http.ResponseWriter, r *http.Request) {

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

	portfolioFunds, err := h.fundService.GetPortfolioFunds(portfolioID)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{
				"error": "portfolio does not exist",
			})
			return
		}

		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":  "failed to get portfolio funds",
			"detail": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, portfolioFunds)
}

func (h *PortfolioHandler) CreatePortfolio(w http.ResponseWriter, r *http.Request) {

	req, err := parseJSON[request.CreatePortfolioRequest](r)
	if err != nil {
		errorResponse := map[string]string{
			"error":  "invalid request body",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusBadRequest, errorResponse)
		return
	}

	if err := validation.ValidateCreatePortfolio(req); err != nil {
		errorResponse := map[string]string{
			"error":  "validation failed",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusBadRequest, errorResponse)
		return
	}

	portfolio, err := h.portfolioService.CreatePortfolio(r.Context(), req)
	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to create portfolio",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusCreated, portfolio)
}

func (h *PortfolioHandler) UpdatePortfolio(w http.ResponseWriter, r *http.Request) {

	portfolioID := chi.URLParam(r, "portfolioId")

	if err := validation.ValidateUUID(portfolioID); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":  "invalid portfolio ID format",
			"detail": err.Error(),
		})
		return
	}

	var req request.UpdatePortfolioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse := map[string]string{
			"error":  "invalid request body",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusBadRequest, errorResponse)
		return
	}

	if err := validation.ValidateUpdatePortfolio(req); err != nil {
		errorResponse := map[string]string{
			"error":  "validation failed",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusBadRequest, errorResponse)
		return
	}

	portfolio, err := h.portfolioService.UpdatePortfolio(r.Context(), portfolioID, req)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			errorResponse := map[string]string{
				"error":  "Portfolio not found",
				"detail": err.Error(),
			}
			respondJSON(w, http.StatusNotFound, errorResponse)
			return
		}
		errorResponse := map[string]string{
			"error":  "failed to update portfolio",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, portfolio)
}

func (h *PortfolioHandler) DeletePortfolio(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioId")

	if err := validation.ValidateUUID(portfolioID); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":  "invalid portfolio ID format",
			"detail": err.Error(),
		})
		return
	}

	err := h.portfolioService.DeletePortfolio(r.Context(), portfolioID)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			errorResponse := map[string]string{
				"error":  "Portfolio not found",
				"detail": err.Error(),
			}
			respondJSON(w, http.StatusNotFound, errorResponse)
			return
		}
		errorResponse := map[string]string{
			"error":  "failed to delete portfolio",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content
}

// ArchivePortfolio handles POST requests to archive a portfolio.
// Archives the portfolio by setting IsArchived to true.
//
// Endpoint: POST /api/portfolio/{portfolioId}/archive
// Response: 200 OK with updated portfolio
// Error: 400 Bad Request if portfolio ID is invalid
// Error: 404 Not Found if portfolio doesn't exist
// Error: 500 Internal Server Error if update fails
func (h *PortfolioHandler) ArchivePortfolio(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioId")

	if err := validation.ValidateUUID(portfolioID); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":  "invalid portfolio ID format",
			"detail": err.Error(),
		})
		return
	}

	isArchived := true
	req := request.UpdatePortfolioRequest{
		IsArchived: &isArchived,
	}

	portfolio, err := h.portfolioService.UpdatePortfolio(r.Context(), portfolioID, req)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{
				"error":  "portfolio not found",
				"detail": err.Error(),
			})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":  "failed to archive portfolio",
			"detail": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, portfolio)
}

// UnarchivePortfolio handles POST requests to unarchive a portfolio.
// Unarchives the portfolio by setting IsArchived to false.
//
// Endpoint: POST /api/portfolio/{portfolioId}/unarchive
// Response: 200 OK with updated portfolio
// Error: 400 Bad Request if portfolio ID is invalid
// Error: 404 Not Found if portfolio doesn't exist
// Error: 500 Internal Server Error if update fails
func (h *PortfolioHandler) UnarchivePortfolio(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioId")

	if err := validation.ValidateUUID(portfolioID); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":  "invalid portfolio ID format",
			"detail": err.Error(),
		})
		return
	}

	isArchived := false
	req := request.UpdatePortfolioRequest{
		IsArchived: &isArchived,
	}

	portfolio, err := h.portfolioService.UpdatePortfolio(r.Context(), portfolioID, req)
	if err != nil {
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {
			respondJSON(w, http.StatusNotFound, map[string]string{
				"error":  "portfolio not found",
				"detail": err.Error(),
			})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":  "failed to unarchive portfolio",
			"detail": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, portfolio)
}
