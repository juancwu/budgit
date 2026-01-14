package handler

import (
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type dashboardHandler struct{}

func NewDashboardHandler() *dashboardHandler {
	return &dashboardHandler{}
}

func (h *dashboardHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, pages.Dashboard())
}
