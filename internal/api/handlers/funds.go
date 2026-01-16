package handlers

import (
	"net/http"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
)

// FundHandler handles HTTP requests for fund endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the fundService.
type FundHandler struct {
	fundService *service.FundService
}

// NewFundHandler creates a new FundHandler with the provided service dependency.
func NewFundHandler(fundService *service.FundService) *FundHandler {
	return &FundHandler{
		fundService: fundService,
	}
}

// Funds handles GET requests to retrieve all funds.
// Returns a list of all available funds that can be held in portfolios.
//
// Endpoint: GET /api/fund
// Response: 200 OK with array of Fund
// Error: 500 Internal Server Error if retrieval fails
func (h *FundHandler) Funds(w http.ResponseWriter, r *http.Request) {

	funds, err := h.fundService.GetAllFunds()
	if err != nil {
		errorResponse := map[string]string{
			"error":  "failed to retrieve funds",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, funds)
}
