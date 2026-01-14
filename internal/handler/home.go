package handler

import (
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type homeHandler struct{}

func NewHomeHandler() *homeHandler {
	return &homeHandler{}
}

// HomePage will redirect to /auth if not authenticated or to /app/dashboard if authenticated.
func (h *homeHandler) HomePage(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/dashboard", http.StatusSeeOther)
}

func (home *homeHandler) NotFoundPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, pages.NotFound())
}

func (h *homeHandler) ForbiddenPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, pages.Forbidden())
}
