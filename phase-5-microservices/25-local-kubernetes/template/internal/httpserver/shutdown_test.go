package httpserver_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"studyplan/phase5/k8sservice/internal/httpserver"
)

// These two tests exercise only GIVEN code (NewServer's wiring) and pass by design before you
// implement GracefulShutdown — they establish the baseline the shutdown tests below build on.

func TestNewServer_HealthzIsAlwaysOK(t *testing.T) {
	srv, err := httpserver.NewServer("127.0.0.1:0", nil)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	go func() { _ = srv.Serve() }()
	t.Cleanup(func() { _ = srv.GracefulShutdown(context.Background(), time.Second) })

	resp, err := http.Get("http://" + srv.Addr() + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestNewServer_ReadyzIsOKBeforeShutdown(t *testing.T) {
	srv, err := httpserver.NewServer("127.0.0.1:0", nil)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	go func() { _ = srv.Serve() }()
	t.Cleanup(func() { _ = srv.GracefulShutdown(context.Background(), time.Second) })

	if !srv.Ready() {
		t.Error("expected a freshly created server to be Ready()")
	}

	resp, err := http.Get("http://" + srv.Addr() + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

// TestGracefulShutdown_FlipsReadinessBeforeDraining is the core exercise test: it proves
// readiness flips to false BEFORE a slow in-flight request finishes, not after.
//
// Bounded wait: the in-flight handler only unblocks when the test explicitly closes `release`,
// and every wait below has an explicit timeout, so a stuck implementation fails this test in at
// most ~2s instead of hanging the suite.
func TestGracefulShutdown_FlipsReadinessBeforeDraining(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseHandler := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseHandler() // let the handler goroutine finish even if an assertion fails early

	mux := http.NewServeMux()
	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusOK)
	})

	srv, err := httpserver.NewServer("127.0.0.1:0", mux)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	go func() { _ = srv.Serve() }()

	type result struct {
		status int
		err    error
	}
	reqDone := make(chan result, 1)
	go func() {
		resp, err := http.Get("http://" + srv.Addr() + "/slow")
		if err != nil {
			reqDone <- result{err: err}
			return
		}
		defer resp.Body.Close()
		reqDone <- result{status: resp.StatusCode}
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("the /slow request never reached the handler")
	}

	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- srv.GracefulShutdown(context.Background(), 2*time.Second) }()

	// Give GracefulShutdown a moment to flip readiness. It should do so essentially
	// immediately — well before the in-flight request is released below.
	time.Sleep(50 * time.Millisecond)
	if srv.Ready() {
		t.Fatal("expected Ready() to be false while GracefulShutdown is draining an in-flight request")
	}

	releaseHandler() // let the in-flight request finish

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("GracefulShutdown returned an error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("GracefulShutdown did not return in time")
	}

	select {
	case res := <-reqDone:
		if res.err != nil {
			t.Fatalf("in-flight request failed: %v", res.err)
		}
		if res.status != http.StatusOK {
			t.Errorf("in-flight request status = %d, want 200", res.status)
		}
	case <-time.After(time.Second):
		t.Fatal("in-flight request never completed")
	}
}

func TestGracefulShutdown_RejectsNewConnectionsOnceStarted(t *testing.T) {
	srv, err := httpserver.NewServer("127.0.0.1:0", nil)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	go func() { _ = srv.Serve() }()
	addr := srv.Addr()

	if err := srv.GracefulShutdown(context.Background(), time.Second); err != nil {
		t.Fatalf("GracefulShutdown: %v", err)
	}

	client := http.Client{Timeout: time.Second}
	if _, err := client.Get("http://" + addr + "/healthz"); err == nil {
		t.Error("expected a request after shutdown to fail (connection refused), got no error")
	}
}

// TestGracefulShutdown_RespectsDeadline proves GracefulShutdown never blocks past gracePeriod,
// even when a request refuses to finish — the grace period must be an upper bound, not a
// suggestion.
func TestGracefulShutdown_RespectsDeadline(t *testing.T) {
	started := make(chan struct{})
	block := make(chan struct{}) // deliberately never closed by this test

	mux := http.NewServeMux()
	mux.HandleFunc("GET /stuck", func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-block
	})

	srv, err := httpserver.NewServer("127.0.0.1:0", mux)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	go func() { _ = srv.Serve() }()

	go func() {
		resp, err := http.Get("http://" + srv.Addr() + "/stuck")
		if err == nil {
			resp.Body.Close()
		}
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("the /stuck request never reached the handler")
	}

	const gracePeriod = 100 * time.Millisecond
	start := time.Now()
	shutdownErr := srv.GracefulShutdown(context.Background(), gracePeriod)
	elapsed := time.Since(start)

	if shutdownErr == nil {
		t.Error("expected a non-nil error when the grace period elapses with work still in flight")
	}
	if elapsed > 2*time.Second {
		t.Errorf("GracefulShutdown took %v, want roughly the %v grace period, not an unbounded wait", elapsed, gracePeriod)
	}
}
