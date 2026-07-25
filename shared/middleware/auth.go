package middleware

import (
	"context"
	"net/http"

	"nodefall/shared/jwt"
)

// contextKey avoids collisions with other packages' context keys.
type contextKey string

const userIDKey contextKey = "userID"

// WithAuth requires a valid JWT on every request, rejecting with 401
// if it's missing or invalid. On success, the verified user ID is
// made available to the wrapped handler via UserIDFromContext.
//
// The token is read from the "token" query parameter first (needed
// for WebSocket connections, which can't set custom headers during
// the handshake), falling back to a standard "Authorization: Bearer
// <token>" header for regular HTTP requests.
func WithAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			token = bearerToken(r)
		}
		if token == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		userID, err := jwt.Verify(token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext returns the verified user ID set by WithAuth, and
// whether one was present.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if len(auth) > len(prefix) && auth[:len(prefix)] == prefix {
		return auth[len(prefix):]
	}
	return ""
}
