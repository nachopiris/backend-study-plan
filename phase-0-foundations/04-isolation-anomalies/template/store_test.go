package isolation

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func uniqueName(t *testing.T, suffix string) string {
	t.Helper()
	return t.Name() + "-" + suffix
}

func mustExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("setup exec failed: %v", err)
	}
}

func readCounter(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name string) int64 {
	t.Helper()
	var value int64
	if err := pool.QueryRow(ctx, `SELECT value FROM counters WHERE name = $1`, name).Scan(&value); err != nil {
		t.Fatalf("reading counter %q: %v", name, err)
	}
	return value
}

func TestIncrementUnsafe_DemonstratesLostUpdate(t *testing.T) {
	pool := requireDB(t)
	store := NewStore(pool)
	ctx := context.Background()
	name := uniqueName(t, "counter")
	mustExec(t, ctx, pool, `INSERT INTO counters (name, value) VALUES ($1, 0)`, name)

	const n = 50
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := store.IncrementUnsafe(ctx, name); err != nil {
				t.Errorf("IncrementUnsafe: %v", err)
			}
		}()
	}
	wg.Wait()

	got := readCounter(t, ctx, pool, name)
	if got == n {
		t.Fatalf(
			"expected IncrementUnsafe to lose at least one update under %d concurrent calls "+
				"(that is the lost-update anomaly this function exists to demonstrate), but the "+
				"final value was the fully correct %d",
			n, got,
		)
	}
	t.Logf("IncrementUnsafe lost %d of %d increments to the race (this is the expected anomaly)", n-got, n)
}

func TestIncrementAtomic_NoLostUpdates(t *testing.T) {
	pool := requireDB(t)
	store := NewStore(pool)
	ctx := context.Background()
	name := uniqueName(t, "counter")
	mustExec(t, ctx, pool, `INSERT INTO counters (name, value) VALUES ($1, 0)`, name)

	const n = 50
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- store.IncrementAtomic(ctx, name)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("IncrementAtomic: %v", err)
		}
	}

	if got := readCounter(t, ctx, pool, name); got != n {
		t.Fatalf("expected counter to equal %d after %d concurrent IncrementAtomic calls, got %d", n, n, got)
	}
}

func TestIncrementForUpdate_NoLostUpdates(t *testing.T) {
	pool := requireDB(t)
	store := NewStore(pool)
	ctx := context.Background()
	name := uniqueName(t, "counter")
	mustExec(t, ctx, pool, `INSERT INTO counters (name, value) VALUES ($1, 0)`, name)

	const n = 50
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- store.IncrementForUpdate(ctx, name)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("IncrementForUpdate: %v", err)
		}
	}

	if got := readCounter(t, ctx, pool, name); got != n {
		t.Fatalf("expected counter to equal %d after %d concurrent IncrementForUpdate calls, got %d", n, n, got)
	}
}

func TestIncrementSerializable_NoLostUpdates(t *testing.T) {
	pool := requireDB(t)
	store := NewStore(pool)
	ctx := context.Background()
	name := uniqueName(t, "counter")
	mustExec(t, ctx, pool, `INSERT INTO counters (name, value) VALUES ($1, 0)`, name)

	const n = 50
	const maxRetries = 20
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- store.IncrementSerializable(ctx, name, maxRetries)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("IncrementSerializable: %v (consider raising maxRetries if this is flaky under heavy load)", err)
		}
	}

	if got := readCounter(t, ctx, pool, name); got != n {
		t.Fatalf("expected counter to equal %d after %d concurrent IncrementSerializable calls, got %d", n, n, got)
	}
}
