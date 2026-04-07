package handler

import (
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type spaceHandler struct {
	spaceService *service.SpaceService
}

func NewSpaceHandler(spaceService *service.SpaceService) *spaceHandler {
	return &spaceHandler{spaceService: spaceService}
}

func (h *spaceHandler) SpacesPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, pages.Spaces())
}
