// Package retry implements, by hand, a retry loop with exponential backoff
// and full jitter (as described in the AWS Builders' Library article
// "Timeouts, retries, and backoff with jitter"), and an idempotency-key HTTP
// middleware (idempotency.go) that makes retried POST requests safe to
// repeat.
//
// No external dependencies: only the Go standard library.
package retry

import (
	"context"
	"errors"
	"time"
)

// Op is the operation Retry executes. attempt is 1-based (the first call
// has attempt == 1).
type Op func(ctx context.Context, attempt int) error

// Policy configures Retry's attempt count and backoff shape.
type Policy struct {
	// MaxAttempts is the total number of times op may be called, including
	// the first attempt. Must be >= 1.
	MaxAttempts int

	// BaseDelay is the backoff base: the delay before attempt N (N > 1) is
	// drawn uniformly from [0, cap(N)), where
	// cap(N) = min(MaxDelay, BaseDelay * 2^(N-2)).
	BaseDelay time.Duration

	// MaxDelay hard-caps the backoff delay, no matter how many attempts
	// have already happened.
	MaxDelay time.Duration

	// IsRetryable decides whether a given error should trigger another
	// attempt. A nil IsRetryable means every non-nil error is retryable.
	IsRetryable func(err error) bool
}

// Sleeper abstracts time.Sleep so tests can inject a fake clock instead of
// waiting on the wall clock. A correct Sleeper must return promptly with
// ctx.Err() if ctx is done before d elapses — it must never ignore ctx.
type Sleeper func(ctx context.Context, d time.Duration) error

// RealSleeper is a production Sleeper backed by a real timer. It is already
// implemented: you do not need to change it, only use it (or a fake) when
// calling Retry.
func RealSleeper(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Random abstracts the source of jitter so tests can inject a deterministic
// sequence instead of math/rand's. Float64 must return a value in [0, 1).
type Random interface {
	Float64() float64
}

// Retry calls op up to policy.MaxAttempts times, applying exponential
// backoff with full jitter (via sleep and rnd) between attempts.
//
// Contract:
//   - Retry returns nil as soon as op returns nil.
//   - If op returns an error for which policy.IsRetryable returns false (or
//     IsRetryable is nil, where every error is retryable), that distinction
//     only matters while attempts remain: a non-retryable error makes Retry
//     return immediately, without sleeping and without any further attempt,
//     even if attempts remain.
//   - If every attempt is exhausted (policy.MaxAttempts calls to op, all
//     returning an error), Retry returns the error from the LAST attempt.
//   - Before each attempt (including the first) and before each sleep,
//     Retry checks ctx: if ctx is already done, Retry returns ctx.Err()
//     immediately without calling op again.
//   - The delay before attempt N (N > 1) is rnd.Float64() * cap(N), where
//     cap(N) = min(policy.MaxDelay, policy.BaseDelay * 2^(N-2)). There is
//     never a delay before the first attempt, and never a delay after the
//     last attempt (whether it succeeded or not).
//
// Not implemented yet: always returns an error without calling op.
func Retry(ctx context.Context, policy Policy, sleep Sleeper, rnd Random, op Op) error {
	return errors.New("not implemented")
}
