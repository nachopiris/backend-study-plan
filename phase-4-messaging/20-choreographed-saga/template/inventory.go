package saga

import "sync"

// InventoryService reacts to OrderCreated (reserve stock) and PaymentFailed
// (release stock already reserved — the saga's compensating action).
type InventoryService struct {
	mu          sync.Mutex
	stock       map[string]int  // sku -> units currently available
	reservedQty map[string]int  // orderID -> qty reserved for it, while the reservation is held
	released    map[string]bool // orderID -> whether its compensation already ran (idempotency guard)
}

// NewInventoryService builds an InventoryService seeded with initialStock
// (sku -> available units). initialStock is not mutated by this call.
func NewInventoryService(initialStock map[string]int) *InventoryService {
	stock := make(map[string]int, len(initialStock))
	for sku, qty := range initialStock {
		stock[sku] = qty
	}
	return &InventoryService{
		stock:       stock,
		reservedQty: make(map[string]int),
		released:    make(map[string]bool),
	}
}

// Stock returns the current available quantity for sku (0 if unknown).
func (s *InventoryService) Stock(sku string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stock[sku]
}

// RegisterInventoryHandlers subscribes this service's reactions to bus:
//
//   - On OrderCreated{OrderID, SKU, Qty}: if s.stock[SKU] >= Qty, decrement
//     it by Qty, remember Qty in s.reservedQty[OrderID], and publish
//     StockReserved{OrderID}. Otherwise, publish OutOfStock{OrderID,
//     Reason: "..."} WITHOUT changing stock.
//   - On PaymentFailed{OrderID}: if a reservation is currently held for
//     OrderID (present in s.reservedQty), give the stock back (increment by
//     the remembered qty) and publish StockReleased{OrderID}.
//
// Both reactions MUST be idempotent: Bus may duplicate or reorder
// deliveries, so OrderCreated for the same OrderID must never reserve stock
// twice, and PaymentFailed for the same OrderID must never release stock
// twice (or release stock that was never reserved, e.g. for an order that
// was rejected for being OutOfStock in the first place).
//
// Not implemented yet: subscribes to nothing, so inventory never reacts to
// anything.
func (s *InventoryService) RegisterInventoryHandlers(bus *Bus) {
}
