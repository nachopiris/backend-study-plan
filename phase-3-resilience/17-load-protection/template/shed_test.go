package loadshed

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestShed_AllowsRequestsUnderTheLimit(t *testing.T) {
	var calls int32
	handler := Shed(2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if calls != 1 {
		t.Fatalf("handler called %d times, want 1", calls)
	}
}

// TestShed_RejectsBeyondMaxInFlight saturates the handler with blocking
// requests and checks that exactly maxInFlight run concurrently, with the
// rest shed as 503 with Retry-After, never reaching the handler. Run with
// -race.
func TestShed_RejectsBeyondMaxInFlight(t *testing.T) {
	const maxInFlight = 3
	const attempts = 8

	release := make(chan struct{})
	var running int32
	var maxObservedRunning int32
	handler := Shed(maxInFlight)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&running, 1)
		for {
			observed := atomic.LoadInt32(&maxObservedRunning)
			if n <= observed || atomic.CompareAndSwapInt32(&maxObservedRunning, observed, n) {
				break
			}
		}
		<-release
		atomic.AddInt32(&running, -1)
		w.WriteHeader(http.StatusOK)
	}))

	statuses := make([]int, attempts)
	retryAfter := make([]string, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			statuses[i] = rec.Code
			retryAfter[i] = rec.Header().Get(RetryAfterHeader)
		}(i)
	}

	// Give every goroutine time to reach the middleware before releasing
	// the ones that got in. Generous relative to scheduling 8 goroutines.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if got := atomic.LoadInt32(&maxObservedRunning); got > maxInFlight {
		t.Fatalf("observed %d requests running concurrently, want at most %d", got, maxInFlight)
	}

	var ok, shed int
	for i, s := range statuses {
		switch s {
		case http.StatusOK:
			ok++
		case http.StatusServiceUnavailable:
			shed++
			if retryAfter[i] == "" {
				t.Errorf("request %d: shed response missing %s header", i, RetryAfterHeader)
			}
		default:
			t.Errorf("request %d: unexpected status %d", i, s)
		}
	}
	if ok != maxInFlight {
		t.Errorf("got %d 200 responses, want exactly %d", ok, maxInFlight)
	}
	if shed != attempts-maxInFlight {
		t.Errorf("got %d 503 responses, want %d", shed, attempts-maxInFlight)
	}
}
