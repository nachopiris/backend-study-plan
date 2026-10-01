package saga

import "sync"

// PaymentsService reacts to StockReserved by attempting to charge the
// customer for the order.
type PaymentsService struct {
	mu         sync.Mutex
	attempted  map[string]int // orderID -> number of charge attempts recorded
	shouldFail func(orderID string) bool
}

// NewPaymentsService builds a PaymentsService. shouldFail decides whether
// charging a given orderID fails; pass nil to make every charge succeed.
func NewPaymentsService(shouldFail func(orderID string) bool) *PaymentsService {
	if shouldFail == nil {
		shouldFail = func(string) bool { return false }
	}
	return &PaymentsService{attempted: make(map[string]int), shouldFail: shouldFail}
}

// AttemptsFor returns how many times this service recorded a charge attempt
// for orderID (0 if it was never asked to charge that order at all).
func (s *PaymentsService) AttemptsFor(orderID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attempted[orderID]
}

// RegisterPaymentHandlers subscribes this service's reaction to bus:
//
//   - On StockReserved{OrderID}: if this is the FIRST time payments has seen
//     StockReserved for OrderID, record one charge attempt and call
//     s.shouldFail(OrderID) to decide the outcome: publish
//     PaymentSucceeded{OrderID} if it returns false, or
//     PaymentFailed{OrderID, Reason: "..."} if it returns true.
//
// This reaction MUST be idempotent: a duplicated or redelivered
// StockReserved for an order payments already attempted must NOT charge
// (or record an attempt) again — AttemptsFor(orderID) must stay at 1 no
// matter how many times StockReserved arrives for that order.
//
// Not implemented yet: subscribes to nothing, so payments never reacts to
// anything.
func (s *PaymentsService) RegisterPaymentHandlers(bus *Bus) {
}
