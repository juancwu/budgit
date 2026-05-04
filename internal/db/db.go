package db

import (
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// Init opens a PostgreSQL connection pool. The connection string must be a
// libpq-style URL or DSN supported by the pgx stdlib driver.
func Init(connection string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("pgx", connection)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	slog.Info("database connected", "driver", "pgx")

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func Close(db *sqlx.DB) error {
	if db != nil {
		return db.Close()
	}
	return nil
}
