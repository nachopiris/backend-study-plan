package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// hangingServer returns an httptest.Server whose handler accepts the request
// but never writes a response. It unblocks only when the request's context is
// done (i.e. the client gave up and closed the connection), so tests never
// hang during cleanup regardless of how Fetch is implemented.
func hangingServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
}

func TestFetch_TimesOutWhenServerNeverResponds(t *testing.T) {
	srv := hangingServer()
	defer srv.Close()

	const timeout = 100 * time.Millisecond
	start := time.Now()
	_, err := Fetch(context.Background(), srv.URL, timeout)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected Fetch to return an error when the server never responds, got nil")
	}

	// Generous bound: well above the configured timeout, but far below "hangs
	// indefinitely". If your implementation ignores the timeout, this fails.
	const bound = 2 * time.Second
	if elapsed > bound {
		t.Fatalf("Fetch took %s to return after a %s timeout; expected it to fail within %s", elapsed, timeout, bound)
	}
}

func TestFetch_RespectsAlreadyCanceledParentContext(t *testing.T) {
	srv := hangingServer()
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := Fetch(ctx, srv.URL, time.Second)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected Fetch to fail immediately when the parent context is already canceled")
	}
	const bound = 500 * time.Millisecond
	if elapsed > bound {
		t.Fatalf("Fetch took %s to notice an already-canceled context; expected under %s", elapsed, bound)
	}
}

func TestFetch_ReturnsBodyOnSuccess(t *testing.T) {
	want := "hello from server"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(want))
	}))
	defer srv.Close()

	got, err := Fetch(context.Background(), srv.URL, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
