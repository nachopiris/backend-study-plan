package slo

import (
	"math"
	"testing"
	"time"
)

const float64EqualityThreshold = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}

func TestErrorBudget(t *testing.T) {
	tests := []struct {
		name   string
		slo    float64
		window time.Duration
		want   time.Duration
	}{
		{
			name:   "99.9% over 30 days: the SRE Workbook's own worked example",
			slo:    0.999,
			window: 30 * 24 * time.Hour,
			want:   43*time.Minute + 12*time.Second,
		},
		{
			name:   "99% over 30 days: a full order of magnitude looser",
			slo:    0.99,
			window: 30 * 24 * time.Hour,
			want:   7*time.Hour + 12*time.Minute,
		},
		{
			name:   "100% SLO allows zero budget, regardless of window",
			slo:    1.0,
			window: 30 * 24 * time.Hour,
			want:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ErrorBudget(tt.slo, tt.window); got != tt.want {
				t.Errorf("ErrorBudget(%v, %v) = %v, want %v", tt.slo, tt.window, got, tt.want)
			}
		})
	}
}

func TestBudgetRemaining(t *testing.T) {
	tests := []struct {
		name  string
		slo   float64
		good  int64
		total int64
		want  float64
	}{
		{
			name:  "no bad events yet: full budget remaining",
			slo:   0.999,
			good:  1_000_000,
			total: 1_000_000,
			want:  1.0,
		},
		{
			name:  "half the budget spent",
			slo:   0.999,
			good:  999_500,
			total: 1_000_000, // errorRate = 0.0005, allowed = 0.001 -> 0.5 remaining
			want:  0.5,
		},
		{
			name:  "exactly at budget: zero remaining",
			slo:   0.999,
			good:  999_000,
			total: 1_000_000, // errorRate = 0.001 = allowed -> 0 remaining
			want:  0.0,
		},
		{
			name:  "over budget: negative remaining signals HOW far over, not just that it's over",
			slo:   0.999,
			good:  998_000,
			total: 1_000_000, // errorRate = 0.002, double the allowed rate -> -1.0 remaining
			want:  -1.0,
		},
		{
			name:  "no observations yet: degenerate case must not divide by zero",
			slo:   0.999,
			good:  0,
			total: 0,
			want:  1.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BudgetRemaining(tt.slo, tt.good, tt.total)
			if !almostEqual(got, tt.want) {
				t.Errorf("BudgetRemaining(%v, %d, %d) = %v, want %v", tt.slo, tt.good, tt.total, got, tt.want)
			}
		})
	}
}

func TestBurnRate(t *testing.T) {
	tests := []struct {
		name      string
		errorRate float64
		slo       float64
		want      float64
	}{
		{
			name:      "burning at exactly the sustainable rate",
			errorRate: 0.001,
			slo:       0.999,
			want:      1.0,
		},
		{
			name:      "14.4x fast burn: the SRE Workbook's own page-threshold example",
			errorRate: 0.0144,
			slo:       0.999,
			want:      14.4,
		},
		{
			name:      "no errors at all: zero burn",
			errorRate: 0,
			slo:       0.999,
			want:      0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BurnRate(tt.errorRate, tt.slo)
			if !almostEqual(got, tt.want) {
				t.Errorf("BurnRate(%v, %v) = %v, want %v", tt.errorRate, tt.slo, got, tt.want)
			}
		})
	}
}

func TestShouldPage(t *testing.T) {
	const threshold = 14.4
	tests := []struct {
		name            string
		shortWindowBurn float64
		longWindowBurn  float64
		want            bool
	}{
		{
			name:            "both windows sustain a fast burn: page",
			shortWindowBurn: 14.4,
			longWindowBurn:  14.4,
			want:            true,
		},
		{
			name:            "long window burned fast but short window already recovered: do not page",
			shortWindowBurn: 1.0,
			longWindowBurn:  14.4,
			want:            false,
		},
		{
			name:            "short window spikes but long window never sustained it: do not page (noise)",
			shortWindowBurn: 14.4,
			longWindowBurn:  1.0,
			want:            false,
		},
		{
			name:            "both well under threshold: do not page",
			shortWindowBurn: 1.0,
			longWindowBurn:  1.0,
			want:            false,
		},
		{
			name:            "both exactly at threshold: page (>=, not >)",
			shortWindowBurn: threshold,
			longWindowBurn:  threshold,
			want:            true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldPage(tt.shortWindowBurn, tt.longWindowBurn, threshold); got != tt.want {
				t.Errorf("ShouldPage(%v, %v, %v) = %v, want %v", tt.shortWindowBurn, tt.longWindowBurn, threshold, got, tt.want)
			}
		})
	}
}
