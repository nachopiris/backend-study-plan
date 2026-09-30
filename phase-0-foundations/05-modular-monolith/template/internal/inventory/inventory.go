// Package inventory defines the inventory module's domain types and the port
// (Reserver) that other modules depend on. Its concrete in-memory adapter
// lives in the sibling internal/inventory/memory package; callers outside
// this module (namely orders) must depend on this package's Reserver
// interface only, never on the memory package.
package inventory

import (
	"context"
	"errors"
)

// ErrInsufficientStock is returned by Reserve when there is not enough stock
// available to satisfy the requested quantity.
var ErrInsufficientStock = errors.New("inventory: insufficient stock")

// ErrReservationNotFound is returned by Release when given an unknown or
// already-released reservation ID.
var ErrReservationNotFound = errors.New("inventory: reservation not found")

// Reservation represents a hold placed on stock for a specific SKU.
type Reservation struct {
	ID       string
	SKU      string
	Quantity int
}

// Reserver is the port other modules depend on to reserve and release stock.
type Reserver interface {
	// Reserve attempts to hold quantity units of sku. On success it returns
	// the Reservation; if there isn't enough stock, it returns
	// ErrInsufficientStock.
	Reserve(ctx context.Context, sku string, quantity int) (Reservation, error)
	// Release cancels a previous reservation, returning its stock to the
	// available pool. Releasing an unknown or already-released reservation ID
	// returns ErrReservationNotFound.
	Release(ctx context.Context, reservationID string) error
}
