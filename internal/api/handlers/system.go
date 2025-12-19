package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/service"
)

// SystemHandler handles system-related HTTP requests
type SystemHandler struct {
	systemService *service.SystemService
}

// NewSystemHandler creates a new SystemHandler
func NewSystemHandler(systemService *service.SystemService) *SystemHandler {
	return &SystemHandler{
		systemService: systemService,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Error    string `json:"error,omitempty"`
}

// respondJSON sends a JSON response with the given status code
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// Health checks the health of the system and database connectivity
func (h *SystemHandler) Health(w http.ResponseWriter, r *http.Request) {
	// Check database health
	if err := h.systemService.CheckHealth(); err != nil {
		response := HealthResponse{
			Status:   "unhealthy",
			Database: "disconnected",
			Error:    err.Error(),
		}
		respondJSON(w, http.StatusServiceUnavailable, response)
		return
	}

	// System is healthy
	response := HealthResponse{
		Status:   "healthy",
		Database: "connected",
	}
	respondJSON(w, http.StatusOK, response)
}
