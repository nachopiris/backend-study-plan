// Package checkout implements two strategies for selling the last unit(s) of
// stock safely under concurrent buyers: optimistic concurrency control (a
// version column plus bounded retry) and pessimistic concurrency control
// (SELECT ... FOR UPDATE).
package checkout

import (
	"context"
	"errors"

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

// CheckoutOptimistic attempts to buy one unit of sku for buyerID using
// optimistic concurrency control: read the current stock and version, and if
// stock is at least 1, try to apply the decrement with an UPDATE that only
// succeeds if the version has not changed since you read it (`UPDATE
// inventory SET stock = stock - 1, version = version + 1 WHERE sku = $1 AND
// version = $2`). If that UPDATE affects zero rows, someone else won the
// race for this version — re-read and retry, up to maxRetries times.
//
// On success, CheckoutOptimistic also inserts a row into orders for (sku,
// buyerID), in the same transaction as the stock decrement.
//
// Returns (true, nil) if the purchase succeeded, (false, nil) if sku was out
// of stock (this is an expected, ordinary outcome under contention for the
// last unit — not an error), or a non-nil error for an infrastructure
// failure or for exhausting maxRetries while stock was still available
// (which signals maxRetries was too low for the observed contention, not
// that the item sold out).
func (s *Store) CheckoutOptimistic(ctx context.Context, sku, buyerID string, maxRetries int) (bool, error) {
	return false, errors.New("not implemented")
}

// CheckoutPessimistic attempts to buy one unit of sku for buyerID using
// pessimistic concurrency control: inside a transaction, `SELECT stock FROM
// inventory WHERE sku = $1 FOR UPDATE` to lock the row before reading it: a
// second concurrent call targeting the same sku blocks until the first
// transaction commits or rolls back. If stock is at least 1, decrement it,
// insert the order row, and commit; otherwise roll back without changing
// anything.
//
// Same return contract as CheckoutOptimistic: (true, nil) on a successful
// purchase, (false, nil) if sku was out of stock, non-nil error otherwise.
func (s *Store) CheckoutPessimistic(ctx context.Context, sku, buyerID string) (bool, error) {
	return false, errors.New("not implemented")
}
