package breaker

import (
	"context"
	"errors"
)

// ErrBulkheadFull is returned by Bulkhead.Execute when both the concurrency
// limit and the waiting-queue limit are exhausted.
var ErrBulkheadFull = errors.New("bulkhead: no room to run or wait")

// Bulkhead limits how much concurrent work a dependency can receive from
// this process, independent of whether that dependency is healthy. The
// name comes from a ship's bulkheads: compartments that keep one flooded
// section from sinking the whole ship. Here, a slow or saturated dependency
// can only ever occupy maxConcurrent + maxWaiting units of this process's
// capacity, never all of it.
type Bulkhead struct {
	maxConcurrent int
	maxWaiting    int
}

// NewBulkhead builds a Bulkhead that allows at most maxConcurrent calls to
// be running fn at the same time, with at most maxWaiting more allowed to
// queue up waiting for a slot. Both must be >= 0; maxConcurrent should
// normally be >= 1 for the bulkhead to be useful at all.
func NewBulkhead(maxConcurrent, maxWaiting int) *Bulkhead {
	return &Bulkhead{maxConcurrent: maxConcurrent, maxWaiting: maxWaiting}
}

// Execute runs fn if a concurrency slot is available now or becomes
// available while waiting in the queue; otherwise, it returns
// ErrBulkheadFull without running fn.
//
// Contract:
//   - If fewer than maxConcurrent calls are currently running fn, fn runs
//     immediately.
//   - Otherwise, if fewer than maxWaiting callers are already queued, this
//     call queues and blocks until a slot frees up (or ctx is canceled,
//     whichever happens first).
//   - Otherwise (both the concurrency limit and the queue are full),
//     Execute returns ErrBulkheadFull immediately, without waiting and
//     without running fn.
//   - If ctx is canceled while this call is queued, Execute returns
//     ctx.Err() promptly and gives up its place in the queue.
//   - Execute returns fn's own error unchanged when fn was actually called.
//
// Not implemented yet: always returns ErrBulkheadFull without running fn.
func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
	return ErrBulkheadFull
}
