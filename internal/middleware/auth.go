package middleware

import (
	"net/http"

	"git.juancwu.dev/juancwu/budgething/internal/ctxkeys"
)

// RequireGuest ensures request is not authenticated
func RequireGuest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := ctxkeys.User(r.Context())
		if user != nil {
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/app/dashboard")
				w.WriteHeader(http.StatusSeeOther)
				return
			}
			http.Redirect(w, r, "/app/dashboard", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	}
}
