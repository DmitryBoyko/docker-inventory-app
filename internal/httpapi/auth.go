package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// WithAuth enforces Bearer / access_token when token is non-empty (ADR-013).
// Static UI paths are not protected. GET /api/v1/health and /health are exempt.
func WithAuth(token string) Middleware {
	token = strings.TrimSpace(token)
	if token == "" {
		return func(next http.Handler) http.Handler { return next }
	}
	expected := []byte(token)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !requiresAuth(r) {
				next.ServeHTTP(w, r)
				return
			}
			got := extractAccessToken(r)
			if subtle.ConstantTimeCompare([]byte(got), expected) != 1 {
				w.Header().Set("WWW-Authenticate", `Bearer realm="docker-visualizer"`)
				writeErr(w, http.StatusUnauthorized, "unauthorized", "missing or invalid auth token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requiresAuth(r *http.Request) bool {
	path := r.URL.Path
	if !strings.HasPrefix(path, "/api/") {
		return false
	}
	if r.Method == http.MethodGet && (path == "/api/v1/health" || path == "/health") {
		return false
	}
	return true
}

func extractAccessToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		const prefix = "Bearer "
		if strings.HasPrefix(h, prefix) {
			return strings.TrimSpace(h[len(prefix):])
		}
		// Also accept raw token in Authorization for simple clients.
		if !strings.Contains(h, " ") {
			return strings.TrimSpace(h)
		}
	}
	return strings.TrimSpace(r.URL.Query().Get("access_token"))
}
