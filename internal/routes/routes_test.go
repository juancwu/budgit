package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/app"
	"git.juancwu.dev/juancwu/budgit/internal/model"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/routeurl"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func newTestApp(dbi testutil.DBInfo) *app.App {
	cfg := testutil.TestConfig()
	cfg.DBDriver = dbi.Driver

	userRepo := repository.NewUserRepository(dbi.DB)
	tokenRepo := repository.NewTokenRepository(dbi.DB)
	spaceRepo := repository.NewSpaceRepository(dbi.DB)
	accountRepo := repository.NewAccountRepository(dbi.DB)
	inviteRepo := repository.NewInvitationRepository(dbi.DB)

	spaceSvc := service.NewSpaceService(spaceRepo)
	accountSvc := service.NewAccountService(accountRepo)
	emailSvc := service.NewEmailService(nil, "test@example.com", "http://localhost:9999", "Budgit Test", false)
	authSvc := service.NewAuthService(emailSvc, userRepo, tokenRepo, spaceSvc, accountSvc, cfg.JWTSecret, cfg.JWTExpiry, cfg.TokenMagicLinkExpiry, false)
	userSvc := service.NewUserService(userRepo)
	inviteSvc := service.NewInviteService(inviteRepo, spaceRepo, userRepo, emailSvc, nil)

	return &app.App{
		Cfg:            cfg,
		DB:             dbi.DB,
		UserService:    userSvc,
		AuthService:    authSvc,
		EmailService:   emailSvc,
		SpaceService:   spaceSvc,
		AccountService: accountSvc,
		InviteService:  inviteSvc,
	}
}

// authCookie generates a valid JWT cookie for the given user.
func authCookie(a *app.App, user *model.User) *http.Cookie {
	token, err := a.AuthService.GenerateJWT(user)
	if err != nil {
		panic("failed to generate test JWT: " + err.Error())
	}
	return &http.Cookie{
		Name:  "auth_token",
		Value: token,
	}
}

func TestSetupRoutes_PublicRoutes(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		a := newTestApp(dbi)
		handler := SetupRoutes(a)

		routes := []string{"/forbidden", "/privacy", "/terms"}

		for _, path := range routes {
			t.Run(path, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
				assert.Equal(t, http.StatusOK, w.Code)
			})
		}
	})
}

func TestSetupRoutes_HomeRedirects(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		a := newTestApp(dbi)
		handler := SetupRoutes(a)

		// Unauthenticated → redirect to /auth
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/auth", w.Header().Get("Location"))

		// Authenticated → redirect to /app/spaces
		name := "Test User"
		user := testutil.CreateTestUserWithName(t, dbi.DB, "home@example.com", &name)
		req = httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(authCookie(a, user))
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/app/spaces", w.Header().Get("Location"))
	})
}

func TestSetupRoutes_GuestRoutes(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		a := newTestApp(dbi)
		handler := SetupRoutes(a)

		routes := []string{"/auth", "/auth/password"}

		for _, path := range routes {
			t.Run(path, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
				assert.Equal(t, http.StatusOK, w.Code)
			})
		}
	})
}

func TestSetupRoutes_GuestRoutes_RedirectAuthenticated(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		a := newTestApp(dbi)
		handler := SetupRoutes(a)

		name := "Test User"
		user := testutil.CreateTestUserWithName(t, dbi.DB, "auth@example.com", &name)

		req := httptest.NewRequest(http.MethodGet, "/auth", nil)
		req.AddCookie(authCookie(a, user))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/app/dashboard", w.Header().Get("Location"))
	})
}

func TestSetupRoutes_AuthRequired_RedirectUnauthenticated(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		a := newTestApp(dbi)
		handler := SetupRoutes(a)

		routes := []string{"/app/spaces", "/app/settings"}

		for _, path := range routes {
			t.Run(path, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
				assert.Equal(t, http.StatusSeeOther, w.Code)
				assert.Equal(t, "/auth", w.Header().Get("Location"))
			})
		}
	})
}

func TestSetupRoutes_AuthRequired_Authenticated(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		a := newTestApp(dbi)
		handler := SetupRoutes(a)

		name := "Test User"
		user := testutil.CreateTestUserWithName(t, dbi.DB, "appuser@example.com", &name)

		routes := []string{"/app/spaces", "/app/settings"}

		for _, path := range routes {
			t.Run(path, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.AddCookie(authCookie(a, user))
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
				assert.Equal(t, http.StatusOK, w.Code)
			})
		}
	})
}

func TestSetupRoutes_OnboardingRedirect(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		a := newTestApp(dbi)
		handler := SetupRoutes(a)

		// User without name → needs onboarding
		user := testutil.CreateTestUser(t, dbi.DB, "noname@example.com", nil)

		req := httptest.NewRequest(http.MethodGet, "/app/spaces", nil)
		req.AddCookie(authCookie(a, user))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Equal(t, "/auth/onboarding", w.Header().Get("Location"))
	})
}

func TestSetupRoutes_PermanentRedirect(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		a := newTestApp(dbi)
		handler := SetupRoutes(a)

		req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMovedPermanently, w.Code)
		assert.Equal(t, "/app/spaces", w.Header().Get("Location"))
	})
}

func TestSetupRoutes_NotFound(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		a := newTestApp(dbi)
		handler := SetupRoutes(a)

		req := httptest.NewRequest(http.MethodGet, "/nonexistent/page", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// The catch-all route renders the NotFound page (handler returns 200).
		// This test verifies the catch-all route is registered and handles unknown paths.
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestURL_ResolvesNamedRoute(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		a := newTestApp(dbi)
		SetupRoutes(a)

		assert.Equal(t, "/privacy", routeurl.URL("page.public.privacy"))
		assert.Equal(t, "/app/spaces", routeurl.URL("page.app.spaces"))
		assert.Equal(t, "/join/abc123", routeurl.URL("page.public.join-space", "token", "abc123"))
		assert.Equal(t, "#", routeurl.URL("does.not.exist"))
	})
}

func TestSetupRoutes_StaticAssets(t *testing.T) {
	testutil.ForEachDB(t, func(t *testing.T, dbi testutil.DBInfo) {
		a := newTestApp(dbi)
		// Force the embedded-FS branch so the test is independent of CWD;
		// in dev we serve from ./assets on disk (see SetupRoutes).
		a.Cfg.AppEnv = "production"
		handler := SetupRoutes(a)

		req := httptest.NewRequest(http.MethodGet, "/assets/css/output.css", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// Static asset should be served (200) or at minimum not 404 via the catch-all
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Cache-Control"), "public")
	})
}
