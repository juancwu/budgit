package testutil

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"git.juancwu.dev/juancwu/budgit/internal/db"
	"github.com/jmoiron/sqlx"
)

// DBInfo holds a test database connection and its driver name.
type DBInfo struct {
	DB     *sqlx.DB
	Driver string
}

// ForEachDB runs the given test function against both SQLite and PostgreSQL.
// PostgreSQL tests are skipped when BUDGIT_TEST_POSTGRES_URL is unset.
func ForEachDB(t *testing.T, fn func(t *testing.T, dbi DBInfo)) {
	t.Helper()

	t.Run("sqlite", func(t *testing.T) {
		t.Parallel()
		dbi := newSQLiteDB(t)
		fn(t, dbi)
	})

	pgURL := os.Getenv("BUDGIT_TEST_POSTGRES_URL")
	if pgURL == "" {
		t.Log("skipping postgres tests: BUDGIT_TEST_POSTGRES_URL not set")
		return
	}

	t.Run("postgres", func(t *testing.T) {
		t.Parallel()
		dbi := newPostgresDB(t, pgURL)
		fn(t, dbi)
	})
}

func newSQLiteDB(t *testing.T) DBInfo {
	t.Helper()

	// Use a unique in-memory database per test via a unique DSN.
	// Each file::memory:?cache=shared&name=X uses a separate in-memory DB.
	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)", safeName)

	sqliteDB, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to connect to sqlite: %v", err)
	}
	// SQLite in-memory DBs are destroyed when the last connection closes.
	// Keep at least one open so it survives the test.
	sqliteDB.SetMaxOpenConns(1)

	t.Cleanup(func() { sqliteDB.Close() })

	err = db.RunMigrations(sqliteDB.DB, "sqlite")
	if err != nil {
		t.Fatalf("failed to run sqlite migrations: %v", err)
	}

	return DBInfo{DB: sqliteDB, Driver: "sqlite"}
}

func newPostgresDB(t *testing.T, baseURL string) DBInfo {
	t.Helper()

	// Create a unique schema per test to ensure isolation.
	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	safeName = strings.ReplaceAll(safeName, " ", "_")
	schema := fmt.Sprintf("test_%s", safeName)

	// Connect to the base database to create the schema.
	baseDB, err := sqlx.Connect("pgx", baseURL)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}

	_, err = baseDB.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %q", schema))
	if err != nil {
		baseDB.Close()
		t.Fatalf("failed to create schema %s: %v", schema, err)
	}
	baseDB.Close()

	// Connect with a single-connection pool and set search_path to the new schema.
	// MaxOpenConns(1) ensures all queries reuse the same connection where
	// search_path is set (SET is session-level in PostgreSQL).
	pgDB, err := sqlx.Connect("pgx", baseURL)
	if err != nil {
		t.Fatalf("failed to connect to postgres with schema: %v", err)
	}
	pgDB.SetMaxOpenConns(1)

	_, err = pgDB.Exec(fmt.Sprintf(`SET search_path TO "%s"`, schema))
	if err != nil {
		pgDB.Close()
		t.Fatalf("failed to set search_path to %s: %v", schema, err)
	}

	t.Cleanup(func() {
		pgDB.Close()
		// Drop the schema after the test.
		cleanDB, err := sqlx.Connect("pgx", baseURL)
		if err == nil {
			cleanDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema))
			cleanDB.Close()
		}
	})

	err = db.RunMigrations(pgDB.DB, "pgx")
	if err != nil {
		t.Fatalf("failed to run postgres migrations: %v", err)
	}

	return DBInfo{DB: pgDB, Driver: "pgx"}
}
