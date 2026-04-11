package routes

import (
	"io/fs"
	"net/http"
	"time"

	"git.juancwu.dev/juancwu/budgit/assets"
	"git.juancwu.dev/juancwu/budgit/internal/app"
	"git.juancwu.dev/juancwu/budgit/internal/handler"
	"git.juancwu.dev/juancwu/budgit/internal/middleware"
	"git.juancwu.dev/juancwu/budgit/internal/router"
)

func SetupRoutes(a *app.App) http.Handler {
	authH := handler.NewAuthHandler(a.AuthService, a.InviteService, a.SpaceService)
	homeH := handler.NewHomeHandler()
	settingsH := handler.NewSettingsHandler(a.AuthService, a.UserService)
	spaceH := handler.NewSpaceHandler(a.SpaceService)
	redirectH := handler.NewRedirectHandler()

	r := router.New()

	// Global middleware
	r.Use(
		middleware.SecurityHeaders,
		middleware.Config(a.Cfg),
		middleware.RequestLogging,
		middleware.NoCacheDynamic,
		middleware.CSRFProtection,
		middleware.AuthMiddleware(a.AuthService, a.UserService),
		middleware.WithURLPath,
	)

	// Static assets (bypass router groups — registered directly on mux)
	var assetsFS http.FileSystem
	if a.Cfg.IsProduction() {
		sub, _ := fs.Sub(assets.AssetsFS, ".")
		assetsFS = http.FS(sub)
	} else {
		assetsFS = http.Dir("./assets")
	}
	r.Mux().Handle("GET /assets/",
		middleware.CacheStatic(http.StripPrefix("/assets/", http.FileServer(assetsFS))),
	)

	// Public pages
	r.Get("/{$}", homeH.HomePage)
	r.Get("/forbidden", homeH.ForbiddenPage)
	r.Get("/privacy", homeH.PrivacyPage)
	r.Get("/terms", homeH.TermsPage)
	r.Get("/join/{token}", authH.JoinSpace)

	// Permanent redirects
	r.Get("/app/dashboard", redirectH.Spaces)

	// Auth - guest routes
	r.Group("/auth", func(g *router.Group) {
		g.Use(middleware.RequireGuest)
		g.Get("", authH.AuthPage)
		g.Get("/password", authH.PasswordPage)
		g.Get("/magic-link/{token}", authH.VerifyMagicLink)

		g.SubGroup("", func(g *router.Group) {
			g.RateLimit(5, 15*time.Minute)
			g.Post("/magic-link", authH.SendMagicLink)
			g.Post("/password", authH.LoginWithPassword)
		})
	})

	// Auth - authenticated routes
	r.Group("/auth", func(g *router.Group) {
		g.Use(middleware.RequireAuth)
		g.Get("/onboarding", authH.OnboardingPage)
		g.Post("/onboarding", authH.CompleteOnboarding)
	})
	r.Post("/auth/logout", authH.Logout)

	// App routes
	r.Group("/app", func(g *router.Group) {
		g.Use(middleware.RequireAuth)

		g.SubGroup("/spaces", func(g *router.Group) {
			g.Get("", spaceH.SpacesPage)
		})

		g.SubGroup("/settings", func(g *router.Group) {
			g.Get("", settingsH.SettingsPage)

			g.SubGroup("", func(g *router.Group) {
				g.RateLimit(5, 15*time.Minute)
				g.Post("/password", settingsH.SetPassword)
			})
		})
	})

	// 404 catch-all
	r.Get("/{path...}", homeH.NotFoundPage)

	return r.Handler()
}
