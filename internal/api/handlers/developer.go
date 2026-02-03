package handlers

import (
	"net/http"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/request"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/response"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
)

// DeveloperHandler handles HTTP requests for Developer endpoints.
// It serves as the HTTP layer adapter, parsing requests and delegating
// business logic to the DeveloperService.
type DeveloperHandler struct {
	DeveloperService *service.DeveloperService
}

// NewDeveloperHandler creates a new DeveloperHandler with the provided service dependency.
func NewDeveloperHandler(DeveloperService *service.DeveloperService) *DeveloperHandler {
	return &DeveloperHandler{
		DeveloperService: DeveloperService,
	}
}

func (h *DeveloperHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	// Parse filter parameters
	filters, err := request.ParseLogFilters(
		r.URL.Query().Get("level"),
		r.URL.Query().Get("category"),
		r.URL.Query().Get("startDate"),
		r.URL.Query().Get("endDate"),
		r.URL.Query().Get("source"),
		r.URL.Query().Get("message"),
		r.URL.Query().Get("sortDir"),
		r.URL.Query().Get("cursor"),
		r.URL.Query().Get("perPage"),
	)
	if err != nil {
		response.RespondError(w, http.StatusBadRequest, "Invalid filter parameters", err.Error())
		return
	}

	// Call service with filters
	logs, err := h.DeveloperService.GetLogs(r.Context(), filters)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, "Failed to retrieve logs", err.Error())
		return
	}

	response.RespondJSON(w, http.StatusOK, logs)
}
