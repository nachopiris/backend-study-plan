package migration

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testPool   *pgxpool.Pool
	skipReason string
)

// TestMain starts a real Postgres container (via testcontainers-go), running
// migrations/001_initial.sql and migrations/002_expand.sql as init scripts —
// that is, the test suite always starts from "expand already applied,
// nothing backfilled yet". migrations/003_contract.sql is applied later, by
// the tests themselves, once they have exercised the expand phase.
//
// Unless -short was passed or Docker is not available, in which case every
// test skips itself with a clear reason instead of failing.
func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		skipReason = "skipping zero-downtime-migrations tests: -short flag set (these tests require Docker + Postgres)"
		os.Exit(m.Run())
	}

	if err := exec.Command("docker", "info").Run(); err != nil {
		skipReason = fmt.Sprintf("skipping zero-downtime-migrations tests: Docker does not appear to be available: %v", err)
		os.Exit(m.Run())
	}

	ctx := context.Background()

	initial, err := filepath.Abs(filepath.Join("migrations", "001_initial.sql"))
	if err != nil {
		fmt.Println("failed to resolve migrations path:", err)
		os.Exit(1)
	}
	expand, err := filepath.Abs(filepath.Join("migrations", "002_expand.sql"))
	if err != nil {
		fmt.Println("failed to resolve migrations path:", err)
		os.Exit(1)
	}

	pgContainer, err := tcpostgres.Run(ctx, "postgres:17",
		tcpostgres.WithInitScripts(initial, expand),
		tcpostgres.WithDatabase("studyplan"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		fmt.Println("failed to start postgres container:", err)
		os.Exit(1)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Println("failed to build connection string:", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		fmt.Println("failed to create pgx pool:", err)
		os.Exit(1)
	}
	testPool = pool

	code := m.Run()

	pool.Close()
	_ = pgContainer.Terminate(ctx)
	os.Exit(code)
}

// requireDB returns the shared test pool, or skips the calling test with a
// clear reason if the environment does not support these tests.
func requireDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if skipReason != "" {
		t.Skip(skipReason)
	}
	return testPool
}
