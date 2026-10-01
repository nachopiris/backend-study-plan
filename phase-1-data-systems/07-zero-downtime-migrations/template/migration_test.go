package migration

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"studyplan/phase1/migrations/appv1"
	"studyplan/phase1/migrations/appv2"
)

// TestZeroDowntimeMigration walks the whole expand -> backfill -> contract
// lifecycle in order, as three subtests. They are not independent: each one
// depends on the database state the previous one left behind, exactly like
// the real migration they model.
func TestZeroDowntimeMigration(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()

	t.Run("ExpandInterop", func(t *testing.T) {
		v1 := appv1.NewStore(pool)
		v2 := appv2.NewStore(pool)

		// v1 creates a row (writes only `name`); v2 must see it immediately
		// via full_name, which only the sync trigger can have populated.
		id1, err := v1.CreateCustomer(ctx, "Ada Lovelace")
		if err != nil {
			t.Fatalf("appv1.CreateCustomer: %v", err)
		}
		c2, err := v2.GetCustomer(ctx, id1)
		if err != nil {
			t.Fatalf("appv2.GetCustomer(%d): %v", id1, err)
		}
		if c2.FullName != "Ada Lovelace" {
			t.Fatalf("expected appv2 to read full_name %q for a row created by appv1, got %q", "Ada Lovelace", c2.FullName)
		}

		// v2 creates a row (writes only full_name); v1 must see it via
		// `name`.
		id2, err := v2.CreateCustomer(ctx, "Grace Hopper")
		if err != nil {
			t.Fatalf("appv2.CreateCustomer: %v", err)
		}
		c1, err := v1.GetCustomer(ctx, id2)
		if err != nil {
			t.Fatalf("appv1.GetCustomer(%d): %v", id2, err)
		}
		if c1.Name != "Grace Hopper" {
			t.Fatalf("expected appv1 to read name %q for a row created by appv2, got %q", "Grace Hopper", c1.Name)
		}

		// v1 updates its row; v2 must see the update (exercises the UPDATE
		// branch of the trigger, not just INSERT).
		if err := v1.UpdateCustomerName(ctx, id1, "Ada Lovelace (updated)"); err != nil {
			t.Fatalf("appv1.UpdateCustomerName: %v", err)
		}
		c2, err = v2.GetCustomer(ctx, id1)
		if err != nil {
			t.Fatalf("appv2.GetCustomer(%d) after update: %v", id1, err)
		}
		if c2.FullName != "Ada Lovelace (updated)" {
			t.Fatalf("expected appv2 to see appv1's update, got full_name %q", c2.FullName)
		}

		// v2 updates its row; v1 must see the update.
		if err := v2.UpdateCustomerFullName(ctx, id2, "Grace Hopper (updated)"); err != nil {
			t.Fatalf("appv2.UpdateCustomerFullName: %v", err)
		}
		c1, err = v1.GetCustomer(ctx, id2)
		if err != nil {
			t.Fatalf("appv1.GetCustomer(%d) after update: %v", id2, err)
		}
		if c1.Name != "Grace Hopper (updated)" {
			t.Fatalf("expected appv1 to see appv2's update, got name %q", c1.Name)
		}
	})

	t.Run("Backfill", func(t *testing.T) {
		var before int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM customers WHERE full_name IS NULL`).Scan(&before); err != nil {
			t.Fatalf("counting rows pending backfill: %v", err)
		}
		if before == 0 {
			t.Fatal("test fixture assumption broken: expected legacy rows (from migrations/001_initial.sql) still pending backfill")
		}

		// Deliberately does not evenly divide `before`, to exercise the
		// last, partial batch too.
		const batchSize = 47
		wantBatches := (before + batchSize - 1) / batchSize

		batches, err := Backfill(ctx, pool, batchSize)
		if err != nil {
			t.Fatalf("Backfill: %v", err)
		}
		if batches <= 1 {
			t.Fatalf("expected Backfill to run in multiple batches (batchSize=%d, %d rows pending), got %d batch(es) — a single UPDATE over the whole table is exactly what this practice asks you to avoid", batchSize, before, batches)
		}
		if batches != wantBatches {
			t.Fatalf("expected %d batches for %d rows with batchSize=%d, got %d", wantBatches, before, batchSize, batches)
		}

		var after int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM customers WHERE full_name IS NULL`).Scan(&after); err != nil {
			t.Fatalf("counting rows pending backfill after Backfill: %v", err)
		}
		if after != 0 {
			t.Fatalf("expected 0 rows with full_name IS NULL after Backfill, got %d", after)
		}
	})

	t.Run("Contract", func(t *testing.T) {
		applyMigrationFile(t, ctx, pool, "migrations/003_contract.sql")

		v2 := appv2.NewStore(pool)
		id, err := v2.CreateCustomer(ctx, "Katherine Johnson")
		if err != nil {
			t.Fatalf("appv2.CreateCustomer after contract: %v", err)
		}
		c, err := v2.GetCustomer(ctx, id)
		if err != nil {
			t.Fatalf("appv2.GetCustomer after contract: %v", err)
		}
		if c.FullName != "Katherine Johnson" {
			t.Fatalf("expected full_name %q after contract, got %q", "Katherine Johnson", c.FullName)
		}

		var hasNameColumn bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'customers' AND column_name = 'name'
			)`,
		).Scan(&hasNameColumn); err != nil {
			t.Fatalf("checking for customers.name: %v", err)
		}
		if hasNameColumn {
			t.Fatal("expected migrations/003_contract.sql to drop customers.name")
		}
	})
}

// applyMigrationFile executes every statement in a SQL file against pool, one
// at a time. It is a minimal multi-statement runner for the test harness
// only (production migration tooling would use a proper migration runner);
// it does not need to handle dollar-quoted function bodies because
// migrations/003_contract.sql only drops objects created earlier.
func applyMigrationFile(t *testing.T, ctx context.Context, pool *pgxpool.Pool, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	// Strip comments from the whole file BEFORE splitting on ";" — a comment
	// line is free-form prose and may itself contain a semicolon, which
	// would otherwise cut a statement in the wrong place.
	for _, stmt := range strings.Split(stripSQLComments(string(raw)), ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("executing statement from %s (%q): %v", path, stmt, err)
		}
	}
}

func stripSQLComments(block string) string {
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			lines[i] = line[:idx]
		}
	}
	return strings.Join(lines, "\n")
}
