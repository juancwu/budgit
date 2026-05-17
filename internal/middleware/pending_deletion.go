package middleware

import (
	"net/http"
	"strings"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

// pendingDeletionAllowedPaths is the small set of endpoints a user marked
// for deletion is still allowed to reach. Everything else is redirected (GET)
// or rejected (mutation) so no further data can be created or changed while
// the deletion job is in flight.
var pendingDeletionAllowedPaths = map[string]struct{}{
	"/account-pending-deletion": {},
	"/auth/logout":              {},
	"/healthz":                  {},
	"/privacy":                  {},
	"/terms":                    {},
	"/forbidden":                {},
}

// BlockPendingDeletion locks out users whose accounts are pending deletion.
// Runs after AuthMiddleware so it can read the user from context. For
// unauthenticated requests and static assets it is a no-op.
func BlockPendingDeletion(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := ctxkeys.User(r.Context())
		if user == nil || !user.IsPendingDeletion() {
			next.ServeHTTP(w, r)
			return
		}

		// Always permit static assets so the pending page can render, and
		// the dynamic deletion-status URL the user got in their email.
		if strings.HasPrefix(r.URL.Path, "/assets/") ||
			strings.HasPrefix(r.URL.Path, "/account-deletion-status/") {
			next.ServeHTTP(w, r)
			return
		}

		if _, ok := pendingDeletionAllowedPaths[r.URL.Path]; ok {
			next.ServeHTTP(w, r)
			return
		}

		// Mutations are hard-rejected so the client gets a clear signal.
		// Safe methods are redirected to the pending-deletion landing page.
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusForbidden)
			ui.Render(w, r, pages.AccountPendingDeletion(*user.PendingDeletionAt, ""))
			return
		}

		ui.Render(w, r, pages.AccountPendingDeletion(*user.PendingDeletionAt, ""))
	}
}
