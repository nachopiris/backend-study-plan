package outbox

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestConsumer_Process_IsIdempotentByMessageID(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	consumer := NewConsumer(pool, 3)

	msg := Message{ID: newID("msg"), AggregateID: "order-x", EventType: "OrderPlaced", Payload: []byte(`{}`)}

	var runs int
	effect := func(ctx context.Context, tx pgx.Tx, m Message) error {
		runs++
		return nil
	}

	for i := 0; i < 5; i++ {
		if err := consumer.Process(ctx, msg, effect); err != nil {
			t.Fatalf("Process call %d: %v", i, err)
		}
	}
	if runs != 1 {
		t.Fatalf("effect ran %d times across 5 deliveries of the same message, want 1", runs)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM processed_messages WHERE message_id = $1`, msg.ID).Scan(&count); err != nil {
		t.Fatalf("querying processed_messages: %v", err)
	}
	if count != 1 {
		t.Fatalf("processed_messages has %d row(s) for message %q, want 1", count, msg.ID)
	}
}

func TestConsumer_Process_EffectSideEffectCommitsWithProcessedMarker(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	consumer := NewConsumer(pool, 3)

	msg := Message{ID: newID("msg"), AggregateID: "order-y", EventType: "OrderPlaced", Payload: []byte(`{}`)}

	effect := func(ctx context.Context, tx pgx.Tx, m Message) error {
		_, err := tx.Exec(ctx, `INSERT INTO orders (id, customer_id, amount_cents) VALUES ($1, $2, $3)`,
			m.AggregateID, "customer-y", 999)
		return err
	}
	if err := consumer.Process(ctx, msg, effect); err != nil {
		t.Fatalf("Process: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE id = $1`, "order-y").Scan(&count); err != nil {
		t.Fatalf("querying orders: %v", err)
	}
	if count != 1 {
		t.Fatalf("effect's own write (order row) was not committed: count=%d, want 1", count)
	}
}

func TestConsumer_Process_PoisonMessageEndsInDeadLetterAfterMaxAttempts(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	const maxAttempts = 3
	consumer := NewConsumer(pool, maxAttempts)

	msg := Message{ID: newID("msg"), AggregateID: "order-poison", EventType: "OrderPlaced", Payload: []byte(`{"poison":true}`)}
	boom := errors.New("effect always fails")
	alwaysFails := func(ctx context.Context, tx pgx.Tx, m Message) error {
		return boom
	}

	for attempt := 1; attempt < maxAttempts; attempt++ {
		err := consumer.Process(ctx, msg, alwaysFails)
		if !errors.Is(err, boom) {
			t.Fatalf("attempt %d: Process() = %v, want the effect's own error (not dead-lettered yet)", attempt, err)
		}
	}

	// The maxAttempts-th failure must dead-letter the message.
	err := consumer.Process(ctx, msg, alwaysFails)
	if !errors.Is(err, ErrDeadLettered) {
		t.Fatalf("final attempt: Process() = %v, want ErrDeadLettered", err)
	}

	var dlqCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM dead_letters WHERE message_id = $1`, msg.ID).Scan(&dlqCount); err != nil {
		t.Fatalf("querying dead_letters: %v", err)
	}
	if dlqCount != 1 {
		t.Fatalf("dead_letters has %d row(s) for message %q, want 1", dlqCount, msg.ID)
	}

	// A dead-lettered message must stay dead: it is never retried again,
	// even if called again.
	var furtherCalls int
	neverCalled := func(ctx context.Context, tx pgx.Tx, m Message) error {
		furtherCalls++
		return nil
	}
	if err := consumer.Process(ctx, msg, neverCalled); !errors.Is(err, ErrDeadLettered) {
		t.Fatalf("Process() on an already dead-lettered message = %v, want ErrDeadLettered", err)
	}
	if furtherCalls != 0 {
		t.Fatalf("effect was called %d times for an already dead-lettered message, want 0", furtherCalls)
	}
}
