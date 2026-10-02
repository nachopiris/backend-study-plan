// Command service is a small HTTP service used to exercise graceful shutdown against a real
// Kubernetes pod lifecycle (see ../../k8s and ../../README.md). GIVEN, working code — you do not
// need to modify this file; the exercise is GracefulShutdown in
// ../../internal/httpserver/shutdown.go.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"studyplan/phase5/k8sservice/internal/httpserver"
)

// gracePeriod bounds how long GracefulShutdown waits for in-flight requests to finish. It must
// stay comfortably BELOW the Pod's terminationGracePeriodSeconds (k8s/deployment.yaml) —
// Kubernetes sends SIGKILL unconditionally once that deadline passes, regardless of what this
// process is doing.
const gracePeriod = 10 * time.Second

func main() {
	addr := ":8080"
	if v := os.Getenv("ADDR"); v != "" {
		addr = v
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /work", workHandler)

	srv, err := httpserver.NewServer(addr, mux)
	if err != nil {
		log.Fatalf("failed to start listening on %s: %v", addr, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.Serve(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received, draining in-flight requests...")

	if err := srv.GracefulShutdown(context.Background(), gracePeriod); err != nil {
		log.Printf("shutdown did not complete cleanly within %s: %v", gracePeriod, err)
		os.Exit(1)
	}
	log.Println("shutdown complete")
}

// workHandler simulates work that takes a configurable amount of time, so graceful shutdown can
// be observed with real in-flight requests: GET /work?ms=2000 sleeps ~2s before responding 200,
// unless the client disconnects or the server starts shutting down first.
func workHandler(w http.ResponseWriter, r *http.Request) {
	ms := 0
	if v := r.URL.Query().Get("ms"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			ms = parsed
		}
	}

	select {
	case <-time.After(time.Duration(ms) * time.Millisecond):
	case <-r.Context().Done():
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("done\n"))
}
