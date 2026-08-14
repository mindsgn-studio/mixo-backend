package admin

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/mindsgn-studio/mixo-backend/internal/config"
)

// BasicAuth protects the /admin interface (and every /admin/* route) with HTTP
// Basic Auth. Authentication is enabled only when ADMIN_PASSWORD is configured
// in .env. When it is empty the middleware is a no-op, preserving the default
// open behaviour.
func BasicAuth(cfg *config.Config) func(http.Handler) http.Handler {
	expectedUser := cfg.AdminUsername
	if expectedUser == "" {
		expectedUser = "admin"
	}
	expectedPass := cfg.AdminPassword

	return func(next http.Handler) http.Handler {
		if expectedPass == "" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isAdminPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			user, pass, ok := r.BasicAuth()
			if !ok ||
				subtle.ConstantTimeCompare([]byte(user), []byte(expectedUser)) != 1 ||
				subtle.ConstantTimeCompare([]byte(pass), []byte(expectedPass)) != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="Hackday Radio admin"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isAdminPath reports whether the request targets the admin interface.
func isAdminPath(path string) bool {
	return path == "/admin" || strings.HasPrefix(path, "/admin/")
}
