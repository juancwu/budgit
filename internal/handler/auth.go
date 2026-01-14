package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/toast"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
	"git.juancwu.dev/juancwu/budgit/internal/validation"
)

type authHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *authHandler {
	return &authHandler{authService: authService}
}

func (h *authHandler) AuthPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, pages.Auth(""))
}

func (h *authHandler) PasswordPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, pages.AuthPassword(""))
}

func (h *authHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.authService.ClearJWTCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *authHandler) SendMagicLink(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email"))

	if email == "" {
		ui.Render(w, r, pages.Auth("Email is required"))
		return
	}

	err := validation.ValidateEmail(email)
	if err != nil {
		ui.Render(w, r, pages.Auth("Please provide a valid email address"))
		return
	}

	err = h.authService.SendMagicLink(email)
	if err != nil {
		slog.Warn("magic link send failed", "error", err, "email", email)
	}

	if r.URL.Query().Get("resend") == "true" {
		ui.RenderOOB(w, r, toast.Toast(toast.Props{
			Title:       "Magic link sent",
			Description: "Check your email for a new magic link",
			Variant:     toast.VariantSuccess,
			Icon:        true,
			Dismissible: true,
			Duration:    5000,
		}), "beforeend:#toast-container")
		return
	}

	ui.Render(w, r, pages.MagicLinkSent(email))
}

func (h *authHandler) VerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	tokenString := r.PathValue("token")

	user, err := h.authService.VerifyMagicLink(tokenString)
	if err != nil {
		slog.Warn("magic link verification failed", "error", err, "token", tokenString)
		ui.Render(w, r, pages.Auth("Invalid or expired magic link. Please try again."))
		return
	}

	jwtToken, err := h.authService.GenerateJWT(user)
	if err != nil {
		slog.Error("failed to generate JWT", "error", err, "user_id", user.ID)
		ui.Render(w, r, pages.Auth("An error occurred. Please try again."))
		return
	}

	h.authService.SetJWTCookie(w, jwtToken, time.Now().Add(7*24*time.Hour))

	needsOnboarding, err := h.authService.NeedsOnboarding(user.ID)
	if err != nil {
		slog.Warn("failed to check onboarding status", "error", err, "user_id", user.ID)
	}

	if needsOnboarding {
		slog.Info("new user needs onboarding", "user_id", user.ID, "email", user.Email)
		http.Redirect(w, r, "/auth/onboarding", http.StatusSeeOther)
		return
	}

	slog.Info("user logged via magic link", "user_id", user.ID, "email", user.Email)
	http.Redirect(w, r, "/app/dashboard", http.StatusSeeOther)
}

func (h *authHandler) OnboardingPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, pages.Onboarding(""))
}

func (h *authHandler) CompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))

	err := h.authService.CompleteOnboarding(user.ID, name)
	if err != nil {
		slog.Error("onboarding failed", "error", err, "user_id", user.ID)
		ui.Render(w, r, pages.Onboarding("Please enter your name"))
		return
	}

	slog.Info("onboarding completed", "user_id", user.ID, "name", name)
	http.Redirect(w, r, "/app/dashboard", http.StatusSeeOther)
}
