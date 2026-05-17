package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/middleware"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/toast"
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

	ui.Render(w, r, pages.AppSettings(fullUser.HasPassword(), fullUser.Email, "", ""))
}

func (h *settingsHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())

	fullUser, err := h.userService.ByID(user.ID)
	if err != nil {
		slog.Error("failed to fetch user for account deletion", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	confirmation := r.FormValue("confirm_email")
	reason := r.FormValue("reason")

	err = h.userService.RequestAccountDeletion(service.RequestAccountDeletionInput{
		UserID:            user.ID,
		ConfirmationEmail: confirmation,
		Reason:            reason,
		IPAddress:         middleware.GetClientIP(r),
	})
	if err != nil {
		slog.Warn("account deletion request failed", "error", err, "user_id", user.ID)

		msg := "We couldn't queue your account for deletion. Please try again."
		if errors.Is(err, service.ErrEmailConfirmationMismatch) {
			msg = "The email you entered does not match your account email."
		} else if errors.Is(err, service.ErrAccountAlreadyPending) {
			// Race with another tab — just send them to the pending page.
			http.Redirect(w, r, "/account-pending-deletion", http.StatusSeeOther)
			return
		}
		ui.Render(w, r, pages.AppSettings(fullUser.HasPassword(), fullUser.Email, "", msg))
		return
	}

	slog.Info("account deletion queued", "user_id", user.ID, "email", fullUser.Email)
	http.Redirect(w, r, "/account-pending-deletion", http.StatusSeeOther)
}

func (h *settingsHandler) AccountPendingDeletionPage(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	if user == nil || !user.IsPendingDeletion() {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	ui.Render(w, r, pages.AccountPendingDeletion(*user.PendingDeletionAt))
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

		ui.Render(w, r, pages.AppSettings(fullUser.HasPassword(), fullUser.Email, msg, ""))
		return
	}

	// Password set successfully — render page with success toast
	ui.Render(w, r, pages.AppSettings(true, fullUser.Email, "", ""))
	ui.RenderToast(w, r, toast.Toast(toast.Props{
		Title:       "Password updated",
		Variant:     toast.VariantSuccess,
		Icon:        true,
		Dismissible: true,
		Duration:    5000,
	}))
}
