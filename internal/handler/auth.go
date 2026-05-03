package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/ui"
	"git.juancwu.dev/juancwu/budgit/internal/ui/components/toast"
	"git.juancwu.dev/juancwu/budgit/internal/ui/pages"
	"git.juancwu.dev/juancwu/budgit/internal/validation"
)

type authHandler struct {
	authService   *service.AuthService
	inviteService *service.InviteService
	spaceService  *service.SpaceService
}

func NewAuthHandler(authService *service.AuthService, inviteService *service.InviteService, spaceService *service.SpaceService) *authHandler {
	return &authHandler{authService: authService, inviteService: inviteService, spaceService: spaceService}
}

func (h *authHandler) AuthPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, pages.Auth(""))
}

func (h *authHandler) PasswordPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, pages.AuthPassword(""))
}

func (h *authHandler) LoginWithPassword(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	if email == "" || password == "" {
		ui.Render(w, r, pages.AuthPassword("Email and password are required"))
		return
	}

	user, err := h.authService.LoginWithPassword(email, password)
	if err != nil {
		slog.Warn("password login failed", "error", err, "email", email)

		msg := "An error occurred. Please try again."
		if errors.Is(err, service.ErrInvalidCredentials) {
			msg = "Invalid email or password"
		} else if errors.Is(err, service.ErrNoPassword) {
			msg = "This account uses passwordless login. Please use a magic link."
		}

		ui.Render(w, r, pages.AuthPassword(msg))
		return
	}

	if err := h.completeLogin(w, r, user, pages.AuthPassword); err != nil {
		return
	}
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
		ui.RenderToast(w, r, toast.Toast(toast.Props{
			Title:       "Magic link sent",
			Description: "Check your email for a new magic link",
			Variant:     toast.VariantSuccess,
			Icon:        true,
			Dismissible: true,
			Duration:    5000,
		}))
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

	if err := h.completeLogin(w, r, user, pages.Auth); err != nil {
		return
	}

	slog.Info("user logged via magic link", "user_id", user.ID, "email", user.Email)
}

// completeLogin handles the post-authentication flow: JWT generation,
// pending invite processing, onboarding check, and redirect.
// Returns an error if the response was already written (caller should return early).
func (h *authHandler) completeLogin(w http.ResponseWriter, r *http.Request, user *model.User, renderError func(string) templ.Component) error {
	jwtToken, err := h.authService.GenerateJWT(user)
	if err != nil {
		slog.Error("failed to generate JWT", "error", err, "user_id", user.ID)
		ui.Render(w, r, renderError("An error occurred. Please try again."))
		return fmt.Errorf("jwt generation failed")
	}

	h.authService.SetJWTCookie(w, jwtToken)

	// Check for pending invite
	inviteCookie, err := r.Cookie("pending_invite")
	if err == nil && inviteCookie.Value != "" {
		spaceID, err := h.inviteService.AcceptInvite(inviteCookie.Value, user.ID)
		if err != nil {
			slog.Error("failed to process pending invite", "error", err, "token", inviteCookie.Value)
		} else {
			slog.Info("accepted pending invite", "user_id", user.ID, "space_id", spaceID)
			http.SetCookie(w, &http.Cookie{
				Name:     "pending_invite",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
		}
	}

	needsOnboarding, err := h.authService.NeedsOnboarding(user.ID)
	if err != nil {
		slog.Warn("failed to check onboarding status", "error", err, "user_id", user.ID)
	}

	if needsOnboarding {
		http.Redirect(w, r, "/auth/onboarding", http.StatusSeeOther)
		return fmt.Errorf("redirected to onboarding")
	}

	http.Redirect(w, r, "/app/dashboard", http.StatusSeeOther)
	return nil
}

func (h *authHandler) OnboardingPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("step") == "name" {
		ui.Render(w, r, pages.OnboardingName(""))
		return
	}
	ui.Render(w, r, pages.OnboardingWelcome())
}

func (h *authHandler) CompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth", http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		ui.Render(w, r, pages.OnboardingName("Please enter your name"))
		return
	}

	if err := h.authService.CompleteOnboarding(user.ID, name); err != nil {
		slog.Error("onboarding failed", "error", err, "user_id", user.ID)
		ui.Render(w, r, pages.OnboardingName("We couldn't finish setting you up. Please try again."))
		return
	}

	http.Redirect(w, r, "/app/dashboard", http.StatusSeeOther)
}

func (h *authHandler) JoinSpace(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	invite, err := h.inviteService.GetByToken(token)
	if err != nil {
		slog.Warn("invite lookup failed", "error", err, "token", token)
		ui.Render(w, r, pages.NotFound())
		return
	}

	if invite.Invitation.Status != model.InvitationStatusPending || time.Now().After(invite.Invitation.ExpiresAt) {
		ui.RenderError(w, r, "This invitation is no longer valid.", http.StatusGone)
		return
	}

	user := ctxkeys.User(r.Context())
	alreadyMember := false
	if user != nil {
		isMember, err := h.spaceService.IsMember(user.ID, invite.Invitation.SpaceID)
		if err != nil {
			slog.Error("failed to check membership", "error", err, "user_id", user.ID)
		}
		alreadyMember = isMember
	}

	ui.Render(w, r, pages.JoinSpaceConfirm(pages.JoinSpaceConfirmProps{
		Token:         token,
		SpaceName:     invite.SpaceName,
		InviterName:   invite.InviterName,
		InviteeEmail:  invite.Invitation.Email,
		IsAuthed:      user != nil,
		AlreadyMember: alreadyMember,
	}))
}

func (h *authHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	invite, err := h.inviteService.GetByToken(token)
	if err != nil {
		slog.Warn("invite lookup failed", "error", err, "token", token)
		ui.Render(w, r, pages.NotFound())
		return
	}

	if invite.Invitation.Status != model.InvitationStatusPending || time.Now().After(invite.Invitation.ExpiresAt) {
		ui.RenderError(w, r, "This invitation is no longer valid.", http.StatusGone)
		return
	}

	user := ctxkeys.User(r.Context())
	if user != nil {
		spaceID, err := h.inviteService.AcceptInvite(token, user.ID)
		if err != nil {
			slog.Error("failed to accept invite", "error", err, "token", token)
			ui.RenderError(w, r, "Failed to join space: "+err.Error(), http.StatusUnprocessableEntity)
			return
		}
		http.Redirect(w, r, "/app/spaces/"+spaceID+"/overview", http.StatusSeeOther)
		return
	}

	// Not logged in: set cookie and redirect to auth so they can log in or sign up
	http.SetCookie(w, &http.Cookie{
		Name:     "pending_invite",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(1 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/auth?invite=true", http.StatusSeeOther)
}
