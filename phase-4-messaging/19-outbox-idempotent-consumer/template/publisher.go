package outbox

import "context"

// Message is what Relay hands to a Publisher: one outbox row's content,
// addressed by a stable, globally unique ID (derived from the outbox row's
// own primary key) that a downstream Consumer uses for deduplication.
type Message struct {
	ID          string // stable idempotency key, derived from the outbox row id
	AggregateID string // the order id this event is about
	EventType   string
	Payload     []byte
}

// Publisher abstracts "hand this message to the message bus" so Relay does
// not need to know about Kafka (or any other broker) at all. A real
// implementation could wrap the Producer from practice 18 (see the README's
// optional step); tests use InMemoryPublisher.
type Publisher interface {
	Publish(ctx context.Context, msg Message) error
}
