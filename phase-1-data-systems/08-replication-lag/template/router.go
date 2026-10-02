// Package replication implements a read router over a Postgres primary and
// one streaming replica, demonstrating the stale-read anomaly asynchronous
// replication introduces and one way to prevent it: read-your-writes
// consistency via an LSN (log sequence number) token.
package replication

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NaiveRouter always serves reads from the replica, with no regard for
// replication lag between it and the primary.
//
// This is GIVEN, deliberately naive code — do not "fix" it. It exists so the
// test suite can demonstrate the anomaly your Router (below) needs to
// prevent: a caller that writes to the primary and immediately reads from
// the replica can observe a result that does not yet reflect their own
// write, simply because replication has not caught up yet.
type NaiveRouter struct {
	Primary *pgxpool.Pool
	Replica *pgxpool.Pool
}

// NewNaiveRouter builds a NaiveRouter over the given pools.
func NewNaiveRouter(primary, replica *pgxpool.Pool) *NaiveRouter {
	return &NaiveRouter{Primary: primary, Replica: replica}
}

// WriteNote inserts a note on the primary and returns its id.
func (r *NaiveRouter) WriteNote(ctx context.Context, body string) (int64, error) {
	var id int64
	err := r.Primary.QueryRow(ctx, `INSERT INTO notes (body) VALUES ($1) RETURNING id`, body).Scan(&id)
	return id, err
}

// CountNotesWithBody always reads from the replica, regardless of how
// recently the caller wrote to the primary.
func (r *NaiveRouter) CountNotesWithBody(ctx context.Context, body string) (int, error) {
	var count int
	err := r.Replica.QueryRow(ctx, `SELECT count(*) FROM notes WHERE body = $1`, body).Scan(&count)
	return count, err
}

// WriteToken is an opaque marker of how far the primary's write-ahead log
// had progressed at the moment of a particular write. Obtain one from
// Router.WriteNote and pass it to Router.ReadYourWriteCount.
type WriteToken string

// Router routes ordinary reads to the replica (reducing load on the
// primary), except when a caller needs read-your-writes consistency: for
// that, it uses a WriteToken obtained at write time to decide whether the
// replica has replayed far enough to reflect that write, falling back to the
// primary (after waiting at most maxWait for the replica to catch up) when
// it has not.
type Router struct {
	Primary *pgxpool.Pool
	Replica *pgxpool.Pool
}

// NewRouter builds a Router over the given pools.
func NewRouter(primary, replica *pgxpool.Pool) *Router {
	return &Router{Primary: primary, Replica: replica}
}

// WriteNote inserts a note on the primary and returns its id along with a
// WriteToken capturing the primary's write-ahead log position immediately
// after the write (`SELECT pg_current_wal_lsn()`), for later use with
// ReadYourWriteCount.
func (r *Router) WriteNote(ctx context.Context, body string) (int64, WriteToken, error) {
	return 0, "", errors.New("not implemented")
}

// CountNotesWithBody reads from the replica unconditionally. Use this for
// reads that do not need to reflect the caller's own very recent writes —
// for example, an aggregate dashboard where being a second or two behind is
// an acceptable, known tradeoff for not loading the primary.
func (r *Router) CountNotesWithBody(ctx context.Context, body string) (int, error) {
	return 0, errors.New("not implemented")
}

// ReadYourWriteCount returns the count of notes with the given body,
// guaranteeing the result reflects at least everything visible as of token:
// it checks the replica's replay position (`SELECT pg_last_wal_replay_lsn()`)
// against token, reads from the replica once the replica has caught up to
// (or past) it, and otherwise waits and re-checks, up to maxWait in total,
// before giving up and reading from the primary instead (which is always
// current by definition). ReadYourWriteCount must never block longer than
// approximately maxWait before falling back.
func (r *Router) ReadYourWriteCount(ctx context.Context, body string, token WriteToken, maxWait time.Duration) (int, error) {
	return 0, errors.New("not implemented")
}
