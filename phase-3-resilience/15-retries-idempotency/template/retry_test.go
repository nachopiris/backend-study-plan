package retry

import (
	"context"
	"errors"
	"math/rand"
	"testing"
	"time"
)

// fakeSleeper records every requested delay instead of actually sleeping,
// so backoff tests run instantly and deterministically.
func fakeSleeper(delays *[]time.Duration) Sleeper {
	return func(ctx context.Context, d time.Duration) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		*delays = append(*delays, d)
		return nil
	}
}

// fakeRandom returns a fixed sequence of Float64 values, repeating the last
// one if Retry asks for more than were provided.
type fakeRandom struct {
	values []float64
	i      int
}

func (f *fakeRandom) Float64() float64 {
	if f.i >= len(f.values) {
		return f.values[len(f.values)-1]
	}
	v := f.values[f.i]
	f.i++
	return v
}

func TestRetry_SucceedsFirstAttempt_NeverSleeps(t *testing.T) {
	var delays []time.Duration
	calls := 0
	err := Retry(context.Background(),
		Policy{MaxAttempts: 3, BaseDelay: 10 * time.Millisecond, MaxDelay: time.Second},
		fakeSleeper(&delays), &fakeRandom{values: []float64{0.5}},
		func(ctx context.Context, attempt int) error {
			calls++
			return nil
		},
	)
	if err != nil {
		t.Fatalf("Retry() = %v, want nil", err)
	}
	if calls != 1 {
		t.Fatalf("op called %d times, want 1", calls)
	}
	if len(delays) != 0 {
		t.Fatalf("recorded %d sleeps, want 0 (op succeeded on the first attempt)", len(delays))
	}
}

func TestRetry_NonRetryableErrorStopsImmediately(t *testing.T) {
	sentinel := errors.New("bad request")
	var delays []time.Duration
	calls := 0
	err := Retry(context.Background(),
		Policy{
			MaxAttempts: 5,
			BaseDelay:   10 * time.Millisecond,
			MaxDelay:    time.Second,
			IsRetryable: func(err error) bool { return !errors.Is(err, sentinel) },
		},
		fakeSleeper(&delays), &fakeRandom{values: []float64{0.5}},
		func(ctx context.Context, attempt int) error {
			calls++
			return sentinel
		},
	)
	if !errors.Is(err, sentinel) {
		t.Fatalf("Retry() = %v, want %v", err, sentinel)
	}
	if calls != 1 {
		t.Fatalf("op called %d times, want 1 (error was not retryable)", calls)
	}
	if len(delays) != 0 {
		t.Fatalf("recorded %d sleeps, want 0 (should not sleep before giving up)", len(delays))
	}
}

func TestRetry_ExhaustsMaxAttemptsAndReturnsLastError(t *testing.T) {
	var delays []time.Duration
	calls := 0
	err := Retry(context.Background(),
		Policy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Second},
		fakeSleeper(&delays), &fakeRandom{values: []float64{0.5}},
		func(ctx context.Context, attempt int) error {
			calls++
			return fmtAttemptError(attempt)
		},
	)
	if calls != 3 {
		t.Fatalf("op called %d times, want 3 (MaxAttempts)", calls)
	}
	if len(delays) != 2 {
		t.Fatalf("recorded %d sleeps, want 2 (one between each pair of attempts)", len(delays))
	}
	want := fmtAttemptError(3)
	if err == nil || err.Error() != want.Error() {
		t.Fatalf("Retry() = %v, want the LAST attempt's error: %v", err, want)
	}
}

func fmtAttemptError(attempt int) error {
	return errors.New(map[int]string{1: "fail-1", 2: "fail-2", 3: "fail-3", 4: "fail-4", 5: "fail-5"}[attempt])
}

func TestRetry_FullJitterIsProportionalToTheCap(t *testing.T) {
	// rnd always returns 0.5: each delay should be exactly half of its cap.
	var delays []time.Duration
	calls := 0
	_ = Retry(context.Background(),
		Policy{MaxAttempts: 5, BaseDelay: 100 * time.Millisecond, MaxDelay: 800 * time.Millisecond},
		fakeSleeper(&delays), &fakeRandom{values: []float64{0.5}},
		func(ctx context.Context, attempt int) error {
			calls++
			return errors.New("always fails")
		},
	)
	if calls != 5 {
		t.Fatalf("op called %d times, want 5", calls)
	}
	// caps: 100ms, 200ms, 400ms, 800ms (base*2^(N-2), capped at MaxDelay)
	want := []time.Duration{50 * time.Millisecond, 100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}
	if len(delays) != len(want) {
		t.Fatalf("recorded %d delays, want %d: %v", len(delays), len(want), delays)
	}
	for i, d := range delays {
		if d != want[i] {
			t.Errorf("delays[%d] = %v, want %v (cap %v * 0.5)", i, d, want[i], want[i]*2)
		}
	}
}

func TestRetry_FullJitterNeverExceedsMaxDelay(t *testing.T) {
	// rnd always returns 1.0 (the supremum of the valid [0,1) range, used
	// here to check the upper bound without relying on float rounding
	// elsewhere): every delay should land exactly on its cap, and the cap
	// must never exceed MaxDelay even as BaseDelay*2^(N-2) grows past it.
	var delays []time.Duration
	_ = Retry(context.Background(),
		Policy{MaxAttempts: 4, BaseDelay: 100 * time.Millisecond, MaxDelay: 300 * time.Millisecond},
		fakeSleeper(&delays), &fakeRandom{values: []float64{1.0}},
		func(ctx context.Context, attempt int) error {
			return errors.New("always fails")
		},
	)
	want := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond}
	if len(delays) != len(want) {
		t.Fatalf("recorded %d delays, want %d: %v", len(delays), len(want), delays)
	}
	for i, d := range delays {
		if d != want[i] {
			t.Errorf("delays[%d] = %v, want %v", i, d, want[i])
		}
	}
}

func TestRetry_AlreadyCanceledContext_NeverCallsOp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	var delays []time.Duration
	err := Retry(ctx,
		Policy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Second},
		fakeSleeper(&delays), &fakeRandom{values: []float64{0.5}},
		func(ctx context.Context, attempt int) error {
			calls++
			return nil
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Retry() = %v, want %v", err, context.Canceled)
	}
	if calls != 0 {
		t.Fatalf("op called %d times, want 0 (context was already canceled)", calls)
	}
}

func TestRetry_ContextCanceledDuringSleepStopsPromptly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// This sleeper blocks until ctx is done, no matter what duration is
	// requested — standing in for "a real sleep that would otherwise take
	// far longer than this test should ever wait."
	blockingSleeper := func(ctx context.Context, d time.Duration) error {
		<-ctx.Done()
		return ctx.Err()
	}

	calls := 0
	op := func(ctx context.Context, attempt int) error {
		calls++
		return errors.New("always fails")
	}

	done := make(chan error, 1)
	go func() {
		done <- Retry(ctx,
			Policy{MaxAttempts: 10, BaseDelay: time.Millisecond, MaxDelay: time.Second},
			blockingSleeper, &fakeRandom{values: []float64{0.5}}, op,
		)
	}()

	// Give the goroutine enough time to run the first attempt and reach the
	// sleep before we cancel. This is a real (short) sleep because there is
	// no way to synchronize on "about to call the sleeper" without changing
	// the contract under test; the bound is generous relative to how fast a
	// single failed op call runs.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Retry() = %v, want %v", err, context.Canceled)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Retry did not return promptly after context cancellation during sleep")
	}
	if calls != 1 {
		t.Fatalf("op called %d times, want exactly 1 (no attempt after cancellation during the sleep)", calls)
	}
}

// TestRetry_WorksWithRealRandom is a light smoke test that Retry also
// behaves with math/rand (seeded, so still deterministic across runs),
// not just with a hand-rolled fake.
func TestRetry_WorksWithRealRandom(t *testing.T) {
	var delays []time.Duration
	calls := 0
	err := Retry(context.Background(),
		Policy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond},
		fakeSleeper(&delays), rand.New(rand.NewSource(42)),
		func(ctx context.Context, attempt int) error {
			calls++
			if attempt < 3 {
				return errors.New("not yet")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("Retry() = %v, want nil", err)
	}
	if calls != 3 {
		t.Fatalf("op called %d times, want 3", calls)
	}
	if len(delays) != 2 {
		t.Fatalf("recorded %d sleeps, want 2", len(delays))
	}
	for i, d := range delays {
		if d < 0 || d > 10*time.Millisecond {
			t.Errorf("delays[%d] = %v, out of the [0, MaxDelay] range", i, d)
		}
	}
}
