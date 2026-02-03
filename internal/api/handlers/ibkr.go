package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/response"
	apperrors "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/errors"
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
func (h *IbkrHandler) GetConfig(w http.ResponseWriter, _ *http.Request) {
	config, err := h.ibkrService.GetIbkrConfig()
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveIbkrConfig.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, config)
}

// GetActivePortfolios handles GET requests to retrieve all active portfolios for IBKR import allocation.
// Returns portfolios that are not archived and not excluded from tracking.
//
// Endpoint: GET /api/ibkr/portfolios
// Response: 200 OK with array of Portfolio
// Error: 500 Internal Server Error if retrieval fails
func (h *IbkrHandler) GetActivePortfolios(w http.ResponseWriter, _ *http.Request) {
	config, err := h.ibkrService.GetActivePortfolios()
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrievePortfolios.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, config)
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
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrievePendingDividend.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, pendingDividend)
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
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveInboxTransactions.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, inbox)
}

// GetInboxCount handles GET requests to retrieve the count of IBKR imported transactions.
// Returns the total number of pending transactions in the inbox.
//
// Endpoint: GET /api/ibkr/inbox/count
//
// Response: 200 OK with {"count": <number>}
// Error: 500 Internal Server Error if retrieval fails
func (h *IbkrHandler) GetInboxCount(w http.ResponseWriter, _ *http.Request) {

	count, err := h.ibkrService.GetInboxCount()

	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveInboxTransactions.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, count)
}

// GetTransactionAllocations handles GET /api/ibkr/inbox/{uuid}/allocations
// Retrieves the allocation details for a specific IBKR transaction, showing how it was
// distributed across portfolios including amounts, shares, and fees.
//
// Path parameters:
//   - uuid: UUID of the IBKR transaction (validated by middleware)
//
// Responses:
//   - 200: Success with allocation details
//   - 400: Invalid transaction ID (validated by middleware)
//   - 404: Transaction not found
//   - 500: Internal server error
func (h *IbkrHandler) GetTransactionAllocations(w http.ResponseWriter, r *http.Request) {
	// UUID is already validated by ValidateUUIDMiddleware
	transactionID := chi.URLParam(r, "uuid")

	transactionAllocations, err := h.ibkrService.GetTransactionAllocations(transactionID)
	if err != nil {
		if errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrIBKRTransactionNotFound.Error(), err.Error())
			return
		}

		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToGetTransactionAllocations.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, transactionAllocations)
}

// GetEligiblePortfolios handles GET /api/ibkr/inbox/{uuid}/eligible-portfolios
// Finds portfolios eligible for allocating an IBKR transaction by matching the transaction's
// fund via ISIN or symbol. Returns fund details and the list of portfolios that hold this fund.
//
// Note: This endpoint matches Python API behavior - returns 200 OK with match_info.found=false
// when no fund is found (not 404). This allows clients to handle missing funds gracefully.
//
// Path parameters:
//   - uuid: UUID of the IBKR transaction (validated by middleware)
//
// Responses:
//   - 200: Success with match_info, portfolios, and optional warning (even if fund not found)
//   - 400: Invalid transaction ID (validated by middleware)
//   - 404: Transaction not found
//   - 500: Internal server error
func (h *IbkrHandler) GetEligiblePortfolios(w http.ResponseWriter, r *http.Request) {
	// UUID is already validated by ValidateUUIDMiddleware
	transactionID := chi.URLParam(r, "uuid")

	eligiblePortfolios, err := h.ibkrService.GetEligiblePortfolios(transactionID)
	if err != nil {
		if errors.Is(err, apperrors.ErrIBKRTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrIBKRTransactionNotFound.Error(), err.Error())
			return
		}

		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToGetEligiblePortfolios.Error(), err.Error())
		return
	}

	// Always return 200 OK if fund is not found, Frontend uses match_info.found = false
	response.RespondJSON(w, http.StatusOK, eligiblePortfolios)
}
