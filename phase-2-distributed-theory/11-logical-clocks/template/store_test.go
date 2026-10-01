package clocks

import "testing"

func TestStore_GetOnMissingKeyIsEmpty(t *testing.T) {
	s := NewStore()
	if got := s.Get("x"); len(got) != 0 {
		t.Fatalf("Get on missing key = %v, want empty", got)
	}
}

func TestStore_SinglePutIsVisible(t *testing.T) {
	s := NewStore()

	ctx := NewVectorClock()
	ctx.Increment("A")
	s.Put("x", "v1", ctx)

	got := s.Get("x")
	if len(got) != 1 {
		t.Fatalf("Get(x) returned %d versions, want 1: %v", len(got), got)
	}
	if got[0].Value != "v1" {
		t.Errorf("Get(x)[0].Value = %q, want %q", got[0].Value, "v1")
	}
}

// TestStore_ConcurrentPutsYieldSiblings writes two values with contexts that
// are Concurrent (neither one read the other), as would happen if two
// clients wrote to the same key without coordinating. Both must survive as
// siblings: this is the whole point of a multi-value register.
func TestStore_ConcurrentPutsYieldSiblings(t *testing.T) {
	s := NewStore()

	ctxA := NewVectorClock()
	ctxA.Increment("A")
	s.Put("cart", "added-item-1", ctxA)

	ctxB := NewVectorClock()
	ctxB.Increment("B")
	s.Put("cart", "added-item-2", ctxB)

	got := s.Get("cart")
	if len(got) != 2 {
		t.Fatalf("Get(cart) returned %d versions, want 2 siblings: %v", len(got), got)
	}

	values := map[string]bool{}
	for _, v := range got {
		values[v.Value] = true
	}
	if !values["added-item-1"] || !values["added-item-2"] {
		t.Fatalf("Get(cart) = %v, want both added-item-1 and added-item-2 present as siblings", got)
	}
}

// TestStore_PutWithMergedContextResolvesSiblings scripts the standard
// Dynamo read-reconcile-write cycle: a client reads the two concurrent
// siblings left by TestStore_ConcurrentPutsYieldSiblings-like writes, merges
// their contexts, increments its own node, and writes back a single
// resolved value. That resolved write must causally dominate both siblings,
// so it replaces them instead of becoming a third sibling.
func TestStore_PutWithMergedContextResolvesSiblings(t *testing.T) {
	s := NewStore()

	ctxA := NewVectorClock()
	ctxA.Increment("A")
	s.Put("cart", "added-item-1", ctxA)

	ctxB := NewVectorClock()
	ctxB.Increment("B")
	s.Put("cart", "added-item-2", ctxB)

	if got := len(s.Get("cart")); got != 2 {
		t.Fatalf("precondition failed: Get(cart) returned %d versions, want 2 siblings before resolving", got)
	}

	// A client ("C") reads both siblings, merges their contexts, and writes
	// back a resolved value that accounts for both.
	resolved := NewVectorClock()
	resolved.Merge(ctxA)
	resolved.Merge(ctxB)
	resolved.Increment("C")
	s.Put("cart", "merged-cart", resolved)

	got := s.Get("cart")
	if len(got) != 1 {
		t.Fatalf("Get(cart) after resolving write returned %d versions, want exactly 1: %v", len(got), got)
	}
	if got[0].Value != "merged-cart" {
		t.Errorf("Get(cart)[0].Value = %q, want %q", got[0].Value, "merged-cart")
	}
}

// TestStore_StaleWriteIsDropped writes under a context that is causally
// before a version already stored (as if the writer never saw it), and
// checks that the stale write does not corrupt or hide the existing one.
func TestStore_StaleWriteIsDropped(t *testing.T) {
	s := NewStore()

	ctx := NewVectorClock()
	ctx.Increment("A")
	s.Put("x", "current", ctx)

	// A stale write: an empty context, causally before "current"'s context.
	stale := NewVectorClock()
	s.Put("x", "stale", stale)

	got := s.Get("x")
	if len(got) != 1 {
		t.Fatalf("Get(x) after a stale write returned %d versions, want exactly 1 (the stale write must be dropped): %v", len(got), got)
	}
	if got[0].Value != "current" {
		t.Errorf("Get(x)[0].Value = %q, want %q (stale write must not replace or hide it)", got[0].Value, "current")
	}
}
