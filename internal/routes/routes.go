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
	spaceH := handler.NewSpaceHandler(a.SpaceService, a.AccountService, a.TransactionService, a.CategoryService, a.AllocationService, a.InviteService, a.AuditLogService, a.TxAuditLogService, a.AccountActivitySvc, a.InvestmentService)
	allocationH := handler.NewAllocationHandler(a.AllocationService, a.AccountService)
	recurringH := handler.NewRecurringEventHandler(a.RecurringEventService, a.AccountService, a.SpaceService)
	investmentH := handler.NewInvestmentHandler(a.AccountService, a.SpaceService, a.InvestmentService)
	planH := handler.NewBudgetPlanHandler(a.BudgetPlanService, a.SpaceService)
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
		middleware.BlockPendingDeletion,
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
	r.Get("/account-deletion-status/{requestID}", settingsH.AccountDeletionStatusPage).Name("page.public.account-deletion-status")
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

	// Account pending deletion page — reachable while the deletion worker
	// finishes wiping the user's data.
	r.Get("/account-pending-deletion", settingsH.AccountPendingDeletionPage).Name("page.account.pending-deletion")

	// App routes
	r.Group("/app", func(g *router.Group) {
		g.Use(middleware.RequireAuth)

		g.Get("/home", spaceH.HomePage).Name("page.app.home")

		g.Get("/investments", investmentH.InvestmentsOverviewPage).Name("page.app.investments")

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
				g.Get("/activity", spaceH.SpaceActivityPage).Name("page.app.spaces.space.activity")
				g.Get("/members", spaceH.SpaceMembersPage).Name("page.app.spaces.space.members")
				g.Post("/members/invite", spaceH.HandleInviteMember).Name("action.app.spaces.space.members.invite")
				g.Post("/members/{userID}/remove", spaceH.HandleRemoveMember).Name("action.app.spaces.space.members.remove")
				g.Post("/invitations/{token}/cancel", spaceH.HandleCancelInvite).Name("action.app.spaces.space.invitations.cancel")

				g.Get("/accounts/create", spaceH.SpaceCreateAccountPage).Name("page.app.spaces.space.accounts.create")
				g.Post("/accounts/create", spaceH.HandleCreateAccount).Name("action.app.spaces.space.accounts.create")

				g.Get("/recurring", recurringH.ListPage).Name("page.app.spaces.space.recurring")
				g.Get("/recurring/create", recurringH.CreatePage).Name("page.app.spaces.space.recurring.create")
				g.Post("/recurring/create", recurringH.HandleCreate).Name("action.app.spaces.space.recurring.create")
				g.Get("/recurring/{eventID}/edit", recurringH.EditPage).Name("page.app.spaces.space.recurring.event.edit")
				g.Post("/recurring/{eventID}/edit", recurringH.HandleEdit).Name("action.app.spaces.space.recurring.event.edit")
				g.Post("/recurring/{eventID}/delete", recurringH.HandleDelete).Name("action.app.spaces.space.recurring.event.delete")
				g.Post("/recurring/{eventID}/pause", recurringH.HandlePause).Name("action.app.spaces.space.recurring.event.pause")
				g.Post("/recurring/{eventID}/resume", recurringH.HandleResume).Name("action.app.spaces.space.recurring.event.resume")

				g.Get("/plans", planH.ListPage).Name("page.app.spaces.space.plans")
				g.Post("/plans", planH.HandleCreate).Name("action.app.spaces.space.plans.create")
				g.Get("/plans/{planID}", planH.EditorPage).Name("page.app.spaces.space.plans.plan")
				g.Post("/plans/{planID}/rename", planH.HandleRename).Name("action.app.spaces.space.plans.plan.rename")
				g.Post("/plans/{planID}/delete", planH.HandleDelete).Name("action.app.spaces.space.plans.plan.delete")
				g.Post("/plans/{planID}/lines", planH.HandleAddLine).Name("action.app.spaces.space.plans.plan.lines.create")
				g.Post("/plans/{planID}/lines/{lineID}", planH.HandleUpdateLine).Name("action.app.spaces.space.plans.plan.lines.line.update")
				g.Post("/plans/{planID}/lines/{lineID}/delete", planH.HandleDeleteLine).Name("action.app.spaces.space.plans.plan.lines.line.delete")

				g.SubGroup("/accounts/{accountID}", func(g *router.Group) {
					g.Get("/overview", spaceH.SpaceAccountPage).Name("page.app.spaces.space.accounts.account.overview")
					g.Get("/activity", spaceH.SpaceAccountActivityPage).Name("page.app.spaces.space.accounts.account.activity")
					g.Get("/transactions", spaceH.SpaceAccountTransactionsPage).Name("page.app.spaces.space.accounts.account.transactions")
					g.Get("/transactions/{transactionID}", spaceH.SpaceTransactionPage).Name("page.app.spaces.space.accounts.account.transactions.transaction")
					g.Get("/transactions/{transactionID}/edit", spaceH.SpaceEditTransactionPage).Name("page.app.spaces.space.accounts.account.transactions.transaction.edit")
					g.Post("/transactions/{transactionID}/edit", spaceH.HandleEditTransaction).Name("action.app.spaces.space.accounts.account.transactions.transaction.edit")
					g.Post("/transactions/{transactionID}/delete", spaceH.HandleDeleteTransaction).Name("action.app.spaces.space.accounts.account.transactions.transaction.delete")
					g.Get("/transactions/{transactionID}/activity", spaceH.SpaceTransactionActivityPage).Name("page.app.spaces.space.accounts.account.transactions.transaction.activity")
					g.Get("/settings", spaceH.SpaceAccountSettingsPage).Name("page.app.spaces.space.accounts.account.settings")
					g.Post("/settings/rename", spaceH.HandleRenameAccount).Name("action.app.spaces.space.accounts.account.settings.rename")
					g.Post("/settings/currency", spaceH.HandleChangeAccountCurrency).Name("action.app.spaces.space.accounts.account.settings.currency")
					g.Post("/settings/delete", spaceH.HandleDeleteAccount).Name("action.app.spaces.space.accounts.account.settings.delete")
					g.Post("/settings/investment", spaceH.HandleSetInvestmentFlag).Name("action.app.spaces.space.accounts.account.settings.investment")
					g.Get("/bills/create", spaceH.SpaceCreateBillPage).Name("page.app.spaces.space.accounts.account.bills.create")
					g.Post("/bills/create", spaceH.HandleCreateBill).Name("action.app.spaces.space.accounts.account.bills.create")
					g.Get("/deposits/create", spaceH.SpaceCreateDepositPage).Name("page.app.spaces.space.accounts.account.deposits.create")
					g.Post("/deposits/create", spaceH.HandleCreateDeposit).Name("action.app.spaces.space.accounts.account.deposits.create")
					g.Get("/transfers/create", spaceH.SpaceCreateTransferPage).Name("page.app.spaces.space.accounts.account.transfers.create")
					g.Post("/transfers/create", spaceH.HandleCreateTransfer).Name("action.app.spaces.space.accounts.account.transfers.create")

					g.Get("/categories", spaceH.SpaceCategoriesPage).Name("page.app.spaces.space.accounts.account.categories")
					g.Post("/categories", spaceH.HandleCreateCategory).Name("action.app.spaces.space.accounts.account.categories.create")
					g.Post("/categories/{categoryID}/delete", spaceH.HandleDeleteCategory).Name("action.app.spaces.space.accounts.account.categories.delete")

					g.Get("/reports", spaceH.SpaceReportsPage).Name("page.app.spaces.space.accounts.account.reports")

					g.Post("/allocations/create", allocationH.HandleCreate).Name("action.app.spaces.space.accounts.account.allocations.create")
					g.Post("/allocations/{allocationID}/edit", allocationH.HandleEdit).Name("action.app.spaces.space.accounts.account.allocations.allocation.edit")
					g.Post("/allocations/{allocationID}/delete", allocationH.HandleDelete).Name("action.app.spaces.space.accounts.account.allocations.allocation.delete")

					g.Post("/investments/contribution-room", investmentH.HandleSetContributionRoom).Name("action.app.spaces.space.accounts.account.investments.contribution-room")
					g.Get("/investments/holdings/create", investmentH.CreateHoldingPage).Name("page.app.spaces.space.accounts.account.investments.holdings.create")
					g.Post("/investments/holdings/create", investmentH.HandleCreateHolding).Name("action.app.spaces.space.accounts.account.investments.holdings.create")
					g.Get("/investments/holdings/{holdingID}", investmentH.HoldingDetailPage).Name("page.app.spaces.space.accounts.account.investments.holdings.holding")
					g.Post("/investments/holdings/{holdingID}/delete", investmentH.HandleDeleteHolding).Name("action.app.spaces.space.accounts.account.investments.holdings.holding.delete")
					g.Post("/investments/holdings/{holdingID}/trades/create", investmentH.HandleCreateTrade).Name("action.app.spaces.space.accounts.account.investments.holdings.holding.trades.create")
					g.Post("/investments/holdings/{holdingID}/trades/{tradeID}/delete", investmentH.HandleDeleteTrade).Name("action.app.spaces.space.accounts.account.investments.holdings.holding.trades.trade.delete")
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
				g.Post("/delete-account", settingsH.DeleteAccount).Name("action.app.settings.account.delete")
			})
		})
	})

	// 404 catch-all
	r.Get("/{path...}", homeH.NotFoundPage)

	return r.Handler()
}
