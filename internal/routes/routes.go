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
	auth := handler.NewAuthHandler(a.AuthService)
	home := handler.NewHomeHandler()
	dashboard := handler.NewDashboardHandler()

	mux := http.NewServeMux()

	// ====================================================================================
	// PUBLIC ROUTES
	// ====================================================================================

	// Static
	sub, _ := fs.Sub(assets.AssetsFS, ".")
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(sub))))

	// Auth pages
	authRateLimiter := middleware.RateLimitAuth()

	mux.HandleFunc("GET /auth", middleware.RequireGuest(auth.AuthPage))
	mux.HandleFunc("GET /auth/password", middleware.RequireGuest(auth.PasswordPage))

	// Token Verifications
	mux.HandleFunc("GET /auth/magic-link/{token}", auth.VerifyMagicLink)

	// Auth Actions
	mux.HandleFunc("POST /auth/magic-link", authRateLimiter(middleware.RequireGuest(auth.SendMagicLink)))

	// ====================================================================================
	// PRIVATE ROUTES
	// ====================================================================================

	mux.HandleFunc("GET /auth/onboarding", middleware.RequireAuth(auth.OnboardingPage))
	mux.HandleFunc("POST /auth/onboarding", middleware.RequireAuth(auth.CompleteOnboarding))

	mux.HandleFunc("GET /app/dashboard", middleware.RequireAuth(dashboard.DashboardPage))

	// 404
	mux.HandleFunc("/{path...}", home.NotFoundPage)

	// Global middlewares
	handler := middleware.Chain(
		mux,
		middleware.Config(a.Cfg),
		middleware.RequestLogging,
		middleware.CSRFProtection,
		middleware.AuthMiddleware(a.AuthService, a.UserService, a.ProfileService),
		middleware.WithURLPath,
	)

	return handler
}
