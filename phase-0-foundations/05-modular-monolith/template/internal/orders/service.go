package orders

import (
	"context"
	"errors"

	"studyplan/phase0/modularmonolith/internal/inventory"
	"studyplan/phase0/modularmonolith/internal/payments"
)

// Service places orders by coordinating stock reservation and payment
// charging through their respective ports.
//
// This is the architectural rule this practice is about: Service depends only
// on inventory.Reserver and payments.Charger — the ports exported by each
// module's root package — and must never import
// studyplan/phase0/modularmonolith/internal/inventory/memory or
// studyplan/phase0/modularmonolith/internal/payments/memory. See
// architecture_test.go, which enforces this by parsing this package's
// imports.
type Service struct {
	inventory inventory.Reserver
	payments  payments.Charger
}

// NewService builds a Service backed by the given ports.
func NewService(inv inventory.Reserver, pay payments.Charger) *Service {
	return &Service{inventory: inv, payments: pay}
}

// PlaceOrder reserves stock for sku, charges the customer, and returns the
// resulting Order.
//
// Contract:
//   - If there isn't enough stock, PlaceOrder returns an error that wraps
//     inventory.ErrInsufficientStock (via %w or errors.Join) and must NOT
//     call Charge.
//   - If the stock reservation succeeds but the charge fails, PlaceOrder must
//     call Release on the reservation before returning — an order that was
//     never confirmed should not keep stock held — and the returned error
//     must wrap the payment error (e.g. payments.ErrPaymentDeclined).
//   - On success, PlaceOrder returns an Order with Status == StatusConfirmed
//     and a nil error.
func (s *Service) PlaceOrder(ctx context.Context, customerID, sku string, quantity int, amountCents int64) (Order, error) {
	return Order{}, errors.New("not implemented")
}
