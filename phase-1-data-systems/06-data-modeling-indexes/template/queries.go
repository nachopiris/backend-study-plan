// Package indexes models the core e-commerce catalog (customers, orders,
// order_items, payments, inventory) and exposes three read queries that must
// be served by an index against a realistically sized dataset, not a
// sequential scan.
//
// The schema itself is also part of this exercise: see
// migrations/001_schema.sql.
package indexes

import (
	"context"
	"errors"
	"time"

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

// Order is one row of the orders table.
type Order struct {
	ID         int64
	CustomerID int64
	Status     string
	TotalCents int64
	CreatedAt  time.Time
}

// Payment is one row of the payments table.
type Payment struct {
	ID          int64
	OrderID     int64
	Status      string
	AmountCents int64
	CreatedAt   time.Time
}

// listRecentOrdersQuery must hold the exact SQL executed by ListRecentOrders,
// with its parameters in $1, $2, ... order matching the call below. Assign it
// from your implementation (do not query with a separate inline string) so
// the test suite can run EXPLAIN on the very statement used in production and
// verify it hits an index instead of a sequential scan over `orders`.
var listRecentOrdersQuery string

// ListRecentOrders returns the limit most recent orders for customerID,
// ordered by created_at descending (most recent first).
func (s *Store) ListRecentOrders(ctx context.Context, customerID int64, limit int) ([]Order, error) {
	return nil, errors.New("not implemented")
}

// stalePendingPaymentsQuery must hold the exact SQL executed by
// StalePendingPayments. Same contract as listRecentOrdersQuery.
var stalePendingPaymentsQuery string

// StalePendingPayments returns every payment still in status "pending" whose
// created_at is strictly older than olderThan — the rows a reconciliation job
// would want to investigate or retry.
func (s *Store) StalePendingPayments(ctx context.Context, olderThan time.Time) ([]Payment, error) {
	return nil, errors.New("not implemented")
}

// lowStockSKUsQuery must hold the exact SQL executed by LowStockSKUs. Same
// contract as the other two query variables.
var lowStockSKUsQuery string

// LowStockSKUs returns every SKU in inventory whose quantity is at or below
// threshold, ordered by sku.
func (s *Store) LowStockSKUs(ctx context.Context, threshold int) ([]string, error) {
	return nil, errors.New("not implemented")
}
