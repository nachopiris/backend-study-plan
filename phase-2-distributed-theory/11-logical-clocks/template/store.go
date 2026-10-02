package clocks

// Version is one sibling value stored under a key, tagged with the causal
// context it was written with.
type Version struct {
	Value   string
	Context VectorClock
}

// Store is a tiny Dynamo-style multi-value register: a key-value store
// where a write never silently overwrites a concurrent write. Instead, when
// two writes are Concurrent (neither one's context causally dominates the
// other's), both are kept as siblings, and it is up to a later reader to
// reconcile them — typically by merging their contexts and writing back a
// single resolved value, exactly as Dynamo and Riak do.
type Store struct {
	versions map[string][]Version
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{versions: make(map[string][]Version)}
}

// Put writes value under key, tagged with causal context ctx. ctx should
// come from a prior Get of the same key: a client reads the current
// siblings, merges their contexts together, increments its own node in the
// merged result, and passes that as ctx here — that is what lets Put tell a
// write that supersedes the siblings it read apart from one that raced with
// them.
//
// Reconciliation against the versions already stored under key:
//   - any existing version whose Context is causally before ctx (that
//     version's Context.Compare(ctx) == Before) is dropped: ctx's write
//     already accounted for it and supersedes it;
//   - if, after dropping superseded versions, ctx is itself causally before
//     or equal to any version that remains, this write is stale (it did not
//     see something the store already has) and Put does nothing further —
//     none of the remaining versions are touched;
//   - otherwise (ctx is Concurrent with everything that remains, or nothing
//     remains), value is appended as a new sibling under key.
//
// Not implemented yet: no-op.
func (s *Store) Put(key, value string, ctx VectorClock) {
}

// Get returns every sibling version currently stored under key. A nil (or
// empty) result means key has never been written, or every version written
// to it was later superseded. The order of the returned siblings is
// unspecified.
//
// Not implemented yet: always returns nil.
func (s *Store) Get(key string) []Version {
	return nil
}
