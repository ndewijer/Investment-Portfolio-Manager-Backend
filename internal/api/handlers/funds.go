package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/response"
	apperrors "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/errors"
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

	funds, err := h.fundService.GetFund("")
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, "failed to retrieve funds", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, funds)
}

// GetFund handles GET requests to retrieve a single fund by ID.
// Returns fund details including name, ISIN, symbol, currency, and latest price.
//
// Endpoint: GET /api/fund/{fundID}
// Response: 200 OK with Fund
// Error: 400 Bad Request if fund ID is missing or invalid
// Error: 404 Not Found if no fund exists with the given ID
// Error: 500 Internal Server Error if retrieval fails
func (h *FundHandler) GetFund(w http.ResponseWriter, r *http.Request) {

	fundID := chi.URLParam(r, "uuid")

	if fundID == "" {
		response.RespondError(w, http.StatusBadRequest, "fund ID is required", "")
		return
	}

	if err := validation.ValidateUUID(fundID); err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid fund ID format", err.Error())
		return
	}

	funds, err := h.fundService.GetFund(fundID)
	if err != nil {

		if errors.Is(err, apperrors.ErrFundNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrFundNotFound.Error(), err.Error())
		}

		response.RespondError(w, http.StatusInternalServerError, "failed to retrieve funds", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, funds[0])
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
		response.RespondError(w, http.StatusBadRequest, "Symbol is required", "")
		return
	}

	symbolresponse, err := h.fundService.GetSymbol(symbol)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, "failed to retrieve symbol", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, symbolresponse)
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
		response.RespondError(w, http.StatusBadRequest, "portfolio ID is required", "")
		return
	}

	// Parse date parameters
	startDate, endDate, err := parseDateParams(r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "Invalid date parameters", err.Error())
		return
	}

	fundHistory, err := h.materializedService.GetFundHistoryWithFallback(portfolioID, startDate, endDate)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, "failed to retrieve fund history", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, fundHistory)
}

// GetFundPrices handles GET requests to retrieve historical price data for a fund.
// Returns all available price history from 1970-01-01 to the current date.
//
// Endpoint: GET /api/fund/fund-prices/{fundId}
// Response: 200 OK with array of FundPrice
// Error: 400 Bad Request if fund ID is missing or invalid
// Error: 500 Internal Server Error if retrieval fails
func (h *FundHandler) GetFundPrices(w http.ResponseWriter, r *http.Request) {

	fundID := chi.URLParam(r, "fundId")

	if fundID == "" {
		response.RespondError(w, http.StatusBadRequest, "portfolio ID is required", "")
		return
	}

	if err := validation.ValidateUUID(fundID); err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid portfolio ID format", err.Error())
		return
	}

	startDate, err := time.Parse("2006-01-02", "1970-01-01")
	if err != nil {
		panic("impossible: hardcoded date failed to parse: " + err.Error())
	}
	endDate := time.Now()

	funds, err := h.fundService.LoadFundPrices([]string{fundID}, startDate, endDate, false)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, "failed to retrieve funds", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, funds[fundID])
}
