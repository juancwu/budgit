package handler

import (
	"log/slog"
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/shoppinglist"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/tag"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type SpaceHandler struct {
	spaceService *service.SpaceService
	tagService   *service.TagService
	listService  *service.ShoppingListService
}

func NewSpaceHandler(ss *service.SpaceService, ts *service.TagService, sls *service.ShoppingListService) *SpaceHandler {
	return &SpaceHandler{
		spaceService: ss,
		tagService:   ts,
		listService:  sls,
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

