package middleware

import (
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
)

// WithSidebarState reads the sidebar_state cookie and adds the collapsed
// state to the request context so templates can render the sidebar in the
// correct initial state.
func WithSidebarState(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collapsed := false
		if c, err := r.Cookie("sidebar_state"); err == nil {
			collapsed = c.Value == "false"
		}
		ctx := ctxkeys.WithSidebarCollapsed(r.Context(), collapsed)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
