package breaker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBulkhead_RunsImmediatelyUnderTheConcurrencyLimit(t *testing.T) {
	bh := NewBulkhead(2, 0)
	var called int32
	err := bh.Execute(context.Background(), func() error {
		atomic.AddInt32(&called, 1)
		return nil
	})
	if err != nil {
		t.Fatalf("Execute() = %v, want nil", err)
	}
	if called != 1 {
		t.Fatalf("fn called %d times, want 1", called)
	}
}

func TestBulkhead_ReturnsFnErrorUnchanged(t *testing.T) {
	bh := NewBulkhead(1, 0)
	sentinel := errors.New("downstream failed")
	err := bh.Execute(context.Background(), func() error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("Execute() = %v, want %v", err, sentinel)
	}
}

// TestBulkhead_RejectsBeyondConcurrencyAndQueueLimits saturates both the
// concurrency limit and the wait queue with blocking calls, and checks that
// anything beyond maxConcurrent+maxWaiting is rejected immediately rather
// than queued or run. Run with -race.
func TestBulkhead_RejectsBeyondConcurrencyAndQueueLimits(t *testing.T) {
	const maxConcurrent = 2
	const maxWaiting = 1
	bh := NewBulkhead(maxConcurrent, maxWaiting)

	release := make(chan struct{})
	var running int32
	blockingFn := func() error {
		atomic.AddInt32(&running, 1)
		<-release
		return nil
	}

	// Fill the concurrency slots.
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = bh.Execute(context.Background(), blockingFn)
		}()
	}
	waitUntil(t, func() bool { return atomic.LoadInt32(&running) == maxConcurrent }, time.Second)

	// Fill the queue with one more caller that must block (not reject, not
	// run yet).
	queuedStarted := make(chan struct{})
	queuedDone := make(chan error, 1)
	go func() {
		close(queuedStarted)
		queuedDone <- bh.Execute(context.Background(), func() error {
			atomic.AddInt32(&running, 1)
			<-release
			return nil
		})
	}()
	<-queuedStarted
	// Give the queued goroutine time to actually reach Execute and queue.
	time.Sleep(30 * time.Millisecond)

	// One more call: concurrency AND queue are both full now, so this must
	// be rejected immediately, without blocking at all.
	rejectDone := make(chan error, 1)
	go func() { rejectDone <- bh.Execute(context.Background(), blockingFn) }()

	select {
	case err := <-rejectDone:
		if !errors.Is(err, ErrBulkheadFull) {
			t.Fatalf("Execute() over both limits = %v, want %v", err, ErrBulkheadFull)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Execute() over both limits should have returned immediately, but it blocked")
	}

	// Release the two running calls: the queued one should now get its
	// slot and run.
	close(release)
	wg.Wait()
	select {
	case err := <-queuedDone:
		if err != nil {
			t.Fatalf("queued Execute() = %v, want nil (it should have eventually run)", err)
		}
	case <-time.After(time.Second):
		t.Fatal("queued caller never got a slot after one freed up")
	}
}

func TestBulkhead_ContextCancellationStopsAQueuedWaitPromptly(t *testing.T) {
	bh := NewBulkhead(1, 1)

	release := make(chan struct{})
	defer close(release)
	go func() {
		_ = bh.Execute(context.Background(), func() error {
			<-release
			return nil
		})
	}()
	waitUntilSlotTaken(t, bh)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- bh.Execute(ctx, func() error { return nil })
	}()

	time.Sleep(30 * time.Millisecond) // let it enter the queue
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Execute() = %v, want %v", err, context.Canceled)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("Execute() did not return promptly after context cancellation while queued")
	}
}

func waitUntil(t *testing.T, cond func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition never became true within timeout")
}

// waitUntilSlotTaken has no visibility into Bulkhead's internals (by
// design); it just gives the first goroutine a moment to reach Execute and
// occupy the only concurrency slot before the test proceeds.
func waitUntilSlotTaken(t *testing.T, bh *Bulkhead) {
	t.Helper()
	time.Sleep(30 * time.Millisecond)
}
