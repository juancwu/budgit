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
	auth := handler.NewAuthHandler(a.AuthService, a.InviteService)
	home := handler.NewHomeHandler()
	dashboard := handler.NewDashboardHandler(a.SpaceService, a.ExpenseService)
	settings := handler.NewSettingsHandler(a.AuthService, a.UserService)
	space := handler.NewSpaceHandler(a.SpaceService, a.TagService, a.ShoppingListService, a.ExpenseService, a.InviteService, a.EventBus)

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

	// ====================================================================================
	// PRIVATE ROUTES
	// ====================================================================================

	mux.HandleFunc("GET /auth/onboarding", middleware.RequireAuth(auth.OnboardingPage))
	mux.HandleFunc("POST /auth/onboarding", authRateLimiter(middleware.RequireAuth(auth.CompleteOnboarding)))

	mux.HandleFunc("GET /app/dashboard", middleware.RequireAuth(dashboard.DashboardPage))
	mux.HandleFunc("POST /app/spaces", middleware.RequireAuth(dashboard.CreateSpace))
	mux.HandleFunc("GET /app/settings", middleware.RequireAuth(settings.SettingsPage))
	mux.HandleFunc("POST /app/settings/password", authRateLimiter(middleware.RequireAuth(settings.SetPassword)))

	// Space routes
	spaceDashboardHandler := middleware.RequireAuth(space.DashboardPage)
	spaceDashboardWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(spaceDashboardHandler)
	mux.Handle("GET /app/spaces/{spaceID}", spaceDashboardWithAccess)

	// SSE
	streamHandler := middleware.RequireAuth(space.StreamEvents)
	streamWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(streamHandler)
	mux.Handle("GET /app/spaces/{spaceID}/stream", streamWithAccess)

	listsPageHandler := middleware.RequireAuth(space.ListsPage)
	listsPageWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(listsPageHandler)
	mux.Handle("GET /app/spaces/{spaceID}/lists", listsPageWithAccess)

	createListHandler := middleware.RequireAuth(space.CreateList)
	createListWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(createListHandler)
	mux.Handle("POST /app/spaces/{spaceID}/lists", createListWithAccess)

	listPageHandler := middleware.RequireAuth(space.ListPage)
	listPageWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(listPageHandler)
	mux.Handle("GET /app/spaces/{spaceID}/lists/{listID}", listPageWithAccess)

	updateListHandler := middleware.RequireAuth(space.UpdateList)
	updateListWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(updateListHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/lists/{listID}", updateListWithAccess)

	deleteListHandler := middleware.RequireAuth(space.DeleteList)
	deleteListWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(deleteListHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/lists/{listID}", deleteListWithAccess)

	addItemHandler := middleware.RequireAuth(space.AddItemToList)
	addItemWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(addItemHandler)
	mux.Handle("POST /app/spaces/{spaceID}/lists/{listID}/items", addItemWithAccess)

	toggleItemHandler := middleware.RequireAuth(space.ToggleItem)
	toggleItemWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(toggleItemHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/lists/{listID}/items/{itemID}", toggleItemWithAccess)

	deleteItemHandler := middleware.RequireAuth(space.DeleteItem)
	deleteItemWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(deleteItemHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/lists/{listID}/items/{itemID}", deleteItemWithAccess)

	// Tag routes
	tagsPageHandler := middleware.RequireAuth(space.TagsPage)
	tagsPageWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(tagsPageHandler)
	mux.Handle("GET /app/spaces/{spaceID}/tags", tagsPageWithAccess)

	createTagHandler := middleware.RequireAuth(space.CreateTag)
	createTagWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(createTagHandler)
	mux.Handle("POST /app/spaces/{spaceID}/tags", createTagWithAccess)

	deleteTagHandler := middleware.RequireAuth(space.DeleteTag)
	deleteTagWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(deleteTagHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/tags/{tagID}", deleteTagWithAccess)

	// Expense routes
	expensesPageHandler := middleware.RequireAuth(space.ExpensesPage)
	expensesPageWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(expensesPageHandler)
	mux.Handle("GET /app/spaces/{spaceID}/expenses", expensesPageWithAccess)

	createExpenseHandler := middleware.RequireAuth(space.CreateExpense)
	createExpenseWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(createExpenseHandler)
	mux.Handle("POST /app/spaces/{spaceID}/expenses", createExpenseWithAccess)

	updateExpenseHandler := middleware.RequireAuth(space.UpdateExpense)
	updateExpenseWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(updateExpenseHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/expenses/{expenseID}", updateExpenseWithAccess)

	deleteExpenseHandler := middleware.RequireAuth(space.DeleteExpense)
	deleteExpenseWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(deleteExpenseHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/expenses/{expenseID}", deleteExpenseWithAccess)

	// Component routes (HTMX updates)
	balanceCardHandler := middleware.RequireAuth(space.GetBalanceCard)
	balanceCardWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(balanceCardHandler)
	mux.Handle("GET /app/spaces/{spaceID}/components/balance", balanceCardWithAccess)

	expensesListHandler := middleware.RequireAuth(space.GetExpensesList)
	expensesListWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(expensesListHandler)
	mux.Handle("GET /app/spaces/{spaceID}/components/expenses", expensesListWithAccess)

	shoppingListItemsHandler := middleware.RequireAuth(space.GetShoppingListItems)
	shoppingListItemsWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(shoppingListItemsHandler)
	mux.Handle("GET /app/spaces/{spaceID}/lists/{listID}/items", shoppingListItemsWithAccess)

	cardItemsHandler := middleware.RequireAuth(space.GetListCardItems)
	cardItemsWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(cardItemsHandler)
	mux.Handle("GET /app/spaces/{spaceID}/lists/{listID}/card-items", cardItemsWithAccess)

	listsComponentHandler := middleware.RequireAuth(space.GetLists)
	listsComponentWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(listsComponentHandler)
	mux.Handle("GET /app/spaces/{spaceID}/components/lists", listsComponentWithAccess)

	// Invite routes
	createInviteHandler := middleware.RequireAuth(space.CreateInvite)
	createInviteWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(createInviteHandler)
	mux.Handle("POST /app/spaces/{spaceID}/invites", createInviteWithAccess)

	mux.HandleFunc("GET /join/{token}", space.JoinSpace)

	// 404
	mux.HandleFunc("/{path...}", home.NotFoundPage)

	// Global middlewares
	handler := middleware.Chain(
		mux,
		middleware.Config(a.Cfg),
		middleware.RequestLogging,
		middleware.NoCacheDynamic,
		middleware.CSRFProtection,
		middleware.AuthMiddleware(a.AuthService, a.UserService, a.ProfileService),
		middleware.WithURLPath,
	)

	return handler
}
