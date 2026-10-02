package ring

import (
	"fmt"
	"math"
	"testing"
)

func TestRing_EmptyRingReturnsFalse(t *testing.T) {
	r := NewRing(10)
	if _, ok := r.Get("anything"); ok {
		t.Fatal("expected Get on an empty ring to return ok=false")
	}
}

func TestRing_DeterministicMapping(t *testing.T) {
	r := NewRing(100)
	for _, n := range []string{"node-a", "node-b", "node-c"} {
		r.Add(n)
	}

	const key = "order-12345"
	first, ok := r.Get(key)
	if !ok {
		t.Fatalf("Get(%q): expected ok=true with nodes in the ring", key)
	}

	for i := 0; i < 100; i++ {
		got, ok := r.Get(key)
		if !ok {
			t.Fatalf("Get(%q) attempt %d: expected ok=true", key, i)
		}
		if got != first {
			t.Fatalf("Get(%q) is not deterministic: got %q on attempt %d, expected %q (same as the first call)", key, got, i, first)
		}
	}
}

func TestRing_BalanceAcrossNodes(t *testing.T) {
	const nodeCount = 10
	const vnodes = 100
	const keyCount = 20000

	r := NewRing(vnodes)
	for i := 0; i < nodeCount; i++ {
		r.Add(fmt.Sprintf("node-%d", i))
	}

	counts := make(map[string]int, nodeCount)
	for i := 0; i < keyCount; i++ {
		key := fmt.Sprintf("key-%d", i)
		node, ok := r.Get(key)
		if !ok {
			t.Fatalf("Get(%q): expected ok=true with %d nodes in the ring", key, nodeCount)
		}
		counts[node]++
	}

	if len(counts) != nodeCount {
		t.Fatalf("expected all %d nodes to receive at least one key, only %d did: %v", nodeCount, len(counts), counts)
	}

	mean := float64(keyCount) / float64(nodeCount)
	var variance float64
	for _, c := range counts {
		d := float64(c) - mean
		variance += d * d
	}
	variance /= float64(nodeCount)
	stddev := math.Sqrt(variance)

	// With 100 virtual nodes per real node, the distribution should be
	// reasonably even. This bound is deliberately generous — plain
	// hash/fnv or hash/crc32 over simple "node#i" strings commonly lands
	// around a 0.15-0.30 ratio here, not near-perfect uniformity — so it is
	// meant to catch a broken or degenerate implementation (e.g. one that
	// ignores the virtual node index and places every replica at the same
	// position, which produces a much larger ratio), not to require a
	// particular hash function.
	if ratio := stddev / mean; ratio > 0.35 {
		t.Fatalf("key distribution too uneven: stddev=%.1f mean=%.1f (ratio %.2f, want <= 0.35); counts=%v", stddev, mean, ratio, counts)
	}
}

func TestRing_AddingNodeMovesApproximatelyExpectedFraction(t *testing.T) {
	const nodeCount = 9
	const vnodes = 100
	const keyCount = 20000

	r := NewRing(vnodes)
	for i := 0; i < nodeCount; i++ {
		r.Add(fmt.Sprintf("node-%d", i))
	}

	keys := make([]string, keyCount)
	before := make(map[string]string, keyCount)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
		node, ok := r.Get(keys[i])
		if !ok {
			t.Fatalf("Get(%q): expected ok=true with %d nodes in the ring", keys[i], nodeCount)
		}
		before[keys[i]] = node
	}

	r.Add("node-new")

	moved, movedToNew := 0, 0
	for _, k := range keys {
		node, ok := r.Get(k)
		if !ok {
			t.Fatalf("Get(%q) after Add: expected ok=true", k)
		}
		if node != before[k] {
			moved++
			if node == "node-new" {
				movedToNew++
			}
		}
	}

	if moved != movedToNew {
		t.Fatalf(
			"expected every moved key to move specifically to the new node; %d moved in total but only "+
				"%d moved to node-new — adding one node should only steal keys for that node, never "+
				"reshuffle keys between the existing nodes",
			moved, movedToNew,
		)
	}

	wantFraction := 1.0 / float64(nodeCount+1)
	gotFraction := float64(moved) / float64(keyCount)
	// Generous tolerance (+/- 50% of the expected fraction): this is a
	// statistical property of a hash, not an exact guarantee.
	if gotFraction < wantFraction*0.5 || gotFraction > wantFraction*1.5 {
		t.Fatalf(
			"expected roughly %.1f%% of keys to move after adding one node to %d existing nodes, got %.1f%% (%d of %d keys)",
			wantFraction*100, nodeCount, gotFraction*100, moved, keyCount,
		)
	}
}

func TestRing_RemoveOnlyAffectsItsOwnKeys(t *testing.T) {
	r := NewRing(100)
	r.Add("a")
	r.Add("b")
	r.Add("c")

	const keyCount = 2000
	keys := make([]string, keyCount)
	before := make(map[string]string, keyCount)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
		node, ok := r.Get(keys[i])
		if !ok {
			t.Fatalf("Get(%q): expected ok=true", keys[i])
		}
		before[keys[i]] = node
	}

	r.Remove("b")

	for _, k := range keys {
		node, ok := r.Get(k)
		if !ok {
			t.Fatalf("Get(%q) after Remove: expected ok=true", k)
		}
		if node == "b" {
			t.Fatalf("key %q still maps to removed node %q", k, node)
		}
		if before[k] != "b" && node != before[k] {
			t.Fatalf(
				"key %q did not belong to the removed node (was on %q) but moved to %q — removing a node "+
					"must not affect keys that did not belong to it",
				k, before[k], node,
			)
		}
	}
}
