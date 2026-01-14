package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/shoppinglist"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/tag"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type SpaceHandler struct {
	spaceService   *service.SpaceService
	tagService     *service.TagService
	listService    *service.ShoppingListService
	expenseService *service.ExpenseService
	inviteService  *service.InviteService
}

func NewSpaceHandler(ss *service.SpaceService, ts *service.TagService, sls *service.ShoppingListService, es *service.ExpenseService, is *service.InviteService) *SpaceHandler {
	return &SpaceHandler{
		spaceService:   ss,
		tagService:     ts,
		listService:    sls,
		expenseService: es,
		inviteService:  is,
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

	ui.Render(w, r, pages.SpaceDashboardPage(space, lists, tags))
}

func (h *SpaceHandler) ListsPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	lists, err := h.listService.GetListsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get lists for space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceListsPage(space, lists))
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

	ui.Render(w, r, shoppinglist.ListItem(newList))
}

func (h *SpaceHandler) ListPage(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	listID := r.PathValue("listID")

	space, err := h.spaceService.GetSpace(spaceID)
	if err != nil {
		http.Error(w, "Space not found", http.StatusNotFound)
		return
	}

	list, err := h.listService.GetList(listID)
	if err != nil {
		slog.Error("failed to get list", "error", err, "list_id", listID)
		http.Error(w, "List not found", http.StatusNotFound)
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
	itemID := r.PathValue("itemID")

	item, err := h.listService.GetItem(itemID)
	if err != nil {
		slog.Error("failed to get item", "error", err, "item_id", itemID)
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	updatedItem, err := h.listService.UpdateItem(itemID, item.Name, !item.IsChecked)
	if err != nil {
		slog.Error("failed to toggle item", "error", err, "item_id", itemID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, shoppinglist.ItemDetail(spaceID, updatedItem))
}

func (h *SpaceHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("itemID")

	err := h.listService.DeleteItem(itemID)
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

	lists, err := h.listService.GetListsForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get lists for space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.SpaceExpensesPage(space, expenses, balance, tags, lists))
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
	tagIDs := r.Form["tags"] // For multi-select

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

	// --- DTO Creation & Service Call ---
	dto := service.CreateExpenseDTO{
		SpaceID:     spaceID,
		UserID:      user.ID,
		Description: description,
		Amount:      amountCents,
		Type:        expenseType,
		Date:        date,
		TagIDs:      tagIDs,
		ItemIDs:     []string{}, // TODO: Add item IDs from form
	}

	newExpense, err := h.expenseService.CreateExpense(dto)
	if err != nil {
		slog.Error("failed to create expense", "error", err)
		http.Error(w, "Failed to create expense.", http.StatusInternalServerError)
		return
	}

	balance, err := h.expenseService.GetBalanceForSpace(spaceID)
	if err != nil {
		slog.Error("failed to get balance", "error", err, "space_id", spaceID)
		// Fallback: return just the item if balance fails, but ideally we want both.
		// For now we will just log and continue, potentially showing stale balance.
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

	// TODO: Return a nice UI response (toast or list update)
	w.Write([]byte("Invitation sent!"))
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
