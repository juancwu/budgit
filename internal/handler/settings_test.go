package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func newTestSettingsHandler(dbi testutil.DBInfo) (*settingsHandler, *service.AuthService) {
	cfg := testutil.TestConfig()
	userRepo := repository.NewUserRepository(dbi.DB)
	profileRepo := repository.NewProfileRepository(dbi.DB)
	tokenRepo := repository.NewTokenRepository(dbi.DB)
	spaceRepo := repository.NewSpaceRepository(dbi.DB)
	spaceSvc := service.NewSpaceService(spaceRepo)
	emailSvc := service.NewEmailService(nil, "test@example.com", "http://localhost:9999", "Budgit Test", false)
	authSvc := service.NewAuthService(emailSvc, userRepo, profileRepo, tokenRepo, spaceSvc, cfg.JWTSecret, cfg.JWTExpiry, cfg.TokenMagicLinkExpiry, false)
	userSvc := service.NewUserService(userRepo)
	return NewSettingsHandler(authSvc, userSvc), authSvc
}

func TestSettingsHandler_SettingsPage(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h, _ := newTestSettingsHandler(dbi)

		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test User")

		req := testutil.NewAuthenticatedRequest(t, http.MethodGet, "/app/settings", user, profile, nil)
		w := httptest.NewRecorder()
		h.SettingsPage(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSettingsHandler_SetPassword(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h, _ := newTestSettingsHandler(dbi)

		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test User")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/settings/password", user, profile, url.Values{
			"new_password":     {"testpassword1"},
			"confirm_password": {"testpassword1"},
		})
		w := httptest.NewRecorder()
		h.SetPassword(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSettingsHandler_SetPassword_Mismatch(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		h, _ := newTestSettingsHandler(dbi)

		user, profile := testutil.CreateTestUserWithProfile(t, dbi.DB, "test@example.com", "Test User")

		req := testutil.NewAuthenticatedRequest(t, http.MethodPost, "/app/settings/password", user, profile, url.Values{
			"new_password":     {"testpassword1"},
			"confirm_password": {"differentpassword"},
		})
		w := httptest.NewRecorder()
		h.SetPassword(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
