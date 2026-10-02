package outbox

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrDeadLettered is returned by Consumer.Process when msg has just failed
// for the maxAttempts-th time and has been moved to dead_letters: the caller
// must NOT retry it again.
var ErrDeadLettered = errors.New("message dead-lettered after exhausting delivery attempts")

// Effect performs the business-logic side effect for one message, using the
// SAME transaction Process will use to record the message as processed.
// Returning a non-nil error aborts that transaction: neither the side
// effect's writes nor the processed_messages row are persisted.
type Effect func(ctx context.Context, tx pgx.Tx, msg Message) error

// Consumer enforces idempotent, at-most-N-attempts processing of messages
// coming from a Publisher (or any message bus): no matter how many times
// Process is called for the same Message.ID, effect runs at most once
// successfully.
type Consumer struct {
	pool        *pgxpool.Pool
	maxAttempts int
}

// NewConsumer builds a Consumer backed by pool, dead-lettering a message
// after maxAttempts failed attempts.
func NewConsumer(pool *pgxpool.Pool, maxAttempts int) *Consumer {
	return &Consumer{pool: pool, maxAttempts: maxAttempts}
}

// Process handles one incoming message, exactly as a message-bus handler
// would call it for every delivery (including redeliveries and duplicates).
//
// Contract:
//   - If msg.ID is already present in processed_messages, Process returns
//     nil immediately WITHOUT calling effect again: the side effect already
//     happened for this message, so this call is a harmless duplicate.
//   - If msg.ID is already present in dead_letters, Process returns
//     ErrDeadLettered immediately WITHOUT calling effect: this message is
//     permanently abandoned and must not be retried.
//   - Otherwise, Process runs effect inside ONE transaction together with
//     `INSERT INTO processed_messages (message_id) VALUES ($1) ON CONFLICT
//     DO NOTHING`, and commits only if effect returns nil.
//   - If effect returns a non-nil error, Process records the failed attempt
//     in delivery_attempts (incrementing its counter) and returns that
//     error — UNLESS this was the maxAttempts-th failure for msg.ID, in
//     which case Process instead inserts msg into dead_letters (with the
//     failure reason and the attempt count) and returns ErrDeadLettered.
//
// Not implemented yet: always returns an error without touching the
// database or calling effect.
func (c *Consumer) Process(ctx context.Context, msg Message, effect Effect) error {
	return errors.New("not implemented")
}
