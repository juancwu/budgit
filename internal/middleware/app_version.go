package middleware

import (
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
)

func AppVersion(version string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := ctxkeys.WithAppVersion(r.Context(), version)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
