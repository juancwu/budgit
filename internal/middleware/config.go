package middleware

import (
	"net/http"

	"git.juancwu.dev/juancwu/budgething/internal/config"
	"git.juancwu.dev/juancwu/budgething/internal/ctxkeys"
)

// Config middleware adds the sanitized app configuration to the request context.
// Sensitive values like JWTSecret and DBPath are excluded for security.
func Config(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := ctxkeys.WithConfig(r.Context(), cfg.Sanitized())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
