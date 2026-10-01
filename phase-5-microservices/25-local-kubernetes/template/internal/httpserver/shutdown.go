package httpserver

import (
	"context"
	"errors"
	"time"
)

// GracefulShutdown is the exercise. It implements the shutdown sequence a process running
// inside Kubernetes needs: stop LOOKING ready immediately, then stop ACCEPTING new work, then
// wait — but not forever — for the work already in flight to finish.
//
// Contract:
//   - Marks the server NOT ready (so Ready() and GET /readyz immediately start reporting
//     unready) BEFORE doing anything else. This ordering matters: a load balancer or kubelet
//     that is still polling /readyz during the drain needs to see the failing probe as early as
//     possible, ideally well before the listener actually stops accepting connections (that gap
//     is what a Kubernetes preStop hook is for — see the README and k8s/deployment.yaml).
//   - Then stops the underlying server from accepting NEW connections and waits for in-flight
//     connections to go idle, bounded by gracePeriod — net/http's own Server.Shutdown already
//     implements exactly this waiting behavior; this function's job is to call it correctly and
//     bound it.
//   - If every in-flight request finishes before gracePeriod elapses, returns nil.
//   - If gracePeriod elapses first, forcibly closes any still-open connections (so this function
//     never blocks past gracePeriod) and returns a non-nil error — the caller (main.go) needs to
//     know the shutdown was not entirely graceful, for example to log it or set a non-zero exit
//     code.
//   - Calling GracefulShutdown more than once must not panic (Kubernetes can, in principle,
//     deliver more than one termination signal).
func (s *Server) GracefulShutdown(ctx context.Context, gracePeriod time.Duration) error {
	return errors.New("not implemented")
}
