package handlers

import (
	"encoding/json"
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
