package clocks

import "testing"

func TestVectorClock_IncrementOwnNode(t *testing.T) {
	vc := NewVectorClock()
	vc.Increment("A")
	vc.Increment("A")
	vc.Increment("B")

	if got := vc.Count("A"); got != 2 {
		t.Errorf("Count(A) = %d, want 2", got)
	}
	if got := vc.Count("B"); got != 1 {
		t.Errorf("Count(B) = %d, want 1", got)
	}
	if got := vc.Count("C"); got != 0 {
		t.Errorf("Count(C) = %d, want 0 (never incremented)", got)
	}
}

func TestVectorClock_Compare_Equal(t *testing.T) {
	a := NewVectorClock()
	a.Increment("A")
	a.Increment("B")

	b := NewVectorClock()
	b.Increment("A")
	b.Increment("B")

	if got := a.Compare(b); got != Equal {
		t.Errorf("a.Compare(b) = %v, want Equal", got)
	}
	if got := b.Compare(a); got != Equal {
		t.Errorf("b.Compare(a) = %v, want Equal", got)
	}
}

func TestVectorClock_Compare_BeforeAfter(t *testing.T) {
	older := NewVectorClock()
	older.Increment("A")

	newer := NewVectorClock()
	newer.Increment("A")
	newer.Increment("A")
	newer.Increment("B")

	if got := older.Compare(newer); got != Before {
		t.Errorf("older.Compare(newer) = %v, want Before (A:1 is dominated by A:2,B:1)", got)
	}
	if got := newer.Compare(older); got != After {
		t.Errorf("newer.Compare(older) = %v, want After", got)
	}
}

func TestVectorClock_Compare_Concurrent(t *testing.T) {
	a := NewVectorClock()
	a.Increment("A")

	b := NewVectorClock()
	b.Increment("B")

	if got := a.Compare(b); got != Concurrent {
		t.Errorf("a.Compare(b) = %v, want Concurrent (A:1,B:0 vs A:0,B:1: neither dominates)", got)
	}
	if got := b.Compare(a); got != Concurrent {
		t.Errorf("b.Compare(a) = %v, want Concurrent", got)
	}
}

func TestVectorClock_Merge(t *testing.T) {
	a := NewVectorClock()
	a.Increment("A")
	a.Increment("A") // A:2

	b := NewVectorClock()
	b.Increment("A") // A:1
	b.Increment("B")
	b.Increment("B")
	b.Increment("B") // B:3

	merged := NewVectorClock()
	merged.Merge(a)
	merged.Merge(b)

	if got := merged.Count("A"); got != 2 {
		t.Errorf("merged.Count(A) = %d, want 2 (max(2,1))", got)
	}
	if got := merged.Count("B"); got != 3 {
		t.Errorf("merged.Count(B) = %d, want 3 (max(0,3))", got)
	}

	// The merged clock must dominate both inputs.
	if got := merged.Compare(a); got != After {
		t.Errorf("merged.Compare(a) = %v, want After", got)
	}
	if got := merged.Compare(b); got != After {
		t.Errorf("merged.Compare(b) = %v, want After", got)
	}

	// Merge must not have mutated its argument.
	if got := b.Count("A"); got != 1 {
		t.Errorf("b.Count(A) = %d after Merge, want unchanged 1 (Merge must not mutate its argument)", got)
	}
}

func TestVectorClock_CopyIsIndependent(t *testing.T) {
	original := NewVectorClock()
	original.Increment("A")

	snapshot := original.Copy()
	original.Increment("A") // advance the original after taking the snapshot

	if got := snapshot.Count("A"); got != 1 {
		t.Errorf("snapshot.Count(A) = %d after mutating original, want 1 (Copy must not alias the original's storage)", got)
	}
	if got := original.Count("A"); got != 2 {
		t.Errorf("original.Count(A) = %d, want 2", got)
	}

	// And the reverse: mutating the snapshot must not affect the original.
	snapshot.Increment("B")
	if got := original.Count("B"); got != 0 {
		t.Errorf("original.Count(B) = %d after mutating snapshot, want 0", got)
	}
}
