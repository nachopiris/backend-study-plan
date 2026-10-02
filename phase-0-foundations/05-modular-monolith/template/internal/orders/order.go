// Package orders is the seed of the guiding project's core module: it places
// orders by coordinating stock reservation (inventory) and payment charging
// (payments) through their ports only — see service.go for the boundary rule
// this practice is really about.
package orders

// Status is the lifecycle state of an Order.
type Status string

const (
	StatusConfirmed Status = "confirmed"
	StatusFailed    Status = "failed"
)

// Order represents a placed (or attempted) customer order.
type Order struct {
	ID          string
	CustomerID  string
	SKU         string
	Quantity    int
	AmountCents int64
	Status      Status
}
