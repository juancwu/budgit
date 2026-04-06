package middleware

import (
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
)

// WithURLPath adds the current URL's path to the context
func WithURLPath(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxWithPath := ctxkeys.WithURLPath(r.Context(), r.URL.Path)
		next.ServeHTTP(w, r.WithContext(ctxWithPath))
	}
}
