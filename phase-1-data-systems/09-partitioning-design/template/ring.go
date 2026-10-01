// Package ring implements consistent hashing with virtual nodes: a way to
// map keys (for example, order IDs) to a changing set of nodes (for example,
// database shards or cache servers) such that adding or removing one node
// moves only a small, predictable fraction of keys instead of reshuffling
// everything, as a naive `hash(key) % node_count` would.
//
// No external dependencies: use only the Go standard library (hash/fnv,
// hash/crc32, or similar, are all reasonable choices).
package ring

// Ring is a consistent hashing ring with virtual nodes.
//
// Each real node added via Add is represented by `vnodes` positions on the
// ring (its virtual nodes), computed deterministically from the node's name
// and an index. Spreading each real node across many positions is what
// makes the key distribution reasonably even even with few real nodes: a
// single position per node would make the size of each node's share of the
// ring depend entirely on how far apart random hash positions happen to
// land, which can be wildly uneven.
type Ring struct {
	vnodes int
}

// NewRing creates an empty Ring that will give each node added via Add
// vnodes virtual positions on the ring. More virtual nodes per node gives a
// more even key distribution, at the cost of more memory and a slightly
// slower Get.
func NewRing(vnodes int) *Ring {
	return &Ring{vnodes: vnodes}
}

// Add adds node to the ring, placing vnodes deterministic virtual positions
// for it. Adding the same node twice should not duplicate its positions.
//
// Not implemented yet: this is a no-op, which is why every test that adds a
// node and then calls Get fails with ok=false below, instead of panicking —
// that failure is the exercise, not a bug in the test file.
func (r *Ring) Add(node string) {
}

// Remove removes node, and all of its virtual positions, from the ring.
// Removing a node that was never added is a no-op.
//
// Not implemented yet: this is a no-op.
func (r *Ring) Remove(node string) {
}

// Get returns the node responsible for key: conceptually, hash key onto the
// same ring used for virtual positions, then walk clockwise (toward larger
// hash values, wrapping around past the maximum back to zero) until you hit
// the first virtual position — that position's real node owns key.
//
// Get returns ("", false) if the ring has no nodes.
//
// Not implemented yet: always returns ("", false).
func (r *Ring) Get(key string) (string, bool) {
	return "", false
}
