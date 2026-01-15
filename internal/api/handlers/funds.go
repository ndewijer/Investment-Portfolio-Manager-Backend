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

// NewPortfolioHandler creates a new FundHandler with the provided service dependency.
func NewFundHandler(fundService *service.FundService) *FundHandler {
	return &FundHandler{
		fundService: fundService,
	}
}

// Funds handles GET requests to retrieve all funds.
//
// Endpoint: GET /api/fund
// Response: 200 OK with array of PortfoliosResponse
// Error: 500 Internal Server Error if retrieval fails
func (h *FundHandler) Funds(w http.ResponseWriter, r *http.Request) {

	funds, err := h.fundService.GetAlFunds()
	if err != nil {
		errorResponse := map[string]string{
			"error":  "Failed to retreive funds",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	respondJSON(w, http.StatusOK, funds)
}
