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
	"git.juancwu.dev/juancwu/budgit/internal/routeurl"
)

func SetupRoutes(a *app.App) http.Handler {
	routeurl.Reset()

	authH := handler.NewAuthHandler(a.AuthService, a.InviteService, a.SpaceService)
	homeH := handler.NewHomeHandler()
	settingsH := handler.NewSettingsHandler(a.AuthService, a.UserService)
	spaceH := handler.NewSpaceHandler(a.SpaceService, a.AccountService, a.TransactionService, a.InviteService)
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
		middleware.WithSidebarState,
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
	r.Get("/{$}", homeH.HomePage).Name("page.public.home")
	r.Get("/forbidden", homeH.ForbiddenPage).Name("page.public.forbidden")
	r.Get("/privacy", homeH.PrivacyPage).Name("page.public.privacy")
	r.Get("/terms", homeH.TermsPage).Name("page.public.terms")
	r.Get("/join/{token}", authH.JoinSpace).Name("page.public.join-space")
	r.Post("/join/{token}/accept", authH.AcceptInvite).Name("action.public.join-space.accept")

	// Permanent redirects
	r.Get("/app/dashboard", redirectH.Spaces)

	// Auth - guest routes
	r.Group("/auth", func(g *router.Group) {
		g.Use(middleware.RequireGuest)
		g.Get("", authH.AuthPage).Name("page.auth.index")
		g.Get("/password", authH.PasswordPage).Name("page.auth.password")
		g.Get("/magic-link/{token}", authH.VerifyMagicLink).Name("page.auth.magic-link.verify")

		g.SubGroup("", func(g *router.Group) {
			g.RateLimit(5, 15*time.Minute)
			g.Post("/magic-link", authH.SendMagicLink).Name("action.auth.magic-link.send")
			g.Post("/password", authH.LoginWithPassword).Name("action.auth.password.login")
		})
	})

	// Auth - authenticated routes
	r.Group("/auth", func(g *router.Group) {
		g.Use(middleware.RequireAuth)
		g.Get("/onboarding", authH.OnboardingPage).Name("page.auth.onboarding")
		g.Post("/onboarding", authH.CompleteOnboarding).Name("action.auth.onboarding.complete")
	})
	r.Post("/auth/logout", authH.Logout).Name("action.auth.logout")

	// App routes
	r.Group("/app", func(g *router.Group) {
		g.Use(middleware.RequireAuth)

		g.SubGroup("/spaces", func(g *router.Group) {
			g.Get("", spaceH.SpacesPage).Name("page.app.spaces")
			g.Get("/create", spaceH.CreateSpacePage).Name("page.app.spaces.create")
			g.Post("/create", spaceH.HandleCreateSpace).Name("action.app.spaces.create")
			g.SubGroup("/{spaceID}", func(g *router.Group) {
				spaceAccessMw := middleware.RequireSpaceAccess(a.SpaceService)
				g.Use(spaceAccessMw)
				g.Get("/overview", spaceH.SpaceOverviewPage).Name("page.app.spaces.space.overview")
				g.Get("/settings", spaceH.SpaceSettingsPage).Name("page.app.spaces.space.settings")
				g.Post("/settings/rename", spaceH.HandleRenameSpace).Name("action.app.spaces.space.settings.rename")
				g.Post("/settings/delete", spaceH.HandleDeleteSpace).Name("action.app.spaces.space.settings.delete")
				g.Get("/members", spaceH.SpaceMembersPage).Name("page.app.spaces.space.members")
				g.Post("/members/invite", spaceH.HandleInviteMember).Name("action.app.spaces.space.members.invite")
				g.Post("/members/{userID}/remove", spaceH.HandleRemoveMember).Name("action.app.spaces.space.members.remove")
				g.Post("/invitations/{token}/cancel", spaceH.HandleCancelInvite).Name("action.app.spaces.space.invitations.cancel")
				g.Get("/accounts/create", spaceH.SpaceCreateAccountPage).Name("page.app.spaces.space.accounts.create")
				g.Post("/accounts/create", spaceH.HandleCreateAccount).Name("action.app.spaces.space.accounts.create")

				g.SubGroup("/accounts/{accountID}", func(g *router.Group) {
					g.Get("/overview", spaceH.SpaceAccountPage).Name("page.app.spaces.space.accounts.account.overview")
					g.Get("/transactions", spaceH.SpaceAccountTransactionsPage).Name("page.app.spaces.space.accounts.account.transactions")
					g.Get("/settings", spaceH.SpaceAccountSettingsPage).Name("page.app.spaces.space.accounts.account.settings")
					g.Post("/settings/rename", spaceH.HandleRenameAccount).Name("action.app.spaces.space.accounts.account.settings.rename")
					g.Post("/settings/delete", spaceH.HandleDeleteAccount).Name("action.app.spaces.space.accounts.account.settings.delete")
					g.Get("/bills/create", spaceH.SpaceCreateBillPage).Name("page.app.spaces.space.accounts.account.bills.create")
					g.Post("/bills/create", spaceH.HandleCreateBill).Name("action.app.spaces.space.accounts.account.bills.create")
					g.Get("/deposits/create", spaceH.SpaceCreateDepositPage).Name("page.app.spaces.space.accounts.account.deposits.create")
					g.Post("/deposits/create", spaceH.HandleCreateDeposit).Name("action.app.spaces.space.accounts.account.deposits.create")
				})
			})
		})

		g.SubGroup("/shared-with-me", func(g *router.Group) {
			g.Get("", spaceH.SharedSpacesPage).Name("page.app.shared-with-me")
		})

		g.SubGroup("/settings", func(g *router.Group) {
			g.Get("", settingsH.SettingsPage).Name("page.app.settings")

			g.SubGroup("", func(g *router.Group) {
				g.RateLimit(5, 15*time.Minute)
				g.Post("/password", settingsH.SetPassword).Name("action.app.settings.password.set")
			})
		})
	})

	// 404 catch-all
	r.Get("/{path...}", homeH.NotFoundPage)

	return r.Handler()
}
