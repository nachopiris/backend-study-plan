// Package loadshed implements, by hand, a token-bucket rate limiter
// (limiter.go) and a load-shedding HTTP middleware (shed.go) — two
// complementary ways of protecting a service from more load than it can
// handle. golang.org/x/time/rate is used ONLY in limiter_compare_test.go,
// to compare your by-hand limiter against a real library's; the limiter and
// middleware themselves have no external dependencies.
package loadshed

import (
	"sync"
	"time"
)

// Clock abstracts time.Now so tests can control the passage of time instead
// of waiting on real refill intervals.
type Clock interface {
	Now() time.Time
}

// RealClock is a production Clock backed by time.Now(). Already
// implemented: use it outside of tests.
type RealClock struct{}

// Now returns the current wall-clock time.
func (RealClock) Now() time.Time { return time.Now() }

// Limiter is a token-bucket rate limiter: it holds up to burst tokens,
// refills at rate tokens per second, and each call to Allow consumes one
// token if one is available.
type Limiter struct {
	clock Clock
	rate  float64 // tokens added per second
	burst float64 // maximum tokens the bucket can hold

	mu         sync.Mutex
	tokens     float64
	lastRefill time.Time
}

// NewLimiter builds a Limiter whose bucket starts FULL (burst tokens
// available immediately), refilling at rate tokens per second up to a
// maximum of burst tokens. Both rate and burst must be > 0.
func NewLimiter(clock Clock, rate, burst float64) *Limiter {
	return &Limiter{
		clock:      clock,
		rate:       rate,
		burst:      burst,
		tokens:     burst,
		lastRefill: clock.Now(),
	}
}

// Allow reports whether a single request may proceed right now. If it
// returns true, exactly one token was consumed from the bucket. If it
// returns false, the bucket was empty and NO token was consumed.
//
// Contract:
//   - Before deciding, Allow refills the bucket based on how much time has
//     passed (per Clock) since the last refill, at `rate` tokens per
//     second, never exceeding `burst` tokens total.
//   - Refilling is fractional: advancing the clock by half of 1/rate
//     seconds adds exactly half a token, not zero and not a whole token.
//
// Not implemented yet: always returns false.
func (l *Limiter) Allow() bool {
	return false
}
