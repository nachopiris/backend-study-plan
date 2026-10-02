package clocks

// Ordering is the result of comparing two VectorClocks.
type Ordering int

const (
	// Equal means both clocks have exactly the same counter for every node.
	Equal Ordering = iota
	// Before means the receiver happened-before the argument: every one of
	// the receiver's counters is <= the argument's corresponding counter,
	// and at least one is strictly less.
	Before
	// After means the argument happened-before the receiver (the symmetric
	// case of Before).
	After
	// Concurrent means neither clock happened-before the other: at least
	// one counter is greater in the receiver, and at least one is greater
	// in the argument. This is the relationship a LamportClock timestamp
	// alone cannot distinguish from Before or After.
	Concurrent
)

// VectorClock tracks one counter per node (process, replica, client — any
// identifier a system uses to tell writers apart). A node absent from a
// VectorClock is treated as having counter 0, exactly as if it had been
// present with that value.
type VectorClock struct {
	counts map[string]uint64
}

// NewVectorClock returns an empty VectorClock (every node implicitly at 0).
func NewVectorClock() VectorClock {
	return VectorClock{counts: make(map[string]uint64)}
}

// Increment records one local event at node: it raises node's own counter
// in vc by exactly one, leaving every other node's counter untouched. This
// mutates vc in place, the same way LamportClock.Tick mutates its receiver.
//
// Calling Increment on the zero value of VectorClock (one not built with
// NewVectorClock) is undefined; always start from NewVectorClock.
//
// Not implemented yet: no-op.
func (vc VectorClock) Increment(node string) {
}

// Merge folds other into vc in place: for every node that appears in either
// clock, vc's counter for that node becomes the larger of vc's and other's
// counters for it (treating an absent node as 0). other is left unchanged.
//
// Merge is what a process uses when it learns about another clock (for
// example, a reader that saw several sibling versions, or a receiver
// catching up with a sender) and wants its own clock to reflect everything
// it now knows about, before it increments its own node and proceeds.
//
// Not implemented yet: no-op.
func (vc VectorClock) Merge(other VectorClock) {
}

// Compare reports how vc relates to other. See the Ordering constants for
// the exact meaning of each result.
//
// Not implemented yet: always returns Equal.
func (vc VectorClock) Compare(other VectorClock) Ordering {
	return Equal
}

// Copy returns an independent copy of vc. Mutating the copy via Increment or
// Merge must never affect vc, and mutating vc afterwards must never affect
// the copy — a VectorClock holds a map internally, and maps are reference
// types in Go, so a naive copy (just returning vc as-is) would alias the
// same underlying storage instead of truly duplicating it.
//
// Not implemented yet: returns the zero value VectorClock (an uninitialized,
// unusable clock — see Increment).
func (vc VectorClock) Copy() VectorClock {
	return VectorClock{}
}

// Count returns vc's current counter for node, or 0 if node has never been
// incremented in vc (including when vc is the zero value).
func (vc VectorClock) Count(node string) uint64 {
	return vc.counts[node]
}
