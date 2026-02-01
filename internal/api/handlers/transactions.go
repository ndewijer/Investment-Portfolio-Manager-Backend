package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/response"
	apperrors "github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/errors"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/validation"
)

// TransactionHandler handles HTTP requests for transaction endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the transactionService.
type TransactionHandler struct {
	transactionService *service.TransactionService
}

// NewTransactionHandler creates a new TransactionHandler with the provided service dependency.
func NewTransactionHandler(transactionService *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
	}
}

// TransactionPerPortfolio handles GET requests to retrieve all transactions for a specific portfolio.
// Returns transaction details including fund information, dates, shares, and IBKR linkage status.
//
// Endpoint: GET /api/transaction/portfolio/{portfolioId}
// Response: 200 OK with array of TransactionResponse
// Error: 400 Bad Request if portfolio ID is missing or invalid
// Error: 500 Internal Server Error if retrieval fails
func (h *TransactionHandler) TransactionPerPortfolio(w http.ResponseWriter, r *http.Request) {

	portfolioID := chi.URLParam(r, "portfolioId")
	if portfolioID == "" {
		response.RespondError(w, http.StatusBadRequest, "portfolio ID is required", "")
		return
	}

	transactions, err := h.transactionService.GetTransactionsperPortfolio(portfolioID)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, "failed to retrieve transactions", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, transactions)
}

// AllTransactions handles GET requests to retrieve all transactions across all portfolios.
// Returns transaction details including fund information, dates, shares, and IBKR linkage status.
//
// Endpoint: GET /api/transaction
// Response: 200 OK with array of TransactionResponse
// Error: 500 Internal Server Error if retrieval fails
func (h *TransactionHandler) AllTransactions(w http.ResponseWriter, _ *http.Request) {

	transactions, err := h.transactionService.GetTransactionsperPortfolio("")
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, "failed to retrieve transactions", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, transactions)
}

// GetTransaction handles GET requests to retrieve a single transaction by ID.
// Returns transaction details including fund information, date, shares, and IBKR linkage status.
//
// Endpoint: GET /api/transaction/{transactionId}
// Response: 200 OK with TransactionResponse
// Error: 400 Bad Request if transaction ID is missing or invalid
// Error: 500 Internal Server Error if retrieval fails
func (h *TransactionHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {

	transactionID := chi.URLParam(r, "transactionId")
	if transactionID == "" {
		response.RespondError(w, http.StatusBadRequest, "transactions ID is required", "")
		return
	}

	if err := validation.ValidateUUID(transactionID); err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid transaction ID format", err.Error())
		return
	}

	transaction, err := h.transactionService.GetTransaction(transactionID)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, "failed to retrieve transaction", err.Error())
		return
	}

	if transaction.ID == "" {
		response.RespondError(w, http.StatusNotFound, "Transaction not found", apperrors.ErrIBKRTransactionNotFound.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, transaction)
}
