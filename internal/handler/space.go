package handler

import (
	"log/slog"
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/misc/currency"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/blocks"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
	"github.com/shopspring/decimal"
)

type spaceHandler struct {
	spaceService *service.SpaceService
}

func NewSpaceHandler(spaceService *service.SpaceService) *spaceHandler {
	return &spaceHandler{spaceService: spaceService}
}

func (h *spaceHandler) SpacesPage(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	if user == nil {
		ui.RenderError(w, r, "Unauthorized", http.StatusUnauthorized)
		return
	}

	spaces, err := h.spaceService.GetSpacesForUser(user.ID)
	if err != nil {
		slog.Error("failed to load spaces", "error", err, "user_id", user.ID)
		ui.RenderError(w, r, "Failed to load spaces", http.StatusInternalServerError)
		return
	}

	cards := make([]blocks.SpaceCardInfo, 0, len(spaces))
	for _, sp := range spaces {
		members, err := h.spaceService.GetMembers(sp.ID)
		if err != nil {
			slog.Error("failed to load space members", "error", err, "space_id", sp.ID)
			members = nil
		}
		cards = append(cards, blocks.SpaceCardInfo{
			ID:           sp.ID,
			Name:         sp.Name,
			MemberCount:  len(members),
			TotalBalance: decimal.Zero,
			Currency:     currency.CAD,
		})
	}

	ui.Render(w, r, pages.Spaces(cards))
}
