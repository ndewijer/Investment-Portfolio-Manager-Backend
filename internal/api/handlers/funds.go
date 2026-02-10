package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/response"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/validation"
)

// FundHandler handles HTTP requests for fund endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the fundService.
type FundHandler struct {
	fundService *service.FundService

	materializedService *service.MaterializedService
}

// NewFundHandler creates a new FundHandler with the provided service dependency.
func NewFundHandler(fundService *service.FundService, materializedService *service.MaterializedService) *FundHandler {
	return &FundHandler{
		fundService: fundService,

		materializedService: materializedService,
	}
}

// GetAllFunds handles GET requests to retrieve all funds.
// Returns a list of all available funds that can be held in portfolios,
// including their latest prices.
//
// Endpoint: GET /api/fund
// Response: 200 OK with array of Fund
// Error: 500 Internal Server Error if retrieval fails
func (h *FundHandler) GetAllFunds(w http.ResponseWriter, _ *http.Request) {

	funds, err := h.fundService.GetAllFunds()
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveFunds.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, funds)
}

// GetFund handles GET requests to retrieve a single fund by ID.
// Returns fund details including name, ISIN, symbol, currency, and latest price.
//
// Endpoint: GET /api/fund/{uuid}
// Response: 200 OK with Fund
// Error: 400 Bad Request if fund ID is missing or invalid (validated by middleware)
// Error: 404 Not Found if no fund exists with the given ID
// Error: 500 Internal Server Error if retrieval fails
func (h *FundHandler) GetFund(w http.ResponseWriter, r *http.Request) {

	fundID := chi.URLParam(r, "uuid")

	fund, err := h.fundService.GetFund(fundID)
	if err != nil {
		if errors.Is(err, apperrors.ErrFundNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrFundNotFound.Error(), err.Error())
			return
		}

		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveFund.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, fund)
}

// GetSymbol handles GET requests to retrieve symbol information by ticker symbol.
// Returns symbol metadata including name, exchange, currency, and ISIN.
//
// Endpoint: GET /api/fund/symbol/{Symbol}
// Response: 200 OK with Symbol
// Error: 400 Bad Request if symbol is missing
// Error: 500 Internal Server Error if retrieval fails
func (h *FundHandler) GetSymbol(w http.ResponseWriter, r *http.Request) {

	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidSymbol.Error(), "")
		return
	}

	symbolresponse, err := h.fundService.GetSymbol(symbol)
	if err != nil {
		if errors.Is(err, apperrors.ErrSymbolNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrSymbolNotFound.Error(), err.Error())
			return
		}
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveSymbol.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, symbolresponse)
}

// GetFundHistory handles GET requests to retrieve historical fund data for a portfolio.
// Returns time-series data showing individual fund values within a portfolio over time.
//
// Endpoint: GET /api/fund/history/{uuid}
// Query Parameters:
//   - start_date (optional): Start date (YYYY-MM-DD), defaults to 1970-01-01
//   - end_date (optional): End date (YYYY-MM-DD), defaults to current date
//
// Response: 200 OK with array of FundHistoryResponse
// Error: 400 Bad Request if portfolio ID is invalid (validated by middleware) or date parsing fails
// Error: 500 Internal Server Error if retrieval fails
func (h *FundHandler) GetFundHistory(w http.ResponseWriter, r *http.Request) {

	portfolioID := chi.URLParam(r, "uuid")

	startDate, endDate, err := parseDateParams(r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "Invalid date parameters", err.Error())
		return
	}

	fundHistory, err := h.materializedService.GetFundHistoryWithFallback(portfolioID, startDate, endDate)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveFundHistory.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, fundHistory)
}

// GetFundPrices handles GET requests to retrieve historical price data for a fund.
// Returns all available price history from 1970-01-01 to the current date.
//
// Endpoint: GET /api/fund/fund-prices/{uuid}
// Response: 200 OK with array of FundPrice
// Error: 400 Bad Request if fund ID is invalid (validated by middleware)
// Error: 500 Internal Server Error if retrieval fails
func (h *FundHandler) GetFundPrices(w http.ResponseWriter, r *http.Request) {

	fundID := chi.URLParam(r, "uuid")

	startDate, err := time.Parse("2006-01-02", "1970-01-01")
	if err != nil {
		panic("impossible: hardcoded date failed to parse: " + err.Error())
	}
	endDate := time.Now()

	funds, err := h.fundService.LoadFundPrices([]string{fundID}, startDate, endDate, false)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveFunds.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, funds[fundID])
}

// CheckUsage handles GET requests to check if a fund is currently in use by any portfolios.
// Returns usage information including whether the fund is in use and which portfolios use it.
//
// Endpoint: GET /api/fund/check-usage/{uuid}
// Response: 200 OK with FundUsage
// Error: 400 Bad Request if fund ID is invalid (validated by middleware)
// Error: 500 Internal Server Error if retrieval fails
func (h *FundHandler) CheckUsage(w http.ResponseWriter, r *http.Request) {
	fundID := chi.URLParam(r, "uuid")

	fundUsage, err := h.fundService.CheckUsage(fundID)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveUsage.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, fundUsage)
}

// CreateFund handles POST requests to create a new fund.
// Validates the request and creates a fund with the provided details.
//
// Endpoint: POST /api/fund
// Request Body: CreateFundRequest (JSON)
//   - name: Fund name (required, max 100 chars)
//   - isin: ISIN code (required, format: 2 letters + 9 alphanumeric + 1 digit)
//   - currency: Currency code (required, max 3 chars, e.g., USD, EUR)
//   - exchange: Exchange name (required, max 15 chars, e.g., NYSE, AMS)
//   - investment_type: Type of investment (required, must be "FUND" or "STOCK")
//   - dividend_type: Type of dividend (required, must be "CASH", "STOCK", or "NONE")
//   - symbol: Trading symbol (optional, max 10 chars)
//
// Response: 201 Created with Fund
// Error: 400 Bad Request if validation fails
// Error: 500 Internal Server Error if creation fails
func (h *FundHandler) CreateFund(w http.ResponseWriter, r *http.Request) {
	req, err := parseJSON[request.CreateFundRequest](r)
	if err != nil {

		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateCreateFund(req); err != nil {

		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	portfolio, err := h.fundService.CreateFund(r.Context(), req)
	if err != nil {

		response.RespondError(w, http.StatusInternalServerError, "failed to create fund", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusCreated, portfolio)
}

// UpdateFund handles PUT requests to update an existing fund.
// Validates the request and updates the fund with the provided fields.
// All fields are optional; only provided fields are updated, omitted fields remain unchanged.
// However, if a field is provided, it must meet the validation requirements.
//
// Endpoint: PUT /api/fund/{uuid}
// Request Body: UpdateFundRequest (JSON) - all fields optional but validated if provided:
//   - name: New fund name (max 100 chars)
//   - isin: New ISIN code (format: 2 letters + 9 alphanumeric + 1 digit)
//   - currency: New currency code (max 3 chars, e.g., USD, EUR)
//   - exchange: New exchange name (max 15 chars, e.g., NYSE, AMS)
//   - investment_type: New investment type (must be "FUND" or "STOCK")
//   - dividend_type: New dividend type (must be "CASH", "STOCK", or "NONE")
//   - symbol: New trading symbol (max 10 chars)
//
// Response: 200 OK with updated Fund
// Error: 400 Bad Request if validation fails or fund ID is invalid
// Error: 404 Not Found if fund does not exist
// Error: 500 Internal Server Error if update fails
func (h *FundHandler) UpdateFund(w http.ResponseWriter, r *http.Request) {
	fundID := chi.URLParam(r, "uuid")

	req, err := parseJSON[request.UpdateFundRequest](r)
	if err != nil {

		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateUpdateFund(req); err != nil {

		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	portfolio, err := h.fundService.UpdateFund(r.Context(), fundID, req)
	if err != nil {
		if errors.Is(err, apperrors.ErrFundNotFound) {

			response.RespondError(w, http.StatusNotFound, apperrors.ErrFundNotFound.Error(), err.Error())
			return
		}

		response.RespondError(w, http.StatusInternalServerError, "failed to update fund", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, portfolio)
}

// DeleteFund handles DELETE requests to remove a fund.
// Checks if the fund exists and is not in use before deletion.
// A fund cannot be deleted if it has been used in any portfolios (has transactions),
// as this would destroy portfolio history and fund price data.
//
// Endpoint: DELETE /api/fund/{uuid}
// Response: 204 No Content on successful deletion
// Error: 400 Bad Request if fund ID is invalid (validated by middleware)
// Error: 404 Not Found if fund does not exist
// Error: 409 Conflict if fund is in use by portfolios
// Error: 500 Internal Server Error if deletion fails
func (h *FundHandler) DeleteFund(w http.ResponseWriter, r *http.Request) {
	fundID := chi.URLParam(r, "uuid")

	err := h.fundService.DeleteFund(r.Context(), fundID)
	if err != nil {
		if errors.Is(err, apperrors.ErrFundNotFound) {

			response.RespondError(w, http.StatusNotFound, apperrors.ErrFundNotFound.Error(), err.Error())
			return
		}

		if errors.Is(err, apperrors.ErrFundInUse) {

			response.RespondError(w, http.StatusConflict, "cannot delete fund: in use by portfolio", err.Error())
			return
		}

		response.RespondError(w, http.StatusInternalServerError, "failed to delete fund", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusNoContent, nil)
}

// UpdateFundPrice updates fund prices based on the requested type.
// Handles both current price updates (yesterday's closing price) and historical
// price backfilling from the earliest transaction date.
//
// Query Parameters:
//   - type: Required. Must be "today" or "historical"
//   - "today": Updates the latest available price (yesterday's close)
//   - "historical": Backfills all missing prices from earliest transaction to yesterday
//
// URL Parameters:
//   - uuid: The fund ID to update prices for
//
// Returns:
//   - 200 OK: Price update completed successfully
//   - 400 Bad Request: Invalid or missing type parameter
//   - 500 Internal Server Error: Price update failed
func (h *FundHandler) UpdateFundPrice(w http.ResponseWriter, r *http.Request) {
	fundID := chi.URLParam(r, "uuid")
	updateType := r.URL.Query().Get("type")

	if updateType != "today" && updateType != "historical" {
		response.RespondError(w, http.StatusBadRequest, "type requires 'today' or 'historical'", "")
		return
	}

	if updateType == "today" {
		_, err := h.fundService.UpdateCurrentFundPrice(r.Context(), fundID)
		if err != nil {
			response.RespondError(w, http.StatusInternalServerError, "cannot update current fund price", err.Error())
			return
		}
	} else {
		err := h.fundService.UpdateHistoricalFundPrice(r.Context(), fundID)
		if err != nil {
			response.RespondError(w, http.StatusInternalServerError, "cannot update historical fund prices", err.Error())
			return
		}
	}

	response.RespondJSON(w, http.StatusOK, nil)
}
