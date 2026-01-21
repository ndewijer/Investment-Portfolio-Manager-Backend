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
