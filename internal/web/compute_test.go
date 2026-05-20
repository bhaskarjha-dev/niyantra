package web

import (
	"testing"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/store"
)

// TestComputeSimpleRateSkipsResetIntervals verifies that intervals where
// remaining percentage INCREASES (indicating a quota reset) are excluded
// from the rate calculation instead of diluting the average.
func TestComputeSimpleRateSkipsResetIntervals(t *testing.T) {
	now := time.Now()

	// Scenario: 3 snapshots. First two show usage (100% → 80%), then a reset
	// back to 100%. The rate should only reflect the usage interval.
	snaps := []store.ClaudeSnapshot{
		{CapturedAt: now.Add(-2 * time.Hour), FiveHourPct: 0},   // 0% used = 100% remaining
		{CapturedAt: now.Add(-1 * time.Hour), FiveHourPct: 20},  // 20% used = 80% remaining
		{CapturedAt: now, FiveHourPct: 0},                        // 0% used = 100% remaining (RESET)
	}

	// Extractor: FiveHourPct is % used, we need % remaining
	extractor := func(s store.ClaudeSnapshot) float64 {
		return 100 - s.FiveHourPct
	}

	rate, remaining := computeSimpleRate(snaps, extractor)

	// Rate should reflect only the 100→80 interval (20% over 1 hour = 20%/hr)
	// NOT be diluted by the reset interval
	if rate <= 0 {
		t.Fatalf("expected positive rate, got %f", rate)
	}

	// Remaining should be 100% (post-reset)
	if remaining != 100 {
		t.Fatalf("expected remaining = 100, got %f", remaining)
	}

	// The rate should be approximately 20%/hr (not ~10%/hr which the diluted version produced)
	if rate < 15 || rate > 25 {
		t.Fatalf("expected rate ~20%%/hr, got %f", rate)
	}
}

// TestComputeSimpleRateAllResets verifies that if all intervals are resets,
// the function returns zero rate (no usage data).
func TestComputeSimpleRateAllResets(t *testing.T) {
	now := time.Now()

	snaps := []store.ClaudeSnapshot{
		{CapturedAt: now.Add(-2 * time.Hour), FiveHourPct: 50}, // 50% remaining
		{CapturedAt: now.Add(-1 * time.Hour), FiveHourPct: 30}, // 70% remaining (RESET)
		{CapturedAt: now, FiveHourPct: 10},                      // 90% remaining (RESET)
	}

	extractor := func(s store.ClaudeSnapshot) float64 {
		return 100 - s.FiveHourPct
	}

	rate, _ := computeSimpleRate(snaps, extractor)
	if rate != 0 {
		t.Fatalf("expected rate = 0 when all intervals are resets, got %f", rate)
	}
}

// TestComputeSimpleRateTooFewSnapshots verifies minimum data requirement.
func TestComputeSimpleRateTooFewSnapshots(t *testing.T) {
	snaps := []store.ClaudeSnapshot{
		{CapturedAt: time.Now(), FiveHourPct: 50},
	}

	extractor := func(s store.ClaudeSnapshot) float64 {
		return 100 - s.FiveHourPct
	}

	rate, remaining := computeSimpleRate(snaps, extractor)
	if rate != 0 || remaining != -1 {
		t.Fatalf("expected (0, -1) for single snapshot, got (%f, %f)", rate, remaining)
	}
}

// TestComputeCodexRateSkipsResets verifies the Codex rate engine has the same fix.
func TestComputeCodexRateSkipsResets(t *testing.T) {
	now := time.Now()

	snaps := []*store.CodexSnapshot{
		{CapturedAt: now.Add(-2 * time.Hour), FiveHourPct: 0},
		{CapturedAt: now.Add(-1 * time.Hour), FiveHourPct: 30},
		{CapturedAt: now, FiveHourPct: 0}, // RESET
	}

	extractor := func(s *store.CodexSnapshot) float64 {
		return 100 - s.FiveHourPct
	}

	rate, _ := computeCodexRate(snaps, extractor)
	if rate <= 0 {
		t.Fatalf("expected positive rate (excluding reset interval), got %f", rate)
	}

	// Should be ~30%/hr, not diluted
	if rate < 20 || rate > 40 {
		t.Fatalf("expected rate ~30%%/hr, got %f", rate)
	}
}

// TestTTXSeverityThresholds verifies the severity classification.
func TestTTXSeverityThresholds(t *testing.T) {
	tests := []struct {
		hours    float64
		expected string
	}{
		{-1, "critical"},
		{0, "critical"},
		{0.3, "critical"},
		{0.7, "warning"},
		{2.0, "caution"},
		{5.0, "safe"},
	}

	for _, tc := range tests {
		got := ttxSeverity(tc.hours)
		if got != tc.expected {
			t.Errorf("ttxSeverity(%f) = %q, want %q", tc.hours, got, tc.expected)
		}
	}
}
