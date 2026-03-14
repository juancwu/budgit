package routes

import (
	"io/fs"
	"net/http"

	"git.juancwu.dev/juancwu/budgit/assets"
	"git.juancwu.dev/juancwu/budgit/internal/app"
	"git.juancwu.dev/juancwu/budgit/internal/handler"
	"git.juancwu.dev/juancwu/budgit/internal/middleware"
)

// spaceRoute registers a space-protected route (no rate limit).
func spaceRoute(mux *http.ServeMux, spaceAccess func(http.HandlerFunc) http.HandlerFunc, pattern string, h http.HandlerFunc) {
	mux.HandleFunc(pattern, middleware.RequireAuth(spaceAccess(h)))
}

// spaceRouteLimited registers a rate-limited space-protected route.
func spaceRouteLimited(mux *http.ServeMux, spaceAccess func(http.HandlerFunc) http.HandlerFunc, limiter func(http.Handler) http.Handler, pattern string, h http.HandlerFunc) {
	mux.Handle(pattern, limiter(middleware.RequireAuth(spaceAccess(h))))
}

func SetupRoutes(a *app.App) http.Handler {
	auth := handler.NewAuthHandler(a.AuthService, a.InviteService, a.SpaceService)
	home := handler.NewHomeHandler()
	settings := handler.NewSettingsHandler(a.AuthService, a.UserService, a.ProfileService)
	space := handler.NewSpaceHandler(a.SpaceService, a.ExpenseService, a.MoneyAccountService, a.ReportService, a.BudgetService, a.RecurringExpenseService, a.ShoppingListService, a.TagService, a.PaymentMethodService, a.LoanService, a.ReceiptService, a.RecurringReceiptService)
	lists := handler.NewListHandler(a.SpaceService, a.ShoppingListService)
	tags := handler.NewTagHandler(a.SpaceService, a.TagService)
	expenses := handler.NewExpenseHandler(a.SpaceService, a.ExpenseService, a.TagService, a.ShoppingListService, a.MoneyAccountService, a.PaymentMethodService)
	accounts := handler.NewAccountHandler(a.SpaceService, a.MoneyAccountService, a.ExpenseService)
	methods := handler.NewMethodHandler(a.SpaceService, a.PaymentMethodService)
	recurring := handler.NewRecurringHandler(a.SpaceService, a.RecurringExpenseService, a.TagService, a.PaymentMethodService)
	budgets := handler.NewBudgetHandler(a.SpaceService, a.BudgetService, a.TagService, a.ReportService)
	spaceSettings := handler.NewSpaceSettingsHandler(a.SpaceService, a.InviteService)

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
	mux.HandleFunc("POST /app/settings/timezone", middleware.RequireAuth(settings.SetTimezone))

	// Space routes — wrapping order: Auth(SpaceAccess(handler))
	// Auth runs first (outer), then SpaceAccess (inner), then the handler.
	sa := middleware.RequireSpaceAccess(a.SpaceService)
	cl := crudLimiter

	// Overview & Reports
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}", space.OverviewPage)
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/reports", space.ReportsPage)

	// Shopping Lists
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/lists", lists.ListsPage)
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/lists", lists.CreateList)
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/lists/{listID}", lists.ListPage)
	spaceRouteLimited(mux, sa, cl, "PATCH /app/spaces/{spaceID}/lists/{listID}", lists.UpdateList)
	spaceRouteLimited(mux, sa, cl, "DELETE /app/spaces/{spaceID}/lists/{listID}", lists.DeleteList)
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/lists/{listID}/items", lists.AddItemToList)
	spaceRouteLimited(mux, sa, cl, "PATCH /app/spaces/{spaceID}/lists/{listID}/items/{itemID}", lists.ToggleItem)
	spaceRouteLimited(mux, sa, cl, "DELETE /app/spaces/{spaceID}/lists/{listID}/items/{itemID}", lists.DeleteItem)

	// Tags
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/tags", tags.TagsPage)
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/tags", tags.CreateTag)
	spaceRouteLimited(mux, sa, cl, "DELETE /app/spaces/{spaceID}/tags/{tagID}", tags.DeleteTag)

	// Expenses
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/expenses", expenses.ExpensesPage)
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/expenses", expenses.CreateExpense)
	spaceRouteLimited(mux, sa, cl, "PATCH /app/spaces/{spaceID}/expenses/{expenseID}", expenses.UpdateExpense)
	spaceRouteLimited(mux, sa, cl, "DELETE /app/spaces/{spaceID}/expenses/{expenseID}", expenses.DeleteExpense)

	// Money Accounts
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/accounts", accounts.AccountsPage)
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/accounts", accounts.CreateAccount)
	spaceRouteLimited(mux, sa, cl, "PATCH /app/spaces/{spaceID}/accounts/{accountID}", accounts.UpdateAccount)
	spaceRouteLimited(mux, sa, cl, "DELETE /app/spaces/{spaceID}/accounts/{accountID}", accounts.DeleteAccount)
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/accounts/{accountID}/transfers", accounts.CreateTransfer)
	spaceRouteLimited(mux, sa, cl, "DELETE /app/spaces/{spaceID}/accounts/{accountID}/transfers/{transferID}", accounts.DeleteTransfer)

	// Payment Methods
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/payment-methods", methods.PaymentMethodsPage)
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/payment-methods", methods.CreatePaymentMethod)
	spaceRouteLimited(mux, sa, cl, "PATCH /app/spaces/{spaceID}/payment-methods/{methodID}", methods.UpdatePaymentMethod)
	spaceRouteLimited(mux, sa, cl, "DELETE /app/spaces/{spaceID}/payment-methods/{methodID}", methods.DeletePaymentMethod)

	// Recurring Expenses
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/recurring", recurring.RecurringExpensesPage)
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/recurring", recurring.CreateRecurringExpense)
	spaceRouteLimited(mux, sa, cl, "PATCH /app/spaces/{spaceID}/recurring/{recurringID}", recurring.UpdateRecurringExpense)
	spaceRouteLimited(mux, sa, cl, "DELETE /app/spaces/{spaceID}/recurring/{recurringID}", recurring.DeleteRecurringExpense)
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/recurring/{recurringID}/toggle", recurring.ToggleRecurringExpense)

	// Budgets
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/budgets", budgets.BudgetsPage)
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/budgets", budgets.CreateBudget)
	spaceRouteLimited(mux, sa, cl, "PATCH /app/spaces/{spaceID}/budgets/{budgetID}", budgets.UpdateBudget)
	spaceRouteLimited(mux, sa, cl, "DELETE /app/spaces/{spaceID}/budgets/{budgetID}", budgets.DeleteBudget)

	// Component routes (HTMX partial updates)
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/components/budgets", budgets.GetBudgetsList)
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/components/report-charts", budgets.GetReportCharts)
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/components/transfer-history", accounts.GetTransferHistory)
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/components/balance", expenses.GetBalanceCard)
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/components/expenses", expenses.GetExpensesList)
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/lists/{listID}/items", lists.GetShoppingListItems)
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/lists/{listID}/card-items", lists.GetListCardItems)
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/components/lists", lists.GetLists)

	// Space Settings
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/settings", spaceSettings.SettingsPage)
	spaceRouteLimited(mux, sa, cl, "PATCH /app/spaces/{spaceID}/settings/name", spaceSettings.UpdateSpaceName)
	spaceRouteLimited(mux, sa, cl, "PATCH /app/spaces/{spaceID}/settings/timezone", spaceSettings.UpdateSpaceTimezone)
	spaceRouteLimited(mux, sa, cl, "DELETE /app/spaces/{spaceID}/members/{userID}", spaceSettings.RemoveMember)
	spaceRouteLimited(mux, sa, cl, "DELETE /app/spaces/{spaceID}/invites/{token}", spaceSettings.CancelInvite)
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/settings/invites", spaceSettings.GetPendingInvites)
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/invites", spaceSettings.CreateInvite)

	// Loans
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/loans", space.LoansPage)
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/loans", space.CreateLoan)
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/loans/{loanID}", space.LoanDetailPage)
	spaceRouteLimited(mux, sa, cl, "PATCH /app/spaces/{spaceID}/loans/{loanID}", space.UpdateLoan)
	spaceRouteLimited(mux, sa, cl, "DELETE /app/spaces/{spaceID}/loans/{loanID}", space.DeleteLoan)

	// Receipts
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/loans/{loanID}/receipts", space.CreateReceipt)
	spaceRouteLimited(mux, sa, cl, "PATCH /app/spaces/{spaceID}/loans/{loanID}/receipts/{receiptID}", space.UpdateReceipt)
	spaceRouteLimited(mux, sa, cl, "DELETE /app/spaces/{spaceID}/loans/{loanID}/receipts/{receiptID}", space.DeleteReceipt)
	spaceRoute(mux, sa, "GET /app/spaces/{spaceID}/loans/{loanID}/components/receipts", space.GetReceiptsList)

	// Recurring Receipts
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/loans/{loanID}/recurring", space.CreateRecurringReceipt)
	spaceRouteLimited(mux, sa, cl, "PATCH /app/spaces/{spaceID}/loans/{loanID}/recurring/{recurringReceiptID}", space.UpdateRecurringReceipt)
	spaceRouteLimited(mux, sa, cl, "DELETE /app/spaces/{spaceID}/loans/{loanID}/recurring/{recurringReceiptID}", space.DeleteRecurringReceipt)
	spaceRouteLimited(mux, sa, cl, "POST /app/spaces/{spaceID}/loans/{loanID}/recurring/{recurringReceiptID}/toggle", space.ToggleRecurringReceipt)

	mux.HandleFunc("GET /join/{token}", spaceSettings.JoinSpace)

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
		middleware.AuthMiddleware(a.AuthService, a.UserService, a.ProfileService),
		middleware.WithURLPath,
	)

	return handler
}
