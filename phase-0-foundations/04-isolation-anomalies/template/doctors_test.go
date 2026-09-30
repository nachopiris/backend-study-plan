package isolation

import (
	"context"
	"sync"
	"testing"
)

func TestGoOffCall_NeverLeavesZeroDoctorsOnCall(t *testing.T) {
	pool := requireDB(t)
	store := NewStore(pool)
	ctx := context.Background()

	d1 := uniqueName(t, "doctor-a")
	d2 := uniqueName(t, "doctor-b")
	mustExec(t, ctx, pool,
		`INSERT INTO on_call_doctors (doctor_name, on_call) VALUES ($1, true), ($2, true)`,
		d1, d2,
	)

	// Both doctors race to go off call, many times concurrently. At most one
	// of them may ever actually end up off call.
	const attemptsPerDoctor = 50
	const maxRetries = 20
	var wg sync.WaitGroup
	for i := 0; i < attemptsPerDoctor; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.GoOffCall(ctx, d1, maxRetries)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.GoOffCall(ctx, d2, maxRetries)
		}()
	}
	wg.Wait()

	var onCall int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM on_call_doctors WHERE doctor_name = ANY($1) AND on_call = true`,
		[]string{d1, d2},
	).Scan(&onCall); err != nil {
		t.Fatalf("reading on-call count: %v", err)
	}

	if onCall == 0 {
		t.Fatal("write skew: both doctors ended up off call at the same time, violating the invariant")
	}
	if onCall != 1 {
		t.Fatalf("expected exactly 1 doctor to remain on call after the race, got %d", onCall)
	}
}
