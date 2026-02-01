// Package response provides utilities for sending consistent HTTP responses.
// It includes helpers for JSON responses and standardized error responses.
package response

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorResponse represents a structured error response returned by the API.
// The Details field is optional and can contain additional context about the error.
type ErrorResponse struct {
	Error   string      `json:"error"`
	Details interface{} `json:"details,omitempty"`
}

// RespondJSON sends a JSON response with the given status code.
// Sets the Content-Type header to application/json and writes the status code.
// If data is nil, only the status code is sent (useful for 204 No Content).
// Logs encoding errors but does not fail the response.
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("failed to encode JSON response: %v", err)
		}
	}
}

// RespondError sends a structured error response with the given status code.
// The message should be a user-friendly error description.
// The details parameter can be an error string, additional context, or nil.
//
// Example:
//
//	response.RespondError(w, http.StatusBadRequest, "validation failed", err.Error())
//	response.RespondError(w, http.StatusNotFound, "resource not found", "")
func RespondError(w http.ResponseWriter, status int, message string, details interface{}) {
	response := ErrorResponse{
		Error:   message,
		Details: details,
	}
	RespondJSON(w, status, response)
}
