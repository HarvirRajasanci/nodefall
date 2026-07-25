// Package middleware provides HTTP middleware shared across Nodefall's
// services — CORS handling, and eventually JWT verification for
// endpoints that require an authenticated caller.
package middleware

import "net/http"

// WithCORS wraps a handler with permissive CORS headers, allowing any
// origin to call it. Suitable for local development; the allowed
// origin should be restricted before this is deployed anywhere real.
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
