// Package pool implements a bounded-concurrency worker pool with context-aware
// cancellation, built by hand (no errgroup, no third-party concurrency helpers).
package pool

import (
	"context"
	"errors"
)

// Run processes every item in jobs using at most workers concurrent goroutines,
// each calling fn(ctx, job) for one job at a time.
//
// Contract:
//   - Every job in jobs is passed to fn exactly once, unless processing stops
//     early because of a cancellation or an error (see below).
//   - At no point are more than workers calls to fn running concurrently.
//   - If ctx is canceled (or a deadline is exceeded) before or during
//     processing, Run must stop dispatching new jobs and return promptly once
//     in-flight calls to fn finish; it must not wait for jobs that were never
//     started. The returned error must reflect the cancellation (ctx.Err() or
//     an error that wraps it).
//   - If any call to fn returns a non-nil error, Run must stop dispatching new
//     jobs (jobs not yet started are simply not processed) and return that
//     first error. "First" means first in the sense of "first to be observed
//     by Run", which — because this is concurrent — does not have to be the
//     first job in the slice.
//   - The context passed to each call of fn must be one derived internally by
//     Run (e.g. via context.WithCancel(ctx)) and canceled by Run as soon as it
//     observes the first error, in addition to being canceled whenever ctx
//     itself is canceled. This lets cooperative jobs — ones that select on
//     their ctx.Done() — stop early instead of running to completion after
//     the outcome no longer matters. This is the same idea errgroup.WithContext
//     uses; you are building it by hand.
//   - If every job completes without error and ctx is never canceled, Run
//     returns nil.
//   - Run must not leak goroutines in any of the above scenarios: once Run
//     returns, no goroutine it started should still be running or blocked.
//   - workers must be treated as at least 1; how you handle workers <= 0 is
//     your call, but document the decision in NOTES.md.
func Run[T any](ctx context.Context, jobs []T, workers int, fn func(ctx context.Context, job T) error) error {
	return errors.New("not implemented")
}
