// Package middleware provides HTTP middleware for request validation and processing.
package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/api/response"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/validation"
)

// ValidatePortfolioIDMiddleware validates that the portfolioId URL parameter is present and is a valid UUID.
// Returns 400 Bad Request if the portfolio ID is missing or invalid.
// This middleware should be applied to routes that require a valid portfolio ID in the URL path.
//
// Example usage in router:
//
//	r.Route("/{portfolioId}", func(r chi.Router) {
//	    r.Use(middleware.ValidatePortfolioIDMiddleware)
//	    r.Get("/", handler.GetPortfolio)
//	    r.Put("/", handler.UpdatePortfolio)
//	})
func ValidateUUIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		UUID := chi.URLParam(r, "uuid")

		if UUID == "" {
			response.RespondError(w, http.StatusBadRequest, "valid UUID is required", "")
			return
		}

		if err := validation.ValidateUUID(UUID); err != nil {
			response.RespondError(w, http.StatusBadRequest, "invalid UUID format", err.Error())
			return
		}

		next.ServeHTTP(w, r)
	})
}
