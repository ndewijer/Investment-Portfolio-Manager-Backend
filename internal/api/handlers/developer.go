package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/response"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/apperrors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
)

// DeveloperHandler handles HTTP requests for Developer endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the DeveloperService.
type DeveloperHandler struct {
	DeveloperService *service.DeveloperService
}

// NewDeveloperHandler creates a new DeveloperHandler with the provided service dependency.
func NewDeveloperHandler(DeveloperService *service.DeveloperService) *DeveloperHandler {
	return &DeveloperHandler{
		DeveloperService: DeveloperService,
	}
}

// GetLogs handles GET requests to retrieve system logs with filtering and pagination.
// Returns logs with cursor-based pagination for efficient traversal of large result sets.
// Supports filtering by level, category, date range, source, and message content.
//
// Query Parameters:
//   - level: Comma-separated log levels (debug, info, warning, error, critical)
//   - category: Comma-separated categories (portfolio, fund, transaction, etc.)
//   - startDate: Filter logs from this date (YYYY-MM-DD format)
//   - endDate: Filter logs until this date (YYYY-MM-DD format)
//   - source: Filter by source field (partial match)
//   - message: Filter by message content (partial match)
//   - sortDir: Sort direction (asc or desc, default: desc)
//   - cursor: Pagination cursor from previous response
//   - perPage: Number of results per page (1-100, default: 50)
//
// Endpoint: GET /api/developer/logs
// Response: 200 OK with LogResponse containing logs, pagination info
// Error: 400 Bad Request if filter parameters are invalid
// Error: 500 Internal Server Error if retrieval fails
func (h *DeveloperHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	// Parse filter parameters
	filters, err := request.ParseLogFilters(
		r.URL.Query().Get("level"),
		r.URL.Query().Get("category"),
		r.URL.Query().Get("startDate"),
		r.URL.Query().Get("endDate"),
		r.URL.Query().Get("source"),
		r.URL.Query().Get("message"),
		r.URL.Query().Get("sortDir"),
		r.URL.Query().Get("cursor"),
		r.URL.Query().Get("perPage"),
	)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "Invalid filter parameters", err.Error())
		return
	}

	// Call service with filters
	logs, err := h.DeveloperService.GetLogs(r.Context(), filters)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveLogs.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, logs)
}

// GetLoggingConfig handles GET requests to retrieve the current logging configuration.
// Returns the enabled status and logging level settings.
//
// Endpoint: GET /api/developer/system-settings/logging
// Response: 200 OK with LoggingSetting
// Error: 500 Internal Server Error if retrieval fails
func (h *DeveloperHandler) GetLoggingConfig(w http.ResponseWriter, _ *http.Request) {
	setting, err := h.DeveloperService.GetLoggingConfig()
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveLoggingConfig.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, setting)
}

// GetFundPriceCSVTemplate handles GET requests to retrieve the CSV template for fund price imports.
// Returns the expected CSV headers, an example row, and a description of the format.
//
// Endpoint: GET /api/developer/csv/fund-prices/template
// Response: 200 OK with TemplateModel
func (h *DeveloperHandler) GetFundPriceCSVTemplate(w http.ResponseWriter, _ *http.Request) {

	headers := []string{"date", "price"}
	example := map[string]string{
		"date":  "2024-03-21",
		"price": "150.75",
	}
	description := `CSV file should contain the following columns:
- date: Price date in YYYY-MM-DD format
- price: Fund price (decimal numbers)`

	template := model.TemplateModel{
		Headers:     headers,
		Example:     example,
		Description: description,
	}

	response.RespondJSON(w, http.StatusOK, template)
}

// GetTransactionCSVTemplate handles GET requests to retrieve the CSV template for transaction imports.
// Returns the expected CSV headers, an example row, and a description of the format.
//
// Endpoint: GET /api/developer/csv/transactions/template
// Response: 200 OK with TemplateModel
func (h *DeveloperHandler) GetTransactionCSVTemplate(w http.ResponseWriter, _ *http.Request) {

	headers := []string{"date", "type", "shares", "cost_per_share"}
	example := map[string]string{
		"date":           "2024-03-21",
		"type":           "buy/sell",
		"shares":         "10.5",
		"cost_per_share": "150.75",
	}
	description := `CSV file should contain the following columns:
- date: Transaction date in YYYY-MM-DD format
- type: Transaction type, either "buy" or "sell"
- shares: Number of shares (decimal numbers allowed)
- cost_per_share: Cost per share in the fund's currency`

	template := model.TemplateModel{
		Headers:     headers,
		Example:     example,
		Description: description,
	}

	response.RespondJSON(w, http.StatusOK, template)
}

// GetExchangeRate handles GET requests to retrieve an exchange rate for a specific currency pair and date.
// Returns the exchange rate if found, or null if no rate exists for the given parameters.
// Always returns 200 OK with the wrapper, even if the rate is not found (rate will be null).
//
// Query Parameters:
//   - fromCurrency: Source currency code (required)
//   - toCurrency: Target currency code (required)
//   - date: Date for the exchange rate in YYYY-MM-DD format (required)
//
// Endpoint: GET /api/developer/exchange-rate
// Response: 200 OK with ExchangeRateWrapper (rate may be null if not found)
// Error: 400 Bad Request if required parameters are missing or date format is invalid
// Error: 500 Internal Server Error if retrieval fails
func (h *DeveloperHandler) GetExchangeRate(w http.ResponseWriter, r *http.Request) {

	fromCurrency := r.URL.Query().Get("fromCurrency")
	toCurrency := r.URL.Query().Get("toCurrency")
	dateStr := r.URL.Query().Get("date")

	if fromCurrency == "" {
		response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidCurrency.Error(), "fromCurrency is required")
		return
	}
	if toCurrency == "" {
		response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidCurrency.Error(), "toCurrency is required")
		return
	}
	if dateStr == "" {
		response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidDate.Error(), "date is required")
		return
	}

	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "Invalid date format", "Date must be in YYYY-MM-DD format")
		return
	}
	date := parsedDate.Format("2006-01-02")

	exchangeResponse := model.ExchangeRateWrapper{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		Date:         date,
	}

	exchangeRate, err := h.DeveloperService.GetExchangeRate(fromCurrency, toCurrency, parsedDate)
	if err != nil {
		if errors.Is(err, apperrors.ErrExchangeRateNotFound) {
			exchangeResponse.Rate = nil
			response.RespondJSON(w, http.StatusOK, exchangeResponse)
			return
		}
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveExchangeRate.Error(), err.Error())
		return
	}

	exchangeResponse.Rate = exchangeRate

	response.RespondJSON(w, http.StatusOK, exchangeResponse)
}

// GetFundPrice handles GET requests to retrieve a fund's price for a specific date.
// Returns the fund price if found for the given fund and date.
//
// Query Parameters:
//   - fundId: The fund UUID (required)
//   - date: Date for the fund price in YYYY-MM-DD format (required)
//
// Endpoint: GET /api/developer/fund-price
// Response: 200 OK with FundPrice
// Error: 400 Bad Request if required parameters are missing or date format is invalid
// Error: 404 Not Found if no price exists for the given fund and date
// Error: 500 Internal Server Error if retrieval fails
func (h *DeveloperHandler) GetFundPrice(w http.ResponseWriter, r *http.Request) {
	fundID := r.URL.Query().Get("fundId")
	dateStr := r.URL.Query().Get("date")

	if fundID == "" {
		response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidFundID.Error(), "fundId is required")
		return
	}
	if dateStr == "" {
		response.RespondError(w, http.StatusBadRequest, apperrors.ErrInvalidDate.Error(), "date is required")
		return
	}

	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "Invalid date format", "Date must be in YYYY-MM-DD format")
		return
	}

	fundPrice, err := h.DeveloperService.GetFundPrice(fundID, parsedDate)
	if err != nil {
		if errors.Is(err, apperrors.ErrFundPriceNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrFundPriceNotFound.Error(), err.Error())
			return
		}
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveFundPrice.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, fundPrice)
}
