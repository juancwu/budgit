package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/shopspring/decimal"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/routeurl"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/blocks"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type budgetPlanHandler struct {
	planService  *service.BudgetPlanService
	spaceService *service.SpaceService
}

func NewBudgetPlanHandler(planService *service.BudgetPlanService, spaceService *service.SpaceService) *budgetPlanHandler {
	return &budgetPlanHandler{planService: planService, spaceService: spaceService}
}

// loadPlan resolves the plan from the URL and verifies it belongs to the space
// in the path. Returns false (and renders 404) when missing or mismatched.
func (h *budgetPlanHandler) loadPlan(w http.ResponseWriter, r *http.Request) (*model.BudgetPlan, bool) {
	spaceID := r.PathValue("spaceID")
	planID := r.PathValue("planID")
	plan, err := h.planService.GetPlan(planID)
	if err != nil || plan.SpaceID != spaceID {
		ui.Render(w, r, pages.NotFound())
		return nil, false
	}
	return plan, true
}

// ListPage lists every budget plan in a space.
func (h *budgetPlanHandler) ListPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		ui.Render(w, r, pages.NotFound())
		return
	}
	plans, err := h.planService.ListPlans(spaceID)
	if err != nil {
		slog.Error("failed to list budget plans", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to load plans", http.StatusInternalServerError)
		return
	}
	ui.Render(w, r, pages.SpaceBudgetPlansPage(pages.SpaceBudgetPlansPageProps{
		SpaceID:   spaceID,
		SpaceName: space.Name,
		Plans:     plans,
	}))
}

func (h *budgetPlanHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	if _, err := h.spaceService.GetSpace(spaceID); err != nil {
		ui.RenderError(w, r, "Space not found", http.StatusNotFound)
		return
	}
	plan, err := h.planService.CreatePlan(
		spaceID,
		r.FormValue("name"),
		r.FormValue("note"),
		r.FormValue("currency"),
	)
	if err != nil {
		slog.Error("failed to create budget plan", "error", err, "space_id", spaceID)
		ui.RenderError(w, r, "Failed to create plan", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, routeurl.URL("page.app.spaces.space.plans.plan", "spaceID", spaceID, "planID", plan.ID), http.StatusSeeOther)
}

// EditorPage renders the full plan editor.
func (h *budgetPlanHandler) EditorPage(w http.ResponseWriter, r *http.Request) {
	plan, ok := h.loadPlan(w, r)
	if !ok {
		return
	}
	space, err := h.spaceService.GetSpace(plan.SpaceID)
	if err != nil {
		ui.Render(w, r, pages.NotFound())
		return
	}
	summary, err := h.planService.Summarize(plan.ID)
	if err != nil {
		slog.Error("failed to summarize plan", "error", err, "plan_id", plan.ID)
		ui.RenderError(w, r, "Failed to load plan", http.StatusInternalServerError)
		return
	}
	categories, err := h.planService.Categories()
	if err != nil {
		slog.Error("failed to load categories", "error", err)
		categories = nil
	}
	ui.Render(w, r, pages.BudgetPlanEditorPage(pages.BudgetPlanEditorPageProps{
		SpaceID:    plan.SpaceID,
		SpaceName:  space.Name,
		Plan:       plan,
		Summary:    summary,
		Categories: categories,
	}))
}

func (h *budgetPlanHandler) HandleRename(w http.ResponseWriter, r *http.Request) {
	plan, ok := h.loadPlan(w, r)
	if !ok {
		return
	}
	if err := h.planService.RenamePlan(plan.ID, r.FormValue("name")); err != nil {
		slog.Error("failed to rename plan", "error", err, "plan_id", plan.ID)
	}
	http.Redirect(w, r, routeurl.URL("page.app.spaces.space.plans.plan", "spaceID", plan.SpaceID, "planID", plan.ID), http.StatusSeeOther)
}

func (h *budgetPlanHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	plan, ok := h.loadPlan(w, r)
	if !ok {
		return
	}
	if err := h.planService.DeletePlan(plan.ID); err != nil {
		slog.Error("failed to delete plan", "error", err, "plan_id", plan.ID)
		ui.RenderError(w, r, "Failed to delete plan", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, routeurl.URL("page.app.spaces.space.plans", "spaceID", plan.SpaceID), http.StatusSeeOther)
}

// ---------- Lines ----------

func (h *budgetPlanHandler) HandleAddLine(w http.ResponseWriter, r *http.Request) {
	plan, ok := h.loadPlan(w, r)
	if !ok {
		return
	}
	kind := strings.TrimSpace(r.FormValue("kind"))
	if !model.IsValidPlanLineKind(kind) {
		http.Error(w, "invalid kind", http.StatusBadRequest)
		return
	}
	isIncome := model.PlanLineKind(kind) == model.PlanLineKindIncome

	label := strings.TrimSpace(r.FormValue("label"))
	amountStr := strings.TrimSpace(r.FormValue("amount"))
	state := blocks.LineFormState{Label: label, Amount: amountStr, CategoryID: strings.TrimSpace(r.FormValue("category_id"))}

	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		state.Err = "Enter a valid amount (e.g. 12.34)."
		h.renderBoardFormError(w, r, plan, isIncome, state)
		return
	}
	if _, err := h.planService.AddLine(service.AddPlanLineInput{
		PlanID:     plan.ID,
		Kind:       model.PlanLineKind(kind),
		CategoryID: parseCategoryID(r.FormValue("category_id")),
		Label:      label,
		Amount:     amount,
	}); err != nil {
		state.Err = err.Error()
		h.renderBoardFormError(w, r, plan, isIncome, state)
		return
	}
	h.renderBoard(w, r, plan, blocks.BudgetPlanBoardProps{})
}

func (h *budgetPlanHandler) HandleUpdateLine(w http.ResponseWriter, r *http.Request) {
	plan, ok := h.loadPlan(w, r)
	if !ok {
		return
	}
	lineID := r.PathValue("lineID")
	line, err := h.planService.GetLine(lineID)
	if err != nil || line.PlanID != plan.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	label := strings.TrimSpace(r.FormValue("label"))
	amountStr := strings.TrimSpace(r.FormValue("amount"))
	state := blocks.LineFormState{Label: label, Amount: amountStr, CategoryID: strings.TrimSpace(r.FormValue("category_id"))}

	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		state.Err = "Enter a valid amount (e.g. 12.34)."
		h.renderBoard(w, r, plan, blocks.BudgetPlanBoardProps{EditLineID: lineID, EditForm: state})
		return
	}
	if err := h.planService.UpdateLine(line, label, amount, parseCategoryID(r.FormValue("category_id"))); err != nil {
		state.Err = err.Error()
		h.renderBoard(w, r, plan, blocks.BudgetPlanBoardProps{EditLineID: lineID, EditForm: state})
		return
	}
	h.renderBoard(w, r, plan, blocks.BudgetPlanBoardProps{})
}

func (h *budgetPlanHandler) HandleDeleteLine(w http.ResponseWriter, r *http.Request) {
	plan, ok := h.loadPlan(w, r)
	if !ok {
		return
	}
	lineID := r.PathValue("lineID")
	line, err := h.planService.GetLine(lineID)
	if err != nil || line.PlanID != plan.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := h.planService.DeleteLine(lineID); err != nil {
		slog.Error("failed to delete plan line", "error", err, "line_id", lineID)
		ui.RenderError(w, r, "Failed to delete line", http.StatusInternalServerError)
		return
	}
	h.renderBoard(w, r, plan, blocks.BudgetPlanBoardProps{})
}

// renderBoard summarizes the plan and renders the #plan-board fragment so a
// single HTMX swap refreshes the lines and totals together.
func (h *budgetPlanHandler) renderBoard(w http.ResponseWriter, r *http.Request, plan *model.BudgetPlan, props blocks.BudgetPlanBoardProps) {
	summary, err := h.planService.Summarize(plan.ID)
	if err != nil {
		slog.Error("failed to summarize plan", "error", err, "plan_id", plan.ID)
		ui.RenderError(w, r, "Failed to load plan", http.StatusInternalServerError)
		return
	}
	categories, err := h.planService.Categories()
	if err != nil {
		categories = nil
	}
	props.SpaceID = plan.SpaceID
	props.PlanID = plan.ID
	props.Currency = plan.Currency
	props.Summary = summary
	props.Categories = categories
	ui.Render(w, r, blocks.BudgetPlanBoard(props))
}

func (h *budgetPlanHandler) renderBoardFormError(w http.ResponseWriter, r *http.Request, plan *model.BudgetPlan, isIncome bool, state blocks.LineFormState) {
	props := blocks.BudgetPlanBoardProps{}
	if isIncome {
		props.IncomeForm = state
		props.ShowIncomeForm = true
	} else {
		props.ExpenseForm = state
		props.ShowExpenseForm = true
	}
	h.renderBoard(w, r, plan, props)
}

func parseCategoryID(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
