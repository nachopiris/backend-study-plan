package outbox

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Relay periodically moves unsent outbox rows to a Publisher.
type Relay struct {
	pool      *pgxpool.Pool
	publisher Publisher
	batchSize int
}

// NewRelay builds a Relay that claims up to batchSize unsent outbox rows per
// RunOnce call, published through publisher.
func NewRelay(pool *pgxpool.Pool, publisher Publisher, batchSize int) *Relay {
	return &Relay{pool: pool, publisher: publisher, batchSize: batchSize}
}

// RunOnce claims up to r.batchSize unsent outbox rows (sent_at IS NULL),
// publishes each one through r.publisher, and marks the batch as sent — all
// in ONE transaction.
//
// Contract:
//   - Claim rows with `SELECT ... FOR UPDATE SKIP LOCKED`, ordered by id, so
//     that two Relay instances (or two goroutines) calling RunOnce
//     concurrently against the same table never claim the same row: each
//     row is claimed by at most one concurrent RunOnce call at a time.
//   - Each claimed row's Message.ID must be a stable string derived from the
//     outbox row's own id (e.g. "outbox_<id>"), NOT a freshly generated
//     random ID — a row that gets published again later (see below) must
//     publish with the EXACT SAME Message.ID every time, so a downstream
//     idempotent consumer can recognize it as a duplicate.
//   - If r.publisher.Publish fails for any row in the batch, RunOnce rolls
//     back the ENTIRE transaction immediately: no row in this batch is
//     marked sent, not even ones published successfully earlier in the same
//     batch. They will all be claimed and republished on a later call. This
//     is deliberate: it keeps the relay's own logic simple by pushing the
//     "publishing the same row twice is safe" requirement entirely onto the
//     consumer side (see consumer.go) — which is exactly why this whole
//     practice also builds an idempotent consumer.
//   - On success, commits the transaction (marking every claimed row's
//     sent_at) and returns the number of rows sent.
//   - If there are no unsent rows, RunOnce returns (0, nil) — this is not an
//     error.
//
// Not implemented yet: always returns an error without touching the
// database or the publisher.
func (r *Relay) RunOnce(ctx context.Context) (int, error) {
	return 0, errors.New("not implemented")
}
