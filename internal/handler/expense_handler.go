package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/expense"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/toast"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type ExpenseHandler struct {
	spaceService   *service.SpaceService
	expenseService *service.ExpenseService
	tagService     *service.TagService
	listService    *service.ShoppingListService
	accountService *service.MoneyAccountService
	methodService  *service.PaymentMethodService
}

func NewExpenseHandler(ss *service.SpaceService, es *service.ExpenseService, ts *service.TagService, sls *service.ShoppingListService, mas *service.MoneyAccountService, pms *service.PaymentMethodService) *ExpenseHandler {
	return &ExpenseHandler{
		spaceService:   ss,
		expenseService: es,
		tagService:     ts,
		listService:    sls,
		accountService: mas,
		methodService:  pms,
	}
}

// getExpenseForSpace fetches an expense and verifies it belongs to the given space.
func (h *ExpenseHandler) getExpenseForSpace(w http.ResponseWriter, spaceID, expenseID string) *model.Expense {
	exp, err := h.expenseService.GetExpense(expenseID)
	if err != nil {
		http.Error(w, "Expense not found", http.StatusNotFound)
		return nil
	}
	if exp.SpaceID != spaceID {
		http.Error(w, "Not Found", http.StatusNotFound)
		return nil
	}
	return exp
}

func (h *ExpenseHandler) ExpensesPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	expenses, totalPages, err := h.expenseService.GetExpensesWithTagsAndMethodsForSpacePaginated(spaceID, page)
	if err != nil {
		slog.Error("failed to get expenses for space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	balance, err := h.expenseService.GetBalanceForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get balance for space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	totalAllocated, err := h.accountService.GetTotalAllocatedForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get total allocated", "error", err, "space_id", spaceID)
		totalAllocated = decimal.Zero
	}
	balance = balance.Sub(totalAllocated)

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

	ui.Render(w, r, pages.SpaceExpensesPage(space, expenses, balance, totalAllocated, tags, listsWithItems, methods, page, totalPages))

	if r.URL.Query().Get("created") == "true" {
		ui.Render(w, r, toast.Toast(toast.Props{
			Title:       "Expense created",
			Description: "Your transaction has been recorded.",
			Variant:     toast.VariantSuccess,
			Icon:        true,
			Dismissible: true,
			Duration:    5000,
		}))
	}
}

func (h *ExpenseHandler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	user := ctxkeys.User(r.Context())

	if err := r.ParseForm(); err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	// --- Form Parsing ---
	description := r.FormValue("description")
	amountStr := r.FormValue("amount")
	typeStr := r.FormValue("type")
	dateStr := r.FormValue("date")
	tagNames := r.Form["tags"] // Contains tag names

	// --- Validation & Conversion ---
	if description == "" || amountStr == "" || typeStr == "" || dateStr == "" {
		ui.RenderError(w, r, "All fields are required.", http.StatusUnprocessableEntity)
		return
	}

	amountDecimal, err := decimal.NewFromString(amountStr)
	if err != nil {
		ui.RenderError(w, r, "Invalid amount format.", http.StatusUnprocessableEntity)
		return
	}
	amount := amountDecimal

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		ui.RenderError(w, r, "Invalid date format.", http.StatusUnprocessableEntity)
		return
	}

	expenseType := model.ExpenseType(typeStr)
	if expenseType != model.ExpenseTypeExpense && expenseType != model.ExpenseTypeTopup {
		ui.RenderError(w, r, "Invalid transaction type.", http.StatusUnprocessableEntity)
		return
	}

	// --- Tag Processing ---
	existingTags, err := h.tagService.GetTagsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get tags for space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	existingTagsMap := make(map[string]string)
	for _, t := range existingTags {
		existingTagsMap[t.Name] = t.ID
	}

	var finalTagIDs []string
	processedTags := make(map[string]bool)

	for _, rawTagName := range tagNames {
		tagName := service.NormalizeTagName(rawTagName)
		if tagName == "" {
			continue
		}
		if processedTags[tagName] {
			continue
		}

		if id, exists := existingTagsMap[tagName]; exists {
			finalTagIDs = append(finalTagIDs, id)
		} else {
			// Create new tag
			newTag, err := h.tagService.CreateTag(spaceID, tagName, nil)
			if err != nil {
				slog.Error("failed to create new tag from expense form", "error", err, "tag_name", tagName)
				continue
			}
			finalTagIDs = append(finalTagIDs, newTag.ID)
			existingTagsMap[tagName] = newTag.ID
		}
		processedTags[tagName] = true
	}

	// Parse payment_method_id
	var paymentMethodID *string
	if pmid := r.FormValue("payment_method_id"); pmid != "" {
		paymentMethodID = &pmid
	}

	// Parse linked shopping list items
	itemIDs := r.Form["item_ids"]
	itemAction := r.FormValue("item_action")

	// Only link items for expense type, not topup
	if expenseType != model.ExpenseTypeExpense {
		itemIDs = nil
	}

	dto := service.CreateExpenseDTO{
		SpaceID:         spaceID,
		UserID:          user.ID,
		Description:     description,
		Amount:          amount,
		Type:            expenseType,
		Date:            date,
		TagIDs:          finalTagIDs,
		ItemIDs:         itemIDs,
		PaymentMethodID: paymentMethodID,
	}

	_, err = h.expenseService.CreateExpense(dto)
	if err != nil {
		slog.Error("failed to create expense", "error", err)
		http.Error(w, "Failed to create expense.", http.StatusInternalServerError)
		return
	}

	// Process linked items post-creation
	for _, itemID := range itemIDs {
		if itemAction == "delete" {
			if err := h.listService.DeleteItem(itemID); err != nil {
				slog.Error("failed to delete linked item", "error", err, "item_id", itemID)
			}
		} else {
			if err := h.listService.CheckItem(itemID); err != nil {
				slog.Error("failed to check linked item", "error", err, "item_id", itemID)
			}
		}
	}

	// If a redirect URL was provided (e.g. from the overview page), redirect instead of inline swap
	if redirectURL := r.FormValue("redirect"); redirectURL != "" {
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
		return
	}

	balance, err := h.expenseService.GetBalanceForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get balance", "error", err, "space_id", spaceID)
	}

	totalAllocated, err := h.accountService.GetTotalAllocatedForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get total allocated", "error", err, "space_id", spaceID)
		totalAllocated = decimal.Zero
	}
	balance = balance.Sub(totalAllocated)

	// Return the full paginated list for page 1 so the new expense appears
	expenses, totalPages, err := h.expenseService.GetExpensesWithTagsAndMethodsForSpacePaginated(spaceID, 1)
	if err != nil {
		slog.Error("failed to get paginated expenses after create", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Re-fetch tags (may have been auto-created)
	refreshedTags, _ := h.tagService.GetTagsForSpace(spaceID)
	ui.Render(w, r, pages.ExpenseCreatedResponse(spaceID, expenses, balance, totalAllocated, refreshedTags, 1, totalPages))

	// OOB-swap the item selector with fresh data (items may have been deleted/checked)
	listsWithItems, err := h.listService.GetListsWithUncheckedItems(spaceID)
	if err != nil {
		slog.Error("failed to refresh lists with items after create", "error", err, "space_id", spaceID)
		return
	}
	ui.Render(w, r, expense.ItemSelectorSection(listsWithItems, true))
}

func (h *ExpenseHandler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	expenseID := r.PathValue("expenseID")

	if h.getExpenseForSpace(w, spaceID, expenseID) == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	description := r.FormValue("description")
	amountStr := r.FormValue("amount")
	typeStr := r.FormValue("type")
	dateStr := r.FormValue("date")
	tagNames := r.Form["tags"]

	if description == "" || amountStr == "" || typeStr == "" || dateStr == "" {
		ui.RenderError(w, r, "All fields are required.", http.StatusUnprocessableEntity)
		return
	}

	amountDecimal, err := decimal.NewFromString(amountStr)
	if err != nil {
		ui.RenderError(w, r, "Invalid amount format.", http.StatusUnprocessableEntity)
		return
	}
	amount := amountDecimal

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		ui.RenderError(w, r, "Invalid date format.", http.StatusUnprocessableEntity)
		return
	}

	expenseType := model.ExpenseType(typeStr)
	if expenseType != model.ExpenseTypeExpense && expenseType != model.ExpenseTypeTopup {
		ui.RenderError(w, r, "Invalid transaction type.", http.StatusUnprocessableEntity)
		return
	}

	// Tag processing (same as CreateExpense)
	existingTags, err := h.tagService.GetTagsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get tags for space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	existingTagsMap := make(map[string]string)
	for _, t := range existingTags {
		existingTagsMap[t.Name] = t.ID
	}

	var finalTagIDs []string
	processedTags := make(map[string]bool)

	for _, rawTagName := range tagNames {
		tagName := service.NormalizeTagName(rawTagName)
		if tagName == "" || processedTags[tagName] {
			continue
		}

		if id, exists := existingTagsMap[tagName]; exists {
			finalTagIDs = append(finalTagIDs, id)
		} else {
			newTag, err := h.tagService.CreateTag(spaceID, tagName, nil)
			if err != nil {
				slog.Error("failed to create new tag from expense form", "error", err, "tag_name", tagName)
				continue
			}
			finalTagIDs = append(finalTagIDs, newTag.ID)
			existingTagsMap[tagName] = newTag.ID
		}
		processedTags[tagName] = true
	}

	// Parse payment_method_id
	var paymentMethodID *string
	if pmid := r.FormValue("payment_method_id"); pmid != "" {
		paymentMethodID = &pmid
	}

	dto := service.UpdateExpenseDTO{
		ID:              expenseID,
		SpaceID:         spaceID,
		Description:     description,
		Amount:          amount,
		Type:            expenseType,
		Date:            date,
		TagIDs:          finalTagIDs,
		PaymentMethodID: paymentMethodID,
	}

	updatedExpense, err := h.expenseService.UpdateExpense(dto)
	if err != nil {
		slog.Error("failed to update expense", "error", err)
		http.Error(w, "Failed to update expense.", http.StatusInternalServerError)
		return
	}

	tagsMap, _ := h.expenseService.GetTagsByExpenseIDs([]string{updatedExpense.ID})
	methodsMap, _ := h.expenseService.GetPaymentMethodsByExpenseIDs([]string{updatedExpense.ID})
	expWithTagsAndMethod := &model.ExpenseWithTagsAndMethod{
		Expense:       *updatedExpense,
		Tags:          tagsMap[updatedExpense.ID],
		PaymentMethod: methodsMap[updatedExpense.ID],
	}

	balance, err := h.expenseService.GetBalanceForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get balance after update", "error", err, "space_id", spaceID)
	}

	totalAllocated, err := h.accountService.GetTotalAllocatedForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get total allocated", "error", err, "space_id", spaceID)
		totalAllocated = decimal.Zero
	}
	balance = balance.Sub(totalAllocated)

	methods, _ := h.methodService.GetMethodsForSpace(spaceID)
	updatedTags, _ := h.tagService.GetTagsForSpace(spaceID)
	ui.Render(w, r, pages.ExpenseUpdatedResponse(spaceID, expWithTagsAndMethod, balance, totalAllocated, methods, updatedTags))
}

func (h *ExpenseHandler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	expenseID := r.PathValue("expenseID")

	if h.getExpenseForSpace(w, spaceID, expenseID) == nil {
		return
	}

	if err := h.expenseService.DeleteExpense(expenseID, spaceID); err != nil {
		slog.Error("failed to delete expense", "error", err, "expense_id", expenseID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	balance, err := h.expenseService.GetBalanceForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get balance after delete", "error", err, "space_id", spaceID)
	}

	totalAllocated, err := h.accountService.GetTotalAllocatedForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get total allocated", "error", err, "space_id", spaceID)
		totalAllocated = decimal.Zero
	}
	balance = balance.Sub(totalAllocated)

	ui.Render(w, r, expense.BalanceCard(spaceID, balance, totalAllocated, true))
	ui.RenderToast(w, r, toast.Toast(toast.Props{
		Title:       "Expense deleted",
		Variant:     toast.VariantSuccess,
		Icon:        true,
		Dismissible: true,
		Duration:    5000,
	}))
}

func (h *ExpenseHandler) GetExpensesList(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	expenses, totalPages, err := h.expenseService.GetExpensesWithTagsAndMethodsForSpacePaginated(spaceID, page)
	if err != nil {
		slog.Error("failed to get expenses", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	methods, _ := h.methodService.GetMethodsForSpace(spaceID)
	paginatedTags, _ := h.tagService.GetTagsForSpace(spaceID)
	ui.Render(w, r, pages.ExpensesListContent(spaceID, expenses, methods, paginatedTags, page, totalPages))
}

func (h *ExpenseHandler) GetBalanceCard(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	balance, err := h.expenseService.GetBalanceForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get balance", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	totalAllocated, err := h.accountService.GetTotalAllocatedForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get total allocated", "error", err, "space_id", spaceID)
		totalAllocated = decimal.Zero
	}
	balance = balance.Sub(totalAllocated)

	ui.Render(w, r, expense.BalanceCard(spaceID, balance, totalAllocated, false))
}
