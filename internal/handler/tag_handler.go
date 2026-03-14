package handler

import (
	"log/slog"
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/tag"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/toast"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type TagHandler struct {
	spaceService *service.SpaceService
	tagService   *service.TagService
}

func NewTagHandler(ss *service.SpaceService, ts *service.TagService) *TagHandler {
	return &TagHandler{
		spaceService: ss,
		tagService:   ts,
	}
}

// getTagForSpace fetches a tag and verifies it belongs to the given space.
func (h *TagHandler) getTagForSpace(w http.ResponseWriter, spaceID, tagID string) *model.Tag {
	t, err := h.tagService.GetTagByID(tagID)
	if err != nil {
		http.Error(w, "Tag not found", http.StatusNotFound)
		return nil
	}
	if t.SpaceID != spaceID {
		http.Error(w, "Not Found", http.StatusNotFound)
		return nil
	}
	return t
}

func (h *TagHandler) TagsPage(w http.ResponseWriter, r *http.Request) {
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

func (h *TagHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	if err := r.ParseForm(); err != nil {
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
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

func (h *TagHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceID")
	tagID := r.PathValue("tagID")

	if h.getTagForSpace(w, spaceID, tagID) == nil {
		return
	}

	err := h.tagService.DeleteTag(tagID)
	if err != nil {
		slog.Error("failed to delete tag", "error", err, "tag_id", tagID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	ui.RenderToast(w, r, toast.Toast(toast.Props{
		Title:       "Tag deleted",
		Variant:     toast.VariantSuccess,
		Icon:        true,
		Dismissible: true,
		Duration:    5000,
	}))
}
