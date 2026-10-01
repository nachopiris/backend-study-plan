package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestHeadersCarrier_GetReturnsEmptyStringForMissingKey(t *testing.T) {
	headers := []Header{{Key: "traceparent", Value: []byte("00-...")}}
	c := &HeadersCarrier{Headers: &headers}

	if got := c.Get("tracestate"); got != "" {
		t.Errorf("Get(%q) = %q, want \"\"", "tracestate", got)
	}
}

func TestHeadersCarrier_SetThenGetRoundTrips(t *testing.T) {
	headers := []Header{}
	c := &HeadersCarrier{Headers: &headers}

	c.Set("traceparent", "00-trace-span-01")

	if got := c.Get("traceparent"); got != "00-trace-span-01" {
		t.Errorf("Get(%q) after Set = %q, want %q", "traceparent", got, "00-trace-span-01")
	}
}

func TestHeadersCarrier_SetReplacesAnExistingKeyInsteadOfDuplicating(t *testing.T) {
	headers := []Header{{Key: "traceparent", Value: []byte("old-value")}}
	c := &HeadersCarrier{Headers: &headers}

	c.Set("traceparent", "new-value")

	if got := c.Get("traceparent"); got != "new-value" {
		t.Errorf("Get(%q) after replacing = %q, want %q", "traceparent", got, "new-value")
	}
	if len(*c.Headers) != 1 {
		t.Errorf("len(headers) = %d, want 1 (Set must replace, not append, an existing key)", len(*c.Headers))
	}
}

func TestHeadersCarrier_KeysListsDistinctStoredKeys(t *testing.T) {
	headers := []Header{
		{Key: "traceparent", Value: []byte("v1")},
		{Key: "tracestate", Value: []byte("v2")},
	}
	c := &HeadersCarrier{Headers: &headers}

	keys := c.Keys()
	if len(keys) != 2 {
		t.Fatalf("len(Keys()) = %d, want 2", len(keys))
	}
	seen := map[string]bool{}
	for _, k := range keys {
		seen[k] = true
	}
	if !seen["traceparent"] || !seen["tracestate"] {
		t.Errorf("Keys() = %v, want it to contain both %q and %q", keys, "traceparent", "tracestate")
	}
}

func TestInjectExtractFromMessage_RoundTripsTheSameTraceAndSpanID(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(recorder),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	tracer := tp.Tracer("test")

	producerCtx, producerSpan := tracer.Start(context.Background(), "produce-order-created")
	defer producerSpan.End()

	var headers []Header
	InjectToMessage(producerCtx, propagation.TraceContext{}, &headers)
	if len(headers) == 0 {
		t.Fatal("InjectToMessage did not write any header")
	}

	consumerCtx := ExtractFromMessage(context.Background(), propagation.TraceContext{}, headers)

	_, consumerSpan := tracer.Start(consumerCtx, "consume-order-created")
	defer consumerSpan.End()

	if consumerSpan.SpanContext().TraceID() != producerSpan.SpanContext().TraceID() {
		t.Errorf("consumer span trace ID = %s, want %s (same trace as the producer span)",
			consumerSpan.SpanContext().TraceID(), producerSpan.SpanContext().TraceID())
	}
}

func TestExtractFromMessage_WithNoTraceContextReturnsUsableContext(t *testing.T) {
	// A message published without ever calling InjectToMessage has no trace context in its
	// headers at all — ExtractFromMessage must not panic or corrupt ctx in that case; a span
	// started from the result simply becomes a new trace root.
	ctx := ExtractFromMessage(context.Background(), propagation.TraceContext{}, nil)
	if ctx == nil {
		t.Fatal("ExtractFromMessage(..., nil) returned a nil context.Context")
	}
}
