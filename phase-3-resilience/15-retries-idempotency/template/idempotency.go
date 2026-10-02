package retry

import (
	"net/http"
	"sync"
)

// IdempotencyKeyHeader is the HTTP header clients set to make a POST
// request safe to retry.
const IdempotencyKeyHeader = "Idempotency-Key"

// requestState tracks one idempotency key's lifecycle: either a request is
// currently being executed for it ("in_progress"), or a response has
// already been recorded for it and can be replayed ("done").
type requestState int

const (
	stateInProgress requestState = iota
	stateDone
)

// storedResponse is what IdempotencyMiddleware remembers about a completed
// request, so a later request with the same key and body can replay it
// instead of re-executing the handler.
type storedResponse struct {
	state      requestState
	bodyHash   [32]byte
	statusCode int
	body       []byte
	header     http.Header
}

// IdempotencyMiddleware wraps an http.Handler (meant for a POST endpoint
// such as /payments) so that retried requests do not re-execute the
// underlying operation.
//
// Contract (enforced by the tests, choose this exact behavior when you
// implement Wrap):
//   - A request with no Idempotency-Key header always passes through
//     unchanged, every time: idempotency is opt-in per request.
//   - The FIRST request seen for a given key executes the wrapped handler
//     normally, and its response (status code, body, and headers) is
//     stored under that key once the handler returns.
//   - A LATER request with the SAME key and the SAME body (compared by
//     content, not byte-for-byte identity of the io.Reader) must NOT
//     re-execute the handler: it replays the stored response verbatim.
//   - A LATER request with the SAME key but a DIFFERENT body must NOT
//     execute the handler either: it gets 422 Unprocessable Entity,
//     signaling a client bug (reusing a key for a different operation).
//   - A request that arrives for a key WHILE the first request for that
//     key is still executing (concurrently, before any response was
//     stored) must NOT execute the handler and must NOT block waiting for
//     the first one to finish: it gets 409 Conflict immediately. Once the
//     in-flight request finishes, later requests with the same key and
//     body replay its stored response as usual.
//
// This in-memory implementation is for learning and testing only (it is
// never persisted and never expires); a production version would use a
// shared store with a TTL.
type IdempotencyMiddleware struct {
	mu      sync.Mutex
	records map[string]*storedResponse
}

// NewIdempotencyMiddleware returns an IdempotencyMiddleware with no
// recorded keys.
func NewIdempotencyMiddleware() *IdempotencyMiddleware {
	return &IdempotencyMiddleware{records: make(map[string]*storedResponse)}
}

// Wrap returns next wrapped with idempotency-key handling, per the contract
// documented on IdempotencyMiddleware.
//
// Not implemented yet: passes every request through unchanged, with no
// deduplication at all.
func (m *IdempotencyMiddleware) Wrap(next http.Handler) http.Handler {
	return next
}
