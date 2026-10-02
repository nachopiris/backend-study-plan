// Package memory is an in-memory adapter implementing inventory.Reserver.
// It is given, working code — this practice is about the orders module and
// the architecture boundary, not about the adapters.
package memory

import (
	"context"
	"fmt"
	"sync"

	"studyplan/phase0/modularmonolith/internal/inventory"
)

// Adapter is a concurrency-safe, in-memory implementation of
// inventory.Reserver, useful for tests and for running the skeleton without a
// real database.
type Adapter struct {
	mu           sync.Mutex
	stock        map[string]int
	reservations map[string]inventory.Reservation
	nextID       int
}

// NewAdapter builds an Adapter seeded with initialStock (SKU -> quantity).
func NewAdapter(initialStock map[string]int) *Adapter {
	stock := make(map[string]int, len(initialStock))
	for sku, qty := range initialStock {
		stock[sku] = qty
	}
	return &Adapter{
		stock:        stock,
		reservations: make(map[string]inventory.Reservation),
	}
}

func (a *Adapter) Reserve(ctx context.Context, sku string, quantity int) (inventory.Reservation, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.stock[sku] < quantity {
		return inventory.Reservation{}, inventory.ErrInsufficientStock
	}

	a.stock[sku] -= quantity
	a.nextID++
	res := inventory.Reservation{
		ID:       fmt.Sprintf("res-%d", a.nextID),
		SKU:      sku,
		Quantity: quantity,
	}
	a.reservations[res.ID] = res
	return res, nil
}

func (a *Adapter) Release(ctx context.Context, reservationID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	res, ok := a.reservations[reservationID]
	if !ok {
		return inventory.ErrReservationNotFound
	}
	a.stock[res.SKU] += res.Quantity
	delete(a.reservations, reservationID)
	return nil
}
