package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// newTestTelemetry builds a tracer (recording every span into the returned SpanRecorder, in
// memory) and a meter (collectable on demand via the returned ManualReader, in memory) — the
// in-memory test doubles this package's own tests use instead of a real OTel Collector, so no
// network and no Docker are involved in running them.
func newTestTelemetry(t *testing.T) (trace.Tracer, metric.Meter, *tracetest.SpanRecorder, *sdkmetric.ManualReader) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(recorder),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })

	return tp.Tracer("test"), mp.Meter("test"), recorder, reader
}

func newREDMetricsOrFail(t *testing.T, meter metric.Meter) *REDMetrics {
	t.Helper()
	red, err := NewREDMetrics(meter)
	if err != nil {
		t.Fatalf("NewREDMetrics() error = %v, want nil", err)
	}
	return red
}

// attrValue looks up key in a ReadOnlySpan's flat attribute slice, since span.Attributes()
// returns []attribute.KeyValue, not an indexable set.
func attrValue(attrs []attribute.KeyValue, key string) (attribute.Value, bool) {
	for _, kv := range attrs {
		if string(kv.Key) == key {
			return kv.Value, true
		}
	}
	return attribute.Value{}, false
}

func TestServerMiddleware_StartsAServerSpanNamedAfterTheRoute(t *testing.T) {
	tracer, meter, recorder, _ := newTestTelemetry(t)
	red := newREDMetricsOrFail(t, meter)

	handler := ServerMiddleware(tracer, nil, red)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/orders/123", nil)
	req.Pattern = "GET /orders/{id}"
	handler.ServeHTTP(httptest.NewRecorder(), req)

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("len(recorder.Ended()) = %d, want 1", len(ended))
	}
	span := ended[0]
	if span.SpanKind() != trace.SpanKindServer {
		t.Errorf("span.SpanKind() = %v, want %v", span.SpanKind(), trace.SpanKindServer)
	}
	if span.Name() != "GET /orders/{id}" {
		t.Errorf("span.Name() = %q, want %q", span.Name(), "GET /orders/{id}")
	}
}

func TestServerMiddleware_FallsBackToURLPathWhenPatternIsEmpty(t *testing.T) {
	tracer, meter, recorder, _ := newTestTelemetry(t)
	red := newREDMetricsOrFail(t, meter)

	handler := ServerMiddleware(tracer, nil, red)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// req.Pattern deliberately left empty, as if the request never went through a ServeMux.
	req := httptest.NewRequest(http.MethodGet, "/raw-path", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("len(recorder.Ended()) = %d, want 1", len(ended))
	}
	if ended[0].Name() != "/raw-path" {
		t.Errorf("span.Name() = %q, want %q (fallback to r.URL.Path)", ended[0].Name(), "/raw-path")
	}
}

func TestServerMiddleware_SetsRouteAndStatusAttributesAndEndsExactlyOneSpan(t *testing.T) {
	tracer, meter, recorder, _ := newTestTelemetry(t)
	red := newREDMetricsOrFail(t, meter)

	handler := ServerMiddleware(tracer, nil, red)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/orders", nil)
	req.Pattern = "POST /orders"
	handler.ServeHTTP(httptest.NewRecorder(), req)

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("len(recorder.Ended()) = %d, want 1", len(ended))
	}
	span := ended[0]

	route, ok := attrValue(span.Attributes(), "route")
	if !ok || route.AsString() != "POST /orders" {
		t.Errorf("route attribute = %v (present=%v), want %q", route, ok, "POST /orders")
	}
	status, ok := attrValue(span.Attributes(), "status")
	if !ok || status.AsInt64() != http.StatusCreated {
		t.Errorf("status attribute = %v (present=%v), want %d", status, ok, http.StatusCreated)
	}
	if span.Status().Code == codes.Error {
		t.Error("span.Status().Code = Error, want Unset/Ok for a 201 response")
	}
}

func TestServerMiddleware_MarksSpanAsErrorOn5xxWithoutAlteringTheResponse(t *testing.T) {
	tracer, meter, recorder, _ := newTestTelemetry(t)
	red := newREDMetricsOrFail(t, meter)

	handler := ServerMiddleware(tracer, nil, red)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	req.Pattern = "GET /boom"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("recorded response status = %d, want %d (middleware must not alter the response)", rec.Code, http.StatusInternalServerError)
	}

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("len(recorder.Ended()) = %d, want 1", len(ended))
	}
	if ended[0].Status().Code != codes.Error {
		t.Errorf("span.Status().Code = %v, want codes.Error for a 500 response", ended[0].Status().Code)
	}
}

func TestServerMiddleware_RecordsRequestCounterAndDurationHistogram(t *testing.T) {
	tracer, meter, _, reader := newTestTelemetry(t)
	red := newREDMetricsOrFail(t, meter)
	handler := ServerMiddleware(tracer, nil, red)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fails" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	ok1 := httptest.NewRequest(http.MethodGet, "/ok", nil)
	ok1.Pattern = "GET /ok"
	handler.ServeHTTP(httptest.NewRecorder(), ok1)

	ok2 := httptest.NewRequest(http.MethodGet, "/ok", nil)
	ok2.Pattern = "GET /ok"
	handler.ServeHTTP(httptest.NewRecorder(), ok2)

	fail := httptest.NewRequest(http.MethodGet, "/fails", nil)
	fail.Pattern = "GET /fails"
	handler.ServeHTTP(httptest.NewRecorder(), fail)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("reader.Collect() error = %v, want nil", err)
	}

	// This test deliberately does not hardcode your instrument names (NewREDMetrics's doc
	// comment only specifies the three instruments' existence and behavior, not their exact
	// names): it sums every int64 Sum instrument's data points as a stand-in for "the requests
	// counter plus whatever else you created", and separately checks that at least one
	// Histogram[float64] data point exists, as a stand-in for the duration histogram.
	var totalCounterPoints int64
	var sawDuration bool
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch data := m.Data.(type) {
			case metricdata.Sum[int64]:
				for _, dp := range data.DataPoints {
					totalCounterPoints += dp.Value
				}
			case metricdata.Histogram[float64]:
				if len(data.DataPoints) > 0 {
					sawDuration = true
				}
			}
		}
	}
	if totalCounterPoints < 3 {
		t.Errorf("sum of all int64 counter data points = %d, want >= 3 (3 requests handled, each incrementing the requests counter at least)", totalCounterPoints)
	}
	if !sawDuration {
		t.Error("no Histogram[float64] data points recorded; duration should have at least one data point per route/status combination observed")
	}
}
