package middleware

import (
	"net/http"
)

// APIKeyMiddleware validates API key from X-API-Key header
// It compares the header value with the configured API key
func APIKeyMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract API key from header
			providedKey := r.Header.Get("X-API-Key")

			// If no API key provided or it doesn't match, return 401
			if providedKey == "" || providedKey != apiKey {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"invalid or missing API key"}`))
				return
			}

			// API key is valid, proceed to next handler
			next.ServeHTTP(w, r)
		})
	}
}
