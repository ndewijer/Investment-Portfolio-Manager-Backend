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
func ValidatePortfolioIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		portfolioID := chi.URLParam(r, "portfolioId")

		if portfolioID == "" {
			response.RespondError(w, http.StatusBadRequest, "portfolio ID is required", "")
			return
		}

		if err := validation.ValidateUUID(portfolioID); err != nil {
			response.RespondError(w, http.StatusBadRequest, "invalid portfolio ID format", err.Error())
			return
		}

		next.ServeHTTP(w, r)
	})
}
