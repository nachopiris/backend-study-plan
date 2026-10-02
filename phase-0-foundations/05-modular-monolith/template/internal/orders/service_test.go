package orders_test

import (
	"context"
	"errors"
	"testing"

	"studyplan/phase0/modularmonolith/internal/inventory"
	"studyplan/phase0/modularmonolith/internal/orders"
	"studyplan/phase0/modularmonolith/internal/payments"
)

// fakeInventory and fakePayments implement the ports directly, with
// configurable behavior and call tracking, so these tests never depend on the
// real in-memory adapters.

type fakeInventory struct {
	reserveErr   error
	releaseErr   error
	reserveCalls int
	released     []string
}

func (f *fakeInventory) Reserve(ctx context.Context, sku string, quantity int) (inventory.Reservation, error) {
	f.reserveCalls++
	if f.reserveErr != nil {
		return inventory.Reservation{}, f.reserveErr
	}
	return inventory.Reservation{ID: "res-1", SKU: sku, Quantity: quantity}, nil
}

func (f *fakeInventory) Release(ctx context.Context, reservationID string) error {
	f.released = append(f.released, reservationID)
	return f.releaseErr
}

type fakePayments struct {
	chargeErr   error
	chargeCalls int
}

func (f *fakePayments) Charge(ctx context.Context, customerID string, amountCents int64) (payments.Receipt, error) {
	f.chargeCalls++
	if f.chargeErr != nil {
		return payments.Receipt{}, f.chargeErr
	}
	return payments.Receipt{ID: "receipt-1", AmountCents: amountCents}, nil
}

func TestPlaceOrder_HappyPath(t *testing.T) {
	inv := &fakeInventory{}
	pay := &fakePayments{}
	svc := orders.NewService(inv, pay)

	order, err := svc.PlaceOrder(context.Background(), "cust-1", "sku-1", 2, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.Status != orders.StatusConfirmed {
		t.Fatalf("expected status %q, got %q", orders.StatusConfirmed, order.Status)
	}
	if inv.reserveCalls != 1 {
		t.Fatalf("expected exactly 1 Reserve call, got %d", inv.reserveCalls)
	}
	if pay.chargeCalls != 1 {
		t.Fatalf("expected exactly 1 Charge call, got %d", pay.chargeCalls)
	}
	if len(inv.released) != 0 {
		t.Fatalf("did not expect any reservation to be released on the happy path, released: %v", inv.released)
	}
}

func TestPlaceOrder_FailsWithoutChargingWhenOutOfStock(t *testing.T) {
	inv := &fakeInventory{reserveErr: inventory.ErrInsufficientStock}
	pay := &fakePayments{}
	svc := orders.NewService(inv, pay)

	_, err := svc.PlaceOrder(context.Background(), "cust-1", "sku-1", 2, 1000)
	if !errors.Is(err, inventory.ErrInsufficientStock) {
		t.Fatalf("expected error to wrap inventory.ErrInsufficientStock, got %v", err)
	}
	if pay.chargeCalls != 0 {
		t.Fatalf("expected Charge to never be called when stock reservation fails, got %d calls", pay.chargeCalls)
	}
}

func TestPlaceOrder_ReleasesReservationWhenPaymentFails(t *testing.T) {
	inv := &fakeInventory{}
	pay := &fakePayments{chargeErr: payments.ErrPaymentDeclined}
	svc := orders.NewService(inv, pay)

	_, err := svc.PlaceOrder(context.Background(), "cust-1", "sku-1", 2, 1000)
	if !errors.Is(err, payments.ErrPaymentDeclined) {
		t.Fatalf("expected error to wrap payments.ErrPaymentDeclined, got %v", err)
	}
	if len(inv.released) != 1 || inv.released[0] != "res-1" {
		t.Fatalf("expected reservation %q to be released exactly once, released: %v", "res-1", inv.released)
	}
}
