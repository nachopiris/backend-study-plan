package router

import (
	"context"
	"fmt"
	"sync"
)

// LegacyCharger is the GIVEN, working legacy in-process payments implementation this strangler
// facade gradually routes traffic away from. It mirrors practice 05's in-memory payments
// adapter: every charge succeeds unless the customer was explicitly marked declined. You do not
// need to (and should not need to) modify this file — this practice is about PaymentsRouter and
// the new gRPC service, not about the code being strangled.
type LegacyCharger struct {
	mu      sync.Mutex
	nextID  int
	decline map[string]bool
}

// NewLegacyCharger builds a LegacyCharger where every charge succeeds by default.
func NewLegacyCharger() *LegacyCharger {
	return &LegacyCharger{decline: make(map[string]bool)}
}

// DeclineCustomer marks customerID so that future Charge calls for it return ErrPaymentDeclined.
func (l *LegacyCharger) DeclineCustomer(customerID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.decline[customerID] = true
}

// Charge implements Charger.
func (l *LegacyCharger) Charge(ctx context.Context, customerID string, amountCents int64) (Receipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.decline[customerID] {
		return Receipt{}, ErrPaymentDeclined
	}

	l.nextID++
	return Receipt{ID: fmt.Sprintf("legacy-receipt-%d", l.nextID), AmountCents: amountCents}, nil
}
