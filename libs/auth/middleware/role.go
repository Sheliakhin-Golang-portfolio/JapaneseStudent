package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/japanesestudent/libs/auth/service"
)

// RoleMiddleware validates JWT access token and checks if user's role is >= requiredRole
func RoleMiddleware(tokenGenerator *service.TokenGenerator, requiredRole int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header or cookie
			var token string

			// Try Authorization header first
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				// Expected format: "Bearer <token>"
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					token = parts[1]
				}
			}

			// If not in header, try cookie
			if token == "" {
				cookie, err := r.Cookie("access_token")
				if err == nil {
					token = cookie.Value
				}
			}

			// If no token found, return 401
			if token == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"authentication required"}`))
				return
			}

			// Validate token and extract userID and role
			userID, role, err := tokenGenerator.ValidateAccessToken(token)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"invalid or expired token"}`))
				return
			}

			// Check if role is sufficient
			if role < requiredRole {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"insufficient permissions"}`))
				return
			}

			// Role is sufficient, proceed to next handler
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
