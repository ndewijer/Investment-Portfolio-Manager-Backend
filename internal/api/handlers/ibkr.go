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
