// Package slo implements, by hand, the small set of arithmetic behind SLO-based alerting as
// described in the Google SRE Book (chapter 4, "Service Level Objectives") and the SRE Workbook
// (chapter 5, "Alerting on SLOs"): how big an error budget is, how much of it is left, how fast
// you are burning it, and when a burn rate is serious enough to wake someone up.
//
// No external dependencies: only the Go standard library. This package has no HTTP, no
// Prometheus client, and no I/O — it is pure arithmetic, meant to be the part of an alerting
// pipeline you can unit test completely deterministically. template/prometheus/slo-rules.yml
// expresses the SAME ideas as PromQL recording and alerting rules, for comparison.
package slo

import "time"

// ErrorBudget returns how much "bad" time is allowed within window before slo is violated.
//
// Contract:
//   - slo is a target expressed as a fraction, not a percentage (0.999 for "99.9%", not 99.9).
//   - The result is window scaled by (1 - slo): for example, a 99.9% SLO over a 30-day window
//     allows 30 days * 0.1% = 43m12s of bad time — the SRE Workbook's own worked example.
//
// Not implemented yet: always returns 0.
func ErrorBudget(slo float64, window time.Duration) time.Duration {
	return 0
}

// BudgetRemaining returns the fraction of the error budget still unspent, given good events out
// of total events observed so far, for a target slo.
//
// Contract:
//   - Let errorRate = (total - good) / total (the fraction of events that did NOT meet the SLI),
//     and let allowedErrorRate = 1 - slo (the sustainable error rate at exactly the target SLO).
//   - BudgetRemaining returns 1 - (errorRate / allowedErrorRate).
//   - A result of 1.0 means no budget spent yet (errorRate is 0); 0.0 means the budget is exactly
//     exhausted; a NEGATIVE result means the budget has been overspent (errorRate exceeds
//     allowedErrorRate) — this is a valid, meaningful result, not an error condition to clamp
//     away: callers need to know HOW far over budget they are, not just that they are over.
//   - total == 0 is a degenerate case with no observations yet: return 1.0 (full budget,
//     nothing spent), not a division by zero.
//
// Not implemented yet: always returns 0.
func BudgetRemaining(slo float64, good, total int64) float64 {
	return 0
}

// BurnRate returns how many times faster than sustainable the error budget is being consumed,
// given an observed errorRate (a fraction, not a percentage) and a target slo.
//
// Contract:
//   - BurnRate returns errorRate / (1 - slo). A burn rate of 1.0 means "consuming the budget at
//     exactly the sustainable rate" (the whole budget would last exactly the SLO's full
//     compliance window, e.g. 30 days). A burn rate of 14.4 means "consuming the budget 14.4x
//     faster than sustainable" — the SRE Workbook's own worked example for a fast-burn page
//     alert (see template/prometheus/slo-rules.yml): at that rate, a 30-day budget is exhausted
//     in 30/14.4 ≈ 2.08 days, or equivalently 2% of it is consumed in a single hour.
//
// Not implemented yet: always returns 0.
func BurnRate(errorRate, slo float64) float64 {
	return 0
}

// ShouldPage implements the SRE Workbook's multiwindow, multi-burn-rate condition: page only when
// BOTH a short window and a long window show the budget burning at or above threshold.
//
// Contract:
//   - Returns true only if shortWindowBurn >= threshold AND longWindowBurn >= threshold.
//   - The long window (for example 1h) confirms the burn is serious and sustained, not a single
//     noisy scrape; the short window (for example 5m) confirms the problem is STILL happening
//     right now, so the alert clears quickly once the issue is fixed instead of continuing to
//     page on a long window that includes stale data. Neither window alone is sufficient: see
//     the reflection questions in the README for why.
//
// Not implemented yet: always returns false.
func ShouldPage(shortWindowBurn, longWindowBurn, threshold float64) bool {
	return false
}
