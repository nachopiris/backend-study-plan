package loadshed

import (
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// TestLimiter_BehavesLikeGolangXTimeRateOnBurst compares your by-hand
// Limiter against golang.org/x/time/rate on the one property both can be
// checked for without relying on either implementation's internal clock
// injection (x/time/rate always uses the real wall clock): a bucket of
// size N allows exactly N immediate calls, then rejects the next one.
//
// This uses real (short) sleeps because x/time/rate is not clock-injectable
// — that is itself one of the differences worth noticing between "built by
// hand" and "off the shelf".
func TestLimiter_BehavesLikeGolangXTimeRateOnBurst(t *testing.T) {
	const burst = 4
	const ratePerSecond = 100 // fast enough that refill does not interfere within this test

	ours := NewLimiter(RealClock{}, ratePerSecond, burst)
	theirs := rate.NewLimiter(rate.Limit(ratePerSecond), burst)

	for i := 0; i < burst; i++ {
		if !ours.Allow() {
			t.Fatalf("our Limiter.Allow() call %d = false, want true", i)
		}
		if !theirs.Allow() {
			t.Fatalf("x/time/rate Limiter.Allow() call %d = false, want true", i)
		}
	}

	// Both should now be empty; check immediately, before any meaningful
	// refill at 100 tokens/sec could occur.
	if ours.Allow() {
		t.Fatal("our Limiter.Allow() after exhausting the burst = true, want false")
	}
	if theirs.Allow() {
		t.Fatal("x/time/rate Limiter.Allow() after exhausting the burst = true, want false")
	}

	// Sanity check that both eventually refill, so this test is not
	// vacuously true for an Allow() that always returns false.
	time.Sleep(50 * time.Millisecond) // at 100 tokens/s, far more than one token's worth
	if !ours.Allow() {
		t.Fatal("our Limiter.Allow() after waiting for a refill = false, want true")
	}
	if !theirs.Allow() {
		t.Fatal("x/time/rate Limiter.Allow() after waiting for a refill = false, want true")
	}
}
