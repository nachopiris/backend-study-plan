package saga

import "sync"

// OrderStatus is one order's position in the saga.
type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusConfirmed OrderStatus = "confirmed"
	StatusCancelled OrderStatus = "cancelled"
	StatusRejected  OrderStatus = "rejected"
)

// OrderService tracks every order's status as the saga's events arrive.
type OrderService struct {
	mu     sync.Mutex
	orders map[string]OrderStatus
}

// NewOrderService builds an OrderService with no orders yet.
func NewOrderService() *OrderService {
	return &OrderService{orders: make(map[string]OrderStatus)}
}

// Status returns orderID's current status, or StatusPending if OrderService
// has not seen OrderCreated for it yet (which is also where every order
// implicitly starts).
func (s *OrderService) Status(orderID string) OrderStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	if status, ok := s.orders[orderID]; ok {
		return status
	}
	return StatusPending
}

// RegisterOrderHandlers subscribes this service's reactions to bus:
//
//   - On PaymentSucceeded{OrderID}: transition to StatusConfirmed.
//   - On PaymentFailed{OrderID}: transition to StatusCancelled.
//   - On OutOfStock{OrderID}: transition to StatusRejected.
//
// Every transition is final: once an order reaches ANY of Confirmed,
// Cancelled, or Rejected, further events for that OrderID must NOT change
// its status again. This is what makes the saga converge to the correct
// final state even when Bus duplicates or reorders deliveries — e.g. two
// PaymentSucceeded deliveries for the same order must not cause any
// observable difference from receiving just one.
//
// Not implemented yet: subscribes to nothing, so every order stays
// StatusPending forever.
func (s *OrderService) RegisterOrderHandlers(bus *Bus) {
}
