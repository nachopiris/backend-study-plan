// Package outbox implements the transactional outbox pattern (reliably
// publishing a domain event exactly when — and only when — the business
// transaction that caused it commits) and an idempotent consumer (safely
// handling the at-least-once redelivery that the outbox's own relay implies).
package outbox

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps the Postgres pool used for placing orders.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore builds a Store backed by pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// orderPlacedPayload is the JSON body of the "OrderPlaced" outbox event.
type orderPlacedPayload struct {
	OrderID     string `json:"order_id"`
	CustomerID  string `json:"customer_id"`
	AmountCents int64  `json:"amount_cents"`
}

// PlaceOrder creates a new order and, in the SAME database transaction,
// inserts the corresponding "OrderPlaced" outbox row that Relay will later
// publish.
//
// Contract:
//   - Both the INSERT into orders and the INSERT into outbox happen inside
//     ONE transaction: either both are visible after PlaceOrder returns nil,
//     or neither is (on any error, nothing is persisted). This is the entire
//     point of the pattern: it is structurally impossible to end up with an
//     order that has no corresponding outbox row, or an outbox row with no
//     corresponding order — the "dual write" problem (see
//     NaivePlaceOrder in dualwrite_demo.go for what happens without this).
//   - The outbox row's event_type is "OrderPlaced" and its payload is a JSON
//     object with the order's id, customer_id, and amount_cents.
//   - On success, returns the new order's id.
//
// Not implemented yet: always returns an error without writing anything.
func (s *Store) PlaceOrder(ctx context.Context, customerID string, amountCents int64) (string, error) {
	return "", errors.New("not implemented")
}

// marshalOrderPlaced is a small helper you may use from PlaceOrder to build
// the outbox payload; it is already implemented.
func marshalOrderPlaced(orderID, customerID string, amountCents int64) ([]byte, error) {
	return json.Marshal(orderPlacedPayload{OrderID: orderID, CustomerID: customerID, AmountCents: amountCents})
}
