package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// Logger is a middleware that logs HTTP requests
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Sanitize user-supplied values to prevent log injection: strip CR/LF before logging.
		sanitize := strings.NewReplacer("\n", "", "\r", "").Replace
		//nolint:gosec // G706: method and path are sanitized above to strip newlines/carriage-returns before logging.
		log.Printf(
			"%s %s %d %s",
			sanitize(r.Method),
			sanitize(r.URL.Path),
			wrapped.statusCode,
			time.Since(start),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
