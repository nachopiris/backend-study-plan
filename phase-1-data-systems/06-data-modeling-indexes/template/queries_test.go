package indexes

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"testing"
	"time"
)

// explainNode is the subset of Postgres's EXPLAIN (FORMAT JSON) node shape
// this suite cares about.
type explainNode struct {
	NodeType     string        `json:"Node Type"`
	RelationName string        `json:"Relation Name"`
	Plans        []explainNode `json:"Plans"`
}

type explainRow struct {
	Plan explainNode `json:"Plan"`
}

// assertNoSeqScan runs EXPLAIN (FORMAT JSON) on query with the same args used
// against the real query, and fails the test if any node in the resulting
// plan is a sequential scan — meaning Postgres would read every row of that
// table instead of using an index.
func assertNoSeqScan(t *testing.T, ctx context.Context, query string, args ...any) {
	t.Helper()
	if query == "" {
		t.Fatal("query is empty: assign the corresponding *Query package variable from your implementation")
	}

	pool := requireDB(t)
	rows, err := pool.Query(ctx, "EXPLAIN (FORMAT JSON) "+query, args...)
	if err != nil {
		t.Fatalf("EXPLAIN failed: %v", err)
	}
	defer rows.Close()

	var plans []explainRow
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			t.Fatalf("scanning EXPLAIN output: %v", err)
		}
		if err := json.Unmarshal([]byte(raw), &plans); err != nil {
			t.Fatalf("parsing EXPLAIN JSON: %v", err)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("reading EXPLAIN rows: %v", err)
	}
	if len(plans) == 0 {
		t.Fatal("EXPLAIN returned no plan")
	}

	var walk func(explainNode)
	walk = func(n explainNode) {
		if n.NodeType == "Seq Scan" {
			t.Fatalf(
				"query uses a sequential scan on %q instead of an index — "+
					"add an index in migrations/001_schema.sql that serves this query",
				n.RelationName,
			)
		}
		for _, child := range n.Plans {
			walk(child)
		}
	}
	walk(plans[0].Plan)
}

func TestListRecentOrders(t *testing.T) {
	ctx := context.Background()
	pool := ensureSeeded(t)
	store := NewStore(pool)

	// Orders were seeded with customer_id = (i % 1000) + 1, so customer 1
	// owns every order whose generator index i is a multiple of 1000 — 100
	// orders out of the 100k seeded rows.
	const customerID = int64(1)
	const limit = 5

	var wantCount int
	var wantMostRecent time.Time
	if err := pool.QueryRow(ctx,
		`SELECT count(*), max(created_at) FROM orders WHERE customer_id = $1`,
		customerID,
	).Scan(&wantCount, &wantMostRecent); err != nil {
		t.Fatalf("oracle query failed: %v", err)
	}
	if wantCount < limit {
		t.Fatalf("test fixture assumption broken: expected customer %d to have at least %d orders, has %d", customerID, limit, wantCount)
	}

	orders, err := store.ListRecentOrders(ctx, customerID, limit)
	if err != nil {
		t.Fatalf("ListRecentOrders: %v", err)
	}
	if len(orders) != limit {
		t.Fatalf("expected %d orders, got %d", limit, len(orders))
	}
	if !orders[0].CreatedAt.Equal(wantMostRecent) {
		t.Fatalf("expected the most recent order's created_at to be %s, got %s", wantMostRecent, orders[0].CreatedAt)
	}
	for i, o := range orders {
		if o.CustomerID != customerID {
			t.Fatalf("order %d belongs to customer %d, want %d", i, o.CustomerID, customerID)
		}
		if i > 0 && orders[i-1].CreatedAt.Before(o.CreatedAt) {
			t.Fatalf("orders are not sorted by created_at descending: index %d (%s) comes after index %d (%s)", i, o.CreatedAt, i-1, orders[i-1].CreatedAt)
		}
	}

	assertNoSeqScan(t, ctx, listRecentOrdersQuery, customerID, limit)
}

func TestStalePendingPayments(t *testing.T) {
	ctx := context.Background()
	pool := ensureSeeded(t)
	store := NewStore(pool)

	// payments.created_at mirrors orders.created_at, which is spread
	// uniformly over the last ~365 days; cutting off at 30 days leaves a
	// large, statistically certain set of "stale" pending payments without
	// depending on any single hand-picked row.
	olderThan := time.Now().Add(-30 * 24 * time.Hour)

	var wantCount int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM payments WHERE status = 'pending' AND created_at < $1`,
		olderThan,
	).Scan(&wantCount); err != nil {
		t.Fatalf("oracle query failed: %v", err)
	}
	if wantCount == 0 {
		t.Fatal("test fixture assumption broken: expected at least one stale pending payment")
	}

	payments, err := store.StalePendingPayments(ctx, olderThan)
	if err != nil {
		t.Fatalf("StalePendingPayments: %v", err)
	}
	if len(payments) != wantCount {
		t.Fatalf("expected %d stale pending payments, got %d", wantCount, len(payments))
	}
	for _, p := range payments {
		if p.Status != "pending" {
			t.Fatalf("payment %d has status %q, want %q", p.ID, p.Status, "pending")
		}
		if !p.CreatedAt.Before(olderThan) {
			t.Fatalf("payment %d has created_at %s, which is not before %s", p.ID, p.CreatedAt, olderThan)
		}
	}

	assertNoSeqScan(t, ctx, stalePendingPaymentsQuery, olderThan)
}

func TestLowStockSKUs(t *testing.T) {
	ctx := context.Background()
	pool := ensureSeeded(t)
	store := NewStore(pool)

	const threshold = 5

	// inventory was seeded with quantity = (i * 37) % 500 for i in 1..50000;
	// recompute the expected set independently instead of asking Postgres,
	// so this test does not depend on the query under test to define
	// "correct".
	want := make(map[string]struct{})
	for i := 1; i <= 50000; i++ {
		if (i*37)%500 <= threshold {
			want[skuName(i)] = struct{}{}
		}
	}
	if len(want) == 0 {
		t.Fatal("test fixture assumption broken: expected at least one low-stock SKU")
	}

	skus, err := store.LowStockSKUs(ctx, threshold)
	if err != nil {
		t.Fatalf("LowStockSKUs: %v", err)
	}
	if len(skus) != len(want) {
		t.Fatalf("expected %d low-stock SKUs, got %d", len(want), len(skus))
	}
	got := make(map[string]struct{}, len(skus))
	for _, sku := range skus {
		got[sku] = struct{}{}
	}
	var missing []string
	for sku := range want {
		if _, ok := got[sku]; !ok {
			missing = append(missing, sku)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("missing expected low-stock SKUs (showing up to 5): %v", missing[:min(5, len(missing))])
	}

	assertNoSeqScan(t, ctx, lowStockSKUsQuery, threshold)
}

func skuName(i int) string {
	return "SKU-" + strconv.Itoa(i)
}
