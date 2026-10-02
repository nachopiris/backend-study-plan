// Package chaos is a deliberately naive checkout service: it works
// perfectly fine as long as everything downstream of it (Postgres, the
// payment provider) behaves. It is YOUR job to hunt down every place where
// it assumes the happy path and make it degrade instead of hang when a
// dependency gets slow or disappears.
//
// Do not rewrite the whole service. The fix for this practice is small:
// a handful of context deadlines and a timed-out HTTP client, applied at
// the two points where Service talks to something outside the process.
package chaos

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CheckoutRequest is the body of POST /checkout.
type CheckoutRequest struct {
	SKU         string `json:"sku"`
	AmountCents int64  `json:"amount_cents"`
}

// CheckoutResponse is the body returned by POST /checkout, on both the
// happy path and the degraded path. Status is "paid" on success and
// "error" on any failure; Reason is only set when Status is "error", and
// should be a short machine-readable tag (e.g. "payment_timeout"), never a
// raw error message.
type CheckoutResponse struct {
	Status  string `json:"status"`
	OrderID string `json:"order_id,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type paymentChargeRequest struct {
	OrderID     string `json:"order_id"`
	AmountCents int64  `json:"amount_cents"`
}

// Service is the naive checkout service. db is a connection pool to
// Postgres; paymentURL is the base URL of the payment provider (in this
// practice, always a Toxiproxy proxy address, never the provider directly).
type Service struct {
	db         *pgxpool.Pool
	paymentURL string
	httpClient *http.Client
}

// NewService builds a Service. httpClient, if nil, defaults to
// http.DefaultClient — which, notably, has no timeout at all. That default
// is part of what you need to fix.
func NewService(db *pgxpool.Pool, paymentURL string, httpClient *http.Client) *Service {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Service{db: db, paymentURL: paymentURL, httpClient: httpClient}
}

// Handler returns the HTTP handler for this service: a single
// POST /checkout endpoint.
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /checkout", s.handleCheckout)
	return mux
}

func (s *Service) handleCheckout(w http.ResponseWriter, r *http.Request) {
	var req CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// NAIVE: context.Background() means this query waits as long as it
	// takes to acquire a connection from the pool AND run the INSERT, no
	// matter how slow Postgres (or the network to it) gets. There is no
	// deadline anywhere in this call.
	var orderID string
	err := s.db.QueryRow(
		context.Background(),
		`INSERT INTO orders (sku, amount_cents, status)
		 VALUES ($1, $2, 'pending')
		 RETURNING id`,
		req.SKU, req.AmountCents,
	).Scan(&orderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body, _ := json.Marshal(paymentChargeRequest{OrderID: orderID, AmountCents: req.AmountCents})

	// NAIVE: s.httpClient may be http.DefaultClient, which has NO timeout.
	// If the payment provider (or anything between here and it) stalls,
	// this call blocks forever and so does the HTTP response to the
	// caller of /checkout.
	resp, err := s.httpClient.Post(s.paymentURL+"/charge", "application/json", bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "payment provider rejected the charge", http.StatusBadGateway)
		return
	}

	// NAIVE: again context.Background(), same problem as the INSERT above.
	_, err = s.db.Exec(context.Background(), `UPDATE orders SET status = 'paid' WHERE id = $1`, orderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(CheckoutResponse{Status: "paid", OrderID: orderID})
}
