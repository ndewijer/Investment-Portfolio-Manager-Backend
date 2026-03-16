package middleware

import (
	"net"
	"net/http"
	"strings"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/ndewijer/Investment-Portfolio-Manager-Backend/internal/logging"
)

var log = logging.NewLogger("system")

// Logger is a middleware that enriches the request context with logging metadata
// and logs each HTTP request with structured fields.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Enrich context with request info for downstream slog calls.
		reqID := chimw.GetReqID(r.Context())
		ip := clientIP(r)
		ua := r.Header.Get("User-Agent")
		ctx := logging.WithRequestInfo(r.Context(), reqID, ip, ua)
		r = r.WithContext(ctx)

		// Create a response writer wrapper to capture status code.
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler.
		next.ServeHTTP(wrapped, r)

		// Sanitize user-supplied values to prevent log injection: strip CR/LF.
		sanitize := strings.NewReplacer("\n", "", "\r", "").Replace

		log.InfoContext(ctx, "http request",
			"method", sanitize(r.Method),
			"path", sanitize(r.URL.Path),
			"status", wrapped.statusCode,
			"duration", time.Since(start).String(),
		)
	})
}

// clientIP extracts the client IP address from a request.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i != -1 {
			xff = xff[:i]
		}
		return strings.TrimSpace(xff)
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
