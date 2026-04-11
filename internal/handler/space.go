package handler

import (
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/blocks"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
	"github.com/shopspring/decimal"
)

type spaceHandler struct {
	spaceService *service.SpaceService
}

func NewSpaceHandler(spaceService *service.SpaceService) *spaceHandler {
	return &spaceHandler{spaceService: spaceService}
}

func (h *spaceHandler) SpacesPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, pages.Spaces([]blocks.SpaceCardInfo{
		{
			ID: "test-1", Name: "My Space",
			MemberCount: 3, TotalBalance: decimal.NewFromFloat(123.23),
		},
	}))
}
