package app

import (
	"fmt"
	"time"

	"git.juancwu.dev/juancwu/budgit/internal/config"
	"git.juancwu.dev/juancwu/budgit/internal/db"
	"git.juancwu.dev/juancwu/budgit/internal/repository"
	"git.juancwu.dev/juancwu/budgit/internal/service"
	"git.juancwu.dev/juancwu/budgit/internal/worker"
	"github.com/jmoiron/sqlx"
)

type App struct {
	Cfg                   *config.Config
	DB                    *sqlx.DB
	UserService           *service.UserService
	AuthService           *service.AuthService
	EmailService          *service.EmailService
	SpaceService          *service.SpaceService
	AccountService        *service.AccountService
	AllocationService     *service.AllocationService
	TransactionService    *service.TransactionService
	RecurringEventService *service.RecurringEventService
	InviteService         *service.InviteService
	AuditLogService       *service.SpaceAuditLogService
	TxAuditLogService     *service.TransactionAuditLogService
	AccountActivitySvc    *service.AccountActivityService
	InvestmentService     *service.InvestmentService
	BudgetPlanService     *service.BudgetPlanService
	AccountDeletionWorker *worker.AccountDeletionWorker
}

func New(cfg *config.Config) (*App, error) {
	database, err := db.Init(cfg.DBConnection)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	err = db.RunMigrations(database.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	emailClient := service.NewEmailClient(cfg.MailerSMTPHost, cfg.MailerSMTPPort, cfg.MailerIMAPHost, cfg.MailerIMAPPort, cfg.MailerUsername, cfg.MailerPassword)

	// Repositories
	userRepository := repository.NewUserRepository(database)
	tokenRepository := repository.NewTokenRepository(database)
	spaceRepository := repository.NewSpaceRepository(database)
	accountRepository := repository.NewAccountRepository(database)
	allocationRepository := repository.NewAllocationRepository(database)
	transactionRepository := repository.NewTransactionRepository(database)
	categoryRepository := repository.NewCategoryRepository(database)
	invitationRepository := repository.NewInvitationRepository(database)
	auditLogRepository := repository.NewSpaceAuditLogRepository(database)
	txAuditLogRepository := repository.NewTransactionAuditLogRepository(database)
	recurringEventRepository := repository.NewRecurringEventRepository(database)
	accountDeletionRequestRepo := repository.NewAccountDeletionRequestRepository(database)
	contributionRoomRepo := repository.NewInvestmentContributionRoomRepository(database)
	holdingRepo := repository.NewInvestmentHoldingRepository(database)
	tradeRepo := repository.NewInvestmentTradeRepository(database)
	budgetPlanRepo := repository.NewBudgetPlanRepository(database)
	budgetPlanLineRepo := repository.NewBudgetPlanLineRepository(database)

	// Services
	emailService := service.NewEmailService(
		emailClient,
		cfg.MailerEmailFrom,
		cfg.AppURL,
		cfg.AppName,
		cfg.IsProduction(),
	)
	userService := service.NewUserService(database, userRepository, accountDeletionRequestRepo, emailService)
	accountDeletionWorker := worker.NewAccountDeletionWorker(userService, 30*time.Second)
	auditLogService := service.NewSpaceAuditLogService(auditLogRepository)
	txAuditLogService := service.NewTransactionAuditLogService(txAuditLogRepository)
	spaceService := service.NewSpaceService(spaceRepository)
	spaceService.SetAuditLogger(auditLogService)
	accountService := service.NewAccountService(accountRepository)
	accountService.SetAuditLogger(auditLogService)
	accountService.SetAllocationRepository(allocationRepository)
	allocationService := service.NewAllocationService(allocationRepository, accountService)
	allocationService.SetAuditLogger(auditLogService)
	transactionService := service.NewTransactionService(transactionRepository, categoryRepository, accountService)
	transactionService.SetAuditLogger(txAuditLogService)
	transactionService.SetAllocationService(allocationService)
	accountActivityService := service.NewAccountActivityService(auditLogService, txAuditLogService)
	authService := service.NewAuthService(
		emailService,
		userRepository,
		tokenRepository,
		spaceService,
		accountService,
		cfg.JWTSecret,
		cfg.JWTExpiry,
		cfg.TokenMagicLinkExpiry,
		cfg.IsProduction(),
		cfg.DisableRegistration,
	)
	inviteService := service.NewInviteService(invitationRepository, spaceRepository, userRepository, emailService, auditLogService)
	recurringEventService := service.NewRecurringEventService(recurringEventRepository, transactionService, accountService)
	investmentService := service.NewInvestmentService(accountRepository, contributionRoomRepo, holdingRepo, tradeRepo, transactionRepository)
	budgetPlanService := service.NewBudgetPlanService(budgetPlanRepo, budgetPlanLineRepo, categoryRepository)

	return &App{
		Cfg:                   cfg,
		DB:                    database,
		UserService:           userService,
		AuthService:           authService,
		EmailService:          emailService,
		SpaceService:          spaceService,
		AccountService:        accountService,
		AllocationService:     allocationService,
		TransactionService:    transactionService,
		RecurringEventService: recurringEventService,
		InviteService:         inviteService,
		AuditLogService:       auditLogService,
		TxAuditLogService:     txAuditLogService,
		AccountActivitySvc:    accountActivityService,
		InvestmentService:     investmentService,
		BudgetPlanService:     budgetPlanService,
		AccountDeletionWorker: accountDeletionWorker,
	}, nil
}

func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}
