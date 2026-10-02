package checkout

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func uniqueSKU(t testing.TB, suffix string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d-%s", t.Name(), time.Now().UnixNano(), suffix)
}

// runConcurrentCheckouts calls checkout(buyerID) from n concurrent goroutines
// and returns, for each call, whether it reported success, alongside any
// error. It is shared by the optimistic and pessimistic "exactly one winner"
// tests below.
func runConcurrentCheckouts(n int, checkout func(buyerID string) (bool, error)) ([]bool, []error) {
	results := make([]bool, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			buyerID := fmt.Sprintf("buyer-%d", i)
			ok, err := checkout(buyerID)
			results[i] = ok
			errs[i] = err
		}(i)
	}
	wg.Wait()
	return results, errs
}

func TestCheckoutOptimistic_ExactlyOneWinner(t *testing.T) {
	pool := requireDB(t)
	store := NewStore(pool)
	ctx := context.Background()
	sku := uniqueSKU(t, "sku")

	if _, err := pool.Exec(ctx, `INSERT INTO inventory (sku, stock, version) VALUES ($1, 1, 0)`, sku); err != nil {
		t.Fatalf("seeding inventory: %v", err)
	}

	const buyers = 100
	const maxRetries = 200
	results, errs := runConcurrentCheckouts(buyers, func(buyerID string) (bool, error) {
		return store.CheckoutOptimistic(ctx, sku, buyerID, maxRetries)
	})

	wins := 0
	for i, err := range errs {
		if err != nil {
			t.Fatalf("CheckoutOptimistic buyer %d: %v (if this happens under heavy contention, consider "+
				"whether maxRetries is high enough, or whether your implementation re-reads fresh stock "+
				"before each retry instead of retrying blindly)", i, err)
		}
		if results[i] {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("expected exactly 1 winner among %d concurrent buyers for 1 unit of stock, got %d", buyers, wins)
	}

	var stock int
	if err := pool.QueryRow(ctx, `SELECT stock FROM inventory WHERE sku = $1`, sku).Scan(&stock); err != nil {
		t.Fatalf("reading final stock: %v", err)
	}
	if stock != 0 {
		t.Fatalf("expected final stock to be 0, got %d", stock)
	}

	var orderCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE sku = $1`, sku).Scan(&orderCount); err != nil {
		t.Fatalf("counting orders: %v", err)
	}
	if orderCount != 1 {
		t.Fatalf("expected exactly 1 order row for %q, got %d", sku, orderCount)
	}
}

func TestCheckoutPessimistic_ExactlyOneWinner(t *testing.T) {
	pool := requireDB(t)
	store := NewStore(pool)
	ctx := context.Background()
	sku := uniqueSKU(t, "sku")

	if _, err := pool.Exec(ctx, `INSERT INTO inventory (sku, stock, version) VALUES ($1, 1, 0)`, sku); err != nil {
		t.Fatalf("seeding inventory: %v", err)
	}

	const buyers = 100
	results, errs := runConcurrentCheckouts(buyers, func(buyerID string) (bool, error) {
		return store.CheckoutPessimistic(ctx, sku, buyerID)
	})

	wins := 0
	for i, err := range errs {
		if err != nil {
			t.Fatalf("CheckoutPessimistic buyer %d: %v", i, err)
		}
		if results[i] {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("expected exactly 1 winner among %d concurrent buyers for 1 unit of stock, got %d", buyers, wins)
	}

	var stock int
	if err := pool.QueryRow(ctx, `SELECT stock FROM inventory WHERE sku = $1`, sku).Scan(&stock); err != nil {
		t.Fatalf("reading final stock: %v", err)
	}
	if stock != 0 {
		t.Fatalf("expected final stock to be 0, got %d", stock)
	}

	var orderCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE sku = $1`, sku).Scan(&orderCount); err != nil {
		t.Fatalf("counting orders: %v", err)
	}
	if orderCount != 1 {
		t.Fatalf("expected exactly 1 order row for %q, got %d", sku, orderCount)
	}
}

// BenchmarkCheckoutOptimistic and BenchmarkCheckoutPessimistic measure
// throughput under contention over a SKU with ample stock (so neither
// strategy ever hits the "sold out" path mid-benchmark): run with
//
//	go test -bench . -run '^$' -benchtime 2s
//
// and compare ns/op between the two. See the README for what to expect and
// why.

func BenchmarkCheckoutOptimistic(b *testing.B) {
	pool := requireDBForBenchmark(b)
	store := NewStore(pool)
	ctx := context.Background()
	sku := uniqueSKU(b, "bench-optimistic")

	if _, err := pool.Exec(ctx, `INSERT INTO inventory (sku, stock, version) VALUES ($1, $2, 0)`, sku, b.N); err != nil {
		b.Fatalf("seeding inventory: %v", err)
	}

	var buyerSeq int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buyerID := fmt.Sprintf("buyer-%d", atomic.AddInt64(&buyerSeq, 1))
			ok, err := store.CheckoutOptimistic(ctx, sku, buyerID, 100)
			if err != nil {
				b.Fatalf("CheckoutOptimistic: %v", err)
			}
			if !ok {
				b.Fatal("unexpected out-of-stock during benchmark; stock was seeded to b.N, this should not happen")
			}
		}
	})
}

func BenchmarkCheckoutPessimistic(b *testing.B) {
	pool := requireDBForBenchmark(b)
	store := NewStore(pool)
	ctx := context.Background()
	sku := uniqueSKU(b, "bench-pessimistic")

	if _, err := pool.Exec(ctx, `INSERT INTO inventory (sku, stock, version) VALUES ($1, $2, 0)`, sku, b.N); err != nil {
		b.Fatalf("seeding inventory: %v", err)
	}

	var buyerSeq int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buyerID := fmt.Sprintf("buyer-%d", atomic.AddInt64(&buyerSeq, 1))
			ok, err := store.CheckoutPessimistic(ctx, sku, buyerID)
			if err != nil {
				b.Fatalf("CheckoutPessimistic: %v", err)
			}
			if !ok {
				b.Fatal("unexpected out-of-stock during benchmark; stock was seeded to b.N, this should not happen")
			}
		}
	})
}
