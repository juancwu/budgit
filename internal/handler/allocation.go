package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/blocks"
	"github.com/shopspring/decimal"
)

type allocationHandler struct {
	allocationService *service.AllocationService
	accountService    *service.AccountService
}

func NewAllocationHandler(allocation *service.AllocationService, account *service.AccountService) *allocationHandler {
	return &allocationHandler{allocationService: allocation, accountService: account}
}

// ensureAccess validates that the account exists and lives in the requested
// space. Returns false (and writes a 404) when access should be denied.
func (h *allocationHandler) ensureAccess(w http.ResponseWriter, r *http.Request, spaceID, accountID string) bool {
	account, err := h.accountService.GetAccount(accountID)
	if err != nil || account.SpaceID != spaceID {
		ui.RenderError(w, r, "Account not found", http.StatusNotFound)
		return false
	}
	return true
}

func (h *allocationHandler) renderSection(w http.ResponseWriter, r *http.Request, spaceID, accountID string) {
	summary, err := h.allocationService.SummaryForAccount(accountID)
	if err != nil {
		slog.Error("failed to load allocation summary", "error", err, "account_id", accountID)
		ui.RenderError(w, r, "Failed to load savings goals", http.StatusInternalServerError)
		return
	}
	ui.Render(w, r, blocks.AllocationsSection(blocks.AllocationsSectionProps{
		SpaceID: spaceID, AccountID: accountID, Summary: summary,
	}))
}

func parseAllocationForm(r *http.Request) (name string, amount decimal.Decimal, target *decimal.Decimal, state blocks.AllocationFormState) {
	name = strings.TrimSpace(r.FormValue("name"))
	amountInput := strings.TrimSpace(r.FormValue("amount"))
	targetInput := strings.TrimSpace(r.FormValue("target_amount"))

	state = blocks.AllocationFormState{
		Name: name, Amount: amountInput, TargetAmount: targetInput,
	}

	if name == "" {
		state.NameErr = "Name is required."
	}
	if amountInput == "" {
		state.AmountErr = "Amount is required."
	} else {
		parsed, err := decimal.NewFromString(amountInput)
		if err != nil {
			state.AmountErr = "Enter a valid number."
		} else if parsed.IsNegative() {
			state.AmountErr = "Amount cannot be negative."
		} else {
			amount = parsed
		}
	}
	if targetInput != "" {
		parsed, err := decimal.NewFromString(targetInput)
		if err != nil {
			state.TargetErr = "Enter a valid number."
		} else if parsed.IsNegative() {
			state.TargetErr = "Goal cannot be negative."
		} else {
			target = &parsed
		}
	}
	return
}

func (h *allocationHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")
	if !h.ensureAccess(w, r, spaceID, accountID) {
		return
	}

	name, amount, target, state := parseAllocationForm(r)
	if state.NameErr != "" || state.AmountErr != "" || state.TargetErr != "" {
		// Re-render the section with the create form expanded and errors shown.
		h.renderSectionWithCreateError(w, r, spaceID, accountID, state)
		return
	}

	user := ctxkeys.User(r.Context())
	actorID := ""
	if user != nil {
		actorID = user.ID
	}
	if _, err := h.allocationService.Create(service.CreateAllocationInput{
		AccountID: accountID, Name: name, Amount: amount, TargetAmount: target, ActorID: actorID,
	}); err != nil {
		slog.Error("failed to create allocation", "error", err, "account_id", accountID)
		state.GeneralErr = friendlyAllocationError(err)
		h.renderSectionWithCreateError(w, r, spaceID, accountID, state)
		return
	}

	h.renderSection(w, r, spaceID, accountID)
}

func (h *allocationHandler) HandleEdit(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")
	allocationID := r.PathValue("allocationID")
	if !h.ensureAccess(w, r, spaceID, accountID) {
		return
	}

	existing, err := h.allocationService.Get(allocationID)
	if err != nil || existing.AccountID != accountID {
		ui.RenderError(w, r, "Savings goal not found", http.StatusNotFound)
		return
	}

	name, amount, target, state := parseAllocationForm(r)
	if state.NameErr != "" || state.AmountErr != "" || state.TargetErr != "" {
		h.renderSection(w, r, spaceID, accountID) // simplest: re-render fresh; inline edit errors require richer state
		return
	}

	user := ctxkeys.User(r.Context())
	actorID := ""
	if user != nil {
		actorID = user.ID
	}
	if _, err := h.allocationService.Update(service.UpdateAllocationInput{
		AllocationID: allocationID, Name: name, Amount: amount, TargetAmount: target, ActorID: actorID,
	}); err != nil {
		slog.Error("failed to update allocation", "error", err, "allocation_id", allocationID)
		ui.RenderError(w, r, friendlyAllocationError(err), http.StatusBadRequest)
		return
	}

	h.renderSection(w, r, spaceID, accountID)
}

func (h *allocationHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")
	allocationID := r.PathValue("allocationID")
	if !h.ensureAccess(w, r, spaceID, accountID) {
		return
	}

	existing, err := h.allocationService.Get(allocationID)
	if err != nil || existing.AccountID != accountID {
		ui.RenderError(w, r, "Savings goal not found", http.StatusNotFound)
		return
	}

	user := ctxkeys.User(r.Context())
	actorID := ""
	if user != nil {
		actorID = user.ID
	}
	if err := h.allocationService.Delete(allocationID, actorID); err != nil {
		slog.Error("failed to delete allocation", "error", err, "allocation_id", allocationID)
		ui.RenderError(w, r, "Failed to delete savings goal", http.StatusInternalServerError)
		return
	}

	h.renderSection(w, r, spaceID, accountID)
}

// renderSectionWithCreateError re-renders the section but injects the partially-
// filled form state so the user doesn't lose their input when validation fails.
// Implemented by rendering the section then... actually we need a richer block
// for this, so for now just render a fresh section — Phase 5 polishes UX.
func (h *allocationHandler) renderSectionWithCreateError(w http.ResponseWriter, r *http.Request, spaceID, accountID string, state blocks.AllocationFormState) {
	summary, err := h.allocationService.SummaryForAccount(accountID)
	if err != nil {
		slog.Error("failed to load allocation summary", "error", err, "account_id", accountID)
		ui.RenderError(w, r, "Failed to load savings goals", http.StatusInternalServerError)
		return
	}
	ui.Render(w, r, blocks.AllocationsSection(blocks.AllocationsSectionProps{
		SpaceID: spaceID, AccountID: accountID, Summary: summary,
		CreateForm:        &state,
		ShowCreateForm:    true,
	}))
}

func friendlyAllocationError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, repository.ErrAllocationNotFound) {
		return "Savings goal not found."
	}
	msg := err.Error()
	if strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique") {
		return "A savings goal with this name already exists for this account."
	}
	return "Something went wrong. Please try again."
}
