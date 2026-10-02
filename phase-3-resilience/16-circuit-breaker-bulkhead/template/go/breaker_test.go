package breaker

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock is a manually-advanced Clock, so breaker tests never depend on
// wall-clock timing.
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

var errBoom = errors.New("dependency failed")

func TestBreaker_ClosedCountsFailuresWithoutOpeningBelowThreshold(t *testing.T) {
	clock := newFakeClock()
	b := NewBreaker(clock, 3, time.Second, 1)

	for i := 0; i < 2; i++ {
		err := b.Execute(func() error { return errBoom })
		if !errors.Is(err, errBoom) {
			t.Fatalf("Execute() = %v, want %v (fn's own error, breaker still closed)", err, errBoom)
		}
	}
	if got := b.State(); got != Closed {
		t.Fatalf("State() = %v, want %v (threshold is 3, only 2 failures so far)", got, Closed)
	}
}

func TestBreaker_OpensAfterConsecutiveFailureThreshold(t *testing.T) {
	clock := newFakeClock()
	b := NewBreaker(clock, 3, time.Second, 1)

	for i := 0; i < 3; i++ {
		_ = b.Execute(func() error { return errBoom })
	}
	if got := b.State(); got != Open {
		t.Fatalf("State() = %v, want %v after 3 consecutive failures", got, Open)
	}

	var called int32
	err := b.Execute(func() error {
		atomic.AddInt32(&called, 1)
		return nil
	})
	if !errors.Is(err, ErrOpen) {
		t.Fatalf("Execute() while open = %v, want %v", err, ErrOpen)
	}
	if called != 0 {
		t.Fatalf("fn was called while the breaker is open, want never called")
	}
}

func TestBreaker_SuccessResetsConsecutiveFailureCount(t *testing.T) {
	clock := newFakeClock()
	b := NewBreaker(clock, 3, time.Second, 1)

	_ = b.Execute(func() error { return errBoom })
	_ = b.Execute(func() error { return errBoom })
	_ = b.Execute(func() error { return nil }) // resets the count
	_ = b.Execute(func() error { return errBoom })
	_ = b.Execute(func() error { return errBoom })

	if got := b.State(); got != Closed {
		t.Fatalf("State() = %v, want %v (the reset means only 2 consecutive failures since the success)", got, Closed)
	}
}

func TestBreaker_StaysOpenUntilTimeoutElapses(t *testing.T) {
	clock := newFakeClock()
	b := NewBreaker(clock, 1, 10*time.Second, 1)

	_ = b.Execute(func() error { return errBoom }) // opens immediately (threshold 1)
	if got := b.State(); got != Open {
		t.Fatalf("State() = %v, want %v", got, Open)
	}

	clock.Advance(9 * time.Second)
	if got := b.State(); got != Open {
		t.Fatalf("State() = %v, want %v (timeout has not elapsed yet)", got, Open)
	}

	clock.Advance(2 * time.Second) // total 11s > 10s openTimeout
	if got := b.State(); got != HalfOpen {
		t.Fatalf("State() = %v, want %v (timeout elapsed)", got, HalfOpen)
	}
}

func TestBreaker_HalfOpenSuccessClosesTheBreaker(t *testing.T) {
	clock := newFakeClock()
	b := NewBreaker(clock, 1, time.Second, 1)

	_ = b.Execute(func() error { return errBoom }) // Open
	clock.Advance(2 * time.Second)                 // -> HalfOpen on next check

	err := b.Execute(func() error { return nil }) // the probe succeeds
	if err != nil {
		t.Fatalf("Execute() probe = %v, want nil", err)
	}
	if got := b.State(); got != Closed {
		t.Fatalf("State() = %v, want %v after a successful half-open probe", got, Closed)
	}
}

func TestBreaker_HalfOpenFailureReopensAndRestartsTimeout(t *testing.T) {
	clock := newFakeClock()
	b := NewBreaker(clock, 1, time.Second, 1)

	_ = b.Execute(func() error { return errBoom }) // Open
	clock.Advance(2 * time.Second)                 // -> HalfOpen

	err := b.Execute(func() error { return errBoom }) // the probe fails
	if !errors.Is(err, errBoom) {
		t.Fatalf("Execute() probe = %v, want %v", err, errBoom)
	}
	if got := b.State(); got != Open {
		t.Fatalf("State() = %v, want %v after a failed half-open probe", got, Open)
	}

	// The timeout must have restarted: advancing by less than openTimeout
	// again must NOT be enough to leave Open.
	clock.Advance(500 * time.Millisecond)
	if got := b.State(); got != Open {
		t.Fatalf("State() = %v, want %v (openTimeout should have restarted from the failed probe)", got, Open)
	}
}

// TestBreaker_HalfOpenAdmitsOnlyLimitedConcurrentProbes exercises the
// concurrency-sensitive part of the state machine: run with -race.
func TestBreaker_HalfOpenAdmitsOnlyLimitedConcurrentProbes(t *testing.T) {
	clock := newFakeClock()
	const maxProbes = 2
	b := NewBreaker(clock, 1, time.Second, maxProbes)

	_ = b.Execute(func() error { return errBoom }) // Open
	clock.Advance(2 * time.Second)                 // -> HalfOpen

	release := make(chan struct{})
	var admitted int32
	var rejected int32

	const attempts = 5
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := b.Execute(func() error {
				atomic.AddInt32(&admitted, 1)
				<-release
				return nil
			})
			if errors.Is(err, ErrOpen) {
				atomic.AddInt32(&rejected, 1)
			}
		}()
	}

	// Let every goroutine reach Execute before releasing the probes that
	// got in. Generous relative to scheduling 5 goroutines.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if got := atomic.LoadInt32(&admitted); got != maxProbes {
		t.Fatalf("admitted %d concurrent probes, want exactly %d (halfOpenMaxProbes)", got, maxProbes)
	}
	if got := atomic.LoadInt32(&rejected); got != attempts-maxProbes {
		t.Fatalf("rejected %d calls, want %d", got, attempts-maxProbes)
	}
}
