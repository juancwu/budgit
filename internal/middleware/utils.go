package middleware

import (
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

// Redirect handles both HTMX and regular HTTP redirects.
// For HTMX requests, it sets the HX-Redirect header; for regular requests,
// it uses http.Redirect.
func redirect(w http.ResponseWriter, r *http.Request, path string, code int) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", path)
		w.WriteHeader(code)
		return
	}
	http.Redirect(w, r, path, code)
}

func notfound(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, pages.NotFound())
}
