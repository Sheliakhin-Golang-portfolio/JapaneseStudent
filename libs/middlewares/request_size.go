package middlewares

import (
	"net/http"
)

// RequestSizeLimitMiddleware limits the size of request bodies
// maxRequestSize specifies the maximum request body size in bytes
func RequestSizeLimitMiddleware(maxRequestSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxRequestSize {
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				w.Write([]byte(`{"error":"request body too large"}`))
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
			next.ServeHTTP(w, r)
		})
	}
}
