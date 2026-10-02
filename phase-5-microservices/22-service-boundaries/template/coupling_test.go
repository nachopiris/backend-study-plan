package coupling

import (
	"math"
	"testing"
)

func assertMetrics(t *testing.T, got map[string]Metrics, pkg string, wantCa, wantCe int, wantI float64) {
	t.Helper()
	m, ok := got[pkg]
	if !ok {
		t.Fatalf("missing Metrics entry for package %q", pkg)
	}
	if m.Ca != wantCa {
		t.Errorf("%s: Ca = %d, want %d", pkg, m.Ca, wantCa)
	}
	if m.Ce != wantCe {
		t.Errorf("%s: Ce = %d, want %d", pkg, m.Ce, wantCe)
	}
	if math.Abs(m.Instability-wantI) > 1e-9 {
		t.Errorf("%s: Instability = %v, want %v", pkg, m.Instability, wantI)
	}
}

func TestCoupling_LinearChain(t *testing.T) {
	// orders -> inventory -> stdlib-like leaf. A straight line: the top of
	// the chain is maximally unstable, the leaf is maximally stable, the
	// middle is balanced.
	graph := map[string][]string{
		"orders":    {"inventory"},
		"inventory": {"leaf"},
	}

	got := Coupling(graph)

	assertMetrics(t, got, "orders", 0, 1, 1.0)
	assertMetrics(t, got, "inventory", 1, 1, 0.5)
	assertMetrics(t, got, "leaf", 1, 0, 0.0)
}

func TestCoupling_SharedDependencyFanIn(t *testing.T) {
	// Both orders and notifications depend on a shared "events" package.
	// events itself depends on nothing: it should read as maximally
	// stable (high Ca, Ce == 0, Instability 0).
	graph := map[string][]string{
		"orders":        {"events"},
		"notifications": {"events"},
	}

	got := Coupling(graph)

	assertMetrics(t, got, "events", 2, 0, 0.0)
	assertMetrics(t, got, "orders", 0, 1, 1.0)
	assertMetrics(t, got, "notifications", 0, 1, 1.0)
}

func TestCoupling_DependencyWithoutOwnGraphEntry(t *testing.T) {
	// "config" is depended upon but never appears as a key: it must still
	// get a Metrics entry, with Ce == 0 (no outgoing edges are known for
	// it) and Ca counting its one incoming edge.
	graph := map[string][]string{
		"orders": {"config"},
	}

	got := Coupling(graph)

	if len(got) != 2 {
		t.Fatalf("expected 2 entries (orders, config), got %d: %v", len(got), got)
	}
	assertMetrics(t, got, "config", 1, 0, 0.0)
	assertMetrics(t, got, "orders", 0, 1, 1.0)
}

func TestCoupling_IsolatedNodeHasZeroInstabilityByConvention(t *testing.T) {
	graph := map[string][]string{
		"standalone": {},
	}

	got := Coupling(graph)

	assertMetrics(t, got, "standalone", 0, 0, 0.0)
}

func TestCoupling_DuplicateDependenciesCountOnce(t *testing.T) {
	graph := map[string][]string{
		"orders": {"inventory", "inventory", "payments"},
	}

	got := Coupling(graph)

	assertMetrics(t, got, "orders", 0, 2, 1.0)
}

func TestCoupling_SelfImportDoesNotInflateCe(t *testing.T) {
	graph := map[string][]string{
		"orders": {"orders", "payments"},
	}

	got := Coupling(graph)

	assertMetrics(t, got, "orders", 0, 1, 1.0)
}

func TestCoupling_DoesNotMutateInput(t *testing.T) {
	graph := map[string][]string{
		"orders": {"inventory"},
	}

	_ = Coupling(graph)

	if len(graph) != 1 || len(graph["orders"]) != 1 || graph["orders"][0] != "inventory" {
		t.Fatalf("Coupling mutated its input graph: %v", graph)
	}
}
