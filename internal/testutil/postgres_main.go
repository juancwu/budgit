package testutil

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresMain is the TestMain entry point used by every package whose tests touch
// the database. It guarantees a running PostgreSQL 17 instance for the duration of
// the test binary:
//
//   - If BUDGIT_TEST_POSTGRES_URL is already set, it is used as-is. CI and `task test`
//     hit this path.
//   - Otherwise an ephemeral `postgres:17-alpine` container is started on a free
//     local port, BUDGIT_TEST_POSTGRES_URL is exported to it for the test process,
//     and the container is removed when the test binary exits — even on panic, via
//     a deferred cleanup around m.Run().
//
// Usage in each test package:
//
//	func TestMain(m *testing.M) { testutil.PostgresMain(m) }
func PostgresMain(m *testing.M) {
	if os.Getenv("BUDGIT_TEST_POSTGRES_URL") != "" {
		os.Exit(m.Run())
	}

	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Fprintln(os.Stderr, "testutil.PostgresMain: BUDGIT_TEST_POSTGRES_URL is unset and `docker` is not on PATH; cannot run db tests")
		os.Exit(1)
	}

	port, err := freePort()
	if err != nil {
		fmt.Fprintf(os.Stderr, "testutil.PostgresMain: failed to find free port: %v\n", err)
		os.Exit(1)
	}

	containerName := fmt.Sprintf("budgit-test-pg-%d-%d", os.Getpid(), time.Now().UnixNano())

	startCmd := exec.Command("docker", "run", "--rm", "-d",
		"--name", containerName,
		"-p", fmt.Sprintf("%d:5432", port),
		"-e", "POSTGRES_USER=budgit_test",
		"-e", "POSTGRES_PASSWORD=testpass",
		"-e", "POSTGRES_DB=budgit_test",
		// tmpfs for the data dir keeps tests fast — we don't care about durability.
		"--tmpfs", "/var/lib/postgresql/data:rw",
		"postgres:17-alpine",
	)
	if out, err := startCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "testutil.PostgresMain: docker run failed: %v\n%s\n", err, out)
		os.Exit(1)
	}

	stop := func() {
		// `docker rm -f` because --rm only fires on a clean exit; force-stop the
		// container regardless of state so leftover containers don't accumulate.
		_ = exec.Command("docker", "rm", "-f", containerName).Run()
	}

	url := fmt.Sprintf("postgres://budgit_test:testpass@127.0.0.1:%d/budgit_test?sslmode=disable", port)
	if err := waitForPostgres(url, 60*time.Second); err != nil {
		stop()
		fmt.Fprintf(os.Stderr, "testutil.PostgresMain: postgres did not become ready: %v\n", err)
		os.Exit(1)
	}

	if err := os.Setenv("BUDGIT_TEST_POSTGRES_URL", url); err != nil {
		stop()
		fmt.Fprintf(os.Stderr, "testutil.PostgresMain: setenv failed: %v\n", err)
		os.Exit(1)
	}

	// Run tests, then ALWAYS stop the container — including on panic.
	code := func() int {
		defer stop()
		return m.Run()
	}()
	os.Exit(code)
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// waitForPostgres polls until a real client connection succeeds. pg_isready isn't
// sufficient under load — under parallel `go test ./...` we've seen it report ready
// while client connections still fail with "unexpected EOF" because the server is
// still finishing startup.
func waitForPostgres(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		db, err := sql.Open("pgx", url)
		if err == nil {
			if err = db.Ping(); err == nil {
				_ = db.Close()
				return nil
			}
			_ = db.Close()
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out after %s: %w", timeout, lastErr)
}
