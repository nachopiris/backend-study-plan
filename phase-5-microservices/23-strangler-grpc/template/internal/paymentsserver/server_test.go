package paymentsserver_test

import (
	"context"
	"net"
	"testing"

	paymentsv1 "studyplan/phase5/strangler/gen/payments/v1"
	"studyplan/phase5/strangler/internal/paymentsserver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// dialServer starts the given Server behind an in-memory bufconn listener and returns a
// PaymentsServiceClient connected to it, plus a cleanup function. GIVEN test infrastructure —
// bufconn.Listen gives a real net.Listener backed by memory instead of a TCP port, so these
// tests exercise the actual gRPC wire protocol (marshaling, status codes) without binding a
// socket or needing a free port.
func dialServer(t *testing.T, srv paymentsv1.PaymentsServiceServer) paymentsv1.PaymentsServiceClient {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	paymentsv1.RegisterPaymentsServiceServer(grpcServer, srv)
	go func() {
		_ = grpcServer.Serve(lis)
	}()
	t.Cleanup(grpcServer.Stop)

	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return paymentsv1.NewPaymentsServiceClient(conn)
}

func statusCode(t *testing.T, err error) codes.Code {
	t.Helper()
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected a gRPC status error, got %v (%T)", err, err)
	}
	return st.Code()
}

func TestCharge_HappyPath(t *testing.T) {
	client := dialServer(t, paymentsserver.NewServer())

	resp, err := client.Charge(context.Background(), &paymentsv1.ChargeRequest{
		CustomerId:     "cust-1",
		AmountCents:    1500,
		IdempotencyKey: "key-1",
	})
	if err != nil {
		t.Fatalf("Charge: unexpected error: %v", err)
	}
	if resp.ReceiptId == "" {
		t.Error("expected a non-empty receipt_id")
	}
	if resp.AmountCents != 1500 {
		t.Errorf("AmountCents = %d, want 1500", resp.AmountCents)
	}
}

func TestCharge_InvalidArgument(t *testing.T) {
	client := dialServer(t, paymentsserver.NewServer())

	cases := []struct {
		name string
		req  *paymentsv1.ChargeRequest
	}{
		{"missing customer_id", &paymentsv1.ChargeRequest{AmountCents: 100, IdempotencyKey: "k"}},
		{"zero amount_cents", &paymentsv1.ChargeRequest{CustomerId: "c", AmountCents: 0, IdempotencyKey: "k"}},
		{"negative amount_cents", &paymentsv1.ChargeRequest{CustomerId: "c", AmountCents: -1, IdempotencyKey: "k"}},
		{"missing idempotency_key", &paymentsv1.ChargeRequest{CustomerId: "c", AmountCents: 100}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.Charge(context.Background(), tc.req)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if code := statusCode(t, err); code != codes.InvalidArgument {
				t.Errorf("status code = %v, want InvalidArgument", code)
			}
		})
	}
}

func TestCharge_IdempotentRetrySameKey(t *testing.T) {
	client := dialServer(t, paymentsserver.NewServer())
	req := &paymentsv1.ChargeRequest{CustomerId: "cust-1", AmountCents: 2500, IdempotencyKey: "retry-key"}

	first, err := client.Charge(context.Background(), req)
	if err != nil {
		t.Fatalf("first Charge: unexpected error: %v", err)
	}

	second, err := client.Charge(context.Background(), req)
	if err != nil {
		t.Fatalf("second Charge: unexpected error: %v", err)
	}

	if second.ReceiptId != first.ReceiptId {
		t.Errorf("retried charge produced a different receipt: first=%q second=%q", first.ReceiptId, second.ReceiptId)
	}
	if second.AmountCents != first.AmountCents {
		t.Errorf("retried charge produced a different amount: first=%d second=%d", first.AmountCents, second.AmountCents)
	}
}

func TestCharge_ConflictingIdempotencyKeyReuse(t *testing.T) {
	client := dialServer(t, paymentsserver.NewServer())

	_, err := client.Charge(context.Background(), &paymentsv1.ChargeRequest{
		CustomerId: "cust-1", AmountCents: 1000, IdempotencyKey: "shared-key",
	})
	if err != nil {
		t.Fatalf("first Charge: unexpected error: %v", err)
	}

	_, err = client.Charge(context.Background(), &paymentsv1.ChargeRequest{
		CustomerId: "cust-2", AmountCents: 1000, IdempotencyKey: "shared-key",
	})
	if err == nil {
		t.Fatal("expected an error reusing the same idempotency_key with a different customer_id")
	}
	if code := statusCode(t, err); code != codes.FailedPrecondition {
		t.Errorf("status code = %v, want FailedPrecondition", code)
	}
}

func TestRefund_HappyPath(t *testing.T) {
	client := dialServer(t, paymentsserver.NewServer())

	charge, err := client.Charge(context.Background(), &paymentsv1.ChargeRequest{
		CustomerId: "cust-1", AmountCents: 800, IdempotencyKey: "charge-key",
	})
	if err != nil {
		t.Fatalf("Charge: unexpected error: %v", err)
	}

	refund, err := client.Refund(context.Background(), &paymentsv1.RefundRequest{
		ReceiptId: charge.ReceiptId, IdempotencyKey: "refund-key",
	})
	if err != nil {
		t.Fatalf("Refund: unexpected error: %v", err)
	}
	if refund.RefundId == "" {
		t.Error("expected a non-empty refund_id")
	}
	if refund.AmountCents != 800 {
		t.Errorf("AmountCents = %d, want 800", refund.AmountCents)
	}
}

func TestRefund_UnknownReceiptNotFound(t *testing.T) {
	client := dialServer(t, paymentsserver.NewServer())

	_, err := client.Refund(context.Background(), &paymentsv1.RefundRequest{
		ReceiptId: "does-not-exist", IdempotencyKey: "refund-key",
	})
	if err == nil {
		t.Fatal("expected an error for an unknown receipt_id")
	}
	if code := statusCode(t, err); code != codes.NotFound {
		t.Errorf("status code = %v, want NotFound", code)
	}
}

func TestRefund_InvalidArgument(t *testing.T) {
	client := dialServer(t, paymentsserver.NewServer())

	cases := []struct {
		name string
		req  *paymentsv1.RefundRequest
	}{
		{"missing receipt_id", &paymentsv1.RefundRequest{IdempotencyKey: "k"}},
		{"missing idempotency_key", &paymentsv1.RefundRequest{ReceiptId: "r"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.Refund(context.Background(), tc.req)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if code := statusCode(t, err); code != codes.InvalidArgument {
				t.Errorf("status code = %v, want InvalidArgument", code)
			}
		})
	}
}

func TestRefund_IdempotentRetrySameKey(t *testing.T) {
	client := dialServer(t, paymentsserver.NewServer())

	charge, err := client.Charge(context.Background(), &paymentsv1.ChargeRequest{
		CustomerId: "cust-1", AmountCents: 400, IdempotencyKey: "charge-key",
	})
	if err != nil {
		t.Fatalf("Charge: unexpected error: %v", err)
	}

	refundReq := &paymentsv1.RefundRequest{ReceiptId: charge.ReceiptId, IdempotencyKey: "refund-key"}
	first, err := client.Refund(context.Background(), refundReq)
	if err != nil {
		t.Fatalf("first Refund: unexpected error: %v", err)
	}
	second, err := client.Refund(context.Background(), refundReq)
	if err != nil {
		t.Fatalf("second Refund: unexpected error: %v", err)
	}

	if second.RefundId != first.RefundId {
		t.Errorf("retried refund produced a different refund_id: first=%q second=%q", first.RefundId, second.RefundId)
	}
}

func TestRefund_ConflictingKeyOnAlreadyRefundedReceipt(t *testing.T) {
	client := dialServer(t, paymentsserver.NewServer())

	charge, err := client.Charge(context.Background(), &paymentsv1.ChargeRequest{
		CustomerId: "cust-1", AmountCents: 600, IdempotencyKey: "charge-key",
	})
	if err != nil {
		t.Fatalf("Charge: unexpected error: %v", err)
	}

	_, err = client.Refund(context.Background(), &paymentsv1.RefundRequest{
		ReceiptId: charge.ReceiptId, IdempotencyKey: "refund-key-1",
	})
	if err != nil {
		t.Fatalf("first Refund: unexpected error: %v", err)
	}

	_, err = client.Refund(context.Background(), &paymentsv1.RefundRequest{
		ReceiptId: charge.ReceiptId, IdempotencyKey: "refund-key-2",
	})
	if err == nil {
		t.Fatal("expected an error refunding the same receipt twice under a different idempotency_key")
	}
	if code := statusCode(t, err); code != codes.FailedPrecondition {
		t.Errorf("status code = %v, want FailedPrecondition", code)
	}
}
