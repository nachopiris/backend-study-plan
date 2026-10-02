package saga

import "testing"

func newWiredSaga(busOpts BusOptions, stock map[string]int, shouldFail func(string) bool) (*Bus, *OrderService, *InventoryService, *PaymentsService) {
	bus := NewBus(busOpts)
	orders := NewOrderService()
	inventory := NewInventoryService(stock)
	payments := NewPaymentsService(shouldFail)
	orders.RegisterOrderHandlers(bus)
	inventory.RegisterInventoryHandlers(bus)
	payments.RegisterPaymentHandlers(bus)
	return bus, orders, inventory, payments
}

func TestSaga_HappyPath_ConfirmsTheOrder(t *testing.T) {
	bus, orders, inventory, _ := newWiredSaga(BusOptions{Seed: 1}, map[string]int{"widget": 5}, nil)

	bus.Publish(Event{Type: EventOrderCreated, OrderID: "o1", SKU: "widget", Qty: 2})
	bus.Drain()

	if got := orders.Status("o1"); got != StatusConfirmed {
		t.Fatalf("Status(o1) = %q, want %q", got, StatusConfirmed)
	}
	if got := inventory.Stock("widget"); got != 3 {
		t.Fatalf("Stock(widget) = %d, want 3 (5 - 2)", got)
	}
}

func TestSaga_PaymentFailure_CompensatesByReleasingStock(t *testing.T) {
	alwaysFail := func(orderID string) bool { return true }
	bus, orders, inventory, payments := newWiredSaga(BusOptions{Seed: 2}, map[string]int{"widget": 5}, alwaysFail)

	bus.Publish(Event{Type: EventOrderCreated, OrderID: "o2", SKU: "widget", Qty: 2})
	bus.Drain()

	if got := orders.Status("o2"); got != StatusCancelled {
		t.Fatalf("Status(o2) = %q, want %q", got, StatusCancelled)
	}
	if got := inventory.Stock("widget"); got != 5 {
		t.Fatalf("Stock(widget) = %d, want 5 (reservation must be released after payment failure)", got)
	}
	if got := payments.AttemptsFor("o2"); got != 1 {
		t.Fatalf("AttemptsFor(o2) = %d, want 1", got)
	}
}

func TestSaga_OutOfStock_RejectsWithoutAttemptingPayment(t *testing.T) {
	bus, orders, inventory, payments := newWiredSaga(BusOptions{Seed: 3}, map[string]int{"widget": 1}, nil)

	bus.Publish(Event{Type: EventOrderCreated, OrderID: "o3", SKU: "widget", Qty: 5})
	bus.Drain()

	if got := orders.Status("o3"); got != StatusRejected {
		t.Fatalf("Status(o3) = %q, want %q", got, StatusRejected)
	}
	if got := inventory.Stock("widget"); got != 1 {
		t.Fatalf("Stock(widget) = %d, want 1 (unchanged: nothing was reserved)", got)
	}
	if got := payments.AttemptsFor("o3"); got != 0 {
		t.Fatalf("AttemptsFor(o3) = %d, want 0 (payment must never be attempted for an out-of-stock order)", got)
	}
}

func TestSaga_DuplicateDeliveries_NeverDoubleReserveOrDoubleCharge(t *testing.T) {
	// DuplicateRate: 1.0 means EVERY delivery is enqueued twice.
	bus, orders, inventory, payments := newWiredSaga(BusOptions{Seed: 4, DuplicateRate: 1.0}, map[string]int{"widget": 10}, nil)

	bus.Publish(Event{Type: EventOrderCreated, OrderID: "o4", SKU: "widget", Qty: 3})
	bus.Drain()

	if got := orders.Status("o4"); got != StatusConfirmed {
		t.Fatalf("Status(o4) = %q, want %q", got, StatusConfirmed)
	}
	if got := inventory.Stock("widget"); got != 7 {
		t.Fatalf("Stock(widget) = %d, want 7 (10 - 3, decremented exactly ONCE despite duplicate deliveries)", got)
	}
	if got := payments.AttemptsFor("o4"); got != 1 {
		t.Fatalf("AttemptsFor(o4) = %d, want 1 (charged exactly ONCE despite duplicate deliveries)", got)
	}
}

func TestSaga_FinalStatesConvergeUnderReordering(t *testing.T) {
	shouldFail := func(orderID string) bool { return orderID == "fails-payment" }
	stock := map[string]int{"widget": 100, "scarce": 1}

	// ReorderWindow lets deliveries overtake each other within a bounded
	// window; a correct saga converges to the same final states regardless.
	bus, orders, inventory, _ := newWiredSaga(BusOptions{Seed: 7, ReorderWindow: 4}, stock, shouldFail)

	bus.Publish(Event{Type: EventOrderCreated, OrderID: "succeeds", SKU: "widget", Qty: 1})
	bus.Publish(Event{Type: EventOrderCreated, OrderID: "fails-payment", SKU: "widget", Qty: 1})
	bus.Publish(Event{Type: EventOrderCreated, OrderID: "out-of-stock", SKU: "scarce", Qty: 5})
	bus.Drain()

	cases := map[string]OrderStatus{
		"succeeds":      StatusConfirmed,
		"fails-payment": StatusCancelled,
		"out-of-stock":  StatusRejected,
	}
	for orderID, want := range cases {
		if got := orders.Status(orderID); got != want {
			t.Errorf("Status(%s) = %q, want %q", orderID, got, want)
		}
	}
	if got := inventory.Stock("widget"); got != 99 {
		t.Errorf("Stock(widget) = %d, want 99 (only `succeeds` keeps its reservation; `fails-payment` releases its own)", got)
	}
	if got := inventory.Stock("scarce"); got != 1 {
		t.Errorf("Stock(scarce) = %d, want 1 (unchanged: the out-of-stock order never reserved anything)", got)
	}
}
