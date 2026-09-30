// Command api wires the modular monolith's modules together using their
// in-memory adapters and places one example order, so you have something
// runnable to sanity-check your implementation against (no HTTP server yet —
// that comes in a later phase of the study plan).
package main

import (
	"context"
	"fmt"
	"log"

	invmemory "studyplan/phase0/modularmonolith/internal/inventory/memory"
	"studyplan/phase0/modularmonolith/internal/orders"
	paymemory "studyplan/phase0/modularmonolith/internal/payments/memory"
)

func main() {
	inventoryAdapter := invmemory.NewAdapter(map[string]int{"sku-1": 5})
	paymentsAdapter := paymemory.NewAdapter()

	service := orders.NewService(inventoryAdapter, paymentsAdapter)

	order, err := service.PlaceOrder(context.Background(), "customer-1", "sku-1", 2, 4999)
	if err != nil {
		log.Fatalf("place order: %v", err)
	}

	fmt.Printf("order placed: %+v\n", order)
}
