// Command paymentprovider is a fake payment provider used only as
// infrastructure for practice 14. It is already complete: you do not need to
// read or modify it to do the practice, only to know it exists and what it
// does.
//
// It is deliberately fast and reliable on its own. Every failure scenario
// you will reproduce in this practice (latency, timeouts, dropped
// connections, bandwidth limits) comes from Toxiproxy sitting in front of
// it, not from this server.
package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type chargeRequest struct {
	OrderID     string `json:"order_id"`
	AmountCents int64  `json:"amount_cents"`
}

type chargeResponse struct {
	Approved bool   `json:"approved"`
	ChargeID string `json:"charge_id"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /charge", handleCharge)

	log.Printf("paymentprovider: listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func handleCharge(w http.ResponseWriter, r *http.Request) {
	var req chargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// A tiny, realistic amount of processing time. This is NOT the chaos
	// mechanism of this practice; it is just here so a happy-path charge
	// does not look suspiciously instantaneous.
	time.Sleep(time.Duration(rand.Intn(15)) * time.Millisecond)

	resp := chargeResponse{
		Approved: true,
		ChargeID: "ch_" + req.OrderID,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
