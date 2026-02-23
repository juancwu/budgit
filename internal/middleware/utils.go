package middleware

import (
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

func redirect(w http.ResponseWriter, r *http.Request, path string, code int) {
	// For HTMX requests, use HX-Redirect header to force full page redirect
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/auth")
		w.WriteHeader(code)
		return
	}
	// For regular requests, use standard redirect
	http.Redirect(w, r, "/auth", code)
}

func notfound(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, pages.NotFound())
}
