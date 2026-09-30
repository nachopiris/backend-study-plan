package pool

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestRun_ProcessesAllJobs(t *testing.T) {
	jobs := make([]int, 100)
	for i := range jobs {
		jobs[i] = i
	}

	var processed int32
	fn := func(ctx context.Context, job int) error {
		atomic.AddInt32(&processed, 1)
		return nil
	}

	if err := Run(context.Background(), jobs, 5, fn); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&processed); got != int32(len(jobs)) {
		t.Fatalf("expected all %d jobs to be processed, got %d", len(jobs), got)
	}
}

func TestRun_ConcurrencyNeverExceedsWorkers(t *testing.T) {
	const workers = 4
	jobs := make([]int, 50)
	for i := range jobs {
		jobs[i] = i
	}

	var current int32
	var maxSeen int32
	fn := func(ctx context.Context, job int) error {
		n := atomic.AddInt32(&current, 1)
		for {
			old := atomic.LoadInt32(&maxSeen)
			if n <= old || atomic.CompareAndSwapInt32(&maxSeen, old, n) {
				break
			}
		}
		// Small, generous sleep to encourage (not guarantee) overlap between
		// workers; the assertion below is an upper bound, so it holds
		// regardless of whether overlap actually happens on a given run.
		time.Sleep(5 * time.Millisecond)
		atomic.AddInt32(&current, -1)
		return nil
	}

	if err := Run(context.Background(), jobs, workers, fn); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&maxSeen); got > int32(workers) {
		t.Fatalf("observed concurrency %d exceeds workers limit %d", got, workers)
	}
}

func TestRun_ContextCancellationStopsPromptly(t *testing.T) {
	started := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make([]int, 500)
	for i := range jobs {
		jobs[i] = i
	}

	var processed int32
	fn := func(jobCtx context.Context, job int) error {
		atomic.AddInt32(&processed, 1)
		select {
		case started <- struct{}{}:
		default:
		}
		<-jobCtx.Done()
		return jobCtx.Err()
	}

	go func() {
		<-started
		time.Sleep(20 * time.Millisecond) // let a handful of workers start
		cancel()
	}()

	start := time.Now()
	err := Run(ctx, jobs, 5, fn)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error after context cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected error to wrap context.Canceled, got %v", err)
	}
	const bound = 2 * time.Second
	if elapsed > bound {
		t.Fatalf("Run took %s to return after cancellation; expected under %s", elapsed, bound)
	}
	if got := atomic.LoadInt32(&processed); got >= int32(len(jobs)) {
		t.Fatalf("expected cancellation to stop dispatching before all %d jobs started, got %d", len(jobs), got)
	}
}

func TestRun_FirstErrorCancelsRemainingWork(t *testing.T) {
	sentinel := errors.New("boom")
	const triggerJob = 3

	jobs := make([]int, 200)
	for i := range jobs {
		jobs[i] = i
	}

	var processed int32
	fn := func(jobCtx context.Context, job int) error {
		atomic.AddInt32(&processed, 1)
		if job == triggerJob {
			return sentinel
		}
		<-jobCtx.Done()
		return jobCtx.Err()
	}

	start := time.Now()
	err := Run(context.Background(), jobs, 5, fn)
	elapsed := time.Since(start)

	if !errors.Is(err, sentinel) {
		t.Fatalf("expected the first observed error to be %v, got %v", sentinel, err)
	}
	const bound = 2 * time.Second
	if elapsed > bound {
		t.Fatalf("Run took %s to stop after an error; expected under %s", elapsed, bound)
	}
	if got := atomic.LoadInt32(&processed); got >= int32(len(jobs)) {
		t.Fatalf("expected the error to stop dispatching before all %d jobs started, got %d", len(jobs), got)
	}
}

func TestRun_NoGoroutineLeaks(t *testing.T) {
	before := runtime.NumGoroutine()

	jobs := make([]int, 50)
	for i := range jobs {
		jobs[i] = i
	}

	// Happy path.
	happy := func(ctx context.Context, job int) error { return nil }
	if err := Run(context.Background(), jobs, 5, happy); err != nil {
		t.Fatalf("unexpected error on happy path: %v", err)
	}

	// Error path: cooperative jobs must be released by internal cancellation.
	sentinel := errors.New("boom")
	errPath := func(jobCtx context.Context, job int) error {
		if job == 0 {
			return sentinel
		}
		<-jobCtx.Done()
		return jobCtx.Err()
	}
	_ = Run(context.Background(), jobs, 5, errPath)

	// External cancellation path, already canceled before Run starts.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelPath := func(jobCtx context.Context, job int) error {
		<-jobCtx.Done()
		return jobCtx.Err()
	}
	_ = Run(ctx, jobs, 5, cancelPath)

	// Goroutines may take a moment to fully unwind; poll with a generous
	// bound instead of a fixed sleep.
	deadline := time.Now().Add(2 * time.Second)
	var after int
	for {
		runtime.GC()
		after = runtime.NumGoroutine()
		if after <= before+2 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if after > before+2 {
		t.Fatalf("possible goroutine leak: had %d goroutines before, %d after (2-goroutine margin)", before, after)
	}
}
