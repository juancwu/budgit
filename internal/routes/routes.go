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
	space := handler.NewSpaceHandler(a.SpaceService, a.TagService, a.ShoppingListService, a.ExpenseService, a.InviteService, a.MoneyAccountService, a.PaymentMethodService, a.RecurringExpenseService, a.RecurringDepositService, a.BudgetService, a.ReportService)

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

	mux.HandleFunc("GET /app/dashboard", middleware.Redirect("/app/spaces"))
	mux.HandleFunc("GET /app/spaces", middleware.RequireAuth(space.DashboardPage))
	mux.Handle("POST /app/spaces", crudLimiter(middleware.RequireAuth(space.CreateSpace)))
	mux.HandleFunc("GET /app/settings", middleware.RequireAuth(settings.SettingsPage))
	mux.HandleFunc("POST /app/settings/password", authRateLimiter(middleware.RequireAuth(settings.SetPassword)))

	// Space routes — wrapping order: Auth(SpaceAccess(handler))
	// Auth runs first (outer), then SpaceAccess (inner), then the handler.
	spaceOverviewHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.OverviewPage)
	spaceOverviewWithAuth := middleware.RequireAuth(spaceOverviewHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}", spaceOverviewWithAuth)

	reportsPageHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.ReportsPage)
	reportsPageWithAuth := middleware.RequireAuth(reportsPageHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/reports", reportsPageWithAuth)

	listsPageHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.ListsPage)
	listsPageWithAuth := middleware.RequireAuth(listsPageHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/lists", listsPageWithAuth)

	createListHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.CreateList)
	createListWithAuth := middleware.RequireAuth(createListHandler)
	mux.Handle("POST /app/spaces/{spaceID}/lists", crudLimiter(createListWithAuth))

	listPageHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.ListPage)
	listPageWithAuth := middleware.RequireAuth(listPageHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/lists/{listID}", listPageWithAuth)

	updateListHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.UpdateList)
	updateListWithAuth := middleware.RequireAuth(updateListHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/lists/{listID}", crudLimiter(updateListWithAuth))

	deleteListHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.DeleteList)
	deleteListWithAuth := middleware.RequireAuth(deleteListHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/lists/{listID}", crudLimiter(deleteListWithAuth))

	addItemHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.AddItemToList)
	addItemWithAuth := middleware.RequireAuth(addItemHandler)
	mux.Handle("POST /app/spaces/{spaceID}/lists/{listID}/items", crudLimiter(addItemWithAuth))

	toggleItemHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.ToggleItem)
	toggleItemWithAuth := middleware.RequireAuth(toggleItemHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/lists/{listID}/items/{itemID}", crudLimiter(toggleItemWithAuth))

	deleteItemHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.DeleteItem)
	deleteItemWithAuth := middleware.RequireAuth(deleteItemHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/lists/{listID}/items/{itemID}", crudLimiter(deleteItemWithAuth))

	// Tag routes
	tagsPageHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.TagsPage)
	tagsPageWithAuth := middleware.RequireAuth(tagsPageHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/tags", tagsPageWithAuth)

	createTagHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.CreateTag)
	createTagWithAuth := middleware.RequireAuth(createTagHandler)
	mux.Handle("POST /app/spaces/{spaceID}/tags", crudLimiter(createTagWithAuth))

	deleteTagHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.DeleteTag)
	deleteTagWithAuth := middleware.RequireAuth(deleteTagHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/tags/{tagID}", crudLimiter(deleteTagWithAuth))

	// Expense routes
	expensesPageHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.ExpensesPage)
	expensesPageWithAuth := middleware.RequireAuth(expensesPageHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/expenses", expensesPageWithAuth)

	createExpenseHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.CreateExpense)
	createExpenseWithAuth := middleware.RequireAuth(createExpenseHandler)
	mux.Handle("POST /app/spaces/{spaceID}/expenses", crudLimiter(createExpenseWithAuth))

	updateExpenseHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.UpdateExpense)
	updateExpenseWithAuth := middleware.RequireAuth(updateExpenseHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/expenses/{expenseID}", crudLimiter(updateExpenseWithAuth))

	deleteExpenseHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.DeleteExpense)
	deleteExpenseWithAuth := middleware.RequireAuth(deleteExpenseHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/expenses/{expenseID}", crudLimiter(deleteExpenseWithAuth))

	// Money Account routes
	accountsPageHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.AccountsPage)
	accountsPageWithAuth := middleware.RequireAuth(accountsPageHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/accounts", accountsPageWithAuth)

	createAccountHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.CreateAccount)
	createAccountWithAuth := middleware.RequireAuth(createAccountHandler)
	mux.Handle("POST /app/spaces/{spaceID}/accounts", crudLimiter(createAccountWithAuth))

	updateAccountHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.UpdateAccount)
	updateAccountWithAuth := middleware.RequireAuth(updateAccountHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/accounts/{accountID}", crudLimiter(updateAccountWithAuth))

	deleteAccountHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.DeleteAccount)
	deleteAccountWithAuth := middleware.RequireAuth(deleteAccountHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/accounts/{accountID}", crudLimiter(deleteAccountWithAuth))

	createTransferHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.CreateTransfer)
	createTransferWithAuth := middleware.RequireAuth(createTransferHandler)
	mux.Handle("POST /app/spaces/{spaceID}/accounts/{accountID}/transfers", crudLimiter(createTransferWithAuth))

	deleteTransferHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.DeleteTransfer)
	deleteTransferWithAuth := middleware.RequireAuth(deleteTransferHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/accounts/{accountID}/transfers/{transferID}", crudLimiter(deleteTransferWithAuth))

	// Recurring Deposit routes
	createRecurringDepositHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.CreateRecurringDeposit)
	createRecurringDepositWithAuth := middleware.RequireAuth(createRecurringDepositHandler)
	mux.Handle("POST /app/spaces/{spaceID}/accounts/recurring", crudLimiter(createRecurringDepositWithAuth))

	updateRecurringDepositHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.UpdateRecurringDeposit)
	updateRecurringDepositWithAuth := middleware.RequireAuth(updateRecurringDepositHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/accounts/recurring/{recurringDepositID}", crudLimiter(updateRecurringDepositWithAuth))

	deleteRecurringDepositHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.DeleteRecurringDeposit)
	deleteRecurringDepositWithAuth := middleware.RequireAuth(deleteRecurringDepositHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/accounts/recurring/{recurringDepositID}", crudLimiter(deleteRecurringDepositWithAuth))

	toggleRecurringDepositHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.ToggleRecurringDeposit)
	toggleRecurringDepositWithAuth := middleware.RequireAuth(toggleRecurringDepositHandler)
	mux.Handle("POST /app/spaces/{spaceID}/accounts/recurring/{recurringDepositID}/toggle", crudLimiter(toggleRecurringDepositWithAuth))

	// Payment Method routes
	methodsPageHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.PaymentMethodsPage)
	methodsPageWithAuth := middleware.RequireAuth(methodsPageHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/payment-methods", methodsPageWithAuth)

	createMethodHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.CreatePaymentMethod)
	createMethodWithAuth := middleware.RequireAuth(createMethodHandler)
	mux.Handle("POST /app/spaces/{spaceID}/payment-methods", crudLimiter(createMethodWithAuth))

	updateMethodHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.UpdatePaymentMethod)
	updateMethodWithAuth := middleware.RequireAuth(updateMethodHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/payment-methods/{methodID}", crudLimiter(updateMethodWithAuth))

	deleteMethodHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.DeletePaymentMethod)
	deleteMethodWithAuth := middleware.RequireAuth(deleteMethodHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/payment-methods/{methodID}", crudLimiter(deleteMethodWithAuth))

	// Recurring expense routes
	recurringPageHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.RecurringExpensesPage)
	recurringPageWithAuth := middleware.RequireAuth(recurringPageHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/recurring", recurringPageWithAuth)

	createRecurringHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.CreateRecurringExpense)
	createRecurringWithAuth := middleware.RequireAuth(createRecurringHandler)
	mux.Handle("POST /app/spaces/{spaceID}/recurring", crudLimiter(createRecurringWithAuth))

	updateRecurringHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.UpdateRecurringExpense)
	updateRecurringWithAuth := middleware.RequireAuth(updateRecurringHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/recurring/{recurringID}", crudLimiter(updateRecurringWithAuth))

	deleteRecurringHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.DeleteRecurringExpense)
	deleteRecurringWithAuth := middleware.RequireAuth(deleteRecurringHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/recurring/{recurringID}", crudLimiter(deleteRecurringWithAuth))

	toggleRecurringHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.ToggleRecurringExpense)
	toggleRecurringWithAuth := middleware.RequireAuth(toggleRecurringHandler)
	mux.Handle("POST /app/spaces/{spaceID}/recurring/{recurringID}/toggle", crudLimiter(toggleRecurringWithAuth))

	// Budget routes
	budgetsPageHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.BudgetsPage)
	budgetsPageWithAuth := middleware.RequireAuth(budgetsPageHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/budgets", budgetsPageWithAuth)

	createBudgetHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.CreateBudget)
	createBudgetWithAuth := middleware.RequireAuth(createBudgetHandler)
	mux.Handle("POST /app/spaces/{spaceID}/budgets", crudLimiter(createBudgetWithAuth))

	updateBudgetHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.UpdateBudget)
	updateBudgetWithAuth := middleware.RequireAuth(updateBudgetHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/budgets/{budgetID}", crudLimiter(updateBudgetWithAuth))

	deleteBudgetHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.DeleteBudget)
	deleteBudgetWithAuth := middleware.RequireAuth(deleteBudgetHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/budgets/{budgetID}", crudLimiter(deleteBudgetWithAuth))

	budgetsListHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.GetBudgetsList)
	budgetsListWithAuth := middleware.RequireAuth(budgetsListHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/components/budgets", budgetsListWithAuth)

	// Report routes
	reportChartsHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.GetReportCharts)
	reportChartsWithAuth := middleware.RequireAuth(reportChartsHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/components/report-charts", reportChartsWithAuth)

	// Component routes (HTMX updates)
	balanceCardHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.GetBalanceCard)
	balanceCardWithAuth := middleware.RequireAuth(balanceCardHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/components/balance", balanceCardWithAuth)

	expensesListHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.GetExpensesList)
	expensesListWithAuth := middleware.RequireAuth(expensesListHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/components/expenses", expensesListWithAuth)

	shoppingListItemsHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.GetShoppingListItems)
	shoppingListItemsWithAuth := middleware.RequireAuth(shoppingListItemsHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/lists/{listID}/items", shoppingListItemsWithAuth)

	cardItemsHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.GetListCardItems)
	cardItemsWithAuth := middleware.RequireAuth(cardItemsHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/lists/{listID}/card-items", cardItemsWithAuth)

	listsComponentHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.GetLists)
	listsComponentWithAuth := middleware.RequireAuth(listsComponentHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/components/lists", listsComponentWithAuth)

	// Settings routes
	settingsPageHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.SettingsPage)
	settingsPageWithAuth := middleware.RequireAuth(settingsPageHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/settings", settingsPageWithAuth)

	updateSpaceNameHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.UpdateSpaceName)
	updateSpaceNameWithAuth := middleware.RequireAuth(updateSpaceNameHandler)
	mux.Handle("PATCH /app/spaces/{spaceID}/settings/name", crudLimiter(updateSpaceNameWithAuth))

	removeMemberHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.RemoveMember)
	removeMemberWithAuth := middleware.RequireAuth(removeMemberHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/members/{userID}", crudLimiter(removeMemberWithAuth))

	cancelInviteHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.CancelInvite)
	cancelInviteWithAuth := middleware.RequireAuth(cancelInviteHandler)
	mux.Handle("DELETE /app/spaces/{spaceID}/invites/{token}", crudLimiter(cancelInviteWithAuth))

	getPendingInvitesHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.GetPendingInvites)
	getPendingInvitesWithAuth := middleware.RequireAuth(getPendingInvitesHandler)
	mux.HandleFunc("GET /app/spaces/{spaceID}/settings/invites", getPendingInvitesWithAuth)

	// Invite routes
	createInviteHandler := middleware.RequireSpaceAccess(a.SpaceService)(space.CreateInvite)
	createInviteWithAuth := middleware.RequireAuth(createInviteHandler)
	mux.Handle("POST /app/spaces/{spaceID}/invites", crudLimiter(createInviteWithAuth))

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
