package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
)

// Header is a single key/value message header, in the shape most Kafka client libraries use
// (for example franz-go's kgo.Header, used in practice 18) — a []Header slice instead of a map,
// because real message headers may repeat a key and preserve insertion order, which a map cannot.
type Header struct {
	Key   string
	Value []byte
}

// HeadersCarrier adapts a *[]Header slice to propagation.TextMapCarrier (Get/Set/Keys) — the same
// interface satisfied by the HTTP carriers OTel ships (propagation.HeaderCarrier, used in
// client.go), but for asynchronous message headers instead of HTTP headers. This is what lets
// the SAME propagator (propagation.TraceContext{}) that works over HTTP in client.go also work
// over a Kafka message's headers: TextMapPropagator only ever talks to a TextMapCarrier, and
// never needs to know whether the underlying transport is HTTP or Kafka.
//
// Headers is a pointer because Set needs to be able to append a brand new header to the slice,
// which (unlike a map) cannot be done by mutating through a non-pointer receiver alone when the
// slice needs to grow.
type HeadersCarrier struct {
	Headers *[]Header
}

var _ propagation.TextMapCarrier = (*HeadersCarrier)(nil)

// Get returns the string value of the FIRST header matching key (case-sensitive), or "" if no
// header with that key exists.
//
// Not implemented yet: always returns "".
func (c *HeadersCarrier) Get(key string) string {
	return ""
}

// Set stores value under key: if a header with that key already exists, its value is replaced
// (only the first match, same as Get); otherwise a new Header is appended.
//
// Not implemented yet: no-op (nothing is stored).
func (c *HeadersCarrier) Set(key, value string) {
}

// Keys returns the distinct header keys currently stored, in no particular order.
//
// Not implemented yet: returns nil.
func (c *HeadersCarrier) Keys() []string {
	return nil
}

// InjectToMessage injects the trace context active in ctx into headers (in place, via
// HeadersCarrier) using propagator (propagation.TraceContext{} if propagator is nil). Call this
// right before publishing a message — typically from inside a CLIENT or PRODUCER span — so a
// consumer on the other end of the topic can continue the same trace instead of starting an
// unrelated one.
//
// Not implemented yet: no-op (headers is left unchanged).
func InjectToMessage(ctx context.Context, propagator propagation.TextMapPropagator, headers *[]Header) {
}

// ExtractFromMessage returns a context derived from ctx carrying the trace context found in
// headers, if any (via propagator, propagation.TraceContext{} if nil). Call this when a consumer
// receives a message, before starting its own span (typically CONSUMER-kind) for processing it —
// that new span then becomes a CHILD of whatever producer span called InjectToMessage, closing
// the trace across the asynchronous boundary.
//
// Contract: if headers carries no valid trace context (for example, a message published by code
// that never called InjectToMessage), ExtractFromMessage returns ctx unchanged — a span started
// from it becomes a new trace root, not an error.
//
// Not implemented yet: returns ctx unchanged.
func ExtractFromMessage(ctx context.Context, propagator propagation.TextMapPropagator, headers []Header) context.Context {
	return ctx
}
