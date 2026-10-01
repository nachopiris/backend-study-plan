// Package coupling computes structural coupling metrics (and dependency
// cycles) from a package dependency graph, the same shape `go list -deps`
// produces: a map from a package's import path to the import paths it
// depends on directly.
//
// These metrics (afferent/efferent coupling, instability) are Robert C.
// Martin's package-level stability metrics, applied here to Go import
// graphs instead of to his original context (C++/Java package design) — the
// idea generalizes directly: a package many others depend on should change
// rarely (it is costly to change), while a package that depends on many
// others but that nothing depends on can change freely.
package coupling

// Metrics holds the coupling metrics for one package node in a dependency
// graph.
type Metrics struct {
	// Ca (afferent coupling) is the number of OTHER packages in the graph
	// that directly depend on this one. High Ca means many things break if
	// this package's public contract changes.
	Ca int
	// Ce (efferent coupling) is the number of packages this one directly
	// depends on. High Ce means this package is exposed to many reasons to
	// change that are outside its own control.
	Ce int
	// Instability is I = Ce / (Ca + Ce), in the range [0, 1]. I == 0 means
	// maximally stable (only depended upon, never depends on anything); I
	// == 1 means maximally unstable (depends on others, nothing depends on
	// it). A package with Ca == 0 and Ce == 0 (no edges at all) has
	// Instability == 0 by convention.
	Instability float64
}

// Coupling computes Ca, Ce, and Instability for every package that appears
// in graph, either as a key or as a dependency named in some key's value
// slice.
//
// Contract:
//   - The returned map has one entry for every package name that appears
//     anywhere in graph: as a key, or inside any value slice. A package
//     that is only ever a dependency (never a key of graph) still gets a
//     Metrics entry, with Ce == 0 (graph has no edges FROM it, because it
//     has no entry) and whatever Ca its incoming edges add up to.
//   - Ce for package p is the number of distinct packages in graph[p],
//     excluding p itself (a direct self-import, p importing p, must not
//     inflate Ce; see Cycles for how self-imports are reported instead).
//   - Ca for package p is the number of distinct OTHER packages q (q != p)
//     such that p appears in graph[q].
//   - Duplicate entries within a single graph[p] slice count once.
//   - Instability follows the formula on Metrics, including the Ca == 0 &&
//     Ce == 0 convention.
//
// Coupling never mutates graph.
func Coupling(graph map[string][]string) map[string]Metrics {
	return nil
}
