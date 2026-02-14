package app

import (
	"fmt"

	"git.juancwu.dev/juancwu/budgit/internal/config"
	"git.juancwu.dev/juancwu/budgit/internal/db"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"github.com/jmoiron/sqlx"
)

type App struct {
	Cfg                     *config.Config
	DB                      *sqlx.DB
	UserService             *service.UserService
	AuthService             *service.AuthService
	EmailService            *service.EmailService
	ProfileService          *service.ProfileService
	SpaceService            *service.SpaceService
	TagService              *service.TagService
	ShoppingListService     *service.ShoppingListService
	ExpenseService          *service.ExpenseService
	InviteService           *service.InviteService
	MoneyAccountService     *service.MoneyAccountService
	PaymentMethodService    *service.PaymentMethodService
	RecurringExpenseService *service.RecurringExpenseService
	BudgetService           *service.BudgetService
	ReportService           *service.ReportService
}

func New(cfg *config.Config) (*App, error) {
	database, err := db.Init(cfg.DBDriver, cfg.DBConnection)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	err = db.RunMigrations(database.DB, cfg.DBDriver)
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	emailClient := service.NewEmailClient(cfg.MailerSMTPHost, cfg.MailerSMTPPort, cfg.MailerIMAPHost, cfg.MailerIMAPPort, cfg.MailerUsername, cfg.MailerPassword)

	// Repositories
	userRepository := repository.NewUserRepository(database)
	profileRepository := repository.NewProfileRepository(database)
	tokenRepository := repository.NewTokenRepository(database)
	spaceRepository := repository.NewSpaceRepository(database)
	tagRepository := repository.NewTagRepository(database)
	shoppingListRepository := repository.NewShoppingListRepository(database)
	listItemRepository := repository.NewListItemRepository(database)
	expenseRepository := repository.NewExpenseRepository(database)
	invitationRepository := repository.NewInvitationRepository(database)
	moneyAccountRepository := repository.NewMoneyAccountRepository(database)
	paymentMethodRepository := repository.NewPaymentMethodRepository(database)
	recurringExpenseRepository := repository.NewRecurringExpenseRepository(database)
	budgetRepository := repository.NewBudgetRepository(database)

	// Services
	userService := service.NewUserService(userRepository)
	spaceService := service.NewSpaceService(spaceRepository)
	emailService := service.NewEmailService(
		emailClient,
		cfg.MailerEmailFrom,
		cfg.AppURL,
		cfg.AppName,
		cfg.IsProduction(),
	)
	authService := service.NewAuthService(
		emailService,
		userRepository,
		profileRepository,
		tokenRepository,
		spaceService,
		cfg.JWTSecret,
		cfg.JWTExpiry,
		cfg.TokenMagicLinkExpiry,
		cfg.IsProduction(),
	)
	profileService := service.NewProfileService(profileRepository)
	tagService := service.NewTagService(tagRepository)
	shoppingListService := service.NewShoppingListService(shoppingListRepository, listItemRepository)
	expenseService := service.NewExpenseService(expenseRepository)
	inviteService := service.NewInviteService(invitationRepository, spaceRepository, userRepository, emailService)
	moneyAccountService := service.NewMoneyAccountService(moneyAccountRepository)
	paymentMethodService := service.NewPaymentMethodService(paymentMethodRepository)
	recurringExpenseService := service.NewRecurringExpenseService(recurringExpenseRepository, expenseRepository)
	budgetService := service.NewBudgetService(budgetRepository)
	reportService := service.NewReportService(expenseRepository)

	return &App{
		Cfg:                     cfg,
		DB:                      database,
		UserService:             userService,
		AuthService:             authService,
		EmailService:            emailService,
		ProfileService:          profileService,
		SpaceService:            spaceService,
		TagService:              tagService,
		ShoppingListService:     shoppingListService,
		ExpenseService:          expenseService,
		InviteService:           inviteService,
		MoneyAccountService:     moneyAccountService,
		PaymentMethodService:    paymentMethodService,
		RecurringExpenseService: recurringExpenseService,
		BudgetService:           budgetService,
		ReportService:           reportService,
	}, nil
}
func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}
