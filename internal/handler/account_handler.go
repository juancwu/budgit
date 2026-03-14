package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/shopspring/decimal"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/moneyaccount"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/toast"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type AccountHandler struct {
	spaceService   *service.SpaceService
	accountService *service.MoneyAccountService
	expenseService *service.ExpenseService
}

func NewAccountHandler(ss *service.SpaceService, mas *service.MoneyAccountService, es *service.ExpenseService) *AccountHandler {
	return &AccountHandler{
		spaceService:   ss,
		accountService: mas,
		expenseService: es,
	}
}

func (h *AccountHandler) getAccountForSpace(w http.ResponseWriter, spaceID, accountID string) *model.MoneyAccount {
	account, err := h.accountService.GetAccount(accountID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return nil
	}
	if account.SpaceID != spaceID {
		http.Error(w, "Not Found", http.StatusNotFound)
		return nil
	}
	return account
}

func (h *AccountHandler) AccountsPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	accounts, err := h.accountService.GetAccountsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get accounts for space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	totalBalance, err := h.expenseService.GetBalanceForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get balance for space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	totalAllocated, err := h.accountService.GetTotalAllocatedForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get total allocated", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	availableBalance := totalBalance.Sub(totalAllocated)

	transfers, totalPages, err := h.accountService.GetTransfersForSpacePaginated(spaceID, 1)
	if err != nil {
		slog.Error("failed to get transfers", "error", err, "space_id", spaceID)
		transfers = nil
		totalPages = 1
	}

	ui.Render(w, r, pages.SpaceAccountsPage(space, accounts, totalBalance, availableBalance, transfers, 1, totalPages))
}

func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	user := ctxkeys.User(r.Context())

	if err := r.ParseForm(); err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		ui.RenderError(w, r, "Account name is required", http.StatusUnprocessableEntity)
		return
	}

	account, err := h.accountService.CreateAccount(service.CreateMoneyAccountDTO{
		SpaceID:   spaceID,
		Name:      name,
		CreatedBy: user.ID,
	})
	if err != nil {
		slog.Error("failed to create account", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	acctWithBalance := model.MoneyAccountWithBalance{
		MoneyAccount: *account,
		Balance:      decimal.Zero,
	}

	ui.Render(w, r, moneyaccount.AccountCard(spaceID, &acctWithBalance))
}

func (h *AccountHandler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	if h.getAccountForSpace(w, spaceID, accountID) == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		ui.RenderError(w, r, "Account name is required", http.StatusUnprocessableEntity)
		return
	}

	updatedAccount, err := h.accountService.UpdateAccount(service.UpdateMoneyAccountDTO{
		ID:   accountID,
		Name: name,
	})
	if err != nil {
		slog.Error("failed to update account", "error", err, "account_id", accountID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	balance, err := h.accountService.GetAccountBalance(accountID)
	if err != nil {
		slog.Error("failed to get account balance", "error", err, "account_id", accountID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	acctWithBalance := model.MoneyAccountWithBalance{
		MoneyAccount: *updatedAccount,
		Balance:      balance,
	}

	ui.Render(w, r, moneyaccount.AccountCard(spaceID, &acctWithBalance))
}

func (h *AccountHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	if h.getAccountForSpace(w, spaceID, accountID) == nil {
		return
	}

	err := h.accountService.DeleteAccount(accountID)
	if err != nil {
		slog.Error("failed to delete account", "error", err, "account_id", accountID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Return updated balance summary via OOB swap
	totalBalance, err := h.expenseService.GetBalanceForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get balance", "error", err, "space_id", spaceID)
	}
	totalAllocated, err := h.accountService.GetTotalAllocatedForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get total allocated", "error", err, "space_id", spaceID)
	}

	ui.Render(w, r, moneyaccount.BalanceSummaryCard(spaceID, totalBalance, totalBalance.Sub(totalAllocated), true))
	ui.RenderToast(w, r, toast.Toast(toast.Props{
		Title:       "Account deleted",
		Variant:     toast.VariantSuccess,
		Icon:        true,
		Dismissible: true,
		Duration:    5000,
	}))
}

func (h *AccountHandler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")
	user := ctxkeys.User(r.Context())

	if h.getAccountForSpace(w, spaceID, accountID) == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	amountStr := r.FormValue("amount")
	direction := model.TransferDirection(r.FormValue("direction"))
	note := r.FormValue("note")

	amountDecimal, err := decimal.NewFromString(amountStr)
	if err != nil || amountDecimal.LessThanOrEqual(decimal.Zero) {
		ui.RenderError(w, r, "Invalid amount", http.StatusUnprocessableEntity)
		return
	}
	amount := amountDecimal

	// Calculate available space balance for deposit validation
	totalBalance, err := h.expenseService.GetBalanceForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get balance", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	totalAllocated, err := h.accountService.GetTotalAllocatedForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get total allocated", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	availableBalance := totalBalance.Sub(totalAllocated)

	// Validate balance limits before creating transfer
	if direction == model.TransferDirectionDeposit && amount.GreaterThan(availableBalance) {
		ui.RenderError(w, r, fmt.Sprintf("Insufficient available balance. You can deposit up to %s.", model.FormatMoney(availableBalance)), http.StatusUnprocessableEntity)
		return
	}

	if direction == model.TransferDirectionWithdrawal {
		acctBalance, err := h.accountService.GetAccountBalance(accountID)
		if err != nil {
			slog.Error("failed to get account balance", "error", err, "account_id", accountID)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if amount.GreaterThan(acctBalance) {
			ui.RenderError(w, r, fmt.Sprintf("Insufficient account balance. You can withdraw up to %s.", model.FormatMoney(acctBalance)), http.StatusUnprocessableEntity)
			return
		}
	}

	_, err = h.accountService.CreateTransfer(service.CreateTransferDTO{
		AccountID: accountID,
		Amount:    amount,
		Direction: direction,
		Note:      note,
		CreatedBy: user.ID,
	}, availableBalance)
	if err != nil {
		slog.Error("failed to create transfer", "error", err, "account_id", accountID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Return updated account card + OOB balance summary
	accountBalance, err := h.accountService.GetAccountBalance(accountID)
	if err != nil {
		slog.Error("failed to get account balance", "error", err, "account_id", accountID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	account, _ := h.accountService.GetAccount(accountID)
	acctWithBalance := model.MoneyAccountWithBalance{
		MoneyAccount: *account,
		Balance:      accountBalance,
	}

	// Recalculate available balance after transfer
	totalAllocated, _ = h.accountService.GetTotalAllocatedForSpace(spaceID)
	newAvailable := totalBalance.Sub(totalAllocated)

	w.Header().Set("HX-Trigger", "transferSuccess")
	ui.Render(w, r, moneyaccount.AccountCard(spaceID, &acctWithBalance, true))
	ui.Render(w, r, moneyaccount.BalanceSummaryCard(spaceID, totalBalance, newAvailable, true))

	transfers, transferTotalPages, _ := h.accountService.GetTransfersForSpacePaginated(spaceID, 1)
	ui.Render(w, r, moneyaccount.TransferHistoryContent(spaceID, transfers, 1, transferTotalPages, true))
}

func (h *AccountHandler) DeleteTransfer(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	if h.getAccountForSpace(w, spaceID, accountID) == nil {
		return
	}

	transferID := r.PathValue("transferID")
	err := h.accountService.DeleteTransfer(transferID)
	if err != nil {
		slog.Error("failed to delete transfer", "error", err, "transfer_id", transferID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Return updated account card + OOB balance summary
	accountBalance, err := h.accountService.GetAccountBalance(accountID)
	if err != nil {
		slog.Error("failed to get account balance", "error", err, "account_id", accountID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	account, _ := h.accountService.GetAccount(accountID)
	acctWithBalance := model.MoneyAccountWithBalance{
		MoneyAccount: *account,
		Balance:      accountBalance,
	}

	totalBalance, _ := h.expenseService.GetBalanceForSpace(spaceID)
	totalAllocated, _ := h.accountService.GetTotalAllocatedForSpace(spaceID)

	ui.Render(w, r, moneyaccount.AccountCard(spaceID, &acctWithBalance, true))
	ui.Render(w, r, moneyaccount.BalanceSummaryCard(spaceID, totalBalance, totalBalance.Sub(totalAllocated), true))

	transfers, transferTotalPages, _ := h.accountService.GetTransfersForSpacePaginated(spaceID, 1)
	ui.Render(w, r, moneyaccount.TransferHistoryContent(spaceID, transfers, 1, transferTotalPages, true))

	ui.RenderToast(w, r, toast.Toast(toast.Props{
		Title:       "Transfer deleted",
		Variant:     toast.VariantSuccess,
		Icon:        true,
		Dismissible: true,
		Duration:    5000,
	}))
}

func (h *AccountHandler) GetTransferHistory(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	transfers, totalPages, err := h.accountService.GetTransfersForSpacePaginated(spaceID, page)
	if err != nil {
		slog.Error("failed to get transfers", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, moneyaccount.TransferHistoryContent(spaceID, transfers, page, totalPages, false))
}
