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
	Cfg          *config.Config
	DB           *sqlx.DB
	UserService  *service.UserService
	AuthService  *service.AuthService
	EmailService *service.EmailService
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

	userRepository := repository.NewUserRepository(database)

	userService := service.NewUserService(userRepository)
	authService := service.NewAuthService(userRepository)
	emailService := service.NewEmailService(emailClient, cfg.MailerEmailFrom, cfg.MailerEnvelopeFrom, cfg.MailerSupportFrom, cfg.MailerSupportEnvelopeFrom, cfg.AppURL, cfg.AppName, cfg.AppEnv == "development")

	return &App{
		Cfg:          cfg,
		DB:           database,
		UserService:  userService,
		AuthService:  authService,
		EmailService: emailService,
	}, nil
}

func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}
