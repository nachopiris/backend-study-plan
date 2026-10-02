package telemetry

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// fakeDoer is a test double for HTTPDoer: it never makes a real network call, so these tests run
// deterministically and instantly, with no server to stand up.
type fakeDoer struct {
	lastReq  *http.Request
	response *http.Response
	err      error
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.lastReq = req
	return f.response, f.err
}

func newTracerWithRecorder(t *testing.T) (trace.Tracer, *tracetest.SpanRecorder) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(recorder),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return tp.Tracer("test"), recorder
}

func TestTracedClient_InjectsW3CTraceparentHeader(t *testing.T) {
	tracer, recorder := newTracerWithRecorder(t)
	doer := &fakeDoer{response: &http.Response{StatusCode: http.StatusOK}}
	client := &TracedClient{Tracer: tracer, Propagator: propagation.TraceContext{}, Doer: doer}

	ctx, parent := tracer.Start(context.Background(), "parent-operation")
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://payments.internal/status", nil)

	if _, err := client.Do(req); err != nil {
		t.Fatalf("client.Do() error = %v, want nil", err)
	}
	parent.End()

	if doer.lastReq == nil {
		t.Fatal("Doer.Do was never called")
	}
	traceparent := doer.lastReq.Header.Get("traceparent")
	if traceparent == "" {
		t.Fatal("outgoing request has no traceparent header; W3C trace context was not injected")
	}

	ended := recorder.Ended()
	var clientSpan sdktrace.ReadOnlySpan
	for _, s := range ended {
		if s.SpanKind() == trace.SpanKindClient {
			clientSpan = s
		}
	}
	if clientSpan == nil {
		t.Fatal("no CLIENT span was recorded")
	}
	wantTraceID := clientSpan.SpanContext().TraceID().String()
	if want := wantTraceID; traceparent[3:3+32] != want {
		t.Errorf("traceparent header trace ID = %q, want %q (from the CLIENT span's own trace ID)\nfull header: %s", traceparent[3:3+32], want, traceparent)
	}
}

func TestTracedClient_ClientSpanIsAChildOfTheActiveSpan(t *testing.T) {
	tracer, recorder := newTracerWithRecorder(t)
	doer := &fakeDoer{response: &http.Response{StatusCode: http.StatusOK}}
	client := &TracedClient{Tracer: tracer, Doer: doer}

	ctx, parentSpan := tracer.Start(context.Background(), "parent-operation")
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://payments.internal/status", nil)
	if _, err := client.Do(req); err != nil {
		t.Fatalf("client.Do() error = %v, want nil", err)
	}
	parentSpan.End()

	var clientSpan sdktrace.ReadOnlySpan
	for _, s := range recorder.Ended() {
		if s.SpanKind() == trace.SpanKindClient {
			clientSpan = s
		}
	}
	if clientSpan == nil {
		t.Fatal("no CLIENT span was recorded")
	}
	if clientSpan.Parent().SpanID() != parentSpan.SpanContext().SpanID() {
		t.Errorf("client span's parent span ID = %s, want %s (the active span when Do was called)",
			clientSpan.Parent().SpanID(), parentSpan.SpanContext().SpanID())
	}
}

func TestTracedClient_RecordsSuccessStatusAttributeAndReturnsResponseUnchanged(t *testing.T) {
	tracer, recorder := newTracerWithRecorder(t)
	wantResp := &http.Response{StatusCode: http.StatusTeapot}
	doer := &fakeDoer{response: wantResp}
	client := &TracedClient{Tracer: tracer, Doer: doer}

	req, _ := http.NewRequest(http.MethodGet, "http://payments.internal/status", nil)
	gotResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do() error = %v, want nil", err)
	}
	if gotResp != wantResp {
		t.Error("client.Do() returned a different *http.Response than the one Doer returned")
	}

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("len(recorder.Ended()) = %d, want 1", len(ended))
	}
	status, ok := attrValue(ended[0].Attributes(), "http.status_code")
	if !ok || status.AsInt64() != http.StatusTeapot {
		t.Errorf("http.status_code attribute = %v (present=%v), want %d", status, ok, http.StatusTeapot)
	}
}

func TestTracedClient_RecordsErrorAndPropagatesItUnchangedOnFailure(t *testing.T) {
	tracer, recorder := newTracerWithRecorder(t)
	wantErr := errors.New("connection refused")
	doer := &fakeDoer{err: wantErr}
	client := &TracedClient{Tracer: tracer, Doer: doer}

	req, _ := http.NewRequest(http.MethodGet, "http://payments.internal/status", nil)
	_, err := client.Do(req)
	if !errors.Is(err, wantErr) {
		t.Fatalf("client.Do() error = %v, want (wrapping) %v", err, wantErr)
	}

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("len(recorder.Ended()) = %d, want 1", len(ended))
	}
	if ended[0].Status().Code != codes.Error {
		t.Errorf("span.Status().Code = %v, want codes.Error after Doer.Do fails", ended[0].Status().Code)
	}
	if len(ended[0].Events()) == 0 {
		t.Error("span has no recorded events; want span.RecordError to have added one")
	}
}
