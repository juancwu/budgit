package testutil

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/db"
	"github.com/jmoiron/sqlx"
)

// DBInfo holds a test database connection.
type DBInfo struct {
	DB *sqlx.DB
}

// ForEachDB runs the test function against PostgreSQL. Skips when
// BUDGIT_TEST_POSTGRES_URL is not set so quick local runs don't fail
// without a database. CI must always set it.
//
// Each test gets its own schema for isolation; the schema is dropped on
// cleanup. The function name is preserved for backwards compatibility,
// although there is now only one engine.
func ForEachDB(t *testing.T, fn func(t *testing.T, dbi DBInfo)) {
	t.Helper()

	pgURL := os.Getenv("BUDGIT_TEST_POSTGRES_URL")
	if pgURL == "" {
		t.Skip("skipping db tests: BUDGIT_TEST_POSTGRES_URL not set")
		return
	}

	t.Run("postgres", func(t *testing.T) {
		t.Parallel()
		dbi := newPostgresDB(t, pgURL)
		fn(t, dbi)
	})
}

func newPostgresDB(t *testing.T, baseURL string) DBInfo {
	t.Helper()

	// Create a unique schema per test for isolation.
	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	safeName = strings.ReplaceAll(safeName, " ", "_")
	schema := fmt.Sprintf("test_%s", safeName)

	baseDB, err := sqlx.Connect("pgx", baseURL)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
	if _, err := baseDB.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %q", schema)); err != nil {
		baseDB.Close()
		t.Fatalf("failed to create schema %s: %v", schema, err)
	}
	baseDB.Close()

	// MaxOpenConns(1) ensures every query reuses the connection where
	// search_path is set (SET is session-level in PostgreSQL).
	pgDB, err := sqlx.Connect("pgx", baseURL)
	if err != nil {
		t.Fatalf("failed to connect to postgres with schema: %v", err)
	}
	pgDB.SetMaxOpenConns(1)

	if _, err := pgDB.Exec(fmt.Sprintf(`SET search_path TO "%s"`, schema)); err != nil {
		pgDB.Close()
		t.Fatalf("failed to set search_path to %s: %v", schema, err)
	}

	t.Cleanup(func() {
		pgDB.Close()
		cleanDB, err := sqlx.Connect("pgx", baseURL)
		if err == nil {
			cleanDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema))
			cleanDB.Close()
		}
	})

	if err := db.RunMigrations(pgDB.DB); err != nil {
		t.Fatalf("failed to run postgres migrations: %v", err)
	}

	return DBInfo{DB: pgDB}
}
