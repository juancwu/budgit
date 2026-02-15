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
	dashboard := handler.NewDashboardHandler(a.SpaceService, a.ExpenseService)
	settings := handler.NewSettingsHandler(a.AuthService, a.UserService)
	space := handler.NewSpaceHandler(a.SpaceService, a.TagService, a.ShoppingListService, a.ExpenseService, a.InviteService, a.MoneyAccountService, a.PaymentMethodService, a.RecurringExpenseService, a.BudgetService, a.ReportService)

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
	crudLimiter := middleware.RateLimitCRUD()

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
	mux.Handle("POST /auth/onboarding", crudLimiter(http.HandlerFunc(middleware.RequireAuth(auth.CompleteOnboarding))))

	mux.HandleFunc("GET /app/dashboard", middleware.RequireAuth(dashboard.DashboardPage))
	mux.Handle("POST /app/spaces", crudLimiter(http.HandlerFunc(middleware.RequireAuth(dashboard.CreateSpace))))
	mux.HandleFunc("GET /app/settings", middleware.RequireAuth(settings.SettingsPage))
	mux.HandleFunc("POST /app/settings/password", authRateLimiter(middleware.RequireAuth(settings.SetPassword)))

	// Space routes
	spaceDashboardHandler := middleware.RequireAuth(space.DashboardPage)
	spaceDashboardWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(spaceDashboardHandler)
	mux.Handle("GET /app/spaces/{spaceID}", spaceDashboardWithAccess)

	listsPageHandler := middleware.RequireAuth(space.ListsPage)
	listsPageWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(listsPageHandler)
	mux.Handle("GET /app/spaces/{spaceID}/lists", listsPageWithAccess)

	createListHandler := middleware.RequireAuth(space.CreateList)
	createListWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(createListHandler)
	mux.Handle("POST /app/spaces/{spaceID}/lists", crudLimiter(createListWithAccess))

	listPageHandler := middleware.RequireAuth(space.ListPage)
	listPageWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(listPageHandler)
	mux.Handle("GET /app/spaces/{spaceID}/lists/{listID}", listPageWithAccess)

	updateListHandler := middleware.RequireAuth(space.UpdateList)
	updateListWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(updateListHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/lists/{listID}", crudLimiter(updateListWithAccess))

	deleteListHandler := middleware.RequireAuth(space.DeleteList)
	deleteListWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(deleteListHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/lists/{listID}", crudLimiter(deleteListWithAccess))

	addItemHandler := middleware.RequireAuth(space.AddItemToList)
	addItemWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(addItemHandler)
	mux.Handle("POST /app/spaces/{spaceID}/lists/{listID}/items", crudLimiter(addItemWithAccess))

	toggleItemHandler := middleware.RequireAuth(space.ToggleItem)
	toggleItemWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(toggleItemHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/lists/{listID}/items/{itemID}", crudLimiter(toggleItemWithAccess))

	deleteItemHandler := middleware.RequireAuth(space.DeleteItem)
	deleteItemWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(deleteItemHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/lists/{listID}/items/{itemID}", crudLimiter(deleteItemWithAccess))

	// Tag routes
	tagsPageHandler := middleware.RequireAuth(space.TagsPage)
	tagsPageWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(tagsPageHandler)
	mux.Handle("GET /app/spaces/{spaceID}/tags", tagsPageWithAccess)

	createTagHandler := middleware.RequireAuth(space.CreateTag)
	createTagWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(createTagHandler)
	mux.Handle("POST /app/spaces/{spaceID}/tags", crudLimiter(createTagWithAccess))

	deleteTagHandler := middleware.RequireAuth(space.DeleteTag)
	deleteTagWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(deleteTagHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/tags/{tagID}", crudLimiter(deleteTagWithAccess))

	// Expense routes
	expensesPageHandler := middleware.RequireAuth(space.ExpensesPage)
	expensesPageWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(expensesPageHandler)
	mux.Handle("GET /app/spaces/{spaceID}/expenses", expensesPageWithAccess)

	createExpenseHandler := middleware.RequireAuth(space.CreateExpense)
	createExpenseWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(createExpenseHandler)
	mux.Handle("POST /app/spaces/{spaceID}/expenses", crudLimiter(createExpenseWithAccess))

	updateExpenseHandler := middleware.RequireAuth(space.UpdateExpense)
	updateExpenseWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(updateExpenseHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/expenses/{expenseID}", crudLimiter(updateExpenseWithAccess))

	deleteExpenseHandler := middleware.RequireAuth(space.DeleteExpense)
	deleteExpenseWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(deleteExpenseHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/expenses/{expenseID}", crudLimiter(deleteExpenseWithAccess))

	// Money Account routes
	accountsPageHandler := middleware.RequireAuth(space.AccountsPage)
	accountsPageWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(accountsPageHandler)
	mux.Handle("GET /app/spaces/{spaceID}/accounts", accountsPageWithAccess)

	createAccountHandler := middleware.RequireAuth(space.CreateAccount)
	createAccountWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(createAccountHandler)
	mux.Handle("POST /app/spaces/{spaceID}/accounts", crudLimiter(createAccountWithAccess))

	updateAccountHandler := middleware.RequireAuth(space.UpdateAccount)
	updateAccountWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(updateAccountHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/accounts/{accountID}", crudLimiter(updateAccountWithAccess))

	deleteAccountHandler := middleware.RequireAuth(space.DeleteAccount)
	deleteAccountWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(deleteAccountHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/accounts/{accountID}", crudLimiter(deleteAccountWithAccess))

	createTransferHandler := middleware.RequireAuth(space.CreateTransfer)
	createTransferWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(createTransferHandler)
	mux.Handle("POST /app/spaces/{spaceID}/accounts/{accountID}/transfers", crudLimiter(createTransferWithAccess))

	deleteTransferHandler := middleware.RequireAuth(space.DeleteTransfer)
	deleteTransferWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(deleteTransferHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/accounts/{accountID}/transfers/{transferID}", crudLimiter(deleteTransferWithAccess))

	// Payment Method routes
	methodsPageHandler := middleware.RequireAuth(space.PaymentMethodsPage)
	methodsPageWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(methodsPageHandler)
	mux.Handle("GET /app/spaces/{spaceID}/payment-methods", methodsPageWithAccess)

	createMethodHandler := middleware.RequireAuth(space.CreatePaymentMethod)
	createMethodWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(createMethodHandler)
	mux.Handle("POST /app/spaces/{spaceID}/payment-methods", crudLimiter(createMethodWithAccess))

	updateMethodHandler := middleware.RequireAuth(space.UpdatePaymentMethod)
	updateMethodWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(updateMethodHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/payment-methods/{methodID}", crudLimiter(updateMethodWithAccess))

	deleteMethodHandler := middleware.RequireAuth(space.DeletePaymentMethod)
	deleteMethodWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(deleteMethodHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/payment-methods/{methodID}", crudLimiter(deleteMethodWithAccess))

	// Recurring expense routes
	recurringPageHandler := middleware.RequireAuth(space.RecurringExpensesPage)
	recurringPageWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(recurringPageHandler)
	mux.Handle("GET /app/spaces/{spaceID}/recurring", recurringPageWithAccess)

	createRecurringHandler := middleware.RequireAuth(space.CreateRecurringExpense)
	createRecurringWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(createRecurringHandler)
	mux.Handle("POST /app/spaces/{spaceID}/recurring", crudLimiter(createRecurringWithAccess))

	updateRecurringHandler := middleware.RequireAuth(space.UpdateRecurringExpense)
	updateRecurringWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(updateRecurringHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/recurring/{recurringID}", crudLimiter(updateRecurringWithAccess))

	deleteRecurringHandler := middleware.RequireAuth(space.DeleteRecurringExpense)
	deleteRecurringWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(deleteRecurringHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/recurring/{recurringID}", crudLimiter(deleteRecurringWithAccess))

	toggleRecurringHandler := middleware.RequireAuth(space.ToggleRecurringExpense)
	toggleRecurringWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(toggleRecurringHandler)
	mux.Handle("POST /app/spaces/{spaceID}/recurring/{recurringID}/toggle", crudLimiter(toggleRecurringWithAccess))

	// Budget routes
	budgetsPageHandler := middleware.RequireAuth(space.BudgetsPage)
	budgetsPageWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(budgetsPageHandler)
	mux.Handle("GET /app/spaces/{spaceID}/budgets", budgetsPageWithAccess)

	createBudgetHandler := middleware.RequireAuth(space.CreateBudget)
	createBudgetWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(createBudgetHandler)
	mux.Handle("POST /app/spaces/{spaceID}/budgets", crudLimiter(createBudgetWithAccess))

	updateBudgetHandler := middleware.RequireAuth(space.UpdateBudget)
	updateBudgetWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(updateBudgetHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/budgets/{budgetID}", crudLimiter(updateBudgetWithAccess))

	deleteBudgetHandler := middleware.RequireAuth(space.DeleteBudget)
	deleteBudgetWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(deleteBudgetHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/budgets/{budgetID}", crudLimiter(deleteBudgetWithAccess))

	budgetsListHandler := middleware.RequireAuth(space.GetBudgetsList)
	budgetsListWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(budgetsListHandler)
	mux.Handle("GET /app/spaces/{spaceID}/components/budgets", budgetsListWithAccess)

	// Report routes
	reportChartsHandler := middleware.RequireAuth(space.GetReportCharts)
	reportChartsWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(reportChartsHandler)
	mux.Handle("GET /app/spaces/{spaceID}/components/report-charts", reportChartsWithAccess)

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

	// Settings routes
	settingsPageHandler := middleware.RequireAuth(space.SettingsPage)
	settingsPageWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(settingsPageHandler)
	mux.Handle("GET /app/spaces/{spaceID}/settings", settingsPageWithAccess)

	updateSpaceNameHandler := middleware.RequireAuth(space.UpdateSpaceName)
	updateSpaceNameWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(updateSpaceNameHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/settings/name", crudLimiter(updateSpaceNameWithAccess))

	removeMemberHandler := middleware.RequireAuth(space.RemoveMember)
	removeMemberWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(removeMemberHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/members/{userID}", crudLimiter(removeMemberWithAccess))

	cancelInviteHandler := middleware.RequireAuth(space.CancelInvite)
	cancelInviteWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(cancelInviteHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/invites/{token}", crudLimiter(cancelInviteWithAccess))

	getPendingInvitesHandler := middleware.RequireAuth(space.GetPendingInvites)
	getPendingInvitesWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(getPendingInvitesHandler)
	mux.Handle("GET /app/spaces/{spaceID}/settings/invites", getPendingInvitesWithAccess)

	// Invite routes
	createInviteHandler := middleware.RequireAuth(space.CreateInvite)
	createInviteWithAccess := middleware.RequireSpaceAccess(a.SpaceService)(createInviteHandler)
	mux.Handle("POST /app/spaces/{spaceID}/invites", crudLimiter(createInviteWithAccess))

	mux.HandleFunc("GET /join/{token}", space.JoinSpace)

	// 404
	mux.HandleFunc("/{path...}", home.NotFoundPage)

	// Global middlewares
	handler := middleware.Chain(
		mux,
		middleware.SecurityHeaders(),
		middleware.AppVersion(a.Cfg.Version),
		middleware.Config(a.Cfg),
		middleware.RequestLogging,
		middleware.NoCacheDynamic,
		middleware.CSRFProtection,
		middleware.AuthMiddleware(a.AuthService, a.UserService, a.ProfileService),
		middleware.WithURLPath,
	)

	return handler
}
