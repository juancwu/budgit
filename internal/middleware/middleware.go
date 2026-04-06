package middleware

import (
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type Middleware func(http.Handler) http.HandlerFunc

// Chain applies multiple middleware in order (first to last)
// The middleware are executed in the order they are provided
//
// Example:
//
//	handler := Chain(mux,
//	    AuthMiddleware(...),  // Executes first
//	    WithURLPath,          // Executes second
//	    Config(...),          // Executes third
//	)
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	// Apply middleware in reverse order so they execute in the order provided
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

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
