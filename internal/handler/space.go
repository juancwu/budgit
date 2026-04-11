package handler

import (
	"log/slog"
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/blocks"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
	"github.com/shopspring/decimal"
)

type spaceHandler struct {
	spaceService   *service.SpaceService
	accountService *service.AccountService
}

func NewSpaceHandler(spaceService *service.SpaceService, accountService *service.AccountService) *spaceHandler {
	return &spaceHandler{spaceService: spaceService, accountService: accountService}
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
		memberCount, err := h.spaceService.GetMemberCount(sp.ID)
		if err != nil {
			slog.Error("failed to get space member count", "error", err, "space_id", sp.ID)
			memberCount = 0
			err = nil
		}

		accounts, err := h.accountService.GetAccountsForSpace(sp.ID)
		if err != nil {
			slog.Error("failed to get space accounts", "error", err, "space_id", sp.ID)
			accounts = nil
			err = nil
		}

		totalBalance := decimal.Zero
		for _, acc := range accounts {
			totalBalance = totalBalance.Add(acc.Balance)
		}

		cards = append(cards, blocks.SpaceCardInfo{
			ID:           sp.ID,
			Name:         sp.Name,
			MemberCount:  memberCount,
			TotalBalance: totalBalance,
		})
	}

	ui.Render(w, r, pages.Spaces(cards))
}
