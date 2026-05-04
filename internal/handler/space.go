package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/misc/currency"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/routeurl"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/blocks"
	"git.juancwu.dev/juancwu/budgit/internal/ui/forms"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
	"git.juancwu.dev/juancwu/budgit/internal/validation"
	"github.com/shopspring/decimal"
)

type spaceHandler struct {
	spaceService       *service.SpaceService
	accountService     *service.AccountService
	transactionService *service.TransactionService
	allocationService  *service.AllocationService
	inviteService      *service.InviteService
	auditLogService    *service.SpaceAuditLogService
	txAuditLogService  *service.TransactionAuditLogService
	accountActivitySvc *service.AccountActivityService
}

func NewSpaceHandler(
	spaceService *service.SpaceService,
	accountService *service.AccountService,
	transactionService *service.TransactionService,
	allocationService *service.AllocationService,
	inviteService *service.InviteService,
	auditLogService *service.SpaceAuditLogService,
	txAuditLogService *service.TransactionAuditLogService,
	accountActivitySvc *service.AccountActivityService,
) *spaceHandler {
	return &spaceHandler{
		spaceService:       spaceService,
		accountService:     accountService,
		transactionService: transactionService,
		allocationService:  allocationService,
		inviteService:      inviteService,
		auditLogService:    auditLogService,
		txAuditLogService:  txAuditLogService,
		accountActivitySvc: accountActivitySvc,
	}
}

func (h *spaceHandler) HomePage(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	if user == nil {
		ui.RenderError(w, r, "Unauthorized", http.StatusUnauthorized)
		return
	}

	owned, err := h.spaceService.GetOwnedSpaces(user.ID)
	if err != nil {
		slog.Error("failed to load owned spaces", "error", err, "user_id", user.ID)
		ui.RenderError(w, r, "Failed to load spaces", http.StatusInternalServerError)
		return
	}

	shared, err := h.spaceService.GetSharedSpaces(user.ID)
	if err != nil {
		slog.Error("failed to load shared spaces", "error", err, "user_id", user.ID)
		ui.RenderError(w, r, "Failed to load spaces", http.StatusInternalServerError)
		return
	}

	ownedCards := h.buildSpaceCards(owned)
	sharedCards := h.buildSpaceCards(shared)

	total := decimal.Zero
	for _, c := range ownedCards {
		total = total.Add(c.TotalBalance)
	}

	ui.Render(w, r, pages.Home(pages.HomeProps{
		OwnedSpaces:  ownedCards,
		SharedSpaces: sharedCards,
		TotalBalance: total,
	}))
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
			SpaceID:  space.ID,
			ID:       a.ID,
			Name:     a.Name,
			Balance:  a.Balance,
			Currency: a.Currency,
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
	currencyInput := currency.Normalize(r.FormValue("currency"))
	if currencyInput == "" {
		currencyInput = currency.Default
	}

	formProps := forms.CreateAccountProps{
		SpaceID:  spaceID,
		Name:     nameInput,
		Currency: currencyInput,
	}

	hasErr := false
	if nameInput == "" {
		formProps.NameErr = "Account name is required."
		hasErr = true
	}
	if !currency.IsValid(currencyInput) {
		formProps.CurrencyErr = "Choose a supported currency."
		hasErr = true
	}
	if hasErr {
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

	user := ctxkeys.User(r.Context())
	actorID := ""
	if user != nil {
		actorID = user.ID
	}
	account, err := h.accountService.CreateAccount(spaceID, nameInput, currencyInput, actorID)
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

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to load space", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load account", http.StatusInternalServerError)
		return
	}

	recent, err := h.transactionService.ListByAccount(accountID, 5, 0)
	if err != nil {
		slog.Error("failed to load recent transactions", "error", err, "account_id", accountID)
		recent = nil
	}

	allocSummary, err := h.allocationService.SummaryForAccount(accountID)
	if err != nil {
		slog.Error("failed to load allocation summary", "error", err, "account_id", accountID)
		allocSummary = nil
	}

	ui.Render(w, r, pages.SpaceAccountPage(pages.SpaceAccountPageProps{
		SpaceID:                  spaceID,
		SpaceName:                space.Name,
		AccountID:                accountID,
		AccountName:              account.Name,
		AccountBalance:           account.Balance,
		AccountCurrency:          account.Currency,
		RecentTransactions:       recent,
		NonEditableTransactionIDs: h.nonEditableTransactionIDs(recent),
		AllocationSummary:        allocSummary,
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

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to load space", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load transactions", http.StatusInternalServerError)
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
		SpaceID:                  spaceID,
		SpaceName:                space.Name,
		AccountID:                accountID,
		AccountName:              account.Name,
		Transactions:             txns,
		NonEditableTransactionIDs: h.nonEditableTransactionIDs(txns),
		CurrentPage:              page,
		TotalPages:               totalPages,
		TotalCount:               total,
		PerPage:                  perPage,
	}))
}

// nonEditableTransactionIDs returns the subset of the given transactions that
// are part of a transfer pair and therefore not editable. Returns an empty
// (non-nil) map on error so list rendering still works — failure here just
// means stale Edit buttons appear; the service layer will still refuse the
// edit, so it's a UX degradation rather than a correctness issue.
func (h *spaceHandler) nonEditableTransactionIDs(txns []*model.Transaction) map[string]bool {
	if len(txns) == 0 {
		return nil
	}
	ids := make([]string, len(txns))
	for i, t := range txns {
		ids[i] = t.ID
	}
	hits, err := h.transactionService.TransferIDsIn(ids)
	if err != nil {
		slog.Error("failed to look up transfer ids", "error", err)
		return map[string]bool{}
	}
	return hits
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

	if err := h.spaceService.UpdateSpaceName(spaceID, nameInput, user.ID); err != nil {
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

	if err := h.spaceService.DeleteSpace(spaceID, user.ID); err != nil {
		slog.Error("failed to delete space", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to delete space", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", routeurl.URL("page.app.spaces"))
	w.WriteHeader(http.StatusOK)
}

func (h *spaceHandler) SpaceMembersPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to load space", "error", err, "space_id", spaceID)
		ui.Render(w, r, pages.NotFound())
		return
	}

	members, err := h.spaceService.GetMembers(spaceID)
	if err != nil {
		slog.Error("failed to load members", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load members", http.StatusInternalServerError)
		return
	}

	pending, err := h.inviteService.GetPendingInvites(spaceID)
	if err != nil {
		slog.Error("failed to load pending invites", "error", err, "space_id", spaceID)
		pending = nil
	}

	user := ctxkeys.User(r.Context())
	currentUserID := ""
	if user != nil {
		currentUserID = user.ID
	}
	isOwner := user != nil && user.ID == space.OwnerID

	ui.Render(w, r, pages.SpaceMembersPage(pages.SpaceMembersPageProps{
		SpaceID:        space.ID,
		SpaceName:      space.Name,
		OwnerID:        space.OwnerID,
		CurrentUserID:  currentUserID,
		IsOwner:        isOwner,
		Members:        members,
		PendingInvites: pending,
		InviteForm:     forms.InviteMemberProps{SpaceID: space.ID},
	}))
}

func (h *spaceHandler) HandleInviteMember(w http.ResponseWriter, r *http.Request) {
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

	emailInput := strings.TrimSpace(r.FormValue("email"))
	formProps := forms.InviteMemberProps{
		SpaceID: spaceID,
		Email:   emailInput,
	}

	if emailInput == "" {
		formProps.EmailErr = "Email is required."
		ui.Render(w, r, forms.InviteMember(formProps))
		return
	}
	if err := validation.ValidateEmail(emailInput); err != nil {
		formProps.EmailErr = "Enter a valid email address."
		ui.Render(w, r, forms.InviteMember(formProps))
		return
	}

	if _, err := h.inviteService.CreateInvite(spaceID, user.ID, emailInput); err != nil {
		switch {
		case errors.Is(err, service.ErrInviteSelf):
			formProps.EmailErr = "You can't invite yourself."
		case errors.Is(err, service.ErrInviteAlreadyMember):
			formProps.EmailErr = "This person is already a member."
		case errors.Is(err, service.ErrInviteAlreadyPending):
			formProps.EmailErr = "An invitation is already pending for this email."
		default:
			slog.Error("failed to create invite", "error", err, "space_id", spaceID)
			formProps.GeneralErr = "Something went wrong. Please try again."
		}
		ui.Render(w, r, forms.InviteMember(formProps))
		return
	}

	formProps.Email = ""
	formProps.SuccessMsg = "Invitation sent to " + emailInput + "."
	w.Header().Set("HX-Trigger", "members:refresh")
	ui.Render(w, r, forms.InviteMember(formProps))
}

func (h *spaceHandler) HandleRemoveMember(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	userID := r.PathValue("userID")

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
	if userID == space.OwnerID {
		ui.RenderError(w, r, "Cannot remove the owner", http.StatusBadRequest)
		return
	}

	if err := h.spaceService.RemoveMember(spaceID, userID, user.ID); err != nil {
		slog.Error("failed to remove member", "error", err, "space_id", spaceID, "user_id", userID)
		ui.RenderError(w, r, "Failed to remove member", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

func (h *spaceHandler) HandleCancelInvite(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	token := r.PathValue("token")

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

	if err := h.inviteService.CancelInvite(token, user.ID); err != nil {
		slog.Error("failed to cancel invite", "error", err, "token", token)
		ui.RenderError(w, r, "Failed to cancel invitation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

func (h *spaceHandler) SpaceActivityPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to load space", "error", err, "space_id", spaceID)
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

	total, err := h.accountActivitySvc.CountSpace(spaceID)
	if err != nil {
		slog.Error("failed to count space activity", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load activity", http.StatusInternalServerError)
		return
	}

	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	rows, err := h.accountActivitySvc.ListSpace(spaceID, perPage, (page-1)*perPage)
	if err != nil {
		slog.Error("failed to list space activity", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load activity", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceActivityPage(pages.SpaceActivityPageProps{
		SpaceID:     space.ID,
		SpaceName:   space.Name,
		Rows:        rows,
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalCount:  total,
		PerPage:     perPage,
	}))
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

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to load space", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load account settings", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceAccountSettingsPage(pages.SpaceAccountSettingsPageProps{
		SpaceID:         spaceID,
		SpaceName:       space.Name,
		AccountID:       accountID,
		AccountName:     account.Name,
		AccountCurrency: account.Currency,
		UpdateForm: forms.UpdateAccountProps{
			SpaceID:   spaceID,
			AccountID: accountID,
			Name:      account.Name,
		},
		CurrencyForm: forms.ChangeAccountCurrencyProps{
			SpaceID:         spaceID,
			AccountID:       accountID,
			CurrentCurrency: account.Currency,
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

	user := ctxkeys.User(r.Context())
	actorID := ""
	if user != nil {
		actorID = user.ID
	}
	if err := h.accountService.RenameAccount(accountID, nameInput, actorID); err != nil {
		slog.Error("failed to rename account", "error", err, "account_id", accountID)
		formProps.GeneralErr = "Something went wrong. Please try again."
		ui.Render(w, r, forms.UpdateAccount(formProps))
		return
	}

	formProps.SuccessMsg = "Account name updated."
	ui.Render(w, r, forms.UpdateAccount(formProps))
}

func (h *spaceHandler) HandleChangeAccountCurrency(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	account, err := h.accountService.GetAccount(accountID)
	if err != nil || account.SpaceID != spaceID {
		ui.RenderError(w, r, "Account not found", http.StatusNotFound)
		return
	}

	newCurrencyInput := currency.Normalize(r.FormValue("new_currency"))
	rateInput := strings.TrimSpace(r.FormValue("rate"))

	formProps := forms.ChangeAccountCurrencyProps{
		SpaceID:         spaceID,
		AccountID:       accountID,
		CurrentCurrency: account.Currency,
		NewCurrency:     newCurrencyInput,
		ConversionRate:  rateInput,
	}

	hasErr := false
	if newCurrencyInput == "" {
		formProps.NewCurrencyErr = "Choose a currency."
		hasErr = true
	} else if !currency.IsValid(newCurrencyInput) {
		formProps.NewCurrencyErr = "Choose a supported currency."
		hasErr = true
	} else if newCurrencyInput == account.Currency {
		formProps.NewCurrencyErr = "Choose a different currency."
		hasErr = true
	}

	var rate decimal.Decimal
	if rateInput == "" {
		formProps.RateErr = "Conversion rate is required."
		hasErr = true
	} else {
		r, err := decimal.NewFromString(rateInput)
		if err != nil {
			formProps.RateErr = "Enter a valid rate (e.g. 1.2345)."
			hasErr = true
		} else if !r.IsPositive() {
			formProps.RateErr = "Rate must be greater than zero."
			hasErr = true
		} else {
			rate = r
		}
	}

	if hasErr {
		ui.Render(w, r, forms.ChangeAccountCurrency(formProps))
		return
	}

	user := ctxkeys.User(r.Context())
	actorID := ""
	if user != nil {
		actorID = user.ID
	}
	if err := h.accountService.ChangeCurrency(accountID, newCurrencyInput, rate, actorID); err != nil {
		slog.Error("failed to change account currency", "error", err, "account_id", accountID)
		formProps.GeneralErr = "Something went wrong. Please try again."
		ui.Render(w, r, forms.ChangeAccountCurrency(formProps))
		return
	}

	formProps.CurrentCurrency = newCurrencyInput
	formProps.NewCurrency = ""
	formProps.ConversionRate = ""
	formProps.SuccessMsg = "Currency updated. Balance and allocations were converted."
	ui.Render(w, r, forms.ChangeAccountCurrency(formProps))
}

func (h *spaceHandler) HandleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	account, err := h.accountService.GetAccount(accountID)
	if err != nil || account.SpaceID != spaceID {
		ui.RenderError(w, r, "Account not found", http.StatusNotFound)
		return
	}

	user := ctxkeys.User(r.Context())
	actorID := ""
	if user != nil {
		actorID = user.ID
	}
	if err := h.accountService.DeleteAccount(accountID, actorID); err != nil {
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

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to load space", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load page", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceCreateBillPage(pages.SpaceCreateBillPageProps{
		SpaceID:     spaceID,
		SpaceName:   space.Name,
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

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to load space", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load page", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceCreateDepositPage(pages.SpaceCreateDepositPageProps{
		SpaceID:     spaceID,
		SpaceName:   space.Name,
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

	actorID := ""
	if u := ctxkeys.User(r.Context()); u != nil {
		actorID = u.ID
	}
	_, err := h.transactionService.Deposit(service.DepositInput{
		AccountID:   accountID,
		Title:       titleInput,
		Amount:      amount,
		OccurredAt:  occurredAt,
		Description: descriptionInput,
		ActorID:     actorID,
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

func (h *spaceHandler) SpaceTransactionPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")
	transactionID := r.PathValue("transactionID")

	account, err := h.accountService.GetAccount(accountID)
	if err != nil || account.SpaceID != spaceID {
		ui.Render(w, r, pages.NotFound())
		return
	}

	txn, err := h.transactionService.GetTransaction(transactionID)
	if err != nil || txn.AccountID != accountID {
		ui.Render(w, r, pages.NotFound())
		return
	}

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to load space", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load page", http.StatusInternalServerError)
		return
	}

	categoryName := ""
	if txn.Type == model.TransactionTypeWithdrawal {
		categoryID, err := h.transactionService.GetTransactionCategoryID(transactionID)
		if err != nil {
			slog.Error("failed to load transaction category", "error", err, "transaction_id", transactionID)
		} else if categoryID != "" {
			categories, err := h.transactionService.ListCategories()
			if err != nil {
				slog.Error("failed to load categories", "error", err)
			} else {
				for _, c := range categories {
					if c.ID == categoryID {
						categoryName = c.Name
						break
					}
				}
			}
		}
	}

	relatedID, err := h.transactionService.GetRelatedTransactionID(transactionID)
	if err != nil {
		slog.Error("failed to load related transaction", "error", err, "transaction_id", transactionID)
		relatedID = ""
	}
	var relatedTxn *model.Transaction
	var relatedAccount *model.Account
	if relatedID != "" {
		relatedTxn, err = h.transactionService.GetTransaction(relatedID)
		if err != nil {
			slog.Error("failed to load related transaction details", "error", err, "related_id", relatedID)
		} else {
			relatedAccount, err = h.accountService.GetAccount(relatedTxn.AccountID)
			if err != nil {
				slog.Error("failed to load related transaction account", "error", err, "account_id", relatedTxn.AccountID)
				relatedAccount = nil
			}
		}
	}

	recentLogs, err := h.txAuditLogService.List(transactionID, 5, 0)
	if err != nil {
		slog.Error("failed to load transaction audit logs", "error", err, "transaction_id", transactionID)
		recentLogs = nil
	}
	logCount, err := h.txAuditLogService.Count(transactionID)
	if err != nil {
		slog.Error("failed to count transaction audit logs", "error", err, "transaction_id", transactionID)
		logCount = len(recentLogs)
	}

	ui.Render(w, r, pages.SpaceTransactionPage(pages.SpaceTransactionPageProps{
		SpaceID:            spaceID,
		SpaceName:          space.Name,
		AccountID:          accountID,
		AccountName:        account.Name,
		Transaction:        txn,
		CategoryName:       categoryName,
		RecentAuditLogs:    recentLogs,
		AuditLogCount:      logCount,
		RelatedTransaction: relatedTxn,
		RelatedAccount:     relatedAccount,
	}))
}

func (h *spaceHandler) SpaceAccountActivityPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	account, err := h.accountService.GetAccount(accountID)
	if err != nil || account.SpaceID != spaceID {
		ui.Render(w, r, pages.NotFound())
		return
	}

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to load space", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load activity", http.StatusInternalServerError)
		return
	}

	const perPage = 25
	page := 1
	if p := strings.TrimSpace(r.URL.Query().Get("page")); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	total, err := h.accountActivitySvc.Count(accountID)
	if err != nil {
		slog.Error("failed to count account activity", "error", err, "account_id", accountID)
		ui.RenderError(w, r, "Failed to load activity", http.StatusInternalServerError)
		return
	}

	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	rows, err := h.accountActivitySvc.List(accountID, perPage, (page-1)*perPage)
	if err != nil {
		slog.Error("failed to list account activity", "error", err, "account_id", accountID)
		ui.RenderError(w, r, "Failed to load activity", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceAccountActivityPage(pages.SpaceAccountActivityPageProps{
		SpaceID:     spaceID,
		SpaceName:   space.Name,
		AccountID:   accountID,
		AccountName: account.Name,
		Rows:        rows,
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalCount:  total,
		PerPage:     perPage,
	}))
}

func (h *spaceHandler) SpaceTransactionActivityPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")
	transactionID := r.PathValue("transactionID")

	account, err := h.accountService.GetAccount(accountID)
	if err != nil || account.SpaceID != spaceID {
		ui.Render(w, r, pages.NotFound())
		return
	}

	txn, err := h.transactionService.GetTransaction(transactionID)
	if err != nil || txn.AccountID != accountID {
		ui.Render(w, r, pages.NotFound())
		return
	}

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to load space", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load activity", http.StatusInternalServerError)
		return
	}

	const perPage = 25
	page := 1
	if p := strings.TrimSpace(r.URL.Query().Get("page")); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	total, err := h.txAuditLogService.Count(transactionID)
	if err != nil {
		slog.Error("failed to count transaction audit logs", "error", err, "transaction_id", transactionID)
		ui.RenderError(w, r, "Failed to load activity", http.StatusInternalServerError)
		return
	}

	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	logs, err := h.txAuditLogService.List(transactionID, perPage, (page-1)*perPage)
	if err != nil {
		slog.Error("failed to list transaction audit logs", "error", err, "transaction_id", transactionID)
		ui.RenderError(w, r, "Failed to load activity", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceTransactionActivityPage(pages.SpaceTransactionActivityPageProps{
		SpaceID:         spaceID,
		SpaceName:       space.Name,
		AccountID:       accountID,
		AccountName:     account.Name,
		TransactionID:   transactionID,
		TransactionName: txn.Title,
		Logs:            logs,
		CurrentPage:     page,
		TotalPages:      totalPages,
		TotalCount:      total,
		PerPage:         perPage,
	}))
}

func (h *spaceHandler) SpaceEditTransactionPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")
	transactionID := r.PathValue("transactionID")

	account, err := h.accountService.GetAccount(accountID)
	if err != nil || account.SpaceID != spaceID {
		ui.Render(w, r, pages.NotFound())
		return
	}

	txn, err := h.transactionService.GetTransaction(transactionID)
	if err != nil || txn.AccountID != accountID {
		ui.Render(w, r, pages.NotFound())
		return
	}

	// Transfers must be edited as a pair; refuse the edit page entirely.
	relatedID, err := h.transactionService.GetRelatedTransactionID(transactionID)
	if err != nil {
		slog.Error("failed to check transfer linkage", "error", err, "transaction_id", transactionID)
	}
	if relatedID != "" {
		redirectTo := routeurl.URL(
			"page.app.spaces.space.accounts.account.transactions.transaction",
			"spaceID", spaceID,
			"accountID", accountID,
			"transactionID", transactionID,
		)
		http.Redirect(w, r, redirectTo, http.StatusSeeOther)
		return
	}

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to load space", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load page", http.StatusInternalServerError)
		return
	}

	description := ""
	if txn.Description != nil {
		description = *txn.Description
	}

	pageProps := pages.SpaceEditTransactionPageProps{
		SpaceID:         spaceID,
		SpaceName:       space.Name,
		AccountID:       accountID,
		AccountName:     account.Name,
		TransactionType: txn.Type,
	}

	if txn.Type == model.TransactionTypeDeposit {
		pageProps.DepositForm = forms.EditDepositProps{
			SpaceID:       spaceID,
			AccountID:     accountID,
			TransactionID: transactionID,
			Title:         txn.Title,
			Amount:        txn.Value.StringFixedBank(2),
			Date:          txn.OccurredAt.Format("2006-01-02"),
			Description:   description,
		}
	} else {
		categories, err := h.transactionService.ListCategories()
		if err != nil {
			slog.Error("failed to load categories", "error", err)
			ui.RenderError(w, r, "Failed to load categories", http.StatusInternalServerError)
			return
		}
		categoryID, err := h.transactionService.GetTransactionCategoryID(transactionID)
		if err != nil {
			slog.Error("failed to load transaction category", "error", err, "transaction_id", transactionID)
			categoryID = ""
		}
		pageProps.BillForm = forms.EditBillProps{
			SpaceID:       spaceID,
			AccountID:     accountID,
			TransactionID: transactionID,
			Categories:    categories,
			Title:         txn.Title,
			Amount:        txn.Value.StringFixedBank(2),
			Date:          txn.OccurredAt.Format("2006-01-02"),
			Description:   description,
			CategoryID:    categoryID,
		}
	}

	ui.Render(w, r, pages.SpaceEditTransactionPage(pageProps))
}

func (h *spaceHandler) HandleEditTransaction(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")
	transactionID := r.PathValue("transactionID")

	account, err := h.accountService.GetAccount(accountID)
	if err != nil || account.SpaceID != spaceID {
		ui.RenderError(w, r, "Account not found", http.StatusNotFound)
		return
	}

	txn, err := h.transactionService.GetTransaction(transactionID)
	if err != nil || txn.AccountID != accountID {
		ui.RenderError(w, r, "Transaction not found", http.StatusNotFound)
		return
	}

	// Defense in depth — the edit page redirects away for transfers, but a
	// hand-crafted POST shouldn't be able to bypass it.
	if relatedID, err := h.transactionService.GetRelatedTransactionID(transactionID); err != nil {
		slog.Error("failed to check transfer linkage", "error", err, "transaction_id", transactionID)
	} else if relatedID != "" {
		ui.RenderError(w, r, "Transfer transactions cannot be edited.", http.StatusBadRequest)
		return
	}

	titleInput := strings.TrimSpace(r.FormValue("title"))
	amountInput := strings.TrimSpace(r.FormValue("amount"))
	dateInput := strings.TrimSpace(r.FormValue("date"))
	descriptionInput := strings.TrimSpace(r.FormValue("description"))

	titleErr, amountErr, dateErr := "", "", ""
	hasErr := false

	if titleInput == "" {
		titleErr = "Title is required."
		hasErr = true
	}

	var amount decimal.Decimal
	if amountInput == "" {
		amountErr = "Amount is required."
		hasErr = true
	} else {
		amt, err := decimal.NewFromString(amountInput)
		if err != nil {
			amountErr = "Enter a valid amount (e.g. 12.34)."
			hasErr = true
		} else if !amt.IsPositive() {
			amountErr = "Amount must be greater than zero."
			hasErr = true
		} else if amt.Exponent() < -2 {
			amountErr = "Amount can have at most 2 decimal places."
			hasErr = true
		} else {
			amount = amt
		}
	}

	var occurredAt time.Time
	if dateInput == "" {
		dateErr = "Date is required."
		hasErr = true
	} else {
		parsed, err := time.Parse("2006-01-02", dateInput)
		if err != nil {
			dateErr = "Enter a valid date."
			hasErr = true
		} else {
			occurredAt = parsed
		}
	}

	if txn.Type == model.TransactionTypeDeposit {
		formProps := forms.EditDepositProps{
			SpaceID:       spaceID,
			AccountID:     accountID,
			TransactionID: transactionID,
			Title:         titleInput,
			Amount:        amountInput,
			Date:          dateInput,
			Description:   descriptionInput,
			TitleErr:      titleErr,
			AmountErr:     amountErr,
			DateErr:       dateErr,
		}
		if hasErr {
			ui.Render(w, r, forms.EditDeposit(formProps))
			return
		}
		actorID := ""
		if u := ctxkeys.User(r.Context()); u != nil {
			actorID = u.ID
		}
		if _, err := h.transactionService.UpdateDeposit(service.UpdateDepositInput{
			TransactionID: transactionID,
			Title:         titleInput,
			Amount:        amount,
			OccurredAt:    occurredAt,
			Description:   descriptionInput,
			ActorID:       actorID,
		}); err != nil {
			slog.Error("failed to update deposit", "error", err, "transaction_id", transactionID)
			formProps.GeneralErr = "Something went wrong. Please try again."
			ui.Render(w, r, forms.EditDeposit(formProps))
			return
		}
		redirectTo := routeurl.URL(
			"page.app.spaces.space.accounts.account.transactions.transaction",
			"spaceID", spaceID,
			"accountID", accountID,
			"transactionID", transactionID,
		)
		w.Header().Set("HX-Redirect", redirectTo)
		w.WriteHeader(http.StatusOK)
		return
	}

	categoryInput := strings.TrimSpace(r.FormValue("category"))
	categories, err := h.transactionService.ListCategories()
	if err != nil {
		slog.Error("failed to load categories", "error", err)
		ui.RenderError(w, r, "Failed to load categories", http.StatusInternalServerError)
		return
	}
	formProps := forms.EditBillProps{
		SpaceID:       spaceID,
		AccountID:     accountID,
		TransactionID: transactionID,
		Categories:    categories,
		Title:         titleInput,
		Amount:        amountInput,
		Date:          dateInput,
		Description:   descriptionInput,
		CategoryID:    categoryInput,
		TitleErr:      titleErr,
		AmountErr:     amountErr,
		DateErr:       dateErr,
	}
	if hasErr {
		ui.Render(w, r, forms.EditBill(formProps))
		return
	}
	actorID := ""
	if u := ctxkeys.User(r.Context()); u != nil {
		actorID = u.ID
	}
	if _, err := h.transactionService.UpdateBill(service.UpdateBillInput{
		TransactionID: transactionID,
		Title:         titleInput,
		Amount:        amount,
		OccurredAt:    occurredAt,
		Description:   descriptionInput,
		CategoryID:    categoryInput,
		ActorID:       actorID,
	}); err != nil {
		slog.Error("failed to update bill", "error", err, "transaction_id", transactionID)
		formProps.GeneralErr = "Something went wrong. Please try again."
		ui.Render(w, r, forms.EditBill(formProps))
		return
	}
	redirectTo := routeurl.URL(
		"page.app.spaces.space.accounts.account.transactions.transaction",
		"spaceID", spaceID,
		"accountID", accountID,
		"transactionID", transactionID,
	)
	w.Header().Set("HX-Redirect", redirectTo)
	w.WriteHeader(http.StatusOK)
}

func (h *spaceHandler) HandleDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")
	transactionID := r.PathValue("transactionID")

	account, err := h.accountService.GetAccount(accountID)
	if err != nil || account.SpaceID != spaceID {
		ui.RenderError(w, r, "Account not found", http.StatusNotFound)
		return
	}

	txn, err := h.transactionService.GetTransaction(transactionID)
	if err != nil || txn.AccountID != accountID {
		ui.RenderError(w, r, "Transaction not found", http.StatusNotFound)
		return
	}

	actorID := ""
	if u := ctxkeys.User(r.Context()); u != nil {
		actorID = u.ID
	}

	if _, err := h.transactionService.DeleteTransaction(service.DeleteTransactionInput{
		TransactionID: transactionID,
		ActorID:       actorID,
	}); err != nil {
		if errors.Is(err, service.ErrTransactionPartOfTransfer) {
			ui.RenderError(w, r, "Transfer transactions cannot be deleted.", http.StatusBadRequest)
			return
		}
		slog.Error("failed to delete transaction", "error", err, "transaction_id", transactionID)
		ui.RenderError(w, r, "Failed to delete transaction", http.StatusInternalServerError)
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

func (h *spaceHandler) SpaceCreateTransferPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	account, err := h.accountService.GetAccount(accountID)
	if err != nil || account.SpaceID != spaceID {
		ui.Render(w, r, pages.NotFound())
		return
	}

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to load space", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load page", http.StatusInternalServerError)
		return
	}

	dests, err := h.transferDestinations(spaceID, accountID)
	if err != nil {
		slog.Error("failed to load destination accounts", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load page", http.StatusInternalServerError)
		return
	}

	allocSummary, err := h.allocationService.SummaryForAccount(accountID)
	if err != nil {
		slog.Error("failed to load allocation summary", "error", err, "account_id", accountID)
		ui.RenderError(w, r, "Failed to load page", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceCreateTransferPage(pages.SpaceCreateTransferPageProps{
		SpaceID:     spaceID,
		SpaceName:   space.Name,
		AccountID:   accountID,
		AccountName: account.Name,
		Form: forms.CreateTransferProps{
			SpaceID:         spaceID,
			SourceAccountID: accountID,
			SourceCurrency:  account.Currency,
			DestAccounts:    dests,
			SourceAvailable: allocSummary.Available.StringFixedBank(2),
			SourceAllocated: allocSummary.Allocated.StringFixedBank(2),
			SourceOverflow:  allocSummary.Overflow,
			Date:            time.Now().Format("2006-01-02"),
		},
	}))
}

func (h *spaceHandler) HandleCreateTransfer(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	source, err := h.accountService.GetAccount(accountID)
	if err != nil || source.SpaceID != spaceID {
		ui.RenderError(w, r, "Account not found", http.StatusNotFound)
		return
	}

	dests, err := h.transferDestinations(spaceID, accountID)
	if err != nil {
		slog.Error("failed to load destination accounts", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load form", http.StatusInternalServerError)
		return
	}

	titleInput := strings.TrimSpace(r.FormValue("title"))
	amountInput := strings.TrimSpace(r.FormValue("amount"))
	destInput := strings.TrimSpace(r.FormValue("destination"))
	rateInput := strings.TrimSpace(r.FormValue("rate"))
	dateInput := strings.TrimSpace(r.FormValue("date"))
	descriptionInput := strings.TrimSpace(r.FormValue("description"))

	formProps := forms.CreateTransferProps{
		SpaceID:         spaceID,
		SourceAccountID: accountID,
		SourceCurrency:  source.Currency,
		DestAccounts:    dests,
		Title:           titleInput,
		Amount:          amountInput,
		DestAccountID:   destInput,
		ConversionRate:  rateInput,
		Date:            dateInput,
		Description:     descriptionInput,
	}

	if allocSummary, err := h.allocationService.SummaryForAccount(accountID); err != nil {
		slog.Error("failed to load allocation summary", "error", err, "account_id", accountID)
	} else {
		formProps.SourceAvailable = allocSummary.Available.StringFixedBank(2)
		formProps.SourceAllocated = allocSummary.Allocated.StringFixedBank(2)
		formProps.SourceOverflow = allocSummary.Overflow
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

	var destCurrency string
	if destInput == "" {
		formProps.DestErr = "Choose a destination account."
		hasErr = true
	} else if destInput == accountID {
		formProps.DestErr = "Destination must be a different account."
		hasErr = true
	} else {
		// Verify the destination is in the same space (defends against hand-crafted requests).
		destAcct, err := h.accountService.GetAccount(destInput)
		if err != nil || destAcct.SpaceID != spaceID {
			formProps.DestErr = "Destination account not found."
			hasErr = true
		} else {
			destCurrency = destAcct.Currency
		}
	}

	var rate decimal.Decimal
	if destCurrency != "" && destCurrency != source.Currency {
		if rateInput == "" {
			formProps.RateErr = "Conversion rate is required for cross-currency transfers."
			hasErr = true
		} else {
			r, err := decimal.NewFromString(rateInput)
			if err != nil {
				formProps.RateErr = "Enter a valid rate (e.g. 1.2345)."
				hasErr = true
			} else if !r.IsPositive() {
				formProps.RateErr = "Rate must be greater than zero."
				hasErr = true
			} else {
				rate = r
			}
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
		ui.Render(w, r, forms.CreateTransfer(formProps))
		return
	}

	actorID := ""
	if u := ctxkeys.User(r.Context()); u != nil {
		actorID = u.ID
	}

	if _, err := h.transactionService.Transfer(service.TransferInput{
		SourceAccountID: accountID,
		DestAccountID:   destInput,
		Title:           titleInput,
		Amount:          amount,
		ConversionRate:  rate,
		OccurredAt:      occurredAt,
		Description:     descriptionInput,
		ActorID:         actorID,
	}); err != nil {
		slog.Error("failed to create transfer", "error", err, "source", accountID, "dest", destInput)
		formProps.GeneralErr = "Something went wrong. Please try again."
		ui.Render(w, r, forms.CreateTransfer(formProps))
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

// transferDestinations returns every account in the space except the source.
func (h *spaceHandler) transferDestinations(spaceID, sourceAccountID string) ([]*model.Account, error) {
	all, err := h.accountService.GetAccountsForSpace(spaceID)
	if err != nil {
		return nil, err
	}
	out := make([]*model.Account, 0, len(all))
	for _, a := range all {
		if a.ID == sourceAccountID {
			continue
		}
		out = append(out, a)
	}
	return out, nil
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

	actorID := ""
	if u := ctxkeys.User(r.Context()); u != nil {
		actorID = u.ID
	}
	_, err = h.transactionService.PayBill(service.PayBillInput{
		AccountID:   accountID,
		Title:       titleInput,
		Amount:      amount,
		OccurredAt:  occurredAt,
		Description: descriptionInput,
		CategoryID:  categoryInput,
		ActorID:     actorID,
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
