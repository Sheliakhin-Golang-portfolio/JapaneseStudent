package middlewares

import (
	"net/http"
	"slices"
	"strings"
)

// CORSMiddleware creates a CORS middleware with the specified allowed origins
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowedOrigin := getAllowedOrigin(origin, allowedOrigins)

			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getAllowedOrigin checks if the request origin is in the allowed origins list
// Returns the origin if allowed, "*" if all origins are allowed, or empty string if not allowed
func getAllowedOrigin(requestOrigin string, allowedOrigins []string) string {
	// If no origin in request, don't set CORS header
	if requestOrigin == "" {
		return ""
	}

	// Check if all origins are allowed
	if slices.Contains(allowedOrigins, "*") {
		return "*"
	}

	// Check if the request origin matches any allowed origin
	for _, allowed := range allowedOrigins {
		if strings.EqualFold(requestOrigin, allowed) {
			return requestOrigin
		}
	}

	// Origin not allowed
	return ""
}
