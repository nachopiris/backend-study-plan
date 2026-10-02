// Package saga implements a choreographed saga for a purchase flow: orders,
// inventory, and payments react to each other's events, with no central
// coordinator deciding what happens next.
package saga

// EventType identifies which domain event an Event carries.
type EventType string

const (
	// EventOrderCreated kicks off the saga: a customer wants to buy Qty
	// units of SKU for OrderID.
	EventOrderCreated EventType = "OrderCreated"
	// EventStockReserved means inventory successfully reserved the
	// requested quantity for OrderID.
	EventStockReserved EventType = "StockReserved"
	// EventOutOfStock means inventory could NOT reserve the requested
	// quantity: the saga ends here for this order, with no charge ever
	// attempted.
	EventOutOfStock EventType = "OutOfStock"
	// EventPaymentSucceeded means payments charged the customer for
	// OrderID successfully.
	EventPaymentSucceeded EventType = "PaymentSucceeded"
	// EventPaymentFailed means the charge for OrderID failed: the saga
	// must compensate by releasing the reserved stock.
	EventPaymentFailed EventType = "PaymentFailed"
	// EventStockReleased is inventory's compensating action after a
	// PaymentFailed: the stock reserved for OrderID is given back.
	EventStockReleased EventType = "StockReleased"
	// EventOrderConfirmed, EventOrderCancelled and EventOrderRejected are
	// the three possible terminal states for an order.
	EventOrderConfirmed EventType = "OrderConfirmed"
	EventOrderCancelled EventType = "OrderCancelled"
	EventOrderRejected  EventType = "OrderRejected"
)

// Event is the common envelope for every message carried by Bus. Not every
// field is meaningful for every Type — SKU and Qty only matter for
// OrderCreated (handlers that care about a later event's quantity must track
// it themselves, keyed by OrderID), and Reason only carries context for
// OutOfStock and PaymentFailed.
type Event struct {
	Type    EventType
	OrderID string
	SKU     string
	Qty     int
	Reason  string
}
