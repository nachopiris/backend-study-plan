package clocks

import "testing"

func TestLamportClock_TickIncrementsMonotonically(t *testing.T) {
	c := NewLamportClock()

	var last uint64
	for i := 0; i < 5; i++ {
		got := c.Tick()
		if got <= last {
			t.Fatalf("Tick() call %d: got %d, want strictly greater than previous value %d", i, got, last)
		}
		last = got
	}
}

func TestLamportClock_SendIncrementsLikeTick(t *testing.T) {
	c := NewLamportClock()

	a := c.Tick()
	b := c.Send()
	if b <= a {
		t.Fatalf("Send() after Tick() = %d, want strictly greater than %d", b, a)
	}
}

func TestLamportClock_ReceiveJumpsAheadOfStaleLocalCounter(t *testing.T) {
	c := NewLamportClock()
	c.Tick() // local counter is now small (e.g. 1)

	// A message arrives carrying a timestamp far ahead of our own counter,
	// as if it came from a process that has done a lot more locally.
	got := c.Receive(100)
	if got <= 100 {
		t.Fatalf("Receive(100) = %d, want strictly greater than 100 (the incoming timestamp)", got)
	}

	// Once caught up, purely local events must still move strictly forward.
	next := c.Tick()
	if next <= got {
		t.Fatalf("Tick() after Receive(100) = %d, want strictly greater than %d", next, got)
	}
}

// TestLamportClock_ThreeProcessExchange scripts a classic three-process
// causal chain and checks that the resulting timestamps respect
// happens-before end to end:
//
//	P1: tick -> send m1 ------> P2: tick, receive m1 -> send m2 ------> P3: tick, receive m2
//
// Every arrow in that chain is a happens-before relationship, so the
// timestamp at the tail of each arrow must be strictly less than the
// timestamp at its head.
func TestLamportClock_ThreeProcessExchange(t *testing.T) {
	p1 := NewLamportClock()
	p2 := NewLamportClock()
	p3 := NewLamportClock()

	p1LocalEvent := p1.Tick()
	m1 := p1.Send()

	p2LocalEvent := p2.Tick()
	p2Receive := p2.Receive(m1)
	m2 := p2.Send()

	p3LocalEvent := p3.Tick()
	p3Receive := p3.Receive(m2)

	// Causally ordered pairs: each must satisfy a strict happens-before.
	type pair struct {
		name       string
		before     uint64
		after      uint64
		beforeDesc string
		afterDesc  string
	}
	pairs := []pair{
		{"p1 local event before p1 sends m1", p1LocalEvent, m1, "P1 tick", "P1 send m1"},
		{"p1 sends m1 before p2 receives m1", m1, p2Receive, "P1 send m1", "P2 receive m1"},
		{"p2 receives m1 before p2 sends m2", p2Receive, m2, "P2 receive m1", "P2 send m2"},
		{"p2 sends m2 before p3 receives m2", m2, p3Receive, "P2 send m2", "P3 receive m2"},
	}
	for _, p := range pairs {
		if !(p.before < p.after) {
			t.Errorf("%s: expected timestamp(%s)=%d < timestamp(%s)=%d", p.name, p.beforeDesc, p.before, p.afterDesc, p.after)
		}
	}

	// p2's own local event (independent of the message) must still be
	// strictly before p2's own later send, by the single-process ordering
	// Lamport clocks always guarantee.
	if !(p2LocalEvent < m2) {
		t.Errorf("expected P2's local tick (%d) to be strictly before P2's send of m2 (%d)", p2LocalEvent, m2)
	}
	if !(p3LocalEvent < p3Receive) {
		// p3's local tick is not causally related to the P1->P2->P3 chain
		// (it happened independently), but a single process's own clock must
		// never go backwards between successive calls.
		t.Errorf("P3's own events must never go backwards: local tick %d, receive %d", p3LocalEvent, p3Receive)
	}
}
