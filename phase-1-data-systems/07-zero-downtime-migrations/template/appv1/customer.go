// Package appv1 is the "old" version of the customer-facing app: it only
// ever reads and writes the `name` column. It is GIVEN, working code — do
// not change it. It keeps running completely unmodified throughout the
// expand phase and the backfill, relying entirely on the sync trigger
// created in migrations/002_expand.sql to keep `full_name` consistent for
// the benefit of app v2 (template/appv2), which it has no knowledge of.
package appv1

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Customer is the shape app v1 understands.
type Customer struct {
	ID   int64
	Name string
}

// Store wraps a pgx connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore builds a Store backed by pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// CreateCustomer inserts a new customer with the given name.
func (s *Store) CreateCustomer(ctx context.Context, name string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO customers (name) VALUES ($1) RETURNING id`,
		name,
	).Scan(&id)
	return id, err
}

// UpdateCustomerName updates an existing customer's name.
func (s *Store) UpdateCustomerName(ctx context.Context, id int64, name string) error {
	_, err := s.pool.Exec(ctx, `UPDATE customers SET name = $1 WHERE id = $2`, name, id)
	return err
}

// GetCustomer reads a customer by id.
func (s *Store) GetCustomer(ctx context.Context, id int64) (Customer, error) {
	var c Customer
	err := s.pool.QueryRow(ctx,
		`SELECT id, name FROM customers WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name)
	return c, err
}
