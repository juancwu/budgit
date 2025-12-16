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
	auth := handler.NewAuthHandler()
	home := handler.NewHomeHandler()

	mux := http.NewServeMux()

	// ====================================================================================
	// PUBLIC ROUTES
	// ====================================================================================

	// Static
	sub, _ := fs.Sub(assets.AssetsFS, ".")
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(sub))))

	// Auth pages
	mux.HandleFunc("GET /auth", middleware.RequireGuest(auth.AuthPage))

	// 404
	mux.HandleFunc("/{path...}", home.NotFoundPage)

	// Global middlewares
	handler := middleware.Chain(
		mux,
		middleware.Config(a.Cfg),
		middleware.RequestLogging,
		middleware.CSRFProtection,
		middleware.WithURLPath,
	)

	return handler
}
