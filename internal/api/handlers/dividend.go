package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/response"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/apperrors"
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
func (h *DividendHandler) GetAllDividend(w http.ResponseWriter, _ *http.Request) {

	dividends, err := h.dividendService.GetAllDividend()
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

func (h *DividendHandler) GetDividend(w http.ResponseWriter, r *http.Request) {
	dividendID := chi.URLParam(r, "uuid")

	dividends, err := h.dividendService.GetDividend(dividendID)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveDividends.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, dividends)
}

// CreateDividend handles POST requests to create a new dividend.
// Validates the request body and creates a dividend record in the database.
//
// Endpoint: POST /api/dividend
// Request Body: CreateDividendRequest (portfolioFundId, recordDate, exDividendDate, dividendPerShare, and optionally, buyOrderDate, reinvestmentShares and reinvestmentPrice)
// Response: 201 Created with Dividend
// Error: 400 Bad Request if validation fails or request body is invalid
// Error: 500 Internal Server Error if creation fails
func (h *DividendHandler) CreateDividend(w http.ResponseWriter, r *http.Request) {
	req, err := parseJSON[request.CreateDividendRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateCreateDividend(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	dividend, err := h.dividendService.CreateDividend(r.Context(), req)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, "failed to create dividend", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusCreated, dividend)
}

// UpdateDividend handles PUT requests to update an existing dividend.
// Validates the request body and updates the specified dividend fields.
//
// Endpoint: PUT /api/dividend/{uuid}
// Request Body: UpdateDividendRequest (all fields optional)
// Response: 200 OK with updated Dividend
// Error: 400 Bad Request if dividend ID is invalid (validated by middleware) or validation fails
// Error: 404 Not Found if dividend not found
// Error: 500 Internal Server Error if update fails

// to-do
// func (h *DividendHandler) UpdateDividend(w http.ResponseWriter, r *http.Request) {
// 	dividendID := chi.URLParam(r, "uuid")

// 	req, err := parseJSON[request.UpdateDividendRequest](r)
// 	if err != nil {
// 		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
// 		return
// 	}

// 	if err := validation.ValidateUpdateDividend(req); err != nil {
// 		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
// 		return
// 	}

// 	dividend, err := h.dividendService.UpdateDividend(r.Context(), dividendID, req)
// 	if err != nil {
// 		if errors.Is(err, apperrors.ErrDividendNotFound) {
// 			response.RespondError(w, http.StatusNotFound, apperrors.ErrDividendNotFound.Error(), err.Error())
// 			return
// 		}

// 		response.RespondError(w, http.StatusInternalServerError, "failed to update dividend", err.Error())
// 		return
// 	}

// 	response.RespondJSON(w, http.StatusOK, dividend)
// }

// DeleteDividend handles DELETE requests to remove a dividend.
// Validates that the dividend exists before deleting.
//
// Endpoint: DELETE /api/dividend/{uuid}
// Response: 204 No Content on successful deletion
// Error: 400 Bad Request if dividend ID is invalid (validated by middleware)
// Error: 404 Not Found if dividend not found
// Error: 500 Internal Server Error if deletion fails

// to-do
// func (h *DividendHandler) DeleteDividend(w http.ResponseWriter, r *http.Request) {
// 	dividendID := chi.URLParam(r, "uuid")

// 	err := h.dividendService.DeleteDividend(r.Context(), dividendID)
// 	if err != nil {
// 		if errors.Is(err, apperrors.ErrDividendNotFound) {
// 			response.RespondError(w, http.StatusNotFound, apperrors.ErrDividendNotFound.Error(), err.Error())
// 			return
// 		}

// 		response.RespondError(w, http.StatusInternalServerError, "failed to delete dividend", err.Error())
// 		return
// 	}

// 	response.RespondJSON(w, http.StatusNoContent, nil)
// }
