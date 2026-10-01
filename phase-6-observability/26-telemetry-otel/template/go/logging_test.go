package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTraceHandler_AddsTraceAndSpanIDWhenContextCarriesASpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(recorder),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	tracer := tp.Tracer("test")

	var buf bytes.Buffer
	logger := slog.New(NewTraceHandler(slog.NewJSONHandler(&buf, nil)))

	ctx, span := tracer.Start(context.Background(), "handle-order")
	logger.InfoContext(ctx, "processing order", "order_id", "order-1")
	span.End()

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, buf.String())
	}

	wantTraceID := span.SpanContext().TraceID().String()
	wantSpanID := span.SpanContext().SpanID().String()

	if got, _ := record["trace_id"].(string); got != wantTraceID {
		t.Errorf("record[\"trace_id\"] = %q, want %q", got, wantTraceID)
	}
	if got, _ := record["span_id"].(string); got != wantSpanID {
		t.Errorf("record[\"span_id\"] = %q, want %q", got, wantSpanID)
	}
	if got, _ := record["order_id"].(string); got != "order-1" {
		t.Errorf("record[\"order_id\"] = %q, want %q (original attributes must survive)", got, "order-1")
	}
}

func TestTraceHandler_LeavesRecordUnchangedWithoutAValidSpan(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewTraceHandler(slog.NewJSONHandler(&buf, nil)))

	logger.InfoContext(context.Background(), "no span in this context")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if _, present := record["trace_id"]; present {
		t.Errorf("record contains a trace_id attribute %v, want none (no valid span in context)", record["trace_id"])
	}
	if _, present := record["span_id"]; present {
		t.Errorf("record contains a span_id attribute %v, want none (no valid span in context)", record["span_id"])
	}
}

func TestTraceHandler_WithAttrsPreservesCorrelationOnTheResultingLogger(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(recorder),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	tracer := tp.Tracer("test")

	var buf bytes.Buffer
	base := slog.New(NewTraceHandler(slog.NewJSONHandler(&buf, nil)))
	logger := base.With("component", "checkout") // exercises Handler.WithAttrs under the hood.

	ctx, span := tracer.Start(context.Background(), "handle-order")
	logger.InfoContext(ctx, "processing order")
	span.End()

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if got, _ := record["component"].(string); got != "checkout" {
		t.Errorf("record[\"component\"] = %q, want %q (With() attributes must survive)", got, "checkout")
	}
	wantTraceID := span.SpanContext().TraceID().String()
	if got, _ := record["trace_id"].(string); got != wantTraceID {
		t.Errorf("record[\"trace_id\"] = %q, want %q (trace correlation must survive a With() call, see WithAttrs's doc comment)", got, wantTraceID)
	}
}
