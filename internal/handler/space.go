package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/event"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/expense"
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
	eventBus       *event.Broker
}

func NewSpaceHandler(ss *service.SpaceService, ts *service.TagService, sls *service.ShoppingListService, es *service.ExpenseService, is *service.InviteService, eb *event.Broker) *SpaceHandler {
	return &SpaceHandler{
		spaceService:   ss,
		tagService:     ts,
		listService:    sls,
		expenseService: es,
		inviteService:  is,
		eventBus:       eb,
	}
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

func (h *SpaceHandler) StreamEvents(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to events
	eventChan := h.eventBus.Subscribe(spaceID)
	defer h.eventBus.Unsubscribe(spaceID, eventChan)

	// Listen for client disconnect
	ctx := r.Context()

	// Flush immediately to establish connection
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	for {
		select {
		case event := <-eventChan:
			// Write event to stream
			if _, err := w.Write([]byte(event.String())); err != nil {
				return
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-ctx.Done():
			return
		}
	}
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

	ui.Render(w, r, pages.SpaceOverviewPage(space, lists, tags))
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

	expenses, err := h.expenseService.GetExpensesForSpace(spaceID)
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

	ui.Render(w, r, pages.SpaceExpensesPage(space, expenses, balance, tags, listsWithItems))
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

	// Parse linked shopping list items
	itemIDs := r.Form["item_ids"]
	itemAction := r.FormValue("item_action")

	// Only link items for expense type, not topup
	if expenseType != model.ExpenseTypeExpense {
		itemIDs = nil
	}

	dto := service.CreateExpenseDTO{
		SpaceID:     spaceID,
		UserID:      user.ID,
		Description: description,
		Amount:      amountCents,
		Type:        expenseType,
		Date:        date,
		TagIDs:      finalTagIDs,
		ItemIDs:     itemIDs,
	}

	newExpense, err := h.expenseService.CreateExpense(dto)
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

	ui.Render(w, r, pages.ExpenseCreatedResponse(newExpense, balance))
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

	ui.Render(w, r, expense.BalanceCard(spaceID, balance, false))
}

func (h *SpaceHandler) GetExpensesList(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	expenses, err := h.expenseService.GetExpensesForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get expenses", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.ExpensesListContent(expenses))
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
