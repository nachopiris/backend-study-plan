// Package appv2 is the "new" version of the customer-facing app: it only
// ever reads and writes the `full_name` column, and never references `name`
// at all. It is deployed and runs concurrently with app v1 (template/appv1)
// during the expand phase: v2 does not need to know `name` exists, because
// the sync trigger created in migrations/002_expand.sql keeps `name`
// consistent for app v1's benefit on every write v2 makes.
//
// This is a deliberate design choice: an alternative strategy would have
// app v2 write to both columns explicitly from Go code (a "dual write")
// instead of relying on a database trigger. See the README's reflection
// questions for the tradeoffs between the two. Because v2's SQL never
// mentions `name`, it keeps working unchanged after migrations/003_contract.sql
// drops that column — that is the property this design is chosen for.
package appv2

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Customer is the shape app v2 understands.
type Customer struct {
	ID       int64
	FullName string
}

// Store wraps a pgx connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore builds a Store backed by pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// CreateCustomer inserts a new customer with the given full name, writing
// only to full_name.
func (s *Store) CreateCustomer(ctx context.Context, fullName string) (int64, error) {
	return 0, errors.New("not implemented")
}

// UpdateCustomerFullName updates an existing customer's full name, writing
// only to full_name.
func (s *Store) UpdateCustomerFullName(ctx context.Context, id int64, fullName string) error {
	return errors.New("not implemented")
}

// GetCustomer reads a customer by id using full_name.
func (s *Store) GetCustomer(ctx context.Context, id int64) (Customer, error) {
	return Customer{}, errors.New("not implemented")
}
