// Package httpserver wraps an http.Server with liveness/readiness endpoints and graceful
// shutdown, the minimum any process needs to behave correctly inside Kubernetes: a liveness
// probe (/healthz) that tells kubelet "should you restart me?", a readiness probe (/readyz) that
// tells it "should you send me traffic right now?", and a shutdown path that answers "no" to the
// second question before it stops answering the first altogether.
package httpserver

import (
	"errors"
	"net"
	"net/http"
	"sync/atomic"
)

// Server is GIVEN, working infrastructure: wiring, routing, and the liveness/readiness
// endpoints. You do not need to modify this file — the exercise is GracefulShutdown, in
// shutdown.go.
type Server struct {
	httpServer *http.Server
	listener   net.Listener
	ready      atomic.Bool
}

// NewServer binds addr (e.g. ":8080", or ":0" in tests to get any free port) and wires:
//   - GET /healthz: liveness. Always responds 200 while the process is alive. This must NEVER
//     reflect readiness — kubelet restarts the container when liveness fails, which is the
//     wrong response to "a downstream dependency is briefly unavailable" and the right response
//     only to "this process is well and truly stuck."
//   - GET /readyz: readiness. 200 while the server is ready to receive traffic, 503 once
//     GracefulShutdown has started — kubelet stops ROUTING traffic here when readiness fails,
//     without restarting the container.
//   - extraRoutes: your business handlers, mounted at "/". May be nil.
//
// The server is ready (Ready() == true) from the moment NewServer returns successfully.
func NewServer(addr string, extraRoutes http.Handler) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &Server{listener: listener}
	s.ready.Store(true)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if s.ready.Load() {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	if extraRoutes != nil {
		mux.Handle("/", extraRoutes)
	}

	s.httpServer = &http.Server{Handler: mux}
	return s, nil
}

// Addr returns the address the server is actually listening on — useful in tests that bind
// ":0" and need to know which port the OS assigned.
func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

// Ready reports whether the server currently considers itself ready to receive traffic. GET
// /readyz is what Kubernetes actually probes; this is exposed directly so tests (and
// GracefulShutdown itself) do not need to make an HTTP round trip to observe it.
func (s *Server) Ready() bool {
	return s.ready.Load()
}

// Serve blocks, accepting connections until the server is shut down (via GracefulShutdown) or
// an unrecoverable error occurs. It mirrors http.Server.Serve's contract: a shutdown is not
// reported as an error.
func (s *Server) Serve() error {
	err := s.httpServer.Serve(s.listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
