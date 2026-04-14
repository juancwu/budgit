package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/blocks"
	"git.juancwu.dev/juancwu/budgit/internal/ui/forms"
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

func (h *spaceHandler) CreateSpacePage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, pages.CreateSpace())
}

func (h *spaceHandler) HandleCreateSpace(w http.ResponseWriter, r *http.Request) {
	spaceName := strings.TrimSpace(r.FormValue("name"))

	if spaceName == "" {
		ui.Render(w, r, forms.CreateSpace("Space name can't be empty.", spaceName))
		return
	}

	user := ctxkeys.User(r.Context())

	isNameAvailable, err := h.spaceService.IsNameAvailable(spaceName, user.ID)
	if err != nil {
		slog.Error("failed to create new space", "error", err, "user_id", user.ID)
		ui.Render(w, r, forms.CreateSpace("Something went wrong. Please try again later.", spaceName))
		return
	}

	if !isNameAvailable {
		ui.Render(w, r, forms.CreateSpace("Space name is not available. Please use another name.", spaceName))
		return
	}

	sp, err := h.spaceService.CreateSpace(spaceName, user.ID)
	if err != nil {
		slog.Error("failed to create new space", "error", err, "user_id", user.ID)
		ui.Render(w, r, forms.CreateSpace("Something went wrong. Please try again later.", spaceName))
		return
	}

	ui.Render(w, r, forms.CreateSpaceSuccess(sp.ID))
}

func (h *spaceHandler) SpaceOverviewPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to fetch space data", "error", err, "spaceID", spaceID)
		ui.Render(w, r, pages.NotFound())
		return
	}

	accounts, err := h.accountService.GetAccountsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to fetch accounts for space", "error", err, "spaceID", spaceID)
		ui.RenderError(w, r, "Failed to load accounts", http.StatusInternalServerError)
		return
	}

	accountCards := make([]blocks.AccountCardInfo, 0, len(accounts))
	for _, a := range accounts {
		accountCards = append(accountCards, blocks.AccountCardInfo{
			ID:      a.ID,
			Name:    a.Name,
			Balance: a.Balance,
		})
	}

	ui.Render(w, r, pages.SpaceOverview(pages.SpaceOverviewProps{
		SpaceID:   space.ID,
		SpaceName: space.Name,
		Accounts:  accountCards,
	}))
}
