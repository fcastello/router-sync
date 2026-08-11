package mcpserver

import (
	"net/http"
	"strings"
)

// BearerAuthMiddleware requires Authorization: Bearer <token> when token is non-empty.
func BearerAuthMiddleware(token string) func(http.Handler) http.Handler {
	token = strings.TrimSpace(token)
	if token == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	const prefix = "Bearer "
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != prefix+token {
				w.Header().Set("WWW-Authenticate", `Bearer realm="router-sync-mcp"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
