// Package paymentsserver implements the new, standalone gRPC payments service this strangler
// migration extracts from the legacy in-process implementation
// (template/internal/router/legacy.go). It is in-memory and single-process — enough to exercise
// idempotency, validation, and status-code mapping end to end over a real gRPC connection
// (via bufconn in the tests), which is what this practice is about. A production version would
// back this with a database, but persistence is not the pedagogical point here.
package paymentsserver

import (
	"context"
	"sync"

	paymentsv1 "studyplan/phase5/strangler/gen/payments/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements paymentsv1.PaymentsServiceServer.
type Server struct {
	paymentsv1.UnimplementedPaymentsServiceServer

	mu sync.Mutex
	// chargesByKey remembers every Charge call by idempotency_key, so a repeated call with the
	// same key can be detected and answered without creating a new receipt.
	chargesByKey map[string]*chargeRecord
	// receiptsByID lets Refund look up what was charged for a given receipt_id.
	receiptsByID map[string]*chargeRecord
	// refundsByKey remembers every Refund call by idempotency_key, the same way chargesByKey
	// does for Charge.
	refundsByKey map[string]*paymentsv1.RefundResponse
	// refundedReceipts tracks which receipt_ids have already been refunded (under ANY
	// idempotency_key), so a second refund attempt under a NEW key can be rejected instead of
	// refunding twice.
	refundedReceipts map[string]string // receipt_id -> the idempotency_key that refunded it
	nextReceiptID    int
	nextRefundID     int
}

// chargeRecord is what Server remembers about one successfully processed Charge call.
type chargeRecord struct {
	customerID  string
	amountCents int64
	response    *paymentsv1.ChargeResponse
}

// NewServer builds an empty Server, ready to accept Charge and Refund calls.
func NewServer() *Server {
	return &Server{
		chargesByKey:     make(map[string]*chargeRecord),
		receiptsByID:     make(map[string]*chargeRecord),
		refundsByKey:     make(map[string]*paymentsv1.RefundResponse),
		refundedReceipts: make(map[string]string),
	}
}

// Charge implements paymentsv1.PaymentsServiceServer.
//
// Contract:
//   - req.CustomerId == "", req.AmountCents <= 0, or req.IdempotencyKey == "" ->
//     status.Error(codes.InvalidArgument, ...).
//   - The FIRST call seen for a given idempotency_key creates a new receipt (a fresh,
//     never-before-used receipt_id) and remembers it, keyed by BOTH idempotency_key (for
//     Charge's own idempotency) and receipt_id (so Refund can find it later).
//   - A REPEATED call with the SAME idempotency_key, SAME customer_id, and SAME amount_cents
//     returns the EXACT SAME ChargeResponse as the first call (same receipt_id) — this is the
//     guarantee the strangler router (template/internal/router) relies on to retry safely
//     without risking a double charge.
//   - A call that reuses an idempotency_key already seen, but with a DIFFERENT customer_id or a
//     DIFFERENT amount_cents, is a conflicting reuse of that key, not a safe retry ->
//     status.Error(codes.FailedPrecondition, ...).
func (s *Server) Charge(ctx context.Context, req *paymentsv1.ChargeRequest) (*paymentsv1.ChargeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "paymentsserver: Charge not implemented")
}

// Refund implements paymentsv1.PaymentsServiceServer.
//
// Contract:
//   - req.ReceiptId == "" or req.IdempotencyKey == "" -> status.Error(codes.InvalidArgument, ...).
//   - req.ReceiptId that was never produced by a successful Charge ->
//     status.Error(codes.NotFound, ...).
//   - The FIRST refund call seen for a given idempotency_key succeeds (a fresh refund_id,
//     amount_cents equal to the original charge) and is remembered, keyed by idempotency_key.
//   - A REPEATED call with the SAME idempotency_key returns the EXACT SAME RefundResponse as
//     the first call, without refunding twice.
//   - A call that targets a receipt_id that was ALREADY refunded under a DIFFERENT
//     idempotency_key -> status.Error(codes.FailedPrecondition, ...): a receipt can only be
//     refunded once, so a new key for an already-refunded receipt is a conflicting reuse, not a
//     safe retry.
func (s *Server) Refund(ctx context.Context, req *paymentsv1.RefundRequest) (*paymentsv1.RefundResponse, error) {
	return nil, status.Error(codes.Unimplemented, "paymentsserver: Refund not implemented")
}
