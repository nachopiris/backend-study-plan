// Command ordersprovider is a small, working HTTP orders API used ONLY as the real provider that
// Pact JS Verifier checks the BFF's consumer contract against (see
// ../bff/scripts/verify-provider.ts). This is GIVEN, working code — you do not need to (and
// should not) modify it. The exercise in this practice is the BFF client code in ../bff/src.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// orderItem and order mirror the shape the BFF's OrdersClient (../bff/src/orders-client.ts) and
// its Pact consumer test (../bff/test/orders-client.pact.test.ts) expect.
type orderItem struct {
	SKU            string `json:"sku"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unitPriceCents"`
}

type order struct {
	ID         string      `json:"id"`
	CustomerID string      `json:"customerId"`
	Status     string      `json:"status"`
	Items      []orderItem `json:"items"`
	TotalCents int64       `json:"totalCents"`
}

// orders is a fixed, in-memory fixture. A real orders provider would read from a database; this
// one only needs to satisfy the pact file deterministically.
var orders = map[string]order{
	"order-1": {
		ID:         "order-1",
		CustomerID: "cust-1",
		Status:     "confirmed",
		Items: []orderItem{
			{SKU: "sku-1", Quantity: 2, UnitPriceCents: 500},
		},
		TotalCents: 1000,
	},
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		o, ok := orders[r.PathValue("id")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(o)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "4567"
	}
	addr := fmt.Sprintf(":%s", port)
	log.Printf("orders provider listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
