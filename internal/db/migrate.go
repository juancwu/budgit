package db

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/pressly/goose/v3"
)

func setupGoose() error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	migrationsDir, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create sub-filesystem migrations directory: %w", err)
	}

	goose.SetBaseFS(migrationsDir)
	return nil
}

func RunMigrations(db *sql.DB) error {
	if err := setupGoose(); err != nil {
		return err
	}

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("migrations completed successfully")
	return nil
}
