package router_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	paymentsv1 "studyplan/phase5/strangler/gen/payments/v1"
	"studyplan/phase5/strangler/internal/router"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// fakePaymentsServer is GIVEN test infrastructure: a PaymentsServiceServer whose Charge
// behavior is fully controlled by the test (canned responses, canned errors, artificial
// delays), so these tests can exercise PaymentsRouter's own logic (routing, deadlines, status
// mapping) in isolation from the real paymentsserver.Server implementation under test in
// ../paymentsserver.
type fakePaymentsServer struct {
	paymentsv1.UnimplementedPaymentsServiceServer

	mu       sync.Mutex
	calls    int
	lastReq  *paymentsv1.ChargeRequest
	chargeFn func(ctx context.Context, req *paymentsv1.ChargeRequest) (*paymentsv1.ChargeResponse, error)
}

func (f *fakePaymentsServer) Charge(ctx context.Context, req *paymentsv1.ChargeRequest) (*paymentsv1.ChargeResponse, error) {
	f.mu.Lock()
	f.calls++
	f.lastReq = req
	f.mu.Unlock()
	return f.chargeFn(ctx, req)
}

func (f *fakePaymentsServer) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakePaymentsServer) lastRequest() *paymentsv1.ChargeRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastReq
}

// dialFake starts srv behind an in-memory bufconn listener and returns a connected client plus
// cleanup. Same pattern as paymentsserver's own tests.
func dialFake(t *testing.T, srv paymentsv1.PaymentsServiceServer) paymentsv1.PaymentsServiceClient {
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

// --- RouteToGRPC ---

func TestRouteToGRPC_Deterministic(t *testing.T) {
	orderIDs := []string{"order-1", "order-42", "order-abc-999", "order-🎉"}
	for _, id := range orderIDs {
		for _, pct := range []int{1, 10, 50, 90, 99} {
			first := router.RouteToGRPC(id, pct)
			for i := 0; i < 5; i++ {
				if got := router.RouteToGRPC(id, pct); got != first {
					t.Errorf("RouteToGRPC(%q, %d) is not deterministic: got %v, then %v", id, pct, first, got)
				}
			}
		}
	}
}

func TestRouteToGRPC_Boundaries(t *testing.T) {
	orderIDs := []string{"order-1", "order-2", "order-3", "order-4", "order-5"}
	for _, id := range orderIDs {
		if router.RouteToGRPC(id, 0) {
			t.Errorf("RouteToGRPC(%q, 0) = true, want false", id)
		}
		if router.RouteToGRPC(id, -10) {
			t.Errorf("RouteToGRPC(%q, -10) = true, want false", id)
		}
		if !router.RouteToGRPC(id, 100) {
			t.Errorf("RouteToGRPC(%q, 100) = false, want true", id)
		}
		if !router.RouteToGRPC(id, 150) {
			t.Errorf("RouteToGRPC(%q, 150) = false, want true", id)
		}
	}
}

func TestRouteToGRPC_ApproximatesPercentage(t *testing.T) {
	const percentage = 30
	const sampleSize = 20000
	const toleranceFraction = 0.05 // +/- 5 percentage points

	trueCount := 0
	for i := 0; i < sampleSize; i++ {
		if router.RouteToGRPC(fmt.Sprintf("order-%d", i), percentage) {
			trueCount++
		}
	}

	got := float64(trueCount) / float64(sampleSize)
	want := float64(percentage) / 100
	if diff := got - want; diff < -toleranceFraction || diff > toleranceFraction {
		t.Errorf("routed fraction = %.4f, want within %.2f of %.2f", got, toleranceFraction, want)
	}
}

// --- MapStatusToDomainError ---

func TestMapStatusToDomainError_Nil(t *testing.T) {
	if err := router.MapStatusToDomainError(nil); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestMapStatusToDomainError_KnownCodes(t *testing.T) {
	cases := []struct {
		code codes.Code
		want error
	}{
		{codes.InvalidArgument, router.ErrInvalidRequest},
		{codes.FailedPrecondition, router.ErrPaymentDeclined},
		{codes.NotFound, router.ErrReceiptNotFound},
		{codes.DeadlineExceeded, router.ErrTimeout},
	}
	for _, tc := range cases {
		t.Run(tc.code.String(), func(t *testing.T) {
			in := status.Error(tc.code, "simulated failure")
			got := router.MapStatusToDomainError(in)
			if !errors.Is(got, tc.want) {
				t.Errorf("MapStatusToDomainError(%v status) = %v, want errors.Is(..., %v)", tc.code, got, tc.want)
			}
		})
	}
}

func TestMapStatusToDomainError_UnknownStatusCode(t *testing.T) {
	in := status.Error(codes.Unavailable, "backend down")
	got := router.MapStatusToDomainError(in)
	if !errors.Is(got, router.ErrPaymentsUnavailable) {
		t.Errorf("MapStatusToDomainError(Unavailable) = %v, want errors.Is(..., ErrPaymentsUnavailable)", got)
	}
}

func TestMapStatusToDomainError_NonStatusError(t *testing.T) {
	in := errors.New("plain transport error, not a gRPC status")
	got := router.MapStatusToDomainError(in)
	if !errors.Is(got, router.ErrPaymentsUnavailable) {
		t.Errorf("MapStatusToDomainError(plain error) = %v, want errors.Is(..., ErrPaymentsUnavailable)", got)
	}
}

// --- PaymentsRouter.Charge ---

func TestPaymentsRouter_Charge_RoutesToLegacyWhenPercentageZero(t *testing.T) {
	fake := &fakePaymentsServer{
		chargeFn: func(ctx context.Context, req *paymentsv1.ChargeRequest) (*paymentsv1.ChargeResponse, error) {
			return &paymentsv1.ChargeResponse{ReceiptId: "should-not-be-used", AmountCents: req.AmountCents}, nil
		},
	}
	client := dialFake(t, fake)
	legacy := router.NewLegacyCharger()
	r := router.NewPaymentsRouter(client, legacy, 0, time.Second)

	receipt, err := r.Charge(context.Background(), "order-1", "cust-1", 1000)
	if err != nil {
		t.Fatalf("Charge: unexpected error: %v", err)
	}
	if fake.callCount() != 0 {
		t.Errorf("gRPC service was called %d times, want 0 (percentage is 0)", fake.callCount())
	}
	if receipt.ID == "" {
		t.Error("expected a non-empty receipt ID from the legacy path")
	}
}

func TestPaymentsRouter_Charge_RoutesToGRPCWhenPercentageHundred(t *testing.T) {
	fake := &fakePaymentsServer{
		chargeFn: func(ctx context.Context, req *paymentsv1.ChargeRequest) (*paymentsv1.ChargeResponse, error) {
			return &paymentsv1.ChargeResponse{ReceiptId: "grpc-receipt-1", AmountCents: req.AmountCents}, nil
		},
	}
	client := dialFake(t, fake)
	legacy := router.NewLegacyCharger()
	r := router.NewPaymentsRouter(client, legacy, 100, time.Second)

	receipt, err := r.Charge(context.Background(), "order-1", "cust-1", 2000)
	if err != nil {
		t.Fatalf("Charge: unexpected error: %v", err)
	}
	if fake.callCount() != 1 {
		t.Errorf("gRPC service was called %d times, want 1 (percentage is 100)", fake.callCount())
	}
	if receipt.ID != "grpc-receipt-1" {
		t.Errorf("ReceiptID = %q, want %q", receipt.ID, "grpc-receipt-1")
	}
	if receipt.AmountCents != 2000 {
		t.Errorf("AmountCents = %d, want 2000", receipt.AmountCents)
	}
}

func TestPaymentsRouter_Charge_IdempotencyKeyDeterministicPerOrder(t *testing.T) {
	fake := &fakePaymentsServer{
		chargeFn: func(ctx context.Context, req *paymentsv1.ChargeRequest) (*paymentsv1.ChargeResponse, error) {
			return &paymentsv1.ChargeResponse{ReceiptId: "r", AmountCents: req.AmountCents}, nil
		},
	}
	client := dialFake(t, fake)
	r := router.NewPaymentsRouter(client, router.NewLegacyCharger(), 100, time.Second)

	if _, err := r.Charge(context.Background(), "order-77", "cust-1", 500); err != nil {
		t.Fatalf("first Charge: unexpected error: %v", err)
	}
	firstKey := fake.lastRequest().IdempotencyKey
	if firstKey == "" {
		t.Fatal("expected a non-empty idempotency_key to be sent to the gRPC service")
	}

	if _, err := r.Charge(context.Background(), "order-77", "cust-1", 500); err != nil {
		t.Fatalf("second Charge: unexpected error: %v", err)
	}
	secondKey := fake.lastRequest().IdempotencyKey

	if secondKey != firstKey {
		t.Errorf("idempotency_key changed across calls for the same order: first=%q second=%q", firstKey, secondKey)
	}
}

func TestPaymentsRouter_Charge_StatusMapping(t *testing.T) {
	fake := &fakePaymentsServer{
		chargeFn: func(ctx context.Context, req *paymentsv1.ChargeRequest) (*paymentsv1.ChargeResponse, error) {
			return nil, status.Error(codes.FailedPrecondition, "idempotency key reused with different payload")
		},
	}
	client := dialFake(t, fake)
	r := router.NewPaymentsRouter(client, router.NewLegacyCharger(), 100, time.Second)

	_, err := r.Charge(context.Background(), "order-1", "cust-1", 100)
	if !errors.Is(err, router.ErrPaymentDeclined) {
		t.Errorf("Charge error = %v, want errors.Is(..., ErrPaymentDeclined)", err)
	}
}

// This test bounds its own wait: the fake handler returns as soon as ctx is done (which
// happens at the router's configured deadline, ~30ms) or after a 2s hard cap, whichever comes
// first — so a router that never applies a deadline still fails this test in ~2s instead of
// hanging indefinitely.
func TestPaymentsRouter_Charge_DeadlineExceededMapsToTimeout(t *testing.T) {
	fake := &fakePaymentsServer{
		chargeFn: func(ctx context.Context, req *paymentsv1.ChargeRequest) (*paymentsv1.ChargeResponse, error) {
			select {
			case <-ctx.Done():
				return nil, status.FromContextError(ctx.Err()).Err()
			case <-time.After(2 * time.Second):
				return &paymentsv1.ChargeResponse{ReceiptId: "too-slow", AmountCents: req.AmountCents}, nil
			}
		},
	}
	client := dialFake(t, fake)
	r := router.NewPaymentsRouter(client, router.NewLegacyCharger(), 100, 30*time.Millisecond)

	start := time.Now()
	_, err := r.Charge(context.Background(), "order-1", "cust-1", 100)
	elapsed := time.Since(start)

	if !errors.Is(err, router.ErrTimeout) {
		t.Errorf("Charge error = %v, want errors.Is(..., ErrTimeout)", err)
	}
	if elapsed > time.Second {
		t.Errorf("Charge took %v to fail, want well under the 2s hard cap (deadline was 30ms)", elapsed)
	}
}
