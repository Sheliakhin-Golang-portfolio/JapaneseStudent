package middlewares

import (
	"net/http"

	"go.uber.org/zap"
)

// RecoveryMiddleware recovers from panics and logs the error
func RecoveryMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := GetRequestID(r.Context())
					logger.Error("panic recovered",
						zap.String("request_id", requestID),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.Any("error", err),
					)

					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":"internal server error"}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
