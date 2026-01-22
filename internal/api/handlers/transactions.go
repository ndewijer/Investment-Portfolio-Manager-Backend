package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
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

	portfolioId := chi.URLParam(r, "portfolioId")
	if portfolioId == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "portfolio ID is required",
		})
		return
	}

	if err := validation.ValidateUUID(portfolioId); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":  "invalid portfolio ID format",
			"detail": err.Error(),
		})
		return
	}

	transactions, err := h.transactionService.GetTransactionsperPortfolio(portfolioId)
	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to retrieve transactions",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, transactions)
}

// AllTransactions handles GET requests to retrieve all transactions across all portfolios.
// Returns transaction details including fund information, dates, shares, and IBKR linkage status.
//
// Endpoint: GET /api/transaction
// Response: 200 OK with array of TransactionResponse
// Error: 500 Internal Server Error if retrieval fails
func (h *TransactionHandler) AllTransactions(w http.ResponseWriter, r *http.Request) {

	transactions, err := h.transactionService.GetTransactionsperPortfolio("")
	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to retrieve transactions",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, transactions)
}

// GetTransaction handles GET requests to retrieve a single transaction by ID.
// Returns transaction details including fund information, date, shares, and IBKR linkage status.
//
// Endpoint: GET /api/transaction/{transactionId}
// Response: 200 OK with TransactionResponse
// Error: 400 Bad Request if transaction ID is missing or invalid
// Error: 500 Internal Server Error if retrieval fails
func (h *TransactionHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {

	transactionId := chi.URLParam(r, "transactionId")
	if transactionId == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "portfolio ID is required",
		})
		return
	}

	if err := validation.ValidateUUID(transactionId); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":  "invalid portfolio ID format",
			"detail": err.Error(),
		})
		return
	}

	transaction, err := h.transactionService.GetTransaction(transactionId)
	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to retrieve transaction",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, transaction)
}
