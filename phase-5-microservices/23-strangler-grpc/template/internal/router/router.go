package router

import (
	"context"
	"errors"
	"time"

	paymentsv1 "studyplan/phase5/strangler/gen/payments/v1"
)

// PaymentsRouter is the strangler fig facade: it implements RoutingCharger by splitting traffic
// between the legacy in-process LegacyCharger and the new gRPC PaymentsService, so the rest of
// the monolith keeps calling a stable Go interface while the migration happens gradually and
// reversibly (set percentage back to 0 and every call goes back to the legacy path, with no
// code change anywhere else).
type PaymentsRouter struct {
	grpcClient paymentsv1.PaymentsServiceClient
	legacy     Charger
	percentage int
	deadline   time.Duration
}

// NewPaymentsRouter builds a PaymentsRouter. percentage (0-100) is the share of orders routed to
// the new gRPC service; deadline bounds each individual gRPC call made through this router.
func NewPaymentsRouter(grpcClient paymentsv1.PaymentsServiceClient, legacy Charger, percentage int, deadline time.Duration) *PaymentsRouter {
	return &PaymentsRouter{grpcClient: grpcClient, legacy: legacy, percentage: percentage, deadline: deadline}
}

// RouteToGRPC deterministically decides whether orderID's charge should go to the new gRPC
// service, for a given percentage (0-100).
//
// Contract:
//   - Deterministic: the same (orderID, percentage) pair always returns the same bool, in this
//     process or any other — do not use anything randomized or time-based (hash orderID
//     instead, e.g. with hash/fnv or hash/crc32 from the standard library, and compare the
//     result against percentage).
//   - percentage <= 0 always returns false; percentage >= 100 always returns true.
//   - Across a large number of distinct order IDs, approximately percentage% return true. The
//     tests check this within a tolerance (see router_test.go), not an exact count.
func RouteToGRPC(orderID string, percentage int) bool {
	return false
}

// MapStatusToDomainError translates a gRPC error returned by the payments service into one of
// this package's domain errors (port.go), so RoutingCharger callers never need to import
// grpc/status or codes themselves. A nil err returns nil.
//
// Contract (use google.golang.org/grpc/status.FromError to inspect err):
//   - codes.InvalidArgument    -> ErrInvalidRequest
//   - codes.FailedPrecondition -> ErrPaymentDeclined
//   - codes.NotFound           -> ErrReceiptNotFound
//   - codes.DeadlineExceeded   -> ErrTimeout
//   - any other non-nil error (including one that is not a gRPC status at all, e.g. a
//     transport-level failure) -> wraps ErrPaymentsUnavailable, so that
//     errors.Is(result, ErrPaymentsUnavailable) is true and err's own message is still visible
//     in result's message (e.g. via fmt.Errorf("%w: %v", ErrPaymentsUnavailable, err)).
func MapStatusToDomainError(err error) error {
	return errors.New("not implemented")
}

// Charge implements RoutingCharger.
//
// Contract:
//   - Uses RouteToGRPC(orderID, percentage) to decide the path for this specific call.
//   - Legacy path: calls legacy.Charge directly with ctx unmodified, and returns its result
//     unchanged (LegacyCharger already returns this package's own ErrPaymentDeclined).
//   - gRPC path: derives a per-call deadline from ctx using the router's configured deadline
//     (context.WithTimeout composes with any deadline ctx already carries, keeping whichever is
//     tighter — exactly the behavior wanted here), calls the gRPC service's Charge, and on
//     error returns MapStatusToDomainError(err). On success, translates the ChargeResponse into
//     a Receipt.
//   - The idempotency_key sent to the gRPC service is derived deterministically from orderID
//     alone (never randomized), so that retrying the SAME order's charge — for example after a
//     timeout — is itself safe to retry at the gRPC service without risking a double charge.
func (r *PaymentsRouter) Charge(ctx context.Context, orderID, customerID string, amountCents int64) (Receipt, error) {
	return Receipt{}, errors.New("not implemented")
}
