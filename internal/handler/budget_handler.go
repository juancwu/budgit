package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/toast"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type BudgetHandler struct {
	spaceService  *service.SpaceService
	budgetService *service.BudgetService
	tagService    *service.TagService
	reportService *service.ReportService
}

func NewBudgetHandler(ss *service.SpaceService, bs *service.BudgetService, ts *service.TagService, rps *service.ReportService) *BudgetHandler {
	return &BudgetHandler{
		spaceService:  ss,
		budgetService: bs,
		tagService:    ts,
		reportService: rps,
	}
}

func (h *BudgetHandler) getBudgetForSpace(w http.ResponseWriter, spaceID, budgetID string) *model.Budget {
	budget, err := h.budgetService.GetBudget(budgetID)
	if err != nil {
		http.Error(w, "Budget not found", http.StatusNotFound)
		return nil
	}
	if budget.SpaceID != spaceID {
		http.Error(w, "Not Found", http.StatusNotFound)
		return nil
	}
	return budget
}

func (h *BudgetHandler) BudgetsPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	tags, err := h.tagService.GetTagsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get tags", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	budgets, err := h.budgetService.GetBudgetsWithSpent(spaceID)
	if err != nil {
		slog.Error("failed to get budgets", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceBudgetsPage(space, budgets, tags))
}

func (h *BudgetHandler) CreateBudget(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	user := ctxkeys.User(r.Context())

	if err := r.ParseForm(); err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	tagNames := r.Form["tags"]
	amountStr := r.FormValue("amount")
	periodStr := r.FormValue("period")
	startDateStr := r.FormValue("start_date")
	endDateStr := r.FormValue("end_date")

	if len(tagNames) == 0 || amountStr == "" || periodStr == "" || startDateStr == "" {
		ui.RenderError(w, r, "All required fields must be provided.", http.StatusUnprocessableEntity)
		return
	}

	tagIDs, err := processTagNames(h.tagService, spaceID, tagNames)
	if err != nil {
		slog.Error("failed to process tag names", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if len(tagIDs) == 0 {
		ui.RenderError(w, r, "At least one valid tag is required.", http.StatusUnprocessableEntity)
		return
	}

	amountDecimal, err := decimal.NewFromString(amountStr)
	if err != nil {
		ui.RenderError(w, r, "Invalid amount.", http.StatusUnprocessableEntity)
		return
	}
	amount := amountDecimal

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		ui.RenderError(w, r, "Invalid start date.", http.StatusUnprocessableEntity)
		return
	}

	var endDate *time.Time
	if endDateStr != "" {
		ed, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			ui.RenderError(w, r, "Invalid end date.", http.StatusUnprocessableEntity)
			return
		}
		endDate = &ed
	}

	_, err = h.budgetService.CreateBudget(service.CreateBudgetDTO{
		SpaceID:   spaceID,
		TagIDs:    tagIDs,
		Amount:    amount,
		Period:    model.BudgetPeriod(periodStr),
		StartDate: startDate,
		EndDate:   endDate,
		CreatedBy: user.ID,
	})
	if err != nil {
		slog.Error("failed to create budget", "error", err)
		http.Error(w, "Failed to create budget.", http.StatusInternalServerError)
		return
	}

	// Refresh the full budgets list
	tags, _ := h.tagService.GetTagsForSpace(spaceID)
	budgets, _ := h.budgetService.GetBudgetsWithSpent(spaceID)
	ui.Render(w, r, pages.BudgetsList(spaceID, budgets, tags))
}

func (h *BudgetHandler) UpdateBudget(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	budgetID := r.PathValue("budgetID")

	if h.getBudgetForSpace(w, spaceID, budgetID) == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	tagNames := r.Form["tags"]
	amountStr := r.FormValue("amount")
	periodStr := r.FormValue("period")
	startDateStr := r.FormValue("start_date")
	endDateStr := r.FormValue("end_date")

	if len(tagNames) == 0 || amountStr == "" || periodStr == "" || startDateStr == "" {
		ui.RenderError(w, r, "All required fields must be provided.", http.StatusUnprocessableEntity)
		return
	}

	tagIDs, err := processTagNames(h.tagService, spaceID, tagNames)
	if err != nil {
		slog.Error("failed to process tag names", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if len(tagIDs) == 0 {
		ui.RenderError(w, r, "At least one valid tag is required.", http.StatusUnprocessableEntity)
		return
	}

	amountDecimal, err := decimal.NewFromString(amountStr)
	if err != nil {
		ui.RenderError(w, r, "Invalid amount.", http.StatusUnprocessableEntity)
		return
	}
	amount := amountDecimal

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		ui.RenderError(w, r, "Invalid start date.", http.StatusUnprocessableEntity)
		return
	}

	var endDate *time.Time
	if endDateStr != "" {
		ed, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			ui.RenderError(w, r, "Invalid end date.", http.StatusUnprocessableEntity)
			return
		}
		endDate = &ed
	}

	_, err = h.budgetService.UpdateBudget(service.UpdateBudgetDTO{
		ID:        budgetID,
		TagIDs:    tagIDs,
		Amount:    amount,
		Period:    model.BudgetPeriod(periodStr),
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		slog.Error("failed to update budget", "error", err)
		http.Error(w, "Failed to update budget.", http.StatusInternalServerError)
		return
	}

	// Refresh the full budgets list
	tags, _ := h.tagService.GetTagsForSpace(spaceID)
	budgets, _ := h.budgetService.GetBudgetsWithSpent(spaceID)
	ui.Render(w, r, pages.BudgetsList(spaceID, budgets, tags))
}

func (h *BudgetHandler) DeleteBudget(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	budgetID := r.PathValue("budgetID")

	if h.getBudgetForSpace(w, spaceID, budgetID) == nil {
		return
	}

	if err := h.budgetService.DeleteBudget(budgetID); err != nil {
		slog.Error("failed to delete budget", "error", err, "budget_id", budgetID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	ui.RenderToast(w, r, toast.Toast(toast.Props{
		Title:       "Budget deleted",
		Variant:     toast.VariantSuccess,
		Icon:        true,
		Dismissible: true,
		Duration:    5000,
	}))
}

func (h *BudgetHandler) GetBudgetsList(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	tags, _ := h.tagService.GetTagsForSpace(spaceID)
	budgets, err := h.budgetService.GetBudgetsWithSpent(spaceID)
	if err != nil {
		slog.Error("failed to get budgets", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.BudgetsList(spaceID, budgets, tags))
}

func (h *BudgetHandler) GetReportCharts(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	rangeKey := r.URL.Query().Get("range")
	now := time.Now()
	presets := service.GetPresetDateRanges(now)

	var from, to time.Time
	activeRange := "this_month"

	if rangeKey == "custom" {
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
		var err error
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			from = presets[0].From
		}
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			to = presets[0].To
		}
		activeRange = "custom"
	} else {
		for _, p := range presets {
			if p.Key == rangeKey {
				from = p.From
				to = p.To
				activeRange = p.Key
				break
			}
		}
		if from.IsZero() {
			from = presets[0].From
			to = presets[0].To
		}
	}

	report, err := h.reportService.GetSpendingReport(spaceID, from, to)
	if err != nil {
		slog.Error("failed to get report charts", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.ReportCharts(spaceID, report, from, to, presets, activeRange))
}
