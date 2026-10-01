package outbox

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NaivePlaceOrder is GIVEN and fully working: it is deliberately correct Go
// code with an INCORRECT distributed-systems shape, kept here only so its
// test can show you the dual-write problem actually happening, instead of
// just describing it. Do not fix it, and do not use this pattern anywhere
// else in this practice (that is what Store.PlaceOrder + Relay are for).
//
// It does the two things a naive "save to the database and publish to the
// bus" implementation does, in the naive order: insert the order, publish an
// event about it directly (no outbox table involved), and only THEN commit
// the transaction. When failCommit is true, it rolls back instead of
// committing — modeling a transaction that fails or is lost AFTER an
// external side effect (the publish) already became visible to the rest of
// the world. The database ends up with no record of the order, while
// whatever subscribed to publisher already "knows" about it: a ghost event.
func NaivePlaceOrder(ctx context.Context, pool *pgxpool.Pool, publisher Publisher, customerID string, amountCents int64, failCommit bool) (orderID string, err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once committed

	orderID = newID("order")
	if _, err := tx.Exec(ctx, `INSERT INTO orders (id, customer_id, amount_cents) VALUES ($1, $2, $3)`,
		orderID, customerID, amountCents); err != nil {
		return "", fmt.Errorf("insert order: %w", err)
	}

	payload, err := marshalOrderPlaced(orderID, customerID, amountCents)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	// NAIVE: publishing directly, with no outbox row backing it. If the
	// commit below fails, this publish already happened and cannot be
	// taken back.
	if err := publisher.Publish(ctx, Message{
		ID:          newID("msg"),
		AggregateID: orderID,
		EventType:   "OrderPlaced",
		Payload:     payload,
	}); err != nil {
		return "", fmt.Errorf("publish: %w", err)
	}

	if failCommit {
		return orderID, fmt.Errorf("simulated commit failure (order %s was never persisted, but the event above was already published)", orderID)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}
	return orderID, nil
}
