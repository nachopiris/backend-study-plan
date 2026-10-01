// Package breaker implements, by hand, a circuit breaker (breaker.go) and a
// bulkhead (bulkhead.go) — two independent stability patterns that are
// commonly used together but do not depend on each other.
//
// No external dependencies: only the Go standard library.
package breaker

import (
	"errors"
	"sync"
	"time"
)

// State is one of the three states of the circuit breaker's state machine.
type State int

const (
	// Closed is the normal state: calls go through, and consecutive
	// failures are counted.
	Closed State = iota
	// Open rejects every call immediately, without running it, until
	// OpenTimeout has elapsed since the breaker opened.
	Open
	// HalfOpen allows a limited number of probe calls through, to test
	// whether the dependency has recovered, while still rejecting any call
	// beyond that limit.
	HalfOpen
)

func (s State) String() string {
	switch s {
	case Closed:
		return "closed"
	case Open:
		return "open"
	case HalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrOpen is returned by Execute when the breaker does not allow the call
// through, whether because it is Open or because HalfOpen's probe capacity
// is currently exhausted. fn is never called in either case.
var ErrOpen = errors.New("breaker: circuit is open")

// Clock abstracts time.Now so tests can control the passage of time instead
// of waiting on OpenTimeout for real.
type Clock interface {
	Now() time.Time
}

// Breaker is a circuit breaker: it protects a flaky dependency from being
// hammered with calls that are likely to fail, by "opening" after too many
// consecutive failures and rejecting calls for a cooldown period, before
// cautiously trying again.
type Breaker struct {
	clock             Clock
	failureThreshold  int
	openTimeout       time.Duration
	halfOpenMaxProbes int

	mu                   sync.Mutex
	state                State
	consecutiveFailures  int
	openedAt             time.Time
	halfOpenProbesInUse  int
}

// NewBreaker builds a Breaker starting in the Closed state.
//
//   - failureThreshold: number of CONSECUTIVE failures (in Closed) that
//     opens the breaker. Must be >= 1.
//   - openTimeout: how long the breaker stays Open before allowing a probe
//     into HalfOpen.
//   - halfOpenMaxProbes: how many calls may be in flight at once while
//     HalfOpen. Must be >= 1.
func NewBreaker(clock Clock, failureThreshold int, openTimeout time.Duration, halfOpenMaxProbes int) *Breaker {
	return &Breaker{
		clock:             clock,
		failureThreshold:  failureThreshold,
		openTimeout:       openTimeout,
		halfOpenMaxProbes: halfOpenMaxProbes,
		state:             Closed,
	}
}

// State returns the breaker's current state. Calling State may itself
// transition the breaker from Open to HalfOpen if openTimeout has elapsed
// (per Clock), since that transition is driven by the passage of time, not
// by any call to Execute.
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.transitionFromOpenIfTimeoutElapsedLocked()
	return b.state
}

// Execute runs fn if, and only if, the breaker currently allows it:
//   - Closed: fn always runs. If fn returns a non-nil error, that counts as
//     one failure toward failureThreshold; a nil error resets the
//     consecutive-failure count to zero. Reaching failureThreshold
//     transitions to Open.
//   - Open: fn never runs; Execute returns ErrOpen immediately. Once
//     openTimeout has elapsed since the breaker opened, the NEXT call
//     transitions to HalfOpen and is treated as a probe (see below).
//   - HalfOpen: up to halfOpenMaxProbes calls may have fn in flight at the
//     same time; a call beyond that limit returns ErrOpen immediately
//     without running fn. When a probe's fn returns nil, the breaker
//     transitions to Closed (with the failure count reset). When a probe's
//     fn returns a non-nil error, the breaker transitions back to Open
//     (restarting openTimeout from now).
//
// Execute returns fn's own error unchanged when fn was actually called and
// failed; it returns ErrOpen only when fn was NOT called because the
// breaker did not allow it.
func (b *Breaker) Execute(fn func() error) error {
	return errors.New("not implemented")
}

// transitionFromOpenIfTimeoutElapsedLocked must be called with b.mu held.
// It is provided fully implemented: it is plumbing, not the part of the
// exercise about deciding WHEN to call it.
func (b *Breaker) transitionFromOpenIfTimeoutElapsedLocked() {
	if b.state != Open {
		return
	}
	if !b.clock.Now().Before(b.openedAt.Add(b.openTimeout)) {
		b.state = HalfOpen
		b.halfOpenProbesInUse = 0
	}
}
