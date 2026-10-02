package retry

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func countingHandler(calls *int32, status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(calls, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
}

func postWithKey(t *testing.T, wrapped http.Handler, key string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader(body))
	if key != "" {
		req.Header.Set(IdempotencyKeyHeader, key)
	}
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
	return rec
}

func TestIdempotencyMiddleware_NoKeyAlwaysPassesThrough(t *testing.T) {
	var calls int32
	wrapped := NewIdempotencyMiddleware().Wrap(countingHandler(&calls, http.StatusCreated, `{"id":"pay_1"}`))

	for i := 0; i < 3; i++ {
		rec := postWithKey(t, wrapped, "", []byte(`{"amount_cents":100}`))
		if rec.Code != http.StatusCreated {
			t.Fatalf("request %d: status = %d, want %d", i, rec.Code, http.StatusCreated)
		}
	}
	if calls != 3 {
		t.Fatalf("handler called %d times, want 3 (no key means no dedup)", calls)
	}
}

func TestIdempotencyMiddleware_SameKeySameBodyReplaysWithoutReexecuting(t *testing.T) {
	var calls int32
	wrapped := NewIdempotencyMiddleware().Wrap(countingHandler(&calls, http.StatusCreated, `{"id":"pay_1"}`))
	body := []byte(`{"amount_cents":100}`)

	first := postWithKey(t, wrapped, "key-replay", body)
	second := postWithKey(t, wrapped, "key-replay", body)

	if first.Code != http.StatusCreated {
		t.Fatalf("first request status = %d, want %d", first.Code, http.StatusCreated)
	}
	if second.Code != first.Code || second.Body.String() != first.Body.String() {
		t.Fatalf("replayed response = (%d, %q), want identical to first (%d, %q)",
			second.Code, second.Body.String(), first.Code, first.Body.String())
	}
	if calls != 1 {
		t.Fatalf("handler called %d times, want 1 (second request must replay, not re-execute)", calls)
	}
}

func TestIdempotencyMiddleware_SameKeyDifferentBodyIsRejected(t *testing.T) {
	var calls int32
	wrapped := NewIdempotencyMiddleware().Wrap(countingHandler(&calls, http.StatusCreated, `{"id":"pay_1"}`))

	first := postWithKey(t, wrapped, "key-conflict", []byte(`{"amount_cents":100}`))
	second := postWithKey(t, wrapped, "key-conflict", []byte(`{"amount_cents":999}`))

	if first.Code != http.StatusCreated {
		t.Fatalf("first request status = %d, want %d", first.Code, http.StatusCreated)
	}
	if second.Code != http.StatusUnprocessableEntity {
		t.Fatalf("second request (same key, different body) status = %d, want %d", second.Code, http.StatusUnprocessableEntity)
	}
	if calls != 1 {
		t.Fatalf("handler called %d times, want 1 (the differing request must not execute)", calls)
	}
}

// TestIdempotencyMiddleware_ConcurrentSameKeyExecutesExactlyOnce fires
// several concurrent requests with the same key and body while the handler
// is deliberately held open, and checks that exactly one of them actually
// runs the handler; the rest must get 409 Conflict rather than blocking
// forever or re-executing.
func TestIdempotencyMiddleware_ConcurrentSameKeyExecutesExactlyOnce(t *testing.T) {
	var calls int32
	release := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		<-release
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"pay_concurrent"}`))
	})
	wrapped := NewIdempotencyMiddleware().Wrap(handler)
	body := []byte(`{"amount_cents":500}`)

	const n = 5
	statuses := make([]int, n)
	done := make(chan struct{})
	for i := 0; i < n; i++ {
		go func(i int) {
			statuses[i] = postWithKey(t, wrapped, "key-concurrent", body).Code
			done <- struct{}{}
		}(i)
	}

	// Give every goroutine time to reach the middleware and register as
	// in-flight before we let the one real execution finish. Generous
	// relative to how long it takes to schedule 5 goroutines.
	time.Sleep(50 * time.Millisecond)
	close(release)
	for i := 0; i < n; i++ {
		<-done
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("handler called %d times, want exactly 1", got)
	}
	var created, conflict, other int
	for _, s := range statuses {
		switch s {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflict++
		default:
			other++
		}
	}
	if created != 1 {
		t.Errorf("got %d responses with status 201, want exactly 1", created)
	}
	if conflict != n-1 {
		t.Errorf("got %d responses with status 409, want %d; other=%d", conflict, n-1, other)
	}
}
