package replication

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// uniqueBody must stay unique not just within one test run but across
// repeated `go test` invocations: unlike the testcontainers-based practices
// elsewhere in this repo, this practice's primary and replica are long-lived
// (started once with `docker compose up -d`, see README) and keep
// accumulating rows across every `go test` run during development. Keying
// only on t.Name() would let a later run collide with an identically named
// row a previous run already replicated, silently hiding the anomaly this
// suite exists to exercise.
func uniqueBody(t *testing.T, suffix string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d-%s", t.Name(), time.Now().UnixNano(), suffix)
}

func TestNaiveRouter_DemonstratesStaleRead(t *testing.T) {
	primary, replica := requirePools(t)
	router := NewNaiveRouter(primary, replica)
	ctx := context.Background()
	body := uniqueBody(t, "note")

	if _, err := router.WriteNote(ctx, body); err != nil {
		t.Fatalf("WriteNote: %v", err)
	}

	// The replica's recovery_min_apply_delay (see docker-compose.yml) makes
	// it reliably behind by ~1.5s, so a read immediately after the write
	// almost certainly misses it — that miss is the anomaly this test
	// exists to demonstrate, not a bug in NaiveRouter.
	count, err := router.CountNotesWithBody(ctx, body)
	if err != nil {
		t.Fatalf("CountNotesWithBody immediately after write: %v", err)
	}
	if count != 0 {
		t.Logf("replica had already caught up (count=%d) — this can happen in a fast or "+
			"already-warm environment, but defeats this test's demonstration; "+
			"consider raising recovery_min_apply_delay if this becomes flaky", count)
	}

	// Eventually (well past the apply delay) the replica does catch up.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		count, err = router.CountNotesWithBody(ctx, body)
		if err != nil {
			t.Fatalf("CountNotesWithBody while waiting to converge: %v", err)
		}
		if count == 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if count != 1 {
		t.Fatalf("expected the replica to eventually reflect the write, got count=%d after waiting", count)
	}
}

func TestRouter_ReadYourWriteCount_SeesOwnWrite(t *testing.T) {
	primary, replica := requirePools(t)
	router := NewRouter(primary, replica)
	ctx := context.Background()
	body := uniqueBody(t, "note")

	_, token, err := router.WriteNote(ctx, body)
	if err != nil {
		t.Fatalf("WriteNote: %v", err)
	}
	if token == "" {
		t.Fatal("expected WriteNote to return a non-empty WriteToken")
	}

	start := time.Now()
	count, err := router.ReadYourWriteCount(ctx, body, token, 5*time.Second)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("ReadYourWriteCount: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected ReadYourWriteCount to see the caller's own write (count=1), got %d", count)
	}
	const bound = 3 * time.Second
	if elapsed > bound {
		t.Fatalf("ReadYourWriteCount took %s, expected well under %s given maxWait=5s and a ~1.5s replica delay", elapsed, bound)
	}
}

func TestRouter_ReadYourWriteCount_FallsBackWithinBound(t *testing.T) {
	primary, replica := requirePools(t)
	router := NewRouter(primary, replica)
	ctx := context.Background()
	body := uniqueBody(t, "note")

	_, token, err := router.WriteNote(ctx, body)
	if err != nil {
		t.Fatalf("WriteNote: %v", err)
	}

	// maxWait is far shorter than the replica's ~1.5s apply delay, so the
	// replica almost certainly has not caught up to token in time — Router
	// must fall back to the primary instead of blocking until the replica
	// eventually does.
	const maxWait = 200 * time.Millisecond
	start := time.Now()
	count, err := router.ReadYourWriteCount(ctx, body, token, maxWait)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("ReadYourWriteCount: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected the correct count (1) via primary fallback, got %d", count)
	}
	const bound = 1 * time.Second
	if elapsed > bound {
		t.Fatalf("ReadYourWriteCount took %s with maxWait=%s; expected it to fall back to the primary "+
			"well under %s instead of waiting for the replica's full ~1.5s apply delay", elapsed, maxWait, bound)
	}
}
