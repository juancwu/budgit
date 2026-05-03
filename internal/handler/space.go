package handler

import (
	"log/slog"
	"net/http"
	"strconv"
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

	total := decimal.Zero
	for _, c := range cards {
		total = total.Add(c.TotalBalance)
	}

	ui.Render(w, r, pages.Spaces(cards, total))
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

func (h *spaceHandler) SpaceCreateAccountPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to fetch space", "error", err, "space_id", spaceID)
		ui.Render(w, r, pages.NotFound())
		return
	}

	ui.Render(w, r, pages.SpaceCreateAccountPage(pages.SpaceCreateAccountPageProps{
		SpaceID:   space.ID,
		SpaceName: space.Name,
		Form: forms.CreateAccountProps{
			SpaceID: space.ID,
		},
	}))
}

func (h *spaceHandler) HandleCreateAccount(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	nameInput := strings.TrimSpace(r.FormValue("name"))

	formProps := forms.CreateAccountProps{
		SpaceID: spaceID,
		Name:    nameInput,
	}

	if nameInput == "" {
		formProps.NameErr = "Account name is required."
		ui.Render(w, r, forms.CreateAccount(formProps))
		return
	}

	existing, err := h.accountService.GetAccountsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to load accounts", "error", err, "space_id", spaceID)
		formProps.GeneralErr = "Something went wrong. Please try again."
		ui.Render(w, r, forms.CreateAccount(formProps))
		return
	}
	for _, a := range existing {
		if strings.EqualFold(strings.TrimSpace(a.Name), nameInput) {
			formProps.NameErr = "An account with this name already exists in this space."
			ui.Render(w, r, forms.CreateAccount(formProps))
			return
		}
	}

	account, err := h.accountService.CreateAccount(spaceID, nameInput)
	if err != nil {
		slog.Error("failed to create account", "error", err, "space_id", spaceID)
		formProps.GeneralErr = "Something went wrong. Please try again."
		ui.Render(w, r, forms.CreateAccount(formProps))
		return
	}

	redirectTo := routeurl.URL(
		"page.app.spaces.space.accounts.account.overview",
		"spaceID", spaceID,
		"accountID", account.ID,
	)
	w.Header().Set("HX-Redirect", redirectTo)
	w.WriteHeader(http.StatusOK)
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

	recent, err := h.transactionService.ListByAccount(accountID, 5, 0)
	if err != nil {
		slog.Error("failed to load recent transactions", "error", err, "account_id", accountID)
		recent = nil
	}

	ui.Render(w, r, pages.SpaceAccountPage(pages.SpaceAccountPageProps{
		SpaceID:            spaceID,
		AccountID:          accountID,
		AccountName:        account.Name,
		AccountBalance:     account.Balance,
		RecentTransactions: recent,
	}))
}

func (h *spaceHandler) SpaceAccountTransactionsPage(w http.ResponseWriter, r *http.Request) {
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

	const perPage = 25
	page := 1
	if p := strings.TrimSpace(r.URL.Query().Get("page")); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	total, err := h.transactionService.CountByAccount(accountID)
	if err != nil {
		slog.Error("failed to count transactions", "error", err, "account_id", accountID)
		ui.RenderError(w, r, "Failed to load transactions", http.StatusInternalServerError)
		return
	}

	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * perPage
	txns, err := h.transactionService.ListByAccount(accountID, perPage, offset)
	if err != nil {
		slog.Error("failed to load transactions", "error", err, "account_id", accountID)
		ui.RenderError(w, r, "Failed to load transactions", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceAccountTransactionsPage(pages.SpaceAccountTransactionsPageProps{
		SpaceID:      spaceID,
		AccountID:    accountID,
		AccountName:  account.Name,
		Transactions: txns,
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalCount:   total,
		PerPage:      perPage,
	}))
}

func (h *spaceHandler) SpaceSettingsPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to fetch space", "error", err, "space_id", spaceID)
		ui.Render(w, r, pages.NotFound())
		return
	}

	user := ctxkeys.User(r.Context())
	canDelete := user != nil && user.ID == space.OwnerID

	ui.Render(w, r, pages.SpaceSettingsPage(pages.SpaceSettingsPageProps{
		SpaceID:   space.ID,
		SpaceName: space.Name,
		CanDelete: canDelete,
		UpdateForm: forms.UpdateSpaceProps{
			SpaceID: space.ID,
			Name:    space.Name,
		},
	}))
}

func (h *spaceHandler) HandleRenameSpace(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		ui.RenderError(w, r, "Space not found", http.StatusNotFound)
		return
	}

	user := ctxkeys.User(r.Context())
	if user == nil || user.ID != space.OwnerID {
		ui.RenderError(w, r, "Forbidden", http.StatusForbidden)
		return
	}

	nameInput := strings.TrimSpace(r.FormValue("name"))
	formProps := forms.UpdateSpaceProps{
		SpaceID: spaceID,
		Name:    nameInput,
	}

	if nameInput == "" {
		formProps.NameErr = "Space name is required."
		ui.Render(w, r, forms.UpdateSpace(formProps))
		return
	}

	if !strings.EqualFold(nameInput, space.Name) {
		available, err := h.spaceService.IsNameAvailable(nameInput, user.ID)
		if err != nil {
			slog.Error("failed to check name availability", "error", err, "user_id", user.ID)
			formProps.GeneralErr = "Something went wrong. Please try again."
			ui.Render(w, r, forms.UpdateSpace(formProps))
			return
		}
		if !available {
			formProps.NameErr = "You already have a space with this name."
			ui.Render(w, r, forms.UpdateSpace(formProps))
			return
		}
	}

	if err := h.spaceService.UpdateSpaceName(spaceID, nameInput); err != nil {
		slog.Error("failed to rename space", "error", err, "space_id", spaceID)
		formProps.GeneralErr = "Something went wrong. Please try again."
		ui.Render(w, r, forms.UpdateSpace(formProps))
		return
	}

	formProps.SuccessMsg = "Space name updated."
	ui.Render(w, r, forms.UpdateSpace(formProps))
}

func (h *spaceHandler) HandleDeleteSpace(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		ui.RenderError(w, r, "Space not found", http.StatusNotFound)
		return
	}

	user := ctxkeys.User(r.Context())
	if user == nil || user.ID != space.OwnerID {
		ui.RenderError(w, r, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.spaceService.DeleteSpace(spaceID); err != nil {
		slog.Error("failed to delete space", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to delete space", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", routeurl.URL("page.app.spaces"))
	w.WriteHeader(http.StatusOK)
}

func (h *spaceHandler) SpaceAccountSettingsPage(w http.ResponseWriter, r *http.Request) {
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

	ui.Render(w, r, pages.SpaceAccountSettingsPage(pages.SpaceAccountSettingsPageProps{
		SpaceID:     spaceID,
		AccountID:   accountID,
		AccountName: account.Name,
		UpdateForm: forms.UpdateAccountProps{
			SpaceID:   spaceID,
			AccountID: accountID,
			Name:      account.Name,
		},
	}))
}

func (h *spaceHandler) HandleRenameAccount(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	account, err := h.accountService.GetAccount(accountID)
	if err != nil || account.SpaceID != spaceID {
		ui.RenderError(w, r, "Account not found", http.StatusNotFound)
		return
	}

	nameInput := strings.TrimSpace(r.FormValue("name"))
	formProps := forms.UpdateAccountProps{
		SpaceID:   spaceID,
		AccountID: accountID,
		Name:      nameInput,
	}

	if nameInput == "" {
		formProps.NameErr = "Account name is required."
		ui.Render(w, r, forms.UpdateAccount(formProps))
		return
	}

	if !strings.EqualFold(nameInput, account.Name) {
		existing, err := h.accountService.GetAccountsForSpace(spaceID)
		if err != nil {
			slog.Error("failed to load accounts", "error", err, "space_id", spaceID)
			formProps.GeneralErr = "Something went wrong. Please try again."
			ui.Render(w, r, forms.UpdateAccount(formProps))
			return
		}
		for _, a := range existing {
			if a.ID == accountID {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(a.Name), nameInput) {
				formProps.NameErr = "An account with this name already exists in this space."
				ui.Render(w, r, forms.UpdateAccount(formProps))
				return
			}
		}
	}

	if err := h.accountService.RenameAccount(accountID, nameInput); err != nil {
		slog.Error("failed to rename account", "error", err, "account_id", accountID)
		formProps.GeneralErr = "Something went wrong. Please try again."
		ui.Render(w, r, forms.UpdateAccount(formProps))
		return
	}

	formProps.SuccessMsg = "Account name updated."
	ui.Render(w, r, forms.UpdateAccount(formProps))
}

func (h *spaceHandler) HandleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	account, err := h.accountService.GetAccount(accountID)
	if err != nil || account.SpaceID != spaceID {
		ui.RenderError(w, r, "Account not found", http.StatusNotFound)
		return
	}

	if err := h.accountService.DeleteAccount(accountID); err != nil {
		slog.Error("failed to delete account", "error", err, "account_id", accountID)
		ui.RenderError(w, r, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	redirectTo := routeurl.URL("page.app.spaces.space.overview", "spaceID", spaceID)
	w.Header().Set("HX-Redirect", redirectTo)
	w.WriteHeader(http.StatusOK)
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

func (h *spaceHandler) SpaceCreateDepositPage(w http.ResponseWriter, r *http.Request) {
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

	ui.Render(w, r, pages.SpaceCreateDepositPage(pages.SpaceCreateDepositPageProps{
		SpaceID:     spaceID,
		AccountID:   accountID,
		AccountName: account.Name,
		Form: forms.CreateDepositProps{
			SpaceID:   spaceID,
			AccountID: accountID,
			Date:      time.Now().Format("2006-01-02"),
		},
	}))
}

func (h *spaceHandler) HandleCreateDeposit(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	titleInput := strings.TrimSpace(r.FormValue("title"))
	amountInput := strings.TrimSpace(r.FormValue("amount"))
	dateInput := strings.TrimSpace(r.FormValue("date"))
	descriptionInput := strings.TrimSpace(r.FormValue("description"))

	formProps := forms.CreateDepositProps{
		SpaceID:     spaceID,
		AccountID:   accountID,
		Title:       titleInput,
		Amount:      amountInput,
		Date:        dateInput,
		Description: descriptionInput,
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
		ui.Render(w, r, forms.CreateDeposit(formProps))
		return
	}

	_, err := h.transactionService.Deposit(service.DepositInput{
		AccountID:   accountID,
		Title:       titleInput,
		Amount:      amount,
		OccurredAt:  occurredAt,
		Description: descriptionInput,
	})
	if err != nil {
		slog.Error("failed to create deposit", "error", err, "account_id", accountID)
		formProps.GeneralErr = "Something went wrong. Please try again."
		ui.Render(w, r, forms.CreateDeposit(formProps))
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
