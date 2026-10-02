package chaos

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/jackc/pgx/v5/pgxpool"
)

// budget is the maximum time a caller of /checkout should ever have to wait,
// no matter what is happening to Postgres or the payment provider. A
// resilient service answers inside this budget by failing fast — a hang is
// a failure of this test even if the service would have eventually
// succeeded.
const budget = 1500 * time.Millisecond

func TestCheckout_DegradesWithinBudgetWhenPaymentProviderIsSlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: requires the docker-compose stack (-short)")
	}

	toxiproxyURL := os.Getenv("TOXIPROXY_URL")
	postgresDSN := os.Getenv("POSTGRES_DSN")
	paymentURL := os.Getenv("PAYMENT_URL")
	if toxiproxyURL == "" || postgresDSN == "" || paymentURL == "" {
		t.Skip("skipping: TOXIPROXY_URL, POSTGRES_DSN and PAYMENT_URL must all be set (see README)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, postgresDSN)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		t.Fatalf("db.Ping: %v (is `docker compose up -d` running?)", err)
	}

	tx := toxiproxy.NewClient(toxiproxyURL)
	paymentProxy, err := tx.Proxy("payment")
	if err != nil {
		t.Fatalf("toxiproxy: fetching proxy %q: %v", "payment", err)
	}

	// Inject 5s of downstream latency between the checkout service and the
	// payment provider — far more than `budget`. A naive client with no
	// timeout waits out the full latency; a resilient one gives up long
	// before that.
	toxic, err := paymentProxy.AddToxic("checkout-test-latency", "latency", "downstream", 1.0, toxiproxy.Attributes{
		"latency": 5000,
	})
	if err != nil {
		t.Fatalf("toxiproxy: adding latency toxic: %v", err)
	}
	defer func() {
		if err := paymentProxy.RemoveToxic(toxic.Name); err != nil {
			t.Logf("toxiproxy: removing toxic: %v (continuing)", err)
		}
	}()

	svc := NewService(db, paymentURL, &http.Client{Timeout: 2 * time.Second})
	server := httptest.NewServer(svc.Handler())
	defer server.Close()

	body, _ := json.Marshal(CheckoutRequest{SKU: "widget-1", AmountCents: 1999})

	start := time.Now()
	// A client-side timeout well above `budget` so that a hanging service
	// fails this test on the ELAPSED assertion below, with a clear message,
	// instead of just timing out the test binary.
	client := &http.Client{Timeout: 2 * budget}
	resp, reqErr := client.Post(server.URL+"/checkout", "application/json", bytes.NewReader(body))
	elapsed := time.Since(start)

	if elapsed > budget {
		t.Fatalf("checkout took %s, want answered within budget %s (even as a degraded/error response); "+
			"the service is almost certainly calling the payment provider with no timeout", elapsed, budget)
	}
	if reqErr != nil {
		t.Fatalf("POST /checkout: %v (elapsed %s)", reqErr, elapsed)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Fatalf("POST /checkout returned 200 OK while the payment provider was unreachable within budget; "+
			"want a degraded/error response (e.g. 503) instead of a false success")
	}

	var got CheckoutResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decoding response body: %v", err)
	}
	if got.Status != "error" {
		t.Errorf("CheckoutResponse.Status = %q, want %q", got.Status, "error")
	}
	if got.Reason == "" {
		t.Errorf("CheckoutResponse.Reason is empty, want a short machine-readable reason (e.g. %q)", "payment_timeout")
	}
}
