package telemetry

import (
	"context"
	"log/slog"
)

// TraceHandler wraps an slog.Handler so every log record emitted through it gets trace_id and
// span_id attributes added automatically, read from the span context active in the ctx passed to
// the logging call — this is what lets Grafana correlate a log line in Loki with the exact trace
// (and span) in Tempo that produced it.
//
// IMPORTANT: correlation only works with the *Context variants of slog's logging calls
// (logger.InfoContext(ctx, ...), logger.ErrorContext(ctx, ...), etc.) — plain logger.Info(...)
// passes context.Background() internally, which carries no span.
//
// TraceHandler embeds the wrapped Handler so Enabled and WithGroup behave exactly like the
// wrapped handler already does; only Handle and WithAttrs need to preserve the TraceHandler
// wrapping themselves (see WithAttrs's doc comment for why).
type TraceHandler struct {
	slog.Handler
}

// NewTraceHandler wraps next (for example slog.NewJSONHandler(os.Stdout, nil)) to add trace
// correlation to every record handled through the result.
func NewTraceHandler(next slog.Handler) *TraceHandler {
	return &TraceHandler{Handler: next}
}

// Handle implements slog.Handler.
//
// Contract:
//   - If trace.SpanContextFromContext(ctx) is valid (.IsValid()), add two attributes to record
//     before forwarding it to the wrapped Handler: trace_id (the span context's TraceID, as its
//     lowercase hex string — TraceID.String()) and span_id (SpanID.String(), same idea).
//   - If ctx carries no valid span context, forward record to the wrapped Handler completely
//     unchanged — never add an empty or zero trace_id/span_id attribute.
//   - record.Clone() before mutating it if you add attributes — slog.Record documents that a
//     Record must not be modified after it has been passed to a Handler, and h.Handler.Handle
//     is itself a Handler receiving it.
//
// Not implemented yet: forwards to the wrapped Handler completely unchanged — no attribute is
// ever added, so a trace_id/span_id correlation test against this stub fails.
func (h *TraceHandler) Handle(ctx context.Context, record slog.Record) error {
	return h.Handler.Handle(ctx, record)
}

// WithAttrs implements slog.Handler. It must return a *TraceHandler wrapping the Handler returned
// by the embedded Handler's own WithAttrs — returning the raw inner Handler instead would lose
// trace correlation for every subsequent call made through a logger built with
// logger.With(...).
//
// Not implemented yet: returns h unchanged, discarding attrs (the wrapped Handler never sees
// them).
func (h *TraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}
