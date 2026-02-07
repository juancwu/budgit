package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
)

type settingsHandler struct {
	authService *service.AuthService
	userService *service.UserService
}

func NewSettingsHandler(authService *service.AuthService, userService *service.UserService) *settingsHandler {
	return &settingsHandler{
		authService: authService,
		userService: userService,
	}
}

func (h *settingsHandler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())

	// Re-fetch user from DB since middleware strips PasswordHash
	fullUser, err := h.userService.ByID(user.ID)
	if err != nil {
		slog.Error("failed to fetch user for settings", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, pages.AppSettings(fullUser.HasPassword(), ""))
}

func (h *settingsHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	// Re-fetch user to check HasPassword
	fullUser, err := h.userService.ByID(user.ID)
	if err != nil {
		slog.Error("failed to fetch user for set password", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = h.authService.SetPassword(user.ID, currentPassword, newPassword, confirmPassword)
	if err != nil {
		slog.Warn("set password failed", "error", err, "user_id", user.ID)

		msg := "An error occurred. Please try again."
		if errors.Is(err, service.ErrInvalidCredentials) {
			msg = "Current password is incorrect"
		} else if errors.Is(err, service.ErrPasswordsDoNotMatch) {
			msg = "New passwords do not match"
		} else if errors.Is(err, service.ErrWeakPassword) {
			msg = "Password must be at least 12 characters"
		}

		ui.Render(w, r, pages.AppSettings(fullUser.HasPassword(), msg))
		return
	}

	// Password set successfully — render page with success message
	ui.Render(w, r, pages.AppSettings(true, ""))
}
