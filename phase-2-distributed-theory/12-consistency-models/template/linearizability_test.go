package linearizability

import "testing"

func TestCheck_EmptyHistoryIsLinearizable(t *testing.T) {
	if !Check(nil) {
		t.Fatal("Check(nil) = false, want true (an empty history is trivially linearizable)")
	}
}

func TestCheck_SingleWriteThenRead(t *testing.T) {
	history := []Operation{
		{Kind: Write, Value: 1, Call: 0, Return: 10},
		{Kind: Read, Value: 1, Call: 12, Return: 20},
	}
	if !Check(history) {
		t.Fatal("Check: want true (the write completed before the read started, and the read saw its value)")
	}
}

// TestCheck_StaleReadAfterWriteCompleted is the classic linearizability
// violation: a write finishes in real time, and a later operation (one that
// starts after the write returned) reads a value from before the write.
// Real time forbids this: once Write(1) has returned, every operation whose
// interval starts afterward must observe 1 (or whatever a later write
// supersedes it with) — it can never see the pre-write value again.
func TestCheck_StaleReadAfterWriteCompleted(t *testing.T) {
	history := []Operation{
		{Kind: Write, Value: 1, Call: 0, Return: 10},
		{Kind: Read, Value: 0, Call: 15, Return: 20}, // starts after the write returned, yet reads 0
	}
	if Check(history) {
		t.Fatal("Check: want false (the read started after the write completed but observed the stale pre-write value)")
	}
}

// TestCheck_OverlappingOperationsAdmitEitherOrder_ReadsOldValue checks that
// an operation overlapping a write is free to linearize before it: here the
// read's interval overlaps the write's interval, so the read reading the
// pre-write value (0) is legal — nothing in real time forces the write to
// come first.
func TestCheck_OverlappingOperationsAdmitEitherOrder_ReadsOldValue(t *testing.T) {
	history := []Operation{
		{Kind: Write, Value: 1, Call: 0, Return: 20},
		{Kind: Read, Value: 0, Call: 5, Return: 15}, // fully inside the write's interval
	}
	if !Check(history) {
		t.Fatal("Check: want true (the read overlaps the write and may legally linearize before it, reading the old value)")
	}
}

// TestCheck_OverlappingOperationsAdmitEitherOrder_ReadsNewValue is the
// mirror image of the previous case: the same overlapping shape, but the
// read instead observes the write's value, which must also be legal — the
// write can equally well linearize before the read.
func TestCheck_OverlappingOperationsAdmitEitherOrder_ReadsNewValue(t *testing.T) {
	history := []Operation{
		{Kind: Write, Value: 1, Call: 0, Return: 20},
		{Kind: Read, Value: 1, Call: 5, Return: 15},
	}
	if !Check(history) {
		t.Fatal("Check: want true (the read overlaps the write and may legally linearize after it, reading the new value)")
	}
}

// TestCheck_SequenceOfWritesAndReads is a longer, fully non-overlapping
// history: each operation starts only after the previous one returned, so
// the only possible order is real-time order, and the register's value
// must follow it exactly.
func TestCheck_SequenceOfWritesAndReads(t *testing.T) {
	history := []Operation{
		{Kind: Write, Value: 1, Call: 0, Return: 10},
		{Kind: Read, Value: 1, Call: 11, Return: 20},
		{Kind: Write, Value: 2, Call: 21, Return: 30},
		{Kind: Read, Value: 2, Call: 31, Return: 40},
		{Kind: Read, Value: 2, Call: 41, Return: 50},
	}
	if !Check(history) {
		t.Fatal("Check: want true (a fully sequential, non-overlapping history whose reads match real-time order)")
	}
}

func TestCheck_SequenceWithWrongValueIsRejected(t *testing.T) {
	history := []Operation{
		{Kind: Write, Value: 1, Call: 0, Return: 10},
		{Kind: Read, Value: 2, Call: 11, Return: 20}, // 2 was never written
	}
	if Check(history) {
		t.Fatal("Check: want false (the read observed a value that was never written)")
	}
}

// TestCheck_NonLinearizableDespiteOneOverlap combines an overlapping
// operation (which, by itself, would admit either order) with a later,
// non-overlapping operation that pins down what the overlap must have
// resolved to. No linearization satisfies both: the first read is free to
// return either 1 or 2, but once committed to the order that makes it
// return 1, the second write (completed before the final read) still
// forces the final read to see 2 — the stale value 1 the first read saw
// cannot leak forward past Write(2)'s completion.
func TestCheck_NonLinearizableDespiteOneOverlap(t *testing.T) {
	history := []Operation{
		{Kind: Write, Value: 1, Call: 0, Return: 10},
		{Kind: Write, Value: 2, Call: 11, Return: 25}, // overlaps the read below
		{Kind: Read, Value: 1, Call: 20, Return: 22},  // overlaps Write(2); legally may see 1
		{Kind: Read, Value: 1, Call: 30, Return: 40},  // starts after Write(2) returned: must see 2
	}
	if Check(history) {
		t.Fatal("Check: want false (the final read starts after Write(2) completed but still observes the stale value 1)")
	}
}
