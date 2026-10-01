// Package clocks implements logical clocks and a tiny Dynamo-style
// multi-value register from scratch: a Lamport clock, a vector clock, and a
// Store that uses vector clocks to detect concurrent writes.
//
// No external dependencies: only the Go standard library.
package clocks

// LamportClock implements a Lamport logical clock: a single counter per
// process that gives every event a timestamp such that if event a
// happens-before event b (a causes b, directly or through a chain of
// messages), then LamportClock timestamp(a) < timestamp(b).
//
// The converse does not hold: two timestamps being ordered does not mean the
// events are causally related. Two events on different processes that never
// exchanged a message can still get comparable timestamps purely by
// coincidence of local counters. That gap is exactly why VectorClock
// (vectorclock.go) exists: it can tell "happened-before" apart from
// "concurrent," which LamportClock cannot.
type LamportClock struct {
	counter uint64
}

// NewLamportClock creates a clock with its counter at zero. No event has
// happened yet.
func NewLamportClock() *LamportClock {
	return &LamportClock{}
}

// Tick records a local event (one that does not send or receive a message)
// and returns its timestamp.
//
// Not implemented yet: always returns 0.
func (c *LamportClock) Tick() uint64 {
	return 0
}

// Send records the local event of sending a message and returns the
// timestamp that must be attached to that message. The receiving process
// will feed this value into its own clock's Receive.
//
// Not implemented yet: always returns 0.
func (c *LamportClock) Send() uint64 {
	return 0
}

// Receive records the local event of receiving a message that was sent with
// timestamp ts, and returns the new local timestamp for that event. A
// Lamport clock must never go backwards and must always account for
// whatever the sender already knew, even if the sender's clock was ahead of
// the receiver's.
//
// Not implemented yet: always returns 0.
func (c *LamportClock) Receive(ts uint64) uint64 {
	return 0
}
