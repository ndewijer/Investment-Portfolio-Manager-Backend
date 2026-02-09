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

	// Parse date parameters
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

func (h *FundHandler) CheckUsage(w http.ResponseWriter, r *http.Request) {
	fundID := chi.URLParam(r, "uuid")

	fundUsage, err := h.fundService.CheckUsage(fundID)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveUsage.Error(), err.Error())
	}

	response.RespondJSON(w, http.StatusOK, fundUsage)
}

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
		if errors.Is(err, apperrors.ErrPortfolioNotFound) {

			response.RespondError(w, http.StatusNotFound, apperrors.ErrPortfolioNotFound.Error(), err.Error())
			return
		}

		response.RespondError(w, http.StatusInternalServerError, "failed to update portfolio", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, portfolio)
}

func (h *FundHandler) DeleteFund(w http.ResponseWriter, r *http.Request) {
	fundID := chi.URLParam(r, "uuid")

	err := h.fundService.DeleteFund(r.Context(), fundID)
	if err != nil {
		if errors.Is(err, apperrors.ErrFundNotFound) {

			response.RespondError(w, http.StatusNotFound, apperrors.ErrFundNotFound.Error(), err.Error())
			return
		}

		response.RespondError(w, http.StatusInternalServerError, "failed to delete fund", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusNoContent, nil)
}
