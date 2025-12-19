package handlers

import (
	"net/http"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
)

// PortfolioHandler handles system-related HTTP requests
type PortfolioHandler struct {
	portfolioService *service.PortfolioService
}

// NewPortfolioHandler creates a new PortfolioHandler
func NewPortfolioHandler(portfolioService *service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{
		portfolioService: portfolioService,
	}
}

// PortfoliosResponse represents the Portfolios get response
type PortfolioResponse struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	IsArchived          bool   `json:"is_archived"`
	ExcludeFromOverview bool   `json:"exclude_from_overview"`
}

// Gets basic Portfolios information
func (h *PortfolioHandler) Portfolios(w http.ResponseWriter, r *http.Request) {

	portfolios, err := h.portfolioService.GetAllPortfolios()
	if err != nil {
		errorResponse := map[string]string{
			"error": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	response := make([]PortfolioResponse, len(portfolios))
	for i, p := range portfolios {
		response[i] = PortfolioResponse{
			ID:                  p.ID,
			Name:                p.Name,
			Description:         p.Description,
			IsArchived:          p.IsArchived,
			ExcludeFromOverview: p.ExcludeFromOverview,
		}
	}

	respondJSON(w, http.StatusOK, response)
}
