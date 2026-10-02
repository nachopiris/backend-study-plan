package coupling

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// normalizeCycle rotates a cycle so it starts at its lexicographically
// smallest element. Two cycles that are the same loop starting at a
// different point (e.g. {"a","b","c"} and {"b","c","a"}) normalize to the
// same slice. This is safe because an elementary cycle, by this package's
// contract, never repeats a package before closing the loop.
func normalizeCycle(c []string) []string {
	if len(c) == 0 {
		return nil
	}
	minIdx := 0
	for i, v := range c {
		if v < c[minIdx] {
			minIdx = i
		}
	}
	rotated := make([]string, len(c))
	for i := range c {
		rotated[i] = c[(minIdx+i)%len(c)]
	}
	return rotated
}

// cycleKeys converts a set of cycles into a sorted set of comparable string
// keys, independent of cycle order and rotation.
func cycleKeys(cycles [][]string) []string {
	keys := make([]string, 0, len(cycles))
	for _, c := range cycles {
		keys = append(keys, strings.Join(normalizeCycle(c), ">"))
	}
	sort.Strings(keys)
	return keys
}

func assertCycles(t *testing.T, got, want [][]string) {
	t.Helper()
	gotKeys := cycleKeys(got)
	wantKeys := cycleKeys(want)
	if !reflect.DeepEqual(gotKeys, wantKeys) {
		t.Errorf("cycles mismatch:\n got:  %v (raw: %v)\n want: %v (raw: %v)", gotKeys, got, wantKeys, want)
	}
}

func TestCycles_DAGHasNoCycles(t *testing.T) {
	// A diamond: both orders and notifications depend on events, which
	// depends on nothing. Shared dependencies alone are not cycles.
	graph := map[string][]string{
		"orders":        {"events"},
		"notifications": {"events"},
		"events":        {},
	}

	assertCycles(t, Cycles(graph), nil)
}

func TestCycles_SelfImportIsALengthOneCycle(t *testing.T) {
	graph := map[string][]string{
		"orders": {"orders"},
	}

	assertCycles(t, Cycles(graph), [][]string{{"orders"}})
}

func TestCycles_DirectTwoCycle(t *testing.T) {
	graph := map[string][]string{
		"orders":    {"inventory"},
		"inventory": {"orders"},
	}

	assertCycles(t, Cycles(graph), [][]string{{"orders", "inventory"}})
}

func TestCycles_ThreeNodeCycle(t *testing.T) {
	graph := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"a"},
	}

	assertCycles(t, Cycles(graph), [][]string{{"a", "b", "c"}})
}

func TestCycles_MultipleDisjointCycles(t *testing.T) {
	graph := map[string][]string{
		"a": {"b"},
		"b": {"a"},
		"x": {"y"},
		"y": {"x"},
		// z sits outside both cycles entirely.
		"z": {"a"},
	}

	assertCycles(t, Cycles(graph), [][]string{{"a", "b"}, {"x", "y"}})
}

func TestCycles_ReportsEachElementaryCycleOnce(t *testing.T) {
	// A single triangle must be reported once, not once per node it visits.
	graph := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"a"},
	}

	got := Cycles(graph)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 cycle, got %d: %v", len(got), got)
	}
}

func TestCycles_DoesNotMutateInput(t *testing.T) {
	graph := map[string][]string{
		"a": {"b"},
		"b": {"a"},
	}

	_ = Cycles(graph)

	if len(graph) != 2 || len(graph["a"]) != 1 || graph["a"][0] != "b" {
		t.Fatalf("Cycles mutated its input graph: %v", graph)
	}
}

func TestCycles_EmptyGraphHasNoCycles(t *testing.T) {
	assertCycles(t, Cycles(map[string][]string{}), nil)
}
