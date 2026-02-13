package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/expense"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/moneyaccount"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/paymentmethod"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/shoppinglist"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/tag"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/toast"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type SpaceHandler struct {
	spaceService   *service.SpaceService
	tagService     *service.TagService
	listService    *service.ShoppingListService
	expenseService *service.ExpenseService
	inviteService  *service.InviteService
	accountService *service.MoneyAccountService
	methodService  *service.PaymentMethodService
}

func NewSpaceHandler(ss *service.SpaceService, ts *service.TagService, sls *service.ShoppingListService, es *service.ExpenseService, is *service.InviteService, mas *service.MoneyAccountService, pms *service.PaymentMethodService) *SpaceHandler {
	return &SpaceHandler{
		spaceService:   ss,
		tagService:     ts,
		listService:    sls,
		expenseService: es,
		inviteService:  is,
		accountService: mas,
		methodService:  pms,
	}
}

// getExpenseForSpace fetches an expense and verifies it belongs to the given space.
func (h *SpaceHandler) getExpenseForSpace(w http.ResponseWriter, spaceID, expenseID string) *model.Expense {
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

// getListForSpace fetches a shopping list and verifies it belongs to the given space.
// Returns the list on success, or writes an error response and returns nil.
func (h *SpaceHandler) getListForSpace(w http.ResponseWriter, spaceID, listID string) *model.ShoppingList {
	list, err := h.listService.GetList(listID)
	if err != nil {
		http.Error(w, "List not found", http.StatusNotFound)
		return nil
	}
	if list.SpaceID != spaceID {
		http.Error(w, "Not Found", http.StatusNotFound)
		return nil
	}
	return list
}

func (h *SpaceHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to get space", "error", err, "space_id", spaceID)
		// The RequireSpaceAccess middleware should prevent this, but as a fallback.
		http.Error(w, "Space not found.", http.StatusNotFound)
		return
	}

	lists, err := h.listService.GetListsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get shopping lists for space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
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

	accounts, err := h.accountService.GetAccountsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get accounts for space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceOverviewPage(space, lists, tags, listsWithItems, accounts))
}

func (h *SpaceHandler) ListsPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	cards, err := h.buildListCards(spaceID)
	if err != nil {
		slog.Error("failed to build list cards", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceListsPage(space, cards))
}

func (h *SpaceHandler) CreateList(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		// handle error - maybe return a toast
		http.Error(w, "List name is required", http.StatusBadRequest)
		return
	}

	newList, err := h.listService.CreateList(spaceID, name)
	if err != nil {
		slog.Error("failed to create list", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, shoppinglist.ListCard(spaceID, newList, nil, 1, 1))
}

func (h *SpaceHandler) UpdateList(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	listID := r.PathValue("listID")

	if h.getListForSpace(w, spaceID, listID) == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "List name is required", http.StatusBadRequest)
		return
	}

	updatedList, err := h.listService.UpdateList(listID, name)
	if err != nil {
		slog.Error("failed to update list", "error", err, "list_id", listID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("from") == "card" {
		ui.Render(w, r, shoppinglist.ListCardHeader(spaceID, updatedList))
	} else {
		ui.Render(w, r, shoppinglist.ListNameHeader(spaceID, updatedList))
	}
}

func (h *SpaceHandler) DeleteList(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	listID := r.PathValue("listID")

	if h.getListForSpace(w, spaceID, listID) == nil {
		return
	}

	err := h.listService.DeleteList(listID)
	if err != nil {
		slog.Error("failed to delete list", "error", err, "list_id", listID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("from") != "card" {
		w.Header().Set("HX-Redirect", "/app/spaces/"+spaceID+"/lists")
	}
	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) ListPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	listID := r.PathValue("listID")

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	list := h.getListForSpace(w, spaceID, listID)
	if list == nil {
		return
	}

	items, err := h.listService.GetItemsForList(listID)
	if err != nil {
		slog.Error("failed to get items for list", "error", err, "list_id", listID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceListDetailPage(space, list, items))
}

func (h *SpaceHandler) AddItemToList(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	listID := r.PathValue("listID")

	if h.getListForSpace(w, spaceID, listID) == nil {
		return
	}

	user := ctxkeys.User(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Item name cannot be empty", http.StatusBadRequest)
		return
	}

	newItem, err := h.listService.AddItemToList(listID, name, user.ID)
	if err != nil {
		slog.Error("failed to add item to list", "error", err, "list_id", listID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, shoppinglist.ItemDetail(spaceID, newItem))
}

func (h *SpaceHandler) ToggleItem(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	listID := r.PathValue("listID")
	itemID := r.PathValue("itemID")

	if h.getListForSpace(w, spaceID, listID) == nil {
		return
	}

	item, err := h.listService.GetItem(itemID)
	if err != nil {
		slog.Error("failed to get item", "error", err, "item_id", itemID)
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	if item.ListID != listID {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	updatedItem, err := h.listService.UpdateItem(itemID, item.Name, !item.IsChecked)
	if err != nil {
		slog.Error("failed to toggle item", "error", err, "item_id", itemID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("from") == "card" {
		ui.Render(w, r, shoppinglist.CardItemDetail(spaceID, updatedItem))
	} else {
		ui.Render(w, r, shoppinglist.ItemDetail(spaceID, updatedItem))
	}
}

func (h *SpaceHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	listID := r.PathValue("listID")
	itemID := r.PathValue("itemID")

	if h.getListForSpace(w, spaceID, listID) == nil {
		return
	}

	item, err := h.listService.GetItem(itemID)
	if err != nil {
		slog.Error("failed to get item", "error", err, "item_id", itemID)
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	if item.ListID != listID {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	err = h.listService.DeleteItem(itemID)
	if err != nil {
		slog.Error("failed to delete item", "error", err, "item_id", itemID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) TagsPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	tags, err := h.tagService.GetTagsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get tags for space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceTagsPage(space, tags))
}

func (h *SpaceHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	color := r.FormValue("color") // color is optional

	var colorPtr *string
	if color != "" {
		colorPtr = &color
	}

	newTag, err := h.tagService.CreateTag(spaceID, name, colorPtr)
	if err != nil {
		slog.Error("failed to create tag", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, tag.Tag(newTag))
}

func (h *SpaceHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	tagID := r.PathValue("tagID")

	err := h.tagService.DeleteTag(tagID)
	if err != nil {
		slog.Error("failed to delete tag", "error", err, "tag_id", tagID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) ExpensesPage(w http.ResponseWriter, r *http.Request) {
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
		totalAllocated = 0
	}
	balance -= totalAllocated

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

func (h *SpaceHandler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	user := ctxkeys.User(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
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
		http.Error(w, "All fields are required.", http.StatusBadRequest)
		return
	}

	amountFloat, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		http.Error(w, "Invalid amount format.", http.StatusBadRequest)
		return
	}
	amountCents := int(amountFloat * 100)

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date format.", http.StatusBadRequest)
		return
	}

	expenseType := model.ExpenseType(typeStr)
	if expenseType != model.ExpenseTypeExpense && expenseType != model.ExpenseTypeTopup {
		http.Error(w, "Invalid transaction type.", http.StatusBadRequest)
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
		Amount:          amountCents,
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

	balance, err := h.expenseService.GetBalanceForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get balance", "error", err, "space_id", spaceID)
	}

	totalAllocated, err := h.accountService.GetTotalAllocatedForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get total allocated", "error", err, "space_id", spaceID)
		totalAllocated = 0
	}
	balance -= totalAllocated

	if r.URL.Query().Get("from") == "overview" {
		w.Header().Set("HX-Redirect", "/app/spaces/"+spaceID+"/expenses?created=true")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Return the full paginated list for page 1 so the new expense appears
	expenses, totalPages, err := h.expenseService.GetExpensesWithTagsAndMethodsForSpacePaginated(spaceID, 1)
	if err != nil {
		slog.Error("failed to get paginated expenses after create", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.ExpenseCreatedResponse(spaceID, expenses, balance, totalAllocated, 1, totalPages))

	// OOB-swap the item selector with fresh data (items may have been deleted/checked)
	listsWithItems, err := h.listService.GetListsWithUncheckedItems(spaceID)
	if err != nil {
		slog.Error("failed to refresh lists with items after create", "error", err, "space_id", spaceID)
		return
	}
	ui.Render(w, r, expense.ItemSelectorSection(listsWithItems, true))
}

func (h *SpaceHandler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	expenseID := r.PathValue("expenseID")

	if h.getExpenseForSpace(w, spaceID, expenseID) == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	description := r.FormValue("description")
	amountStr := r.FormValue("amount")
	typeStr := r.FormValue("type")
	dateStr := r.FormValue("date")
	tagNames := r.Form["tags"]

	if description == "" || amountStr == "" || typeStr == "" || dateStr == "" {
		http.Error(w, "All fields are required.", http.StatusBadRequest)
		return
	}

	amountFloat, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		http.Error(w, "Invalid amount format.", http.StatusBadRequest)
		return
	}
	amountCents := int(amountFloat * 100)

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date format.", http.StatusBadRequest)
		return
	}

	expenseType := model.ExpenseType(typeStr)
	if expenseType != model.ExpenseTypeExpense && expenseType != model.ExpenseTypeTopup {
		http.Error(w, "Invalid transaction type.", http.StatusBadRequest)
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
		Amount:          amountCents,
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
		totalAllocated = 0
	}
	balance -= totalAllocated

	methods, _ := h.methodService.GetMethodsForSpace(spaceID)
	ui.Render(w, r, pages.ExpenseUpdatedResponse(spaceID, expWithTagsAndMethod, balance, totalAllocated, methods))
}

func (h *SpaceHandler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
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
		totalAllocated = 0
	}
	balance -= totalAllocated

	ui.Render(w, r, expense.BalanceCard(spaceID, balance, totalAllocated, true))
}

func (h *SpaceHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	user := ctxkeys.User(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	if email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	_, err := h.inviteService.CreateInvite(spaceID, user.ID, email)
	if err != nil {
		slog.Error("failed to create invite", "error", err, "space_id", spaceID)
		http.Error(w, "Failed to create invite", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, toast.Toast(toast.Props{
		Title:       "Invitation sent",
		Description: "An email has been sent to " + email,
		Variant:     toast.VariantSuccess,
		Icon:        true,
		Dismissible: true,
		Duration:    5000,
	}))
}

func (h *SpaceHandler) JoinSpace(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	user := ctxkeys.User(r.Context())

	if user != nil {
		spaceID, err := h.inviteService.AcceptInvite(token, user.ID)
		if err != nil {
			slog.Error("failed to accept invite", "error", err, "token", token)
			http.Error(w, "Failed to join space: "+err.Error(), http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, "/app/spaces/"+spaceID, http.StatusSeeOther)
		return
	}

	// Not logged in: set cookie and redirect to auth
	http.SetCookie(w, &http.Cookie{
		Name:     "pending_invite",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(1 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/auth?invite=true", http.StatusTemporaryRedirect)
}

func (h *SpaceHandler) GetBalanceCard(w http.ResponseWriter, r *http.Request) {
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
		totalAllocated = 0
	}
	balance -= totalAllocated

	ui.Render(w, r, expense.BalanceCard(spaceID, balance, totalAllocated, false))
}

func (h *SpaceHandler) GetExpensesList(w http.ResponseWriter, r *http.Request) {
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
	ui.Render(w, r, pages.ExpensesListContent(spaceID, expenses, methods, page, totalPages))
}

func (h *SpaceHandler) GetShoppingListItems(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	listID := r.PathValue("listID")

	if h.getListForSpace(w, spaceID, listID) == nil {
		return
	}

	items, err := h.listService.GetItemsForList(listID)
	if err != nil {
		slog.Error("failed to get items", "error", err, "list_id", listID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.ShoppingListItems(spaceID, items))
}

func (h *SpaceHandler) GetLists(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	cards, err := h.buildListCards(spaceID)
	if err != nil {
		slog.Error("failed to build list cards", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.ListsContainer(spaceID, cards))
}

func (h *SpaceHandler) GetListCardItems(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	listID := r.PathValue("listID")

	if h.getListForSpace(w, spaceID, listID) == nil {
		return
	}

	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	items, totalPages, err := h.listService.GetItemsForListPaginated(listID, page)
	if err != nil {
		slog.Error("failed to get paginated items", "error", err, "list_id", listID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, shoppinglist.ListCardItems(spaceID, listID, items, page, totalPages))
}

func (h *SpaceHandler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	user := ctxkeys.User(r.Context())

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		slog.Error("failed to get space", "error", err, "space_id", spaceID)
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	members, err := h.spaceService.GetMembers(spaceID)
	if err != nil {
		slog.Error("failed to get members", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	isOwner := space.OwnerID == user.ID

	var pendingInvites []*model.SpaceInvitation
	if isOwner {
		pendingInvites, err = h.inviteService.GetPendingInvites(spaceID)
		if err != nil {
			slog.Error("failed to get pending invites", "error", err, "space_id", spaceID)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	ui.Render(w, r, pages.SpaceSettingsPage(space, members, pendingInvites, isOwner, user.ID))
}

func (h *SpaceHandler) UpdateSpaceName(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	user := ctxkeys.User(r.Context())

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	if space.OwnerID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	if err := h.spaceService.UpdateSpaceName(spaceID, name); err != nil {
		slog.Error("failed to update space name", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	userID := r.PathValue("userID")
	user := ctxkeys.User(r.Context())

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	if space.OwnerID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if userID == user.ID {
		http.Error(w, "Cannot remove yourself", http.StatusBadRequest)
		return
	}

	if err := h.spaceService.RemoveMember(spaceID, userID); err != nil {
		slog.Error("failed to remove member", "error", err, "space_id", spaceID, "user_id", userID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) CancelInvite(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	token := r.PathValue("token")
	user := ctxkeys.User(r.Context())

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	if space.OwnerID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.inviteService.CancelInvite(token); err != nil {
		slog.Error("failed to cancel invite", "error", err, "space_id", spaceID, "token", token)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SpaceHandler) GetPendingInvites(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	user := ctxkeys.User(r.Context())

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	if space.OwnerID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	pendingInvites, err := h.inviteService.GetPendingInvites(spaceID)
	if err != nil {
		slog.Error("failed to get pending invites", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.PendingInvitesList(spaceID, pendingInvites))
}

// --- Money Accounts ---

func (h *SpaceHandler) getAccountForSpace(w http.ResponseWriter, spaceID, accountID string) *model.MoneyAccount {
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

func (h *SpaceHandler) AccountsPage(w http.ResponseWriter, r *http.Request) {
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

	availableBalance := totalBalance - totalAllocated

	ui.Render(w, r, pages.SpaceAccountsPage(space, accounts, totalBalance, availableBalance))
}

func (h *SpaceHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	user := ctxkeys.User(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Account name is required", http.StatusBadRequest)
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
		BalanceCents: 0,
	}

	ui.Render(w, r, moneyaccount.AccountCard(spaceID, &acctWithBalance))
}

func (h *SpaceHandler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")

	if h.getAccountForSpace(w, spaceID, accountID) == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Account name is required", http.StatusBadRequest)
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
		BalanceCents: balance,
	}

	ui.Render(w, r, moneyaccount.AccountCard(spaceID, &acctWithBalance))
}

func (h *SpaceHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
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

	ui.Render(w, r, moneyaccount.BalanceSummaryCard(spaceID, totalBalance, totalBalance-totalAllocated, true))
}

func (h *SpaceHandler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	accountID := r.PathValue("accountID")
	user := ctxkeys.User(r.Context())

	if h.getAccountForSpace(w, spaceID, accountID) == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	amountStr := r.FormValue("amount")
	direction := model.TransferDirection(r.FormValue("direction"))
	note := r.FormValue("note")

	amountFloat, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amountFloat <= 0 {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}
	amountCents := int(amountFloat * 100)

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
	availableBalance := totalBalance - totalAllocated

	// Validate balance limits before creating transfer
	if direction == model.TransferDirectionDeposit && amountCents > availableBalance {
		fmt.Fprintf(w, "Insufficient available balance. You can deposit up to $%.2f.", float64(availableBalance)/100.0)
		return
	}

	if direction == model.TransferDirectionWithdrawal {
		acctBalance, err := h.accountService.GetAccountBalance(accountID)
		if err != nil {
			slog.Error("failed to get account balance", "error", err, "account_id", accountID)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if amountCents > acctBalance {
			fmt.Fprintf(w, "Insufficient account balance. You can withdraw up to $%.2f.", float64(acctBalance)/100.0)
			return
		}
	}

	_, err = h.accountService.CreateTransfer(service.CreateTransferDTO{
		AccountID: accountID,
		Amount:    amountCents,
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
		BalanceCents: accountBalance,
	}

	// Recalculate available balance after transfer
	totalAllocated, _ = h.accountService.GetTotalAllocatedForSpace(spaceID)
	newAvailable := totalBalance - totalAllocated

	w.Header().Set("HX-Trigger", "transferSuccess")
	ui.Render(w, r, moneyaccount.AccountCard(spaceID, &acctWithBalance, true))
	ui.Render(w, r, moneyaccount.BalanceSummaryCard(spaceID, totalBalance, newAvailable, true))
}

func (h *SpaceHandler) DeleteTransfer(w http.ResponseWriter, r *http.Request) {
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
		BalanceCents: accountBalance,
	}

	totalBalance, _ := h.expenseService.GetBalanceForSpace(spaceID)
	totalAllocated, _ := h.accountService.GetTotalAllocatedForSpace(spaceID)

	ui.Render(w, r, moneyaccount.AccountCard(spaceID, &acctWithBalance, true))
	ui.Render(w, r, moneyaccount.BalanceSummaryCard(spaceID, totalBalance, totalBalance-totalAllocated, true))
}

// --- Payment Methods ---

func (h *SpaceHandler) getMethodForSpace(w http.ResponseWriter, spaceID, methodID string) *model.PaymentMethod {
	method, err := h.methodService.GetMethod(methodID)
	if err != nil {
		http.Error(w, "Payment method not found", http.StatusNotFound)
		return nil
	}
	if method.SpaceID != spaceID {
		http.Error(w, "Not Found", http.StatusNotFound)
		return nil
	}
	return method
}

func (h *SpaceHandler) PaymentMethodsPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	methods, err := h.methodService.GetMethodsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get payment methods for space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpacePaymentMethodsPage(space, methods))
}

func (h *SpaceHandler) CreatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	user := ctxkeys.User(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	methodType := model.PaymentMethodType(r.FormValue("type"))
	lastFour := r.FormValue("last_four")

	method, err := h.methodService.CreateMethod(service.CreatePaymentMethodDTO{
		SpaceID:   spaceID,
		Name:      name,
		Type:      methodType,
		LastFour:  lastFour,
		CreatedBy: user.ID,
	})
	if err != nil {
		slog.Error("failed to create payment method", "error", err, "space_id", spaceID)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ui.Render(w, r, paymentmethod.MethodItem(spaceID, method))
}

func (h *SpaceHandler) UpdatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	methodID := r.PathValue("methodID")

	if h.getMethodForSpace(w, spaceID, methodID) == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	methodType := model.PaymentMethodType(r.FormValue("type"))
	lastFour := r.FormValue("last_four")

	updatedMethod, err := h.methodService.UpdateMethod(service.UpdatePaymentMethodDTO{
		ID:       methodID,
		Name:     name,
		Type:     methodType,
		LastFour: lastFour,
	})
	if err != nil {
		slog.Error("failed to update payment method", "error", err, "method_id", methodID)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ui.Render(w, r, paymentmethod.MethodItem(spaceID, updatedMethod))
}

func (h *SpaceHandler) DeletePaymentMethod(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	methodID := r.PathValue("methodID")

	if h.getMethodForSpace(w, spaceID, methodID) == nil {
		return
	}

	err := h.methodService.DeleteMethod(methodID)
	if err != nil {
		slog.Error("failed to delete payment method", "error", err, "method_id", methodID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
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
