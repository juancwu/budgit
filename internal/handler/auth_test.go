package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/ctxkeys"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func newTestAuthHandler(dbi testutil.DBInfo) *authHandler {
	cfg := testutil.TestConfig()
	userRepo := repository.NewUserRepository(dbi.DB)
	profileRepo := repository.NewProfileRepository(dbi.DB)
	tokenRepo := repository.NewTokenRepository(dbi.DB)
	spaceRepo := repository.NewSpaceRepository(dbi.DB)
	inviteRepo := repository.NewInvitationRepository(dbi.DB)
	spaceSvc := service.NewSpaceService(spaceRepo)
	emailSvc := service.NewEmailService(nil, "test@example.com", "http://localhost:9999", "Budgit Test", false)
	authSvc := service.NewAuthService(emailSvc, userRepo, profileRepo, tokenRepo, spaceSvc, cfg.JWTSecret, cfg.JWTExpiry, cfg.TokenMagicLinkExpiry, false)
	inviteSvc := service.NewInviteService(inviteRepo, spaceRepo, userRepo, emailSvc)
	return NewAuthHandler(authSvc, inviteSvc, spaceSvc)
}

func guestContext() context.Context {
	ctx := context.Background()
	ctx = ctxkeys.WithConfig(ctx, testutil.TestConfig().Sanitized())
	ctx = ctxkeys.WithCSRFToken(ctx, "test")
	ctx = ctxkeys.WithAppVersion(ctx, "test")
	return ctx
}

func TestAuthHandler_AuthPage(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h := newTestAuthHandler(dbi)

		req := httptest.NewRequest(http.MethodGet, "/auth", nil)
		req = req.WithContext(guestContext())

		w := httptest.NewRecorder()
		h.AuthPage(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthHandler_SendMagicLink(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h := newTestAuthHandler(dbi)

		form := url.Values{"email": {"newuser@example.com"}}
		req := httptest.NewRequest(http.MethodPost, "/auth/magic-link", nil)
		req = req.WithContext(guestContext())
		req.PostForm = form

		w := httptest.NewRecorder()
		h.SendMagicLink(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthHandler_SendMagicLink_EmptyEmail(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h := newTestAuthHandler(dbi)

		form := url.Values{"email": {""}}
		req := httptest.NewRequest(http.MethodPost, "/auth/magic-link", nil)
		req = req.WithContext(guestContext())
		req.PostForm = form

		w := httptest.NewRecorder()
		h.SendMagicLink(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h := newTestAuthHandler(dbi)

		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test User")
		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/auth/logout", user, profile, nil)

		w := httptest.NewRecorder()
		h.Logout(w, req)

		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/", w.Header().Get("Location"))
	})
}

func TestAuthHandler_CompleteOnboarding_Step2(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h := newTestAuthHandler(dbi)

		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/auth/onboarding", user, profile, url.Values{
			"step": {"2"},
			"name": {"John"},
		})

		w := httptest.NewRecorder()
		h.CompleteOnboarding(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthHandler_CompleteOnboarding_Step3(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h := newTestAuthHandler(dbi)

		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/auth/onboarding", user, profile, url.Values{
			"step":       {"3"},
			"name":       {"John"},
			"space_name": {"My Space"},
		})

		w := httptest.NewRecorder()
		h.CompleteOnboarding(w, req)

		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/app/dashboard", w.Header().Get("Location"))
	})
}
