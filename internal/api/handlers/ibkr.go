package handlers

import (
	"net/http"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
)

// IbkrHandler handles HTTP requests for ibkr endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the ibkrService.
type IbkrHandler struct {
	ibkrService *service.IbkrService
}

// NewIbkrHandler creates a new IbkrHandler with the provided service dependency.
func NewIbkrHandler(ibkrService *service.IbkrService) *IbkrHandler {
	return &IbkrHandler{
		ibkrService: ibkrService,
	}
}

// GetConfig handles GET requests to retrieve the IBKR integration configuration.
// Returns configuration details including flex query ID, token expiration, import settings,
// and default allocation settings.
//
// Endpoint: GET /api/ibkr/config
// Response: 200 OK with IbkrConfig
// Error: 500 Internal Server Error if retrieval fails
func (h *IbkrHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.ibkrService.GetIbkrConfig()
	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to retrieve ibkr config",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, config)
}

// GetActivePortfolios handles GET requests to retrieve all active portfolios for IBKR import allocation.
// Returns portfolios that are not archived and not excluded from tracking.
//
// Endpoint: GET /api/ibkr/portfolios
// Response: 200 OK with array of Portfolio
// Error: 500 Internal Server Error if retrieval fails
func (h *IbkrHandler) GetActivePortfolios(w http.ResponseWriter, r *http.Request) {
	config, err := h.ibkrService.GetActivePortfolios()
	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to retrieve portfolios",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, config)
}

// GetPendingDividends handles GET requests to retrieve dividend records awaiting processing.
// Returns dividends with reinvestment_status = 'PENDING' that can be matched to IBKR dividend transactions.
//
// Endpoint: GET /api/ibkr/dividend/pending
// Query params:
//   - symbol: Filter by fund trading symbol (optional)
//   - isin: Filter by fund ISIN (optional)
//
// Response: 200 OK with array of PendingDividend
// Error: 500 Internal Server Error if retrieval fails
func (h *IbkrHandler) GetPendingDividends(w http.ResponseWriter, r *http.Request) {

	symbol := r.URL.Query().Get("symbol")
	isin := r.URL.Query().Get("isin")

	pendingDividend, err := h.ibkrService.GetPendingDividends(symbol, isin)

	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to retrieve pending dividend",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, pendingDividend)
}

// GetInbox handles GET requests to retrieve IBKR imported transactions from the inbox.
// Returns transactions that have been imported from IBKR and are awaiting allocation or processing.
//
// Endpoint: GET /api/ibkr/inbox
// Query params:
//   - status: Filter by transaction status (optional, defaults to "pending")
//   - transaction_type: Filter by transaction type (optional, e.g., "dividend", "trade")
//
// Response: 200 OK with array of IBKRTransaction
// Error: 500 Internal Server Error if retrieval fails
func (h *IbkrHandler) GetInbox(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	transactionType := r.URL.Query().Get("transactionType")

	inbox, err := h.ibkrService.GetInbox(status, transactionType)

	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to retrieve inbox transactions",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, inbox)
}

// GetInboxCount handles GET requests to retrieve the count of IBKR imported transactions.
// Returns the total number of pending transactions in the inbox.
//
// Endpoint: GET /api/ibkr/inbox/count
//
// Response: 200 OK with {"count": <number>}
// Error: 500 Internal Server Error if retrieval fails
func (h *IbkrHandler) GetInboxCount(w http.ResponseWriter, r *http.Request) {

	count, err := h.ibkrService.GetInboxCount()

	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to retrieve inbox transactions",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, count)
}
