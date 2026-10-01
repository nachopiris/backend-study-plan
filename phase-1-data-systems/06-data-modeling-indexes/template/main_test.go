package indexes

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testPool   *pgxpool.Pool
	skipReason string

	seedOnce sync.Once
	seedErr  error
)

// TestMain starts a real Postgres container (via testcontainers-go), running
// migrations/001_schema.sql as its init script, unless -short was passed or
// Docker is not available — in either case every test skips itself with a
// clear reason instead of failing.
func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		skipReason = "skipping data-modeling-indexes tests: -short flag set (these tests require Docker + Postgres)"
		os.Exit(m.Run())
	}

	if err := exec.Command("docker", "info").Run(); err != nil {
		skipReason = fmt.Sprintf("skipping data-modeling-indexes tests: Docker does not appear to be available: %v", err)
		os.Exit(m.Run())
	}

	ctx := context.Background()

	migrationsPath, err := filepath.Abs(filepath.Join("migrations", "001_schema.sql"))
	if err != nil {
		fmt.Println("failed to resolve migrations path:", err)
		os.Exit(1)
	}

	pgContainer, err := tcpostgres.Run(ctx, "postgres:17",
		tcpostgres.WithInitScripts(migrationsPath),
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

// seed statements, run once for the whole package. Deliberately expressed as
// set-based INSERT ... SELECT over generate_series instead of one row at a
// time from Go: seeding ~150k rows this way takes a fraction of a second.
// Every statement here assumes the schema described in
// migrations/001_schema.sql exists; if you have not written it yet, seeding
// fails with a clear "relation does not exist" error — that failure is the
// expected red state, not a bug in this test file.
var seedStatements = []string{
	`INSERT INTO customers (email, full_name)
	 SELECT 'customer' || i || '@example.com', 'Customer ' || i
	 FROM generate_series(1, 1000) AS i`,

	`INSERT INTO orders (customer_id, status, total_cents, created_at)
	 SELECT
	   ((i % 1000) + 1),
	   (ARRAY['pending','paid','shipped','cancelled'])[1 + (i % 4)],
	   (100 + (i % 5000))::bigint,
	   now() - ((i % 365) || ' days')::interval - ((i % 24) || ' hours')::interval
	 FROM generate_series(1, 100000) AS i`,

	`INSERT INTO order_items (order_id, sku, quantity, unit_price_cents)
	 SELECT o.id, 'SKU-' || (1 + (o.id % 200)), 1 + (o.id % 5), 500 + (o.id % 3000)
	 FROM orders o`,

	`INSERT INTO payments (order_id, status, amount_cents, created_at)
	 SELECT o.id,
	   CASE WHEN o.status = 'pending' THEN 'pending' ELSE 'completed' END,
	   o.total_cents,
	   o.created_at
	 FROM orders o`,

	`INSERT INTO inventory (sku, quantity)
	 SELECT 'SKU-' || i, (i * 37) % 500
	 FROM generate_series(1, 50000) AS i`,
}

// ensureSeeded runs the seed data exactly once for the whole test binary and
// fails the calling test with a clear message if seeding did not work —
// almost always because migrations/001_schema.sql has not been written yet.
func ensureSeeded(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := requireDB(t)

	seedOnce.Do(func() {
		ctx := context.Background()
		for i, stmt := range seedStatements {
			if _, err := pool.Exec(ctx, stmt); err != nil {
				seedErr = fmt.Errorf("seed statement %d failed (did you write migrations/001_schema.sql?): %w", i, err)
				return
			}
		}
	})
	if seedErr != nil {
		t.Fatalf("%v", seedErr)
	}
	return pool
}
