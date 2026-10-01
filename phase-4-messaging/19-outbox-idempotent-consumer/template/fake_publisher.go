package outbox

import (
	"context"
	"errors"
	"sync"
	"time"
)

// InMemoryPublisher is a GIVEN, fully working test double for Publisher: it
// stands in for a real message bus (like the Kafka client from practice 18)
// so tests can drive Relay and Consumer together without any real
// infrastructure besides Postgres.
//
// It can be configured to reproduce two real-world failure modes:
//
//   - CrashAfterPublish: simulates the relay process dying in the gap
//     between "the broker acknowledged the publish" and "the relay recorded
//     the row as sent" — Publish still records the delivery (the message
//     really did reach the bus), but returns an error, so Relay's
//     transaction never commits the sent_at update and will retry the row.
//   - Duplicate: simulates a message bus that itself redelivers messages
//     (its own at-least-once guarantee) — every successful Publish call
//     records the message TWICE.
//
// Both are independent of Relay's own correctness: they exist to prove that
// a correct Consumer tolerates both, not to test Relay.
type InMemoryPublisher struct {
	mu                sync.Mutex
	delivered         []Message
	crashAfterPublish bool
	duplicate         bool
}

// NewInMemoryPublisher returns an InMemoryPublisher with no configured
// failure modes and nothing delivered yet.
func NewInMemoryPublisher() *InMemoryPublisher {
	return &InMemoryPublisher{}
}

// CrashAfterPublish enables or disables the "crash after publish, before
// ack" failure mode described on InMemoryPublisher.
func (p *InMemoryPublisher) CrashAfterPublish(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.crashAfterPublish = enabled
}

// Duplicate enables or disables the "the bus itself redelivers" failure mode
// described on InMemoryPublisher.
func (p *InMemoryPublisher) Duplicate(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.duplicate = enabled
}

// Publish records msg as delivered (once, or twice if Duplicate is enabled),
// then returns an error if CrashAfterPublish is enabled.
func (p *InMemoryPublisher) Publish(ctx context.Context, msg Message) error {
	// A small fixed delay, standing in for the network latency a real
	// publish call would have. Without it, two relays racing for the same
	// row in a fast local test could finish so quickly that a missing
	// `FOR UPDATE SKIP LOCKED` would rarely actually overlap, hiding the bug
	// this delay is here to make reliably observable.
	time.Sleep(5 * time.Millisecond)

	p.mu.Lock()
	defer p.mu.Unlock()

	p.delivered = append(p.delivered, msg)
	if p.duplicate {
		p.delivered = append(p.delivered, msg)
	}
	if p.crashAfterPublish {
		return errors.New("simulated crash: the publish reached the bus, but the relay died before it could mark the row as sent")
	}
	return nil
}

// Delivered returns every message this fake has handed to "downstream" so
// far, in delivery order. The returned slice is a copy: mutating it does not
// affect the fake.
func (p *InMemoryPublisher) Delivered() []Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Message, len(p.delivered))
	copy(out, p.delivered)
	return out
}
