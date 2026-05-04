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
	Cfg                *config.Config
	DB                 *sqlx.DB
	UserService        *service.UserService
	AuthService        *service.AuthService
	EmailService       *service.EmailService
	SpaceService       *service.SpaceService
	AccountService     *service.AccountService
	AllocationService  *service.AllocationService
	TransactionService *service.TransactionService
	InviteService      *service.InviteService
	AuditLogService    *service.SpaceAuditLogService
	TxAuditLogService  *service.TransactionAuditLogService
	AccountActivitySvc *service.AccountActivityService
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

	// Services
	userService := service.NewUserService(userRepository)
	auditLogService := service.NewSpaceAuditLogService(auditLogRepository)
	txAuditLogService := service.NewTransactionAuditLogService(txAuditLogRepository)
	spaceService := service.NewSpaceService(spaceRepository)
	spaceService.SetAuditLogger(auditLogService)
	accountService := service.NewAccountService(accountRepository)
	accountService.SetAuditLogger(auditLogService)
	allocationService := service.NewAllocationService(allocationRepository, accountService)
	allocationService.SetAuditLogger(auditLogService)
	transactionService := service.NewTransactionService(transactionRepository, categoryRepository, accountService)
	transactionService.SetAuditLogger(txAuditLogService)
	accountActivityService := service.NewAccountActivityService(auditLogService, txAuditLogService)
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
		tokenRepository,
		spaceService,
		accountService,
		cfg.JWTSecret,
		cfg.JWTExpiry,
		cfg.TokenMagicLinkExpiry,
		cfg.IsProduction(),
	)
	inviteService := service.NewInviteService(invitationRepository, spaceRepository, userRepository, emailService, auditLogService)

	return &App{
		Cfg:                cfg,
		DB:                 database,
		UserService:        userService,
		AuthService:        authService,
		EmailService:       emailService,
		SpaceService:       spaceService,
		AccountService:     accountService,
		AllocationService:  allocationService,
		TransactionService: transactionService,
		InviteService:      inviteService,
		AuditLogService:    auditLogService,
		TxAuditLogService:  txAuditLogService,
		AccountActivitySvc: accountActivityService,
	}, nil
}

func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}
