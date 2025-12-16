package routes

import (
	"io/fs"
	"net/http"

	"git.juancwu.dev/juancwu/budgething/assets"
	"git.juancwu.dev/juancwu/budgething/internal/app"
	"git.juancwu.dev/juancwu/budgething/internal/handler"
	"git.juancwu.dev/juancwu/budgething/internal/middleware"
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
