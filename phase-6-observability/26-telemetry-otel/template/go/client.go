package telemetry

import (
	"net/http"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// HTTPDoer is the shape TracedClient wraps — satisfied by *http.Client, and by any fake you build
// for tests. Keeping it this small (one method) is what lets tests substitute a fake Doer instead
// of making real network calls.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// TracedClient wraps an HTTPDoer so every outbound request:
//   - starts a CLIENT span, child of whatever span is active in the request's context (via
//     trace.WithSpanKind(trace.SpanKindClient)),
//   - injects the active trace context into the request's headers using Propagator
//     (propagation.TraceContext{} if Propagator is nil) and propagation.HeaderCarrier(req.Header),
//     so the callee can continue the same trace (see client_test.go for what the resulting
//     header looks like),
//   - records the outcome on the span, and ends it.
type TracedClient struct {
	Tracer     trace.Tracer
	Propagator propagation.TextMapPropagator // nil means propagation.TraceContext{}
	Doer       HTTPDoer
}

// Do sends req through c.Doer, instrumented as described in TracedClient's doc comment.
//
// Contract:
//   - The span name should identify the call (for example "<METHOD> <path>" using req.Method and
//     req.URL.Path).
//   - Header injection happens AFTER the span is started (so the span's own trace/span IDs are
//     the ones injected) and BEFORE c.Doer.Do is called.
//   - On success, the span gets an "http.status_code" int attribute from resp.StatusCode, and a
//     status code >= 500 marks the span as codes.Error (go.opentelemetry.io/otel/codes); the
//     *http.Response is returned unchanged.
//   - On failure (c.Doer.Do returns a non-nil error), the span records the error
//     (span.RecordError) and is marked codes.Error; the error propagates to the caller unchanged
//     — Do never swallows or wraps it.
//   - The span ends exactly once, regardless of which path above was taken (hint: defer).
//
// Not implemented yet: calls c.Doer.Do(req) directly, without starting any span or injecting any
// header.
func (c *TracedClient) Do(req *http.Request) (*http.Response, error) {
	return c.Doer.Do(req)
}
