// Package linearizability implements a brute-force linearizability checker
// for a single integer read/write register.
//
// Linearizability is a real-time consistency model: it requires that there
// exists some sequential order of all operations such that (1) every read
// returns the value written by the most recent write before it in that
// order (or the register's initial value, 0, if there is none), and (2)
// that order is consistent with real time — if operation a's interval
// (from its call to its return) ends strictly before operation b's interval
// begins, then a must come before b in the order. Operations whose
// intervals overlap are unconstrained relative to each other: either order
// is allowed, as long as some order exists that also satisfies (1).
//
// No external dependencies: only the Go standard library.
package linearizability

// OpKind distinguishes a read from a write in an Operation.
type OpKind int

const (
	Read OpKind = iota
	Write
)

// Operation is one client operation against the register, with its
// real-time interval.
//
// Call and Return are abstract time units (not wall-clock durations): what
// matters is only their relative order, with Call < Return for every valid
// operation. For a Write, Value is the value being written. For a Read,
// Value is the value the client observed (the value the operation claims
// the register held).
type Operation struct {
	Kind   OpKind
	Value  int
	Call   int64
	Return int64
}

// Check reports whether history is linearizable with respect to a single
// integer register whose initial value (before any Write in history) is 0.
//
// history does not need to be sorted in any particular order; Check must
// consider every operation's Call and Return to determine which real-time
// orderings are allowed.
//
// Not implemented yet: always returns false.
func Check(history []Operation) bool {
	return false
}
