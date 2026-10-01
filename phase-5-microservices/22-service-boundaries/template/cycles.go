package coupling

// Cycles returns every elementary cycle present in graph, where graph maps a
// package's import path to the import paths it directly depends on. An
// elementary cycle is a path of distinct packages p0 -> p1 -> ... -> pk that
// returns to p0 (pk depends on p0), without revisiting any package before
// closing the loop.
//
// A real Go build never has import cycles (the compiler rejects them), but
// this function also runs against OTHER dependency graphs you may build by
// hand from code that is not yet a compiled Go program — for example, a
// proposed bounded-context map before it is implemented, or a dependency
// graph extracted from a language that does tolerate cycles. Finding them
// here, before extraction, is cheaper than discovering them as a deploy-time
// deadlock between two services that each wait on the other to come up.
//
// Contract:
//   - Each returned cycle is a []string of package names in traversal
//     order, WITHOUT repeating the start package at the end (a cycle
//     a -> b -> a is {"a", "b"}, never {"a", "b", "a"}).
//   - A package that imports itself directly (a -> a) is a cycle of length
//     1: {"a"}.
//   - The order of the returned cycles, and which package a given cycle
//     starts from, are NOT specified by this contract. Tests normalize
//     before comparing — see the normalization helper in cycles_test.go.
//   - Each elementary cycle is reported once, not once per rotation and not
//     once per package it visits.
//   - If graph has no cycles, Cycles returns an empty (nil is fine) slice.
//
// Cycles never mutates graph.
func Cycles(graph map[string][]string) [][]string {
	return nil
}
