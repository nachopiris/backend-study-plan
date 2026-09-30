// Package memory is an in-memory adapter implementing payments.Charger. It is
// given, working code — this practice is about the orders module and the
// architecture boundary, not about the adapters.
package memory

import (
	"context"
	"fmt"
	"sync"

	"studyplan/phase0/modularmonolith/internal/payments"
)

// Adapter is a concurrency-safe, in-memory implementation of
// payments.Charger. By default every charge succeeds; use DeclineCustomer to
// simulate a declined payment for a given customer.
type Adapter struct {
	mu      sync.Mutex
	nextID  int
	decline map[string]bool
}

// NewAdapter builds an Adapter where every charge succeeds by default.
func NewAdapter() *Adapter {
	return &Adapter{decline: make(map[string]bool)}
}

// DeclineCustomer marks customerID so that future Charge calls for it return
// payments.ErrPaymentDeclined.
func (a *Adapter) DeclineCustomer(customerID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.decline[customerID] = true
}

func (a *Adapter) Charge(ctx context.Context, customerID string, amountCents int64) (payments.Receipt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.decline[customerID] {
		return payments.Receipt{}, payments.ErrPaymentDeclined
	}

	a.nextID++
	return payments.Receipt{
		ID:          fmt.Sprintf("receipt-%d", a.nextID),
		AmountCents: amountCents,
	}, nil
}
