package app

import (
	"fmt"

	"git.juancwu.dev/juancwu/budgething/internal/config"
	"git.juancwu.dev/juancwu/budgething/internal/db"
	"git.juancwu.dev/juancwu/budgething/internal/repository"
	"git.juancwu.dev/juancwu/budgething/internal/service"
	"github.com/jmoiron/sqlx"
)

type App struct {
	Cfg         *config.Config
	DB          *sqlx.DB
	UserService *service.UserService
	AuthService *service.AuthService
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

	userRepository := repository.NewUserRepository(database)

	userService := service.NewUserService(userRepository)
	authService := service.NewAuthService(userRepository)

	return &App{
		Cfg:         cfg,
		DB:          database,
		UserService: userService,
		AuthService: authService,
	}, nil
}

func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}
