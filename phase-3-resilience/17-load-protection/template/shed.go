package loadshed

import (
	"net/http"
)

// RetryAfterHeader is the header Shed sets, in seconds, when it sheds a
// request — a hint to the caller about how long to wait before trying
// again.
const RetryAfterHeader = "Retry-After"

// RetryAfterSeconds is the fixed value Shed writes into RetryAfterHeader.
// A real system would likely compute this dynamically; a constant is
// enough for this practice.
const RetryAfterSeconds = "1"

// Shed returns a middleware that rejects requests with 503 Service
// Unavailable once more than maxInFlight requests are already being
// handled by the wrapped handler, instead of letting them queue up
// (in the OS, in a goroutine, wherever) and risk taking the whole process
// down with them.
//
// Contract:
//   - While fewer than maxInFlight requests are currently being handled,
//     a new request is let through immediately.
//   - Once maxInFlight requests are in flight, a new request is shed
//     immediately: 503 Service Unavailable, with RetryAfterHeader set to
//     RetryAfterSeconds, and the wrapped handler is NEVER called for that
//     request.
//   - A request is "in flight" from the moment the wrapped handler starts
//     to the moment it returns; the in-flight count must be accurate under
//     concurrent requests (this is exercised with -race).
//
// Not implemented yet: lets every request through, with no shedding at all.
func Shed(maxInFlight int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
