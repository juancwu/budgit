package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type dashboardHandler struct {
	spaceService   *service.SpaceService
	expenseService *service.ExpenseService
}

func NewDashboardHandler(ss *service.SpaceService, es *service.ExpenseService) *dashboardHandler {
	return &dashboardHandler{
		spaceService:   ss,
		expenseService: es,
	}
}

func (h *dashboardHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	spaces, err := h.spaceService.GetSpacesForUser(user.ID)
	if err != nil {
		slog.Error("failed to get spaces for user", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var totalBalance int
	for _, space := range spaces {
		balance, err := h.expenseService.GetBalanceForSpace(space.ID)
		if err != nil {
			slog.Error("failed to get balance for space", "error", err, "space_id", space.ID)
			continue
		}
		totalBalance += balance
	}

	ui.Render(w, r, pages.Dashboard(spaces, totalBalance))
}

func (h *dashboardHandler) CreateSpace(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "Space name is required", http.StatusBadRequest)
		return
	}

	space, err := h.spaceService.CreateSpace(name, user.ID)
	if err != nil {
		slog.Error("failed to create space", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/app/spaces/"+space.ID)
	w.WriteHeader(http.StatusOK)
}
