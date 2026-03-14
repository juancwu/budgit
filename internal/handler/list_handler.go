package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/shoppinglist"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/toast"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type ListHandler struct {
	spaceService *service.SpaceService
	listService  *service.ShoppingListService
}

func NewListHandler(ss *service.SpaceService, sls *service.ShoppingListService) *ListHandler {
	return &ListHandler{
		spaceService: ss,
		listService:  sls,
	}
}

// getListForSpace fetches a shopping list and verifies it belongs to the given space.
// Returns the list on success, or writes an error response and returns nil.
func (h *ListHandler) getListForSpace(w http.ResponseWriter, spaceID, listID string) *model.ShoppingList {
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

func (h *ListHandler) ListsPage(w http.ResponseWriter, r *http.Request) {
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

func (h *ListHandler) CreateList(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	err := r.ParseForm()
	if err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		// handle error - maybe return a toast
		ui.RenderError(w, r, "List name is required", http.StatusUnprocessableEntity)
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

func (h *ListHandler) UpdateList(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	listID := r.PathValue("listID")

	if h.getListForSpace(w, spaceID, listID) == nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		ui.RenderError(w, r, "List name is required", http.StatusUnprocessableEntity)
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

func (h *ListHandler) DeleteList(w http.ResponseWriter, r *http.Request) {
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
	ui.RenderToast(w, r, toast.Toast(toast.Props{
		Title:       "List deleted",
		Variant:     toast.VariantSuccess,
		Icon:        true,
		Dismissible: true,
		Duration:    5000,
	}))
}

func (h *ListHandler) ListPage(w http.ResponseWriter, r *http.Request) {
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

func (h *ListHandler) AddItemToList(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	listID := r.PathValue("listID")

	if h.getListForSpace(w, spaceID, listID) == nil {
		return
	}

	user := ctxkeys.User(r.Context())

	if err := r.ParseForm(); err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		ui.RenderError(w, r, "Item name cannot be empty", http.StatusUnprocessableEntity)
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

func (h *ListHandler) ToggleItem(w http.ResponseWriter, r *http.Request) {
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

func (h *ListHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
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
	ui.RenderToast(w, r, toast.Toast(toast.Props{
		Title:       "Item deleted",
		Variant:     toast.VariantSuccess,
		Icon:        true,
		Dismissible: true,
		Duration:    5000,
	}))
}

func (h *ListHandler) GetShoppingListItems(w http.ResponseWriter, r *http.Request) {
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

func (h *ListHandler) GetLists(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")

	cards, err := h.buildListCards(spaceID)
	if err != nil {
		slog.Error("failed to build list cards", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.ListsContainer(spaceID, cards))
}

func (h *ListHandler) GetListCardItems(w http.ResponseWriter, r *http.Request) {
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

func (h *ListHandler) buildListCards(spaceID string) ([]model.ListCardData, error) {
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
