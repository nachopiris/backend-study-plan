package outbox

import (
	"context"
	"testing"
)

// TestNaivePlaceOrder_DemonstratesTheDualWriteProblem does not test any code
// you write: it exercises the GIVEN NaivePlaceOrder to show you the exact
// failure this whole practice exists to prevent. Read it before you write
// anything else.
func TestNaivePlaceOrder_DemonstratesTheDualWriteProblem(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	pub := NewInMemoryPublisher()

	orderID, err := NaivePlaceOrder(ctx, pool, pub, "customer-ghost", 1500, true /* failCommit */)
	if err == nil {
		t.Fatal("NaivePlaceOrder() = nil error, want an error (the commit was configured to fail)")
	}
	if orderID == "" {
		t.Fatal("NaivePlaceOrder did not return the order id it attempted to use, even though it failed")
	}

	// The "ghost event": the bus received a message about an order that, as
	// far as the database is concerned, never happened.
	delivered := pub.Delivered()
	if len(delivered) != 1 {
		t.Fatalf("publisher recorded %d deliveries, want exactly 1 (the naive publish happens before the failed commit)", len(delivered))
	}
	if delivered[0].AggregateID != orderID {
		t.Fatalf("delivered event is for order %q, want %q", delivered[0].AggregateID, orderID)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE id = $1`, orderID).Scan(&count); err != nil {
		t.Fatalf("querying orders: %v", err)
	}
	if count != 0 {
		t.Fatalf("order %q exists in the database with count=%d, want 0 — the transaction was supposed to roll back", orderID, count)
	}

	t.Logf("ghost event confirmed: %d subscriber(s) were told about order %q, which does not exist in the database", len(delivered), orderID)
}
