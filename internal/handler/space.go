package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type SpaceHandler struct {
	spaceService            *service.SpaceService
	expenseService          *service.ExpenseService
	accountService          *service.MoneyAccountService
	reportService           *service.ReportService
	budgetService           *service.BudgetService
	recurringService        *service.RecurringExpenseService
	listService             *service.ShoppingListService
	tagService              *service.TagService
	methodService           *service.PaymentMethodService
	loanService             *service.LoanService
	receiptService          *service.ReceiptService
	recurringReceiptService *service.RecurringReceiptService
}

func NewSpaceHandler(
	ss *service.SpaceService,
	es *service.ExpenseService,
	mas *service.MoneyAccountService,
	rps *service.ReportService,
	bs *service.BudgetService,
	rs *service.RecurringExpenseService,
	sls *service.ShoppingListService,
	ts *service.TagService,
	pms *service.PaymentMethodService,
	ls *service.LoanService,
	rcs *service.ReceiptService,
	rrs *service.RecurringReceiptService,
) *SpaceHandler {
	return &SpaceHandler{
		spaceService:            ss,
		expenseService:          es,
		accountService:          mas,
		reportService:           rps,
		budgetService:           bs,
		recurringService:        rs,
		listService:             sls,
		tagService:              ts,
		methodService:           pms,
		loanService:             ls,
		receiptService:          rcs,
		recurringReceiptService: rrs,
	}
}

func (h *SpaceHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	spaces, err := h.spaceService.GetSpacesForUser(user.ID)
	if err != nil {
		slog.Error("failed to get spaces for user", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.Dashboard(spaces))
}

func (h *SpaceHandler) CreateSpace(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		w.Header().Set("HX-Reswap", "none")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `<p id="create-space-error" hx-swap-oob="true" class="text-sm text-destructive">Space name is required</p>`)
		return
	}

	space, err := h.spaceService.CreateSpace(name, user.ID)
	if err != nil {
		slog.Error("failed to create space", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/app/spaces/"+space.ID)
	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) OverviewPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to get space", "error", err, "space_id", spaceID)
		http.Error(w, "Space not found.", http.StatusNotFound)
		return
	}

	balance, err := h.expenseService.GetBalanceForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get balance", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	allocated, err := h.accountService.GetTotalAllocatedForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get total allocated", "error", err, "space_id", spaceID)
		allocated = 0
	}
	balance -= allocated

	// This month's report
	now := time.Now()
	presets := service.GetPresetDateRanges(now)
	report, err := h.reportService.GetSpendingReport(spaceID, presets[0].From, presets[0].To)
	if err != nil {
		slog.Error("failed to get spending report", "error", err, "space_id", spaceID)
		report = nil
	}

	// Budgets
	budgets, err := h.budgetService.GetBudgetsWithSpent(spaceID)
	if err != nil {
		slog.Error("failed to get budgets", "error", err, "space_id", spaceID)
	}

	// Recurring expenses
	recs, err := h.recurringService.GetRecurringExpensesWithTagsAndMethodsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get recurring expenses", "error", err, "space_id", spaceID)
	}

	// Shopping lists
	cards, err := h.buildListCards(spaceID)
	if err != nil {
		slog.Error("failed to build list cards", "error", err, "space_id", spaceID)
	}

	tags, err := h.tagService.GetTagsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get tags for space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	listsWithItems, err := h.listService.GetListsWithUncheckedItems(spaceID)
	if err != nil {
		slog.Error("failed to get lists with unchecked items", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	methods, err := h.methodService.GetMethodsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get payment methods", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceOverviewPage(pages.OverviewData{
		Space:             space,
		Balance:           balance,
		Allocated:         allocated,
		Report:            report,
		Budgets:           budgets,
		UpcomingRecurring: recs,
		ShoppingLists:     cards,
		Tags:              tags,
		Methods:           methods,
		ListsWithItems:    listsWithItems,
	}))
}

func (h *SpaceHandler) ReportsPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to get space", "error", err, "space_id", spaceID)
		http.Error(w, "Space not found.", http.StatusNotFound)
		return
	}

	now := time.Now()
	presets := service.GetPresetDateRanges(now)
	from := presets[0].From
	to := presets[0].To

	report, err := h.reportService.GetSpendingReport(spaceID, from, to)
	if err != nil {
		slog.Error("failed to get spending report", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceReportsPage(space, report, presets, "this_month"))
}

func (h *SpaceHandler) buildListCards(spaceID string) ([]model.ListCardData, error) {
	lists, err := h.listService.GetListsForSpace(spaceID)
	if err != nil {
		return nil, err
	}

	cards := make([]model.ListCardData, len(lists))
	for i, list := range lists {
		items, totalPages, err := h.listService.GetItemsForListPaginated(list.ID, 1)
		if err != nil {
			return nil, err
		}
		cards[i] = model.ListCardData{
			List:        list,
			Items:       items,
			CurrentPage: 1,
			TotalPages:  totalPages,
		}
	}

	return cards, nil
}
