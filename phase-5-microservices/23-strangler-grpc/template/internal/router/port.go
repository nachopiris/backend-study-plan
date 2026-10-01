// Package router implements the strangler fig facade (PaymentsRouter) that gradually moves
// traffic from the legacy in-process payments implementation (legacy.go) to the new gRPC
// PaymentsService (template/internal/paymentsserver), by a deterministic percentage split on
// order ID — see router.go.
package router

import (
	"context"
	"errors"
)

// Receipt mirrors payments.Receipt from practice 05's payments module
// (studyplan/phase0/modularmonolith/internal/payments).
type Receipt struct {
	ID          string
	AmountCents int64
}

// Charger mirrors practice 05's payments.Charger port — the port orders.Service depends on to
// charge a customer. LegacyCharger implements this directly; PaymentsRouter implements
// RoutingCharger (below), a small superset of this shape.
//
// This package keeps its own copy of this shape instead of importing practice 05's module, so
// this practice's go.mod stays self-contained — each Go template in this plan is its own
// module (see the root README).
type Charger interface {
	Charge(ctx context.Context, customerID string, amountCents int64) (Receipt, error)
}

// RoutingCharger is the port PaymentsRouter exposes: the same shape as Charger, plus an
// explicit orderID the strangler facade needs to make a deterministic routing decision (see
// RouteToGRPC in router.go). payments.Charger itself has no orderID parameter; wiring this
// facade into the real monolith's orders.Service, whose calls only carry a customerID, would
// need a thin adapter at the call site providing it — out of scope for this practice.
type RoutingCharger interface {
	Charge(ctx context.Context, orderID, customerID string, amountCents int64) (Receipt, error)
}

// Domain errors PaymentsRouter.Charge can return. Callers of RoutingCharger never need to
// import grpc/status or codes themselves — translating that gRPC-specific vocabulary into
// stable Go errors at the boundary is the whole point of putting a facade in front of the new
// service.
var (
	// ErrPaymentDeclined mirrors payments.ErrPaymentDeclined from practice 05: the legacy path
	// declined the charge, or the new service rejected it with codes.FailedPrecondition.
	ErrPaymentDeclined = errors.New("router: payment declined")
	// ErrInvalidRequest maps from the new service's codes.InvalidArgument.
	ErrInvalidRequest = errors.New("router: invalid payment request")
	// ErrReceiptNotFound maps from the new service's codes.NotFound.
	ErrReceiptNotFound = errors.New("router: receipt not found")
	// ErrTimeout maps from the new service's codes.DeadlineExceeded, including the router's own
	// per-call deadline being exceeded before the service responded at all.
	ErrTimeout = errors.New("router: payments service call timed out")
	// ErrPaymentsUnavailable is the catch-all for any other failure talking to the new service
	// (a transport error, or a status code with no more specific domain mapping).
	ErrPaymentsUnavailable = errors.New("router: payments service unavailable")
)
