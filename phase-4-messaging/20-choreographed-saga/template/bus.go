package saga

import "math/rand"

// Handler reacts to one delivered event. Bus does not retry a handler that
// panics or misbehaves; a handler that wants its own retry logic must
// implement that itself (deliberately out of scope — see "Para ir más
// allá").
type Handler func(e Event)

// BusOptions configures a Bus's delivery behavior. The zero value
// (DuplicateRate: 0, ReorderWindow: 0) delivers every event exactly once, in
// strict publish order — the simplest possible case, useful for a first
// "happy path" test before testing against duplication or reordering.
type BusOptions struct {
	// Seed drives every pseudo-random decision Bus makes. The same seed
	// always produces the same sequence of decisions, so a test built on a
	// fixed seed is fully reproducible.
	Seed int64
	// DuplicateRate is the probability (0..1) that a given delivery is
	// enqueued a second time, immediately alongside the first.
	DuplicateRate float64
	// ReorderWindow lets Bus insert a newly published delivery at a random
	// position among the last ReorderWindow entries still waiting in the
	// queue, instead of always appending strictly at the end. 0 (or
	// negative) disables reordering.
	ReorderWindow int
}

// pending is one (event, handler) delivery waiting in Bus's queue.
type pending struct {
	event   Event
	handler Handler
}

// Bus is a deterministic, single-goroutine, in-memory publish/subscribe
// event bus for this practice's purchase saga. It is GIVEN and fully
// working: you do not need to (and should not) change it — the practice is
// about the handlers you register on it, in orders.go, inventory.go and
// payments.go.
//
// Bus is NOT safe for concurrent use: Publish and Drain are meant to be
// called from a single goroutine (your test or driver code), exactly like a
// single-threaded event-loop. This keeps the delivery order fully
// deterministic given a seed, which is what makes the duplicate/reorder
// tests reproducible instead of flaky.
type Bus struct {
	subs  map[EventType][]Handler
	queue []pending
	rnd   *rand.Rand
	opts  BusOptions
}

// NewBus builds a Bus configured with opts.
func NewBus(opts BusOptions) *Bus {
	return &Bus{
		subs: make(map[EventType][]Handler),
		rnd:  rand.New(rand.NewSource(opts.Seed)),
		opts: opts,
	}
}

// Subscribe registers handler to be called for every event of type t that
// Bus delivers (see Publish and Drain). Multiple handlers may subscribe to
// the same type; they are all delivered to, in subscription order, for each
// event.
func (b *Bus) Subscribe(t EventType, handler Handler) {
	b.subs[t] = append(b.subs[t], handler)
}

// Publish enqueues e for delivery to every handler currently subscribed to
// e.Type. Publish never calls a handler itself, not even synchronously:
// delivery only happens inside Drain. This means a handler that publishes
// another event while Drain is running never recurses, and never changes
// delivery order by accident — it just adds more entries to the same queue
// Drain is already working through.
func (b *Bus) Publish(e Event) {
	for _, h := range b.subs[e.Type] {
		b.enqueue(pending{event: e, handler: h})
		if b.opts.DuplicateRate > 0 && b.rnd.Float64() < b.opts.DuplicateRate {
			b.enqueue(pending{event: e, handler: h})
		}
	}
}

// enqueue appends p to the queue, or — if reordering is enabled — inserts it
// at a random position among the last ReorderWindow entries, simulating a
// delivery that overtakes some of the ones published just before it.
func (b *Bus) enqueue(p pending) {
	if b.opts.ReorderWindow <= 0 || len(b.queue) == 0 {
		b.queue = append(b.queue, p)
		return
	}
	windowStart := len(b.queue) - b.opts.ReorderWindow
	if windowStart < 0 {
		windowStart = 0
	}
	pos := windowStart + b.rnd.Intn(len(b.queue)-windowStart+1)
	b.queue = append(b.queue, pending{})
	copy(b.queue[pos+1:], b.queue[pos:])
	b.queue[pos] = p
}

// Drain delivers every pending event — including ones published by a
// handler while Drain is still running — until none remain, then returns.
// Call it once after kicking off a flow (typically right after the initial
// Publish(Event{Type: EventOrderCreated, ...})) and before asserting on the
// resulting state.
func (b *Bus) Drain() {
	for len(b.queue) > 0 {
		p := b.queue[0]
		b.queue = b.queue[1:]
		p.handler(p.event)
	}
}
