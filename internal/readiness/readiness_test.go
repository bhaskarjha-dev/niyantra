package readiness

import (
	"testing"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/client"
)

func TestCalculate_NilSnapshots(t *testing.T) {
	result := Calculate(nil, 0.0)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil input, got %d", len(result))
	}
}

func TestCalculate_EmptySlice(t *testing.T) {
	result := Calculate([]*client.Snapshot{}, 0.0)
	if len(result) != 0 {
		t.Errorf("expected empty result for empty input, got %d", len(result))
	}
}

func TestCalculate_SkipsNilEntries(t *testing.T) {
	result := Calculate([]*client.Snapshot{nil, nil}, 0.0)
	if len(result) != 0 {
		t.Errorf("expected empty result for all-nil input, got %d", len(result))
	}
}

func TestCalculate_SingleAccount(t *testing.T) {
	now := time.Now()
	resetTime := now.Add(2 * time.Hour)

	snap := &client.Snapshot{
		AccountID:  1,
		Email:      "test@example.com",
		PlanName:   "Pro",
		CapturedAt: now.Add(-5 * time.Minute),
		Models: []client.ModelQuota{
			{Label: "Claude Sonnet", RemainingFraction: 0.8, RemainingPercent: 80, ResetTime: &resetTime},
		},
	}

	result := Calculate([]*client.Snapshot{snap}, 0.0)
	if len(result) != 1 {
		t.Fatalf("expected 1 account, got %d", len(result))
	}

	ar := result[0]
	if ar.Email != "test@example.com" {
		t.Errorf("email = %q, want %q", ar.Email, "test@example.com")
	}
	if ar.PlanName != "Pro" {
		t.Errorf("plan = %q, want %q", ar.PlanName, "Pro")
	}
	if !ar.IsReady {
		t.Error("expected account to be ready with 80% remaining")
	}
	if len(ar.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(ar.Models))
	}
	if ar.Models[0].RemainingPercent != 80 {
		t.Errorf("model remaining = %f, want 80", ar.Models[0].RemainingPercent)
	}
}

func TestCalculate_ExhaustedAccount(t *testing.T) {
	now := time.Now()
	resetTime := now.Add(1 * time.Hour)

	snap := &client.Snapshot{
		AccountID:  1,
		Email:      "exhausted@example.com",
		PlanName:   "Free",
		CapturedAt: now,
		Models: []client.ModelQuota{
			{Label: "Claude Sonnet", RemainingFraction: 0.0, RemainingPercent: 0, IsExhausted: true, ResetTime: &resetTime},
		},
	}

	result := Calculate([]*client.Snapshot{snap}, 0.0)
	if len(result) != 1 {
		t.Fatalf("expected 1 account, got %d", len(result))
	}

	// With threshold 0.0, even 0% remaining counts as "not ready" because IsExhausted is true
	// Actually, the readiness check is: g.RemainingFraction > threshold
	// 0.0 > 0.0 is false, so IsReady should be false
	if result[0].IsReady {
		t.Error("expected exhausted account to not be ready at threshold 0.0")
	}
}

func TestCalculate_ThresholdFiltering(t *testing.T) {
	now := time.Now()
	resetTime := now.Add(2 * time.Hour)

	snap := &client.Snapshot{
		AccountID:  1,
		Email:      "threshold@example.com",
		PlanName:   "Pro",
		CapturedAt: now,
		Models: []client.ModelQuota{
			{Label: "Claude Sonnet", RemainingFraction: 0.15, RemainingPercent: 15, ResetTime: &resetTime},
		},
	}

	// At threshold 0.0, 15% should be ready (0.15 > 0.0)
	result := Calculate([]*client.Snapshot{snap}, 0.0)
	if !result[0].IsReady {
		t.Error("expected ready at threshold 0.0 with 15% remaining")
	}

	// At threshold 0.2, 15% should NOT be ready (0.15 is NOT > 0.2)
	result = Calculate([]*client.Snapshot{snap}, 0.2)
	if result[0].IsReady {
		t.Error("expected NOT ready at threshold 0.2 with 15% remaining")
	}
}

func TestCalculate_SortingReadyFirst(t *testing.T) {
	now := time.Now()
	resetTime := now.Add(2 * time.Hour)

	exhausted := &client.Snapshot{
		AccountID:  1,
		Email:      "exhausted@example.com",
		CapturedAt: now,
		Models: []client.ModelQuota{
			{Label: "Claude Sonnet", RemainingFraction: 0.0, IsExhausted: true, ResetTime: &resetTime},
		},
	}
	ready := &client.Snapshot{
		AccountID:  2,
		Email:      "ready@example.com",
		CapturedAt: now.Add(-1 * time.Hour),
		Models: []client.ModelQuota{
			{Label: "Claude Sonnet", RemainingFraction: 0.9, RemainingPercent: 90, ResetTime: &resetTime},
		},
	}

	// Pass exhausted first — result should sort ready first
	result := Calculate([]*client.Snapshot{exhausted, ready}, 0.0)
	if len(result) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(result))
	}
	if result[0].Email != "ready@example.com" {
		t.Errorf("expected ready account first, got %q", result[0].Email)
	}
}

func TestCalculate_MultipleModelsGrouping(t *testing.T) {
	now := time.Now()
	resetTime := now.Add(3 * time.Hour)

	snap := &client.Snapshot{
		AccountID:  1,
		Email:      "multi@example.com",
		CapturedAt: now,
		Models: []client.ModelQuota{
			{Label: "Claude Sonnet 4.6 (Thinking)", RemainingFraction: 0.5, RemainingPercent: 50, ResetTime: &resetTime},
			{Label: "GPT-4.1", RemainingFraction: 0.6, RemainingPercent: 60, ResetTime: &resetTime},
			{Label: "Gemini 2.5 Pro", RemainingFraction: 0.8, RemainingPercent: 80, ResetTime: &resetTime},
			{Label: "Gemini 2.5 Flash", RemainingFraction: 0.9, RemainingPercent: 90, ResetTime: &resetTime},
		},
	}

	result := Calculate([]*client.Snapshot{snap}, 0.0)
	if len(result) != 1 {
		t.Fatalf("expected 1 account, got %d", len(result))
	}
	if len(result[0].Models) != 6 {
		t.Errorf("expected 6 models (2 non-Gemini + 4 upgraded Gemini), got %d", len(result[0].Models))
	}
	// Should have groups (exact count depends on GroupModels logic)
	if len(result[0].Groups) == 0 {
		t.Error("expected at least 1 group")
	}
}

func TestFormatStaleness(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{10 * time.Second, "just now"},
		{1 * time.Minute, "1 min ago"},
		{5 * time.Minute, "5 min ago"},
		{1 * time.Hour, "1 hour ago"},
		{3 * time.Hour, "3h ago"},
		{24 * time.Hour, "1 day ago"},
		{72 * time.Hour, "3d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatStaleness(tt.duration)
			if got != tt.want {
				t.Errorf("formatStaleness(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

// TestVeryStaleSnapshotKeepsOriginalValues verifies that a very stale snapshot
// (>24h since reset) keeps original values with very_low confidence. The data is
// too stale to produce a useful 100% estimate — the user may have consumed quota.
func TestStaleSnapshotInfersReset(t *testing.T) {
	pastReset := time.Now().Add(-10 * 24 * time.Hour).Add(5 * time.Hour) // 10 days ago + 5h
	snap := &client.Snapshot{
		AccountID:  1,
		Email:      "stale@example.com",
		PlanName:   "Pro",
		CapturedAt: time.Now().Add(-10 * 24 * time.Hour), // 10 days ago
		Models: []client.ModelQuota{
			{
				ModelID:           "model-1",
				Label:             "Claude Sonnet",
				RemainingFraction: 0.0,
				RemainingPercent:  0,
				IsExhausted:       true,
				ResetTime:         &pastReset,
			},
		},
	}

	result := Calculate([]*client.Snapshot{snap}, 0.0)
	if len(result) != 1 {
		t.Fatalf("expected 1 account, got %d", len(result))
	}

	ar := result[0]
	// Q3: StalenessLabel is now always time-ago format (e.g. "10d ago"), not "Stale"
	if ar.StalenessLabel != "10d ago" {
		t.Errorf("staleness label = %q, want %q", ar.StalenessLabel, "10d ago")
	}
	if len(ar.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(ar.Models))
	}
	// >24h since reset: keep original values (0%), mark as very_low confidence
	if ar.Models[0].RemainingPercent != 0 {
		t.Errorf("very stale model remaining = %f, want 0 (>24h, keep original)", ar.Models[0].RemainingPercent)
	}
	if !ar.Models[0].IsExhausted {
		t.Error("very stale model should remain exhausted (>24h, keep original)")
	}
	if !ar.Models[0].IsEstimated || ar.Models[0].Basis != "snapshot_too_stale" {
		t.Fatalf("very stale model should have basis=snapshot_too_stale, got %+v", ar.Models[0])
	}
	if ar.Models[0].Confidence != "very_low" {
		t.Fatalf("very stale model confidence = %q, want very_low", ar.Models[0].Confidence)
	}
}

// TestFreshSnapshotUnchanged verifies that a fresh snapshot (5 min ago)
// with 0% remaining stays at 0% — it's genuinely exhausted, not stale.
func TestFreshSnapshotUnchanged(t *testing.T) {
	futureReset := time.Now().Add(2 * time.Hour) // reset in 2 hours
	snap := &client.Snapshot{
		AccountID:  1,
		Email:      "fresh@example.com",
		PlanName:   "Pro",
		CapturedAt: time.Now().Add(-5 * time.Minute), // 5 min ago
		Models: []client.ModelQuota{
			{
				ModelID:           "model-1",
				Label:             "Claude Sonnet",
				RemainingFraction: 0.0,
				RemainingPercent:  0,
				IsExhausted:       true,
				ResetTime:         &futureReset,
			},
		},
	}

	result := Calculate([]*client.Snapshot{snap}, 0.0)
	if len(result) != 1 {
		t.Fatalf("expected 1 account, got %d", len(result))
	}

	ar := result[0]
	if ar.StalenessLabel == "Stale" {
		t.Error("fresh snapshot should NOT be labeled Stale")
	}
	if len(ar.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(ar.Models))
	}
	if ar.Models[0].RemainingPercent != 0 {
		t.Errorf("fresh model remaining = %f, want 0 (genuinely exhausted)", ar.Models[0].RemainingPercent)
	}
	if !ar.Models[0].IsExhausted {
		t.Error("fresh model should be marked exhausted")
	}
}

// TestResetPassedInfersFullQuota verifies that a recently-reset model (1 hour ago)
// is estimated at 100% with medium confidence. This matches Google's sprint reset
// behavior where quota restores to full at the reset boundary.
func TestResetPassedDoesNotInferRefillBeforeStalenessThreshold(t *testing.T) {
	now := time.Now()
	pastReset := now.Add(-1 * time.Hour)
	snap := &client.Snapshot{
		AccountID:  1,
		Email:      "recent-reset@example.com",
		PlanName:   "Pro",
		CapturedAt: now.Add(-2 * time.Hour),
		Models: []client.ModelQuota{
			{
				ModelID:           "claude-sonnet",
				Label:             "Claude Sonnet",
				RemainingFraction: 0,
				RemainingPercent:  0,
				IsExhausted:       true,
				ResetTime:         &pastReset,
			},
		},
	}

	result := Calculate([]*client.Snapshot{snap}, 0.0)
	if len(result) != 1 {
		t.Fatalf("expected 1 account, got %d", len(result))
	}
	// Sprint reset 1 hour ago: estimate 100% availability (medium confidence)
	if got := result[0].Models[0].RemainingPercent; got != 100 {
		t.Fatalf("remaining = %.0f, want 100 (sprint reset 1h ago should estimate full quota)", got)
	}
	if result[0].Models[0].IsExhausted {
		t.Fatal("model should NOT be exhausted after sprint reset estimation")
	}
	if !result[0].Models[0].IsEstimated {
		t.Fatal("model should be flagged as estimated")
	}
	if result[0].Models[0].Basis != "sprint_reset_assumed" {
		t.Fatalf("basis = %q, want sprint_reset_assumed", result[0].Models[0].Basis)
	}
	if result[0].Models[0].Confidence != "medium" {
		t.Fatalf("confidence = %q, want medium", result[0].Models[0].Confidence)
	}
}

// TestRecentResetHighConfidence verifies that a very recent reset (<30 min)
// produces a high-confidence 100% estimate.
func TestRecentResetHighConfidence(t *testing.T) {
	now := time.Now()
	pastReset := now.Add(-10 * time.Minute) // reset 10 minutes ago
	snap := &client.Snapshot{
		AccountID:  1,
		Email:      "just-reset@example.com",
		PlanName:   "Pro",
		CapturedAt: now.Add(-1 * time.Hour),
		Models: []client.ModelQuota{
			{
				ModelID:           "claude-sonnet",
				Label:             "Claude Sonnet",
				RemainingFraction: 0,
				RemainingPercent:  0,
				IsExhausted:       true,
				ResetTime:         &pastReset,
			},
		},
	}

	result := Calculate([]*client.Snapshot{snap}, 0.0)
	if len(result) != 1 {
		t.Fatalf("expected 1 account, got %d", len(result))
	}
	model := result[0].Models[0]
	if model.RemainingPercent != 100 {
		t.Fatalf("remaining = %.0f, want 100 (sprint reset 10min ago)", model.RemainingPercent)
	}
	if model.IsExhausted {
		t.Fatal("model should NOT be exhausted after recent reset")
	}
	if model.Basis != "sprint_reset_recent" {
		t.Fatalf("basis = %q, want sprint_reset_recent", model.Basis)
	}
	if model.Confidence != "high" {
		t.Fatalf("confidence = %q, want high (reset was <30 min ago)", model.Confidence)
	}
}

func TestExhaustedModelMakesGroupNotReadyEvenWhenAveragePositive(t *testing.T) {
	now := time.Now()
	resetTime := now.Add(2 * time.Hour)
	snap := &client.Snapshot{
		AccountID:  1,
		Email:      "mixed@example.com",
		PlanName:   "Pro",
		CapturedAt: now,
		Models: []client.ModelQuota{
			{
				ModelID:           "claude-sonnet",
				Label:             "Claude Sonnet",
				RemainingFraction: 0,
				RemainingPercent:  0,
				IsExhausted:       true,
				ResetTime:         &resetTime,
			},
			{
				ModelID:           "gpt-4.1",
				Label:             "GPT-4.1",
				RemainingFraction: 0.8,
				RemainingPercent:  80,
				ResetTime:         &resetTime,
			},
		},
	}

	result := Calculate([]*client.Snapshot{snap}, 0.0)
	if len(result) != 1 {
		t.Fatalf("expected 1 account, got %d", len(result))
	}
	if result[0].IsReady {
		t.Fatal("account should not be ready when a model in its only group is exhausted")
	}
	if len(result[0].Groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(result[0].Groups))
	}
	group := result[0].Groups[0]
	if group.GroupKey != client.GroupClaudeGPT {
		t.Fatalf("group = %q, want %q", group.GroupKey, client.GroupClaudeGPT)
	}
	if !group.IsExhausted {
		t.Fatal("group should be exhausted when any member model is exhausted")
	}
	if group.IsReady {
		t.Fatal("group should not be ready when any member model is exhausted")
	}
	if group.RemainingPercent != 40 {
		t.Fatalf("group remaining = %.0f, want 40", group.RemainingPercent)
	}
}

func TestSnapshotWithNoModelsIsNotReady(t *testing.T) {
	result := Calculate([]*client.Snapshot{{
		AccountID:  1,
		Email:      "empty@example.com",
		CapturedAt: time.Now(),
	}}, 0.0)

	if len(result) != 1 {
		t.Fatalf("expected 1 account, got %d", len(result))
	}
	if result[0].IsReady {
		t.Fatal("account with no model quota data should not be ready")
	}
}
