package loadshed

import (
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestLimiter_StartsFullAndAllowsUpToBurst(t *testing.T) {
	clock := newFakeClock()
	l := NewLimiter(clock, 1, 3)

	for i := 0; i < 3; i++ {
		if !l.Allow() {
			t.Fatalf("Allow() call %d = false, want true (bucket starts full with burst=3)", i)
		}
	}
	if l.Allow() {
		t.Fatal("Allow() after exhausting the burst = true, want false")
	}
}

func TestLimiter_RefillsAtTheConfiguredRate(t *testing.T) {
	clock := newFakeClock()
	l := NewLimiter(clock, 1 /* token/s */, 1 /* burst */)

	if !l.Allow() {
		t.Fatal("Allow() on a full bucket = false, want true")
	}
	if l.Allow() {
		t.Fatal("Allow() on an empty bucket = true, want false")
	}

	clock.Advance(999 * time.Millisecond)
	if l.Allow() {
		t.Fatal("Allow() after 999ms at 1 token/s = true, want false (not quite a full token yet)")
	}

	clock.Advance(1 * time.Millisecond) // total 1s elapsed: exactly 1 token
	if !l.Allow() {
		t.Fatal("Allow() after 1s at 1 token/s = false, want true")
	}
	if l.Allow() {
		t.Fatal("Allow() immediately after consuming the refilled token = true, want false")
	}
}

func TestLimiter_RefillIsFractional(t *testing.T) {
	clock := newFakeClock()
	l := NewLimiter(clock, 2 /* tokens/s */, 1 /* burst */)

	_ = l.Allow() // drain the initial token
	clock.Advance(250 * time.Millisecond) // at 2 tokens/s, that's half a token

	if l.Allow() {
		t.Fatal("Allow() after accumulating only half a token = true, want false")
	}

	clock.Advance(250 * time.Millisecond) // now a full token accumulated
	if !l.Allow() {
		t.Fatal("Allow() after accumulating a full token (two 250ms steps) = false, want true")
	}
}

func TestLimiter_NeverExceedsBurstCapacityNoMatterHowLongIdle(t *testing.T) {
	clock := newFakeClock()
	l := NewLimiter(clock, 10, 3)

	clock.Advance(1000 * time.Hour) // a very long idle period

	for i := 0; i < 3; i++ {
		if !l.Allow() {
			t.Fatalf("Allow() call %d after a long idle period = false, want true", i)
		}
	}
	if l.Allow() {
		t.Fatal("Allow() beyond burst after a long idle period = true, want false (the bucket must still cap at burst)")
	}
}
