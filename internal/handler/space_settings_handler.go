package handler

import (
	"log/slog"
	"net/http"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/toast"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type SpaceSettingsHandler struct {
	spaceService  *service.SpaceService
	inviteService *service.InviteService
}

func NewSpaceSettingsHandler(ss *service.SpaceService, is *service.InviteService) *SpaceSettingsHandler {
	return &SpaceSettingsHandler{
		spaceService:  ss,
		inviteService: is,
	}
}

func (h *SpaceSettingsHandler) SettingsPage(w http.ResponseWriter, r *http.Request) {
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

func (h *SpaceSettingsHandler) UpdateSpaceName(w http.ResponseWriter, r *http.Request) {
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
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		ui.RenderError(w, r, "Name is required", http.StatusUnprocessableEntity)
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

func (h *SpaceSettingsHandler) UpdateSpaceTimezone(w http.ResponseWriter, r *http.Request) {
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
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	tz := r.FormValue("timezone")
	if tz == "" {
		ui.RenderError(w, r, "Timezone is required", http.StatusUnprocessableEntity)
		return
	}

	if err := h.spaceService.UpdateSpaceTimezone(spaceID, tz); err != nil {
		if err == service.ErrInvalidTimezone {
			ui.RenderError(w, r, "Invalid timezone", http.StatusUnprocessableEntity)
			return
		}
		slog.Error("failed to update space timezone", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

func (h *SpaceSettingsHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
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
		ui.RenderError(w, r, "Cannot remove yourself", http.StatusUnprocessableEntity)
		return
	}

	if err := h.spaceService.RemoveMember(spaceID, userID); err != nil {
		slog.Error("failed to remove member", "error", err, "space_id", spaceID, "user_id", userID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	ui.RenderToast(w, r, toast.Toast(toast.Props{
		Title:       "Member removed",
		Variant:     toast.VariantSuccess,
		Icon:        true,
		Dismissible: true,
		Duration:    5000,
	}))
}

func (h *SpaceSettingsHandler) CancelInvite(w http.ResponseWriter, r *http.Request) {
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
	ui.RenderToast(w, r, toast.Toast(toast.Props{
		Title:       "Invitation cancelled",
		Variant:     toast.VariantSuccess,
		Icon:        true,
		Dismissible: true,
		Duration:    5000,
	}))
}

func (h *SpaceSettingsHandler) GetPendingInvites(w http.ResponseWriter, r *http.Request) {
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

func (h *SpaceSettingsHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
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
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	email := r.FormValue("email")
	if email == "" {
		ui.RenderError(w, r, "Email is required", http.StatusUnprocessableEntity)
		return
	}

	_, err = h.inviteService.CreateInvite(spaceID, user.ID, email)
	if err != nil {
		slog.Error("failed to create invite", "error", err, "space_id", spaceID)
		http.Error(w, "Failed to create invite", http.StatusInternalServerError)
		return
	}

	ui.RenderToast(w, r, toast.Toast(toast.Props{
		Title:       "Invitation sent",
		Description: "An email has been sent to " + email,
		Variant:     toast.VariantSuccess,
		Icon:        true,
		Dismissible: true,
		Duration:    5000,
	}))
}

func (h *SpaceSettingsHandler) DeleteSpace(w http.ResponseWriter, r *http.Request) {
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
		ui.RenderError(w, r, "Bad Request", http.StatusUnprocessableEntity)
		return
	}

	confirmationName := r.FormValue("confirmation_name")
	if confirmationName != space.Name {
		ui.RenderError(w, r, "Space name does not match", http.StatusUnprocessableEntity)
		return
	}

	if err := h.spaceService.DeleteSpace(spaceID); err != nil {
		slog.Error("failed to delete space", "error", err, "space_id", spaceID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/app/spaces")
	w.WriteHeader(http.StatusOK)
}

func (h *SpaceSettingsHandler) JoinSpace(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	user := ctxkeys.User(r.Context())

	if user != nil {
		spaceID, err := h.inviteService.AcceptInvite(token, user.ID)
		if err != nil {
			slog.Error("failed to accept invite", "error", err, "token", token)
			ui.RenderError(w, r, "Failed to join space: "+err.Error(), http.StatusUnprocessableEntity)
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
