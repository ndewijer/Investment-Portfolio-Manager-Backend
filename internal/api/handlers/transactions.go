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
// Endpoint: GET /api/transaction/portfolio/{uuid}
// Response: 200 OK with array of TransactionResponse
// Error: 400 Bad Request if portfolio ID is invalid (validated by middleware)
// Error: 500 Internal Server Error if retrieval fails
func (h *TransactionHandler) TransactionPerPortfolio(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "uuid")

	transactions, err := h.transactionService.GetTransactionsperPortfolio(portfolioID)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveTransactions.Error(), err.Error())
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
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveTransactions.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, transactions)
}

// GetTransaction handles GET requests to retrieve a single transaction by ID.
// Returns transaction details including fund information, date, shares, and IBKR linkage status.
//
// Endpoint: GET /api/transaction/{uuid}
// Response: 200 OK with TransactionResponse
// Error: 400 Bad Request if transaction ID is invalid (validated by middleware)
// Error: 404 Not Found if transaction not found
// Error: 500 Internal Server Error if retrieval fails
func (h *TransactionHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {

	transactionID := chi.URLParam(r, "uuid")

	transaction, err := h.transactionService.GetTransaction(transactionID)
	if err != nil {
		if errors.Is(err, apperrors.ErrTransactionNotFound) {
			response.RespondError(w, http.StatusNotFound, apperrors.ErrTransactionNotFound.Error(), err.Error())
			return
		}
		response.RespondError(w, http.StatusInternalServerError, apperrors.ErrFailedToRetrieveTransaction.Error(), err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, transaction)
}

func (h *TransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	req, err := parseJSON[request.CreateTransactionRequest](r)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateCreateTransaction(req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	portfolio, err := h.transactionService.CreateTransaction(r.Context(), req)
	if err != nil {

		response.RespondError(w, http.StatusInternalServerError, "failed to create transaction", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusCreated, portfolio)
}

func (h *TransactionHandler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	fundID := chi.URLParam(r, "uuid")

	req, err := parseJSON[request.UpdateTransactionRequest](r)
	if err != nil {

		response.RespondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := validation.ValidateUpdateTransaction(req); err != nil {

		response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	portfolio, err := h.transactionService.UpdateTransaction(r.Context(), fundID, req)
	if err != nil {
		if errors.Is(err, apperrors.ErrTransactionNotFound) {

			response.RespondError(w, http.StatusNotFound, apperrors.ErrTransactionNotFound.Error(), err.Error())
			return
		}

		response.RespondError(w, http.StatusInternalServerError, "failed to update transaction", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, portfolio)
}
