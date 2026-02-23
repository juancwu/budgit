package middleware

import (
	"log/slog"
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/service"
)

// RequireSpaceAccess validates that a user is a member of the space they are trying to access.
// It expects a URL parameter named "spaceID".
func RequireSpaceAccess(spaceService *service.SpaceService) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user := ctxkeys.User(r.Context())
			if user == nil {
				// This should be caught by RequireAuth first, but as a safeguard.
				redirect(w, r, "/auth", http.StatusSeeOther)
				return
			}

			spaceID := r.PathValue("spaceID")
			if spaceID == "" {
				slog.Warn("RequireSpaceAccess middleware used on a route without a {spaceID} path parameter")
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}

			isMember, err := spaceService.IsMember(user.ID, spaceID)
			if err != nil {
				slog.Error("failed to check space membership", "error", err, "user_id", user.ID, "space_id", spaceID)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if !isMember {
				if r.Header.Get("HX-Request") == "true" {
					w.Header().Set("HX-Redirect", "/forbidden")
					w.WriteHeader(http.StatusSeeOther)
					return
				}
				http.Redirect(w, r, "/forbidden", http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		}
	}
}
