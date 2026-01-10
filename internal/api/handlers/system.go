package handlers

import (
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

type VersionInfoResponse struct {
	AppVersion       string          `json:"app_version"`
	DbVersion        string          `json:"db_version"`
	Features         map[string]bool `json:"features"`
	MigrationNeeded  bool            `json:"migration_needed"`
	MigrationMessage *string         `json:"migration_message"`
}

func (h *SystemHandler) Version(w http.ResponseWriter, r *http.Request) {
	version, err := h.systemService.CheckVersion()
	if err != nil {
		errorResponse := map[string]string{
			"error":  "Failed to get version information",
			"detail": err.Error(),
		}
		respondJSON(w, http.StatusInternalServerError, errorResponse)
		return
	}

	response := VersionInfoResponse{
		AppVersion:       version.AppVersion,
		DbVersion:        version.DbVersion,
		Features:         version.Features,
		MigrationNeeded:  version.MigrationNeeded,
		MigrationMessage: version.MigrationMessage,
	}

	respondJSON(w, http.StatusOK, response)
}
