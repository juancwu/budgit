package routes

import (
	"io/fs"
	"net/http"

	"git.juancwu.dev/juancwu/budgit/assets"
	"git.juancwu.dev/juancwu/budgit/internal/app"
	"git.juancwu.dev/juancwu/budgit/internal/handler"
	"git.juancwu.dev/juancwu/budgit/internal/middleware"
)

func SetupRoutes(a *app.App) http.Handler {
	auth := handler.NewAuthHandler(a.AuthService, a.InviteService, a.SpaceService)
	home := handler.NewHomeHandler()
	settings := handler.NewSettingsHandler(a.AuthService, a.UserService)

	mux := http.NewServeMux()

	// ====================================================================================
	// PUBLIC ROUTES
	// ====================================================================================

	// Static assets with long-lived cache (cache-busted via ?v=<timestamp>)
	sub, _ := fs.Sub(assets.AssetsFS, ".")
	mux.Handle("GET /assets/", middleware.CacheStatic(http.StripPrefix("/assets/", http.FileServer(http.FS(sub)))))

	// Home
	mux.HandleFunc("GET /{$}", home.HomePage)
	mux.HandleFunc("GET /forbidden", home.ForbiddenPage)
	mux.HandleFunc("GET /privacy", home.PrivacyPage)
	mux.HandleFunc("GET /terms", home.TermsPage)

	// Auth pages
	authRateLimiter := middleware.RateLimitAuth()

	mux.HandleFunc("GET /auth", middleware.RequireGuest(auth.AuthPage))
	mux.HandleFunc("GET /auth/password", middleware.RequireGuest(auth.PasswordPage))

	// Token Verifications
	mux.HandleFunc("GET /auth/magic-link/{token}", auth.VerifyMagicLink)

	// Auth Actions
	mux.HandleFunc("POST /auth/magic-link", authRateLimiter(middleware.RequireGuest(auth.SendMagicLink)))
	mux.HandleFunc("POST /auth/password", authRateLimiter(middleware.RequireGuest(auth.LoginWithPassword)))
	mux.HandleFunc("POST /auth/logout", auth.Logout)

	// Join via invite
	mux.HandleFunc("GET /join/{token}", auth.JoinSpace)

	// ====================================================================================
	// PRIVATE ROUTES
	// ====================================================================================

	crudLimiter := middleware.RateLimitCRUD()

	mux.HandleFunc("GET /auth/onboarding", middleware.RequireAuth(auth.OnboardingPage))
	mux.Handle("POST /auth/onboarding", crudLimiter(http.HandlerFunc(middleware.RequireAuth(auth.CompleteOnboarding))))

	mux.HandleFunc("GET /app/settings", middleware.RequireAuth(settings.SettingsPage))
	mux.HandleFunc("POST /app/settings/password", authRateLimiter(middleware.RequireAuth(settings.SetPassword)))

	// 404
	mux.HandleFunc("/{path...}", home.NotFoundPage)

	// Global middlewares
	handler := middleware.Chain(
		mux,
		middleware.SecurityHeaders(),
		middleware.Config(a.Cfg),
		middleware.RequestLogging,
		middleware.NoCacheDynamic,
		middleware.CSRFProtection,
		middleware.AuthMiddleware(a.AuthService, a.UserService),
		middleware.WithURLPath,
	)

	return handler
}
