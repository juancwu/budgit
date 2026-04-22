package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/routeurl"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/blocks"
	"git.juancwu.dev/juancwu/budgit/internal/ui/forms"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
	"github.com/shopspring/decimal"
)

type spaceHandler struct {
	spaceService       *service.SpaceService
	accountService     *service.AccountService
	transactionService *service.TransactionService
}

func NewSpaceHandler(
	spaceService *service.SpaceService,
	accountService *service.AccountService,
	transactionService *service.TransactionService,
) *spaceHandler {
	return &spaceHandler{
		spaceService:       spaceService,
		accountService:     accountService,
		transactionService: transactionService,
	}
}

func (h *spaceHandler) SpacesPage(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	if user == nil {
		ui.RenderError(w, r, "Unauthorized", http.StatusUnauthorized)
		return
	}

	spaces, err := h.spaceService.GetOwnedSpaces(user.ID)
	if err != nil {
		slog.Error("failed to load spaces", "error", err, "user_id", user.ID)
		ui.RenderError(w, r, "Failed to load spaces", http.StatusInternalServerError)
		return
	}

	cards := h.buildSpaceCards(spaces)
	ui.Render(w, r, pages.Spaces(cards))
}

func (h *spaceHandler) SharedSpacesPage(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	if user == nil {
		ui.RenderError(w, r, "Unauthorized", http.StatusUnauthorized)
		return
	}

	spaces, err := h.spaceService.GetSharedSpaces(user.ID)
	if err != nil {
		slog.Error("failed to load shared spaces", "error", err, "user_id", user.ID)
		ui.RenderError(w, r, "Failed to load shared spaces", http.StatusInternalServerError)
		return
	}

	cards := h.buildSpaceCards(spaces)
	ui.Render(w, r, pages.SharedSpaces(cards))
}

func (h *spaceHandler) buildSpaceCards(spaces []*model.Space) []blocks.SpaceCardInfo {
	cards := make([]blocks.SpaceCardInfo, 0, len(spaces))
	for _, sp := range spaces {
		memberCount, err := h.spaceService.GetMemberCount(sp.ID)
		if err != nil {
			slog.Error("failed to get space member count", "error", err, "space_id", sp.ID)
			memberCount = 0
		}

		accounts, err := h.accountService.GetAccountsForSpace(sp.ID)
		if err != nil {
			slog.Error("failed to get space accounts", "error", err, "space_id", sp.ID)
			accounts = nil
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
	return cards
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
			SpaceID: space.ID,
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

func (h *spaceHandler) SpaceAccountPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	account, err := h.accountService.GetAccount(accountID)
	if err != nil {
		slog.Error("failed to load account", "error", err, "account_id", accountID)
		ui.Render(w, r, pages.NotFound())
		return
	}

	if account.SpaceID != spaceID {
		ui.Render(w, r, pages.NotFound())
		return
	}

	ui.Render(w, r, pages.SpaceAccountPage(pages.SpaceAccountPageProps{
		SpaceID:        spaceID,
		AccountID:      accountID,
		AccountName:    account.Name,
		AccountBalance: account.Balance,
	}))
}

func (h *spaceHandler) SpaceCreateBillPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	account, err := h.accountService.GetAccount(accountID)
	if err != nil {
		slog.Error("failed to load account", "error", err, "account_id", accountID)
		ui.Render(w, r, pages.NotFound())
		return
	}
	if account.SpaceID != spaceID {
		ui.Render(w, r, pages.NotFound())
		return
	}

	categories, err := h.transactionService.ListCategories()
	if err != nil {
		slog.Error("failed to load categories", "error", err)
		ui.RenderError(w, r, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceCreateBillPage(pages.SpaceCreateBillPageProps{
		SpaceID:     spaceID,
		AccountID:   accountID,
		AccountName: account.Name,
		Form: forms.CreateBillProps{
			SpaceID:    spaceID,
			AccountID:  accountID,
			Categories: categories,
			Date:       time.Now().Format("2006-01-02"),
		},
	}))
}

func (h *spaceHandler) HandleCreateBill(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	titleInput := strings.TrimSpace(r.FormValue("title"))
	amountInput := strings.TrimSpace(r.FormValue("amount"))
	dateInput := strings.TrimSpace(r.FormValue("date"))
	descriptionInput := strings.TrimSpace(r.FormValue("description"))
	categoryInput := strings.TrimSpace(r.FormValue("category"))

	categories, err := h.transactionService.ListCategories()
	if err != nil {
		slog.Error("failed to load categories", "error", err)
		ui.RenderError(w, r, "Failed to load categories", http.StatusInternalServerError)
		return
	}

	formProps := forms.CreateBillProps{
		SpaceID:     spaceID,
		AccountID:   accountID,
		Categories:  categories,
		Title:       titleInput,
		Amount:      amountInput,
		Date:        dateInput,
		Description: descriptionInput,
		CategoryID:  categoryInput,
	}

	hasErr := false
	if titleInput == "" {
		formProps.TitleErr = "Title is required."
		hasErr = true
	}

	var amount decimal.Decimal
	if amountInput == "" {
		formProps.AmountErr = "Amount is required."
		hasErr = true
	} else {
		amt, err := decimal.NewFromString(amountInput)
		if err != nil {
			formProps.AmountErr = "Enter a valid amount (e.g. 12.34)."
			hasErr = true
		} else if !amt.IsPositive() {
			formProps.AmountErr = "Amount must be greater than zero."
			hasErr = true
		} else if amt.Exponent() < -2 {
			formProps.AmountErr = "Amount can have at most 2 decimal places."
			hasErr = true
		} else {
			amount = amt
		}
	}

	var occurredAt time.Time
	if dateInput == "" {
		formProps.DateErr = "Date is required."
		hasErr = true
	} else {
		parsed, err := time.Parse("2006-01-02", dateInput)
		if err != nil {
			formProps.DateErr = "Enter a valid date."
			hasErr = true
		} else {
			occurredAt = parsed
		}
	}

	if hasErr {
		ui.Render(w, r, forms.CreateBill(formProps))
		return
	}

	_, err = h.transactionService.PayBill(service.PayBillInput{
		AccountID:   accountID,
		Title:       titleInput,
		Amount:      amount,
		OccurredAt:  occurredAt,
		Description: descriptionInput,
		CategoryID:  categoryInput,
	})
	if err != nil {
		slog.Error("failed to create bill", "error", err, "account_id", accountID)
		formProps.GeneralErr = "Something went wrong. Please try again."
		ui.Render(w, r, forms.CreateBill(formProps))
		return
	}

	redirectTo := routeurl.URL(
		"page.app.spaces.space.accounts.account.overview",
		"spaceID", spaceID,
		"accountID", accountID,
	)
	w.Header().Set("HX-Redirect", redirectTo)
	w.WriteHeader(http.StatusOK)
}
