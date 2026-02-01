package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// respondJSON sends a JSON response with the given status code
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("Failed to encode JSON: %v", err)
		}
	}
}

func parseJSON[T any](r *http.Request) (T, error) {
	var req T

	// Limit body size (prevent memory exhaustion attacks)
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1MB max

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Strict parsing

	if err := decoder.Decode(&req); err != nil {
		return req, fmt.Errorf("invalid JSON: %w", err)
	}

	return req, nil
}

func errorResponse(err, detail string) map[string]string {
	return map[string]string{
		"error":  err,
		"detail": detail,
	}
}
