// Package telemetry implements, by hand, the four pieces of OpenTelemetry instrumentation an
// HTTP service crossing a process boundary typically needs:
//
//   - middleware.go: an HTTP server middleware that starts a SERVER span per request and records
//     RED metrics (Rate, Errors, Duration).
//   - client.go: an outbound HTTP client wrapper that starts a CLIENT span per request and
//     injects W3C trace context into the outgoing headers.
//   - carrier.go: a TextMapCarrier over message headers (not HTTP headers), plus
//     InjectToMessage/ExtractFromMessage, so a trace can cross an asynchronous boundary like
//     Kafka (see practice 18) instead of just a synchronous HTTP call.
//   - logging.go: an slog.Handler wrapper that adds trace_id/span_id to every log record, so logs
//     and traces can be correlated in Grafana (Loki logs linked to Tempo traces).
//
// Only go.opentelemetry.io/otel's API and SDK packages are used — the OTel Collector, Tempo,
// Loki, Prometheus and Grafana that this practice's docker-compose.yml brings up are exercised
// manually (see the README), not from this package's tests, which use the SDK's own in-memory
// test doubles instead (tracetest.SpanRecorder and sdk/metric's ManualReader): no network, no
// Docker, fully deterministic.
package telemetry

import (
	"net/http"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Setting the span status on error uses go.opentelemetry.io/otel/codes.Error, and tagging
// attributes uses go.opentelemetry.io/otel/attribute.String / attribute.Int — both referenced in
// the contract above. Neither package is imported here because the stub body below does not use
// them yet; add the imports yourself when you implement ServerMiddleware.

// statusRecordingWriter wraps an http.ResponseWriter to capture the status code written to it.
// GIVEN: http.ResponseWriter does not expose the status code it already sent, so recording RED
// metrics (which need that status) requires this kind of small wrapper — it is not the point of
// this exercise, so it is implemented for you.
type statusRecordingWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusRecordingWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Write implements http.ResponseWriter. If the handler never calls WriteHeader explicitly (the
// common case for a 200 response), the first Write call implies status 200, exactly like the
// standard library's own http.ResponseWriter does.
func (w *statusRecordingWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// REDMetrics holds the three RED (Rate, Errors, Duration) instruments ServerMiddleware records on
// every request, each one tagged with route and status attributes:
//   - requests: a counter of every request handled, regardless of outcome (Rate).
//   - errors: a counter of requests whose response status was >= 500 (Errors).
//   - duration: a histogram, in milliseconds, of how long each request took (Duration).
type REDMetrics struct {
	requests metric.Int64Counter
	errors   metric.Int64Counter
	duration metric.Float64Histogram
}

// NewREDMetrics creates the three RED instruments on meter. Call it once per process (typically
// at startup next to the rest of your telemetry wiring) and reuse the returned *REDMetrics across
// every request handled by ServerMiddleware — creating instruments per-request leaks memory and
// defeats the SDK's own aggregation.
//
// Suggested instrument names: "http.server.requests", "http.server.errors",
// "http.server.duration" (milliseconds). The exact names are yours to choose as long as the three
// instruments exist and record what their doc comments describe — the tests only check the
// collected metric data, not instrument names.
//
// Not implemented yet: returns (nil, an error), so callers fail loudly instead of silently
// recording into broken instruments.
func NewREDMetrics(meter metric.Meter) (*REDMetrics, error) {
	return nil, errNotImplemented
}

// ServerMiddleware wraps next so that every request handled through the returned middleware:
//
//  1. Extracts any inbound W3C trace context from the request's headers (propagator.Extract,
//     using propagation.HeaderCarrier(r.Header) — propagator.TraceContext{} if propagator is
//     nil) and uses the resulting context.Context as the parent for a new SERVER span
//     (trace.WithSpanKind(trace.SpanKindServer)), so this span becomes a CHILD of an upstream
//     caller's span when one was propagated, and a new trace root otherwise.
//  2. Names the span after the request's route: r.Pattern (the ServeMux-matched pattern, set
//     automatically by net/http since Go 1.22 — see the README) if non-empty, otherwise
//     r.URL.Path.
//  3. Calls next with a request whose context carries the new span, and with a
//     statusRecordingWriter wrapping the original ResponseWriter so the final status code can be
//     read afterwards.
//  4. After next returns, sets the span's attributes "route" (string, the same route used for the
//     span name) and "status" (int, the recorded status code), marks the span status as
//     codes.Error when the recorded status is >= 500, and ends the span.
//  5. Records all three instruments in red with the same route/status attributes: increments
//     requests always, increments errors only when status >= 500, and records the elapsed
//     wall-clock duration (in milliseconds) in duration.
//
// Not implemented yet: calls next unchanged, creating no span and recording no metric.
func ServerMiddleware(tracer trace.Tracer, propagator propagation.TextMapPropagator, red *REDMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return next
	}
}
