// Package payments defines the payments module's domain types and the port
// (Charger) that other modules depend on. Its concrete in-memory adapter
// lives in the sibling internal/payments/memory package; callers outside this
// module (namely orders) must depend on this package's Charger interface
// only, never on the memory package.
package payments

import (
	"context"
	"errors"
)

// ErrPaymentDeclined is returned by Charge when the payment could not be
// completed.
var ErrPaymentDeclined = errors.New("payments: payment declined")

// Receipt represents a successful charge.
type Receipt struct {
	ID          string
	AmountCents int64
}

// Charger is the port other modules depend on to charge a customer.
type Charger interface {
	// Charge attempts to charge amountCents to customerID. On success it
	// returns a Receipt; if the charge cannot be completed, it returns
	// ErrPaymentDeclined.
	Charge(ctx context.Context, customerID string, amountCents int64) (Receipt, error)
}
