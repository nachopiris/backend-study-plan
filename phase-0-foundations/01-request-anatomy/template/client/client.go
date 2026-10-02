// Package client provides a minimal HTTP GET helper that must never hang longer
// than an explicit timeout, even if the server accepts the connection and then
// never sends a response.
package client

import (
	"context"
	"errors"
	"time"
)

// Fetch performs an HTTP GET request to url and returns the response body.
//
// Contract:
//   - Fetch must not block longer than timeout, measured from the moment Fetch
//     is called, regardless of what the server does (including a server that
//     accepts the TCP connection and never writes anything back).
//   - Fetch must also respect ctx: if ctx is already canceled, or is canceled
//     while the request is in flight, Fetch must return promptly with an error.
//   - On success, Fetch returns the full response body and a nil error.
//   - On failure (timeout, cancellation, connection error, non-2xx status —
//     your choice how strict to be on status codes, document it in NOTES.md),
//     Fetch returns a nil body and a non-nil, descriptive error.
//
// Implement this using the standard library only (net/http, context, time).
// Think about where the timeout should be applied: on the context passed to
// the outgoing request, on the http.Client itself, or both — and what each
// choice means for an already-canceled parent ctx.
func Fetch(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
	return nil, errors.New("not implemented")
}
