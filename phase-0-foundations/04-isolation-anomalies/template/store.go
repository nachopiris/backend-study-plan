// Package isolation demonstrates classic concurrency anomalies against a real
// Postgres database (lost update, write skew) and three strategies to prevent
// them: a single atomic statement, explicit row locking, and SERIALIZABLE
// isolation with retry.
package isolation

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps a pgx connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore builds a Store backed by pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// IncrementUnsafe increments the named counter using a classic
// read-modify-write sequence: SELECT the current value, add one in Go code,
// then UPDATE.
//
// This function is GIVEN, deliberately incorrect code — do not "fix" it. It
// exists so the test suite can demonstrate the lost update anomaly: under
// concurrent calls against the same row, two callers can read the same value
// before either of them writes back, and one increment silently disappears.
// The small sleep below exists only to widen the race window enough to make
// the anomaly observable in a fast test; a real bug does not need to sleep to
// exist, it can just as easily happen without it.
func (s *Store) IncrementUnsafe(ctx context.Context, name string) error {
	var value int64
	if err := s.pool.QueryRow(ctx, `SELECT value FROM counters WHERE name = $1`, name).Scan(&value); err != nil {
		return err
	}
	time.Sleep(time.Millisecond)
	_, err := s.pool.Exec(ctx, `UPDATE counters SET value = $1 WHERE name = $2`, value+1, name)
	return err
}

// IncrementAtomic increments the named counter using a single SQL statement,
// letting Postgres perform the read-modify-write atomically at the row level.
// There should be no application-level read of the current value, no
// transaction management, and no race window: `UPDATE ... SET value = value + 1`
// is itself atomic from the point of view of concurrent callers.
func (s *Store) IncrementAtomic(ctx context.Context, name string) error {
	return errors.New("not implemented")
}

// IncrementForUpdate increments the named counter inside a transaction that
// first locks the row with `SELECT ... FOR UPDATE`, reads the current value,
// then updates it, then commits. A second concurrent call targeting the same
// row must block until the first transaction commits or rolls back before it
// can acquire the lock — this serializes access to that one row.
func (s *Store) IncrementForUpdate(ctx context.Context, name string) error {
	return errors.New("not implemented")
}

// IncrementSerializable increments the named counter inside a SERIALIZABLE
// transaction: SELECT the current value, add one, UPDATE, commit. If Postgres
// reports a serialization failure (SQLSTATE 40001) at any point — including
// on commit — the whole transaction must be retried from the beginning, up to
// maxRetries times. If it still fails after that many attempts, return the
// last error.
func (s *Store) IncrementSerializable(ctx context.Context, name string, maxRetries int) error {
	return errors.New("not implemented")
}

// isSerializationFailure reports whether err is a Postgres serialization
// failure (SQLSTATE 40001) — the error SERIALIZABLE transactions return when
// they detect that concurrent execution would not be equivalent to running
// serially, and which callers are expected to handle by retrying.
func isSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "40001"
}

// withSerializableRetry is a helper you may use (or ignore and write your own)
// from both IncrementSerializable and GoOffCall: it runs fn inside a
// SERIALIZABLE transaction, retrying the entire transaction when fn or the
// commit fails with a serialization failure, up to maxRetries times.
func withSerializableRetry(ctx context.Context, pool *pgxpool.Pool, maxRetries int, fn func(tx pgx.Tx) error) error {
	return errors.New("not implemented")
}
