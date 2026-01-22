package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string      `json:"error"`
	Details interface{} `json:"details,omitempty"`
}

// RespondJSON sends a JSON response with the given status code
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("failed to encode JSON response: %v", err)
		}
	}
}

// RespondError sends an error response with the given status code
func RespondError(w http.ResponseWriter, status int, message string, details interface{}) {
	response := ErrorResponse{
		Error:   message,
		Details: details,
	}
	RespondJSON(w, status, response)
}
