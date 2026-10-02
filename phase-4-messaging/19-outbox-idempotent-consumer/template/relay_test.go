package outbox

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestRelay_NeverLosesOrInventsEvents(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	store := NewStore(pool)

	const n = 10
	placed := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		orderID, err := store.PlaceOrder(ctx, fmt.Sprintf("customer-%d", i), int64(100*(i+1)))
		if err != nil {
			t.Fatalf("PlaceOrder: %v", err)
		}
		placed[orderID] = true
	}

	pub := NewInMemoryPublisher()
	relay := NewRelay(pool, pub, 4)

	drainRelay(t, relay)

	delivered := pub.Delivered()
	if len(delivered) != n {
		t.Fatalf("publisher delivered %d messages, want exactly %d (no loss, no invented events)", len(delivered), n)
	}
	seen := make(map[string]bool, n)
	for _, msg := range delivered {
		if seen[msg.AggregateID] {
			t.Errorf("order %q delivered more than once", msg.AggregateID)
		}
		seen[msg.AggregateID] = true
		if !placed[msg.AggregateID] {
			t.Errorf("delivered an event for order %q, which was never placed", msg.AggregateID)
		}
	}
	for orderID := range placed {
		if !seen[orderID] {
			t.Errorf("order %q was placed but never delivered", orderID)
		}
	}

	var unsent int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE sent_at IS NULL`).Scan(&unsent); err != nil {
		t.Fatalf("counting unsent rows: %v", err)
	}
	if unsent != 0 {
		t.Fatalf("outbox still has %d unsent row(s) after draining the relay", unsent)
	}
}

func TestRelay_PublishingTwice_YieldsExactlyOneEffectAtTheConsumer(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	store := NewStore(pool)

	orderID, err := store.PlaceOrder(ctx, "customer-retry", 2500)
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	pub := NewInMemoryPublisher()
	pub.CrashAfterPublish(true)
	relay := NewRelay(pool, pub, 10)

	// First RunOnce: the fake publishes (records the delivery) but then
	// reports an error, simulating the relay dying before it could record
	// the row as sent — the row must still be unsent afterwards.
	if _, err := relay.RunOnce(ctx); err == nil {
		t.Fatal("RunOnce() = nil error, want an error (the publisher was configured to crash after publishing)")
	}
	var sentAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT sent_at FROM outbox WHERE aggregate_id = $1`, orderID).Scan(&sentAt); err != nil {
		t.Fatalf("querying outbox row: %v", err)
	}
	if sentAt != nil {
		t.Fatal("outbox row was marked sent despite the publish-time crash")
	}

	// Second RunOnce: no more crashing. The same row gets claimed and
	// published AGAIN — a real duplicate delivery, at the bus level.
	pub.CrashAfterPublish(false)
	sent, err := relay.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce (second, successful): %v", err)
	}
	if sent != 1 {
		t.Fatalf("RunOnce (second) sent %d rows, want 1", sent)
	}

	delivered := pub.Delivered()
	if len(delivered) != 2 {
		t.Fatalf("publisher recorded %d deliveries, want exactly 2 (the row really was published twice)", len(delivered))
	}
	if delivered[0].ID != delivered[1].ID {
		t.Fatalf("the two deliveries have different message IDs (%q vs %q): a downstream idempotent consumer cannot recognize them as the same message", delivered[0].ID, delivered[1].ID)
	}

	// Now feed BOTH deliveries through an idempotent Consumer and check that
	// the business side effect ran exactly once.
	consumer := NewConsumer(pool, 3)
	var runs int
	noopEffect := func(ctx context.Context, tx pgx.Tx, msg Message) error {
		runs++
		return nil
	}
	for _, msg := range delivered {
		if err := consumer.Process(ctx, msg, noopEffect); err != nil {
			t.Fatalf("Consumer.Process: %v", err)
		}
	}
	if runs != 1 {
		t.Fatalf("business effect ran %d times for %d deliveries of the SAME message, want exactly 1", runs, len(delivered))
	}
}

func TestRelay_TwoRelaysInParallel_NeverPublishTheSameRowConcurrently(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	store := NewStore(pool)

	const n = 30
	for i := 0; i < n; i++ {
		if _, err := store.PlaceOrder(ctx, fmt.Sprintf("customer-parallel-%d", i), 500); err != nil {
			t.Fatalf("PlaceOrder: %v", err)
		}
	}

	pub := NewInMemoryPublisher()
	relayA := NewRelay(pool, pub, 1)
	relayB := NewRelay(pool, pub, 1)

	deadline := time.Now().Add(30 * time.Second)
	var wg sync.WaitGroup
	wg.Add(2)
	run := func(r *Relay) {
		defer wg.Done()
		for time.Now().Before(deadline) {
			sent, err := r.RunOnce(ctx)
			if err != nil {
				t.Errorf("RunOnce: %v", err)
				return
			}
			if sent == 0 {
				var remaining int
				if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE sent_at IS NULL`).Scan(&remaining); err != nil {
					t.Errorf("counting unsent rows: %v", err)
					return
				}
				if remaining == 0 {
					return
				}
				time.Sleep(5 * time.Millisecond)
			}
		}
	}
	go run(relayA)
	go run(relayB)
	wg.Wait()

	delivered := pub.Delivered()
	seen := make(map[string]int, n)
	for _, msg := range delivered {
		seen[msg.AggregateID]++
	}
	if len(seen) != n {
		t.Fatalf("delivered events for %d distinct orders, want %d", len(seen), n)
	}
	for orderID, count := range seen {
		if count != 1 {
			t.Errorf("order %q was published %d times by the two parallel relays, want exactly 1 (SKIP LOCKED should prevent this)", orderID, count)
		}
	}
}

// drainRelay calls RunOnce until it reports nothing left to send, with a
// bounded number of iterations so a buggy RunOnce that never drains the
// table fails the test instead of hanging it.
func drainRelay(t *testing.T, r *Relay) {
	t.Helper()
	for i := 0; i < 1000; i++ {
		sent, err := r.RunOnce(context.Background())
		if err != nil {
			t.Fatalf("RunOnce: %v", err)
		}
		if sent == 0 {
			return
		}
	}
	t.Fatal("drainRelay: RunOnce kept reporting sent rows after 1000 calls — does it ever stop claiming rows?")
}
