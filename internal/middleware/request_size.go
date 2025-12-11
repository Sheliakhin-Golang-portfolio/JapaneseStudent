package middleware

import (
	"net/http"
)

const maxRequestSize = 10 * 1024 * 1024 // 10MB

// RequestSizeLimitMiddleware limits the size of request bodies
func RequestSizeLimitMiddleware(next http.Handler) http.Handler {
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
