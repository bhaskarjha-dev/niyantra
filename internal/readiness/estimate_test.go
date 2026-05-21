package readiness

import (
	"testing"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/store"
)

// ═══════════════════════════════════════════════════════════════
//  Codex estimation tests
// ═══════════════════════════════════════════════════════════════

func TestEstimateCodexSnapshot_NilSafe(t *testing.T) {
	// Should not panic
	EstimateCodexSnapshot(nil, time.Now())
}

func TestEstimateCodexSnapshot_FiveHourResetRecent(t *testing.T) {
	resetTime := time.Now().Add(-10 * time.Minute) // Reset 10 min ago
	snap := &store.CodexSnapshot{
		FiveHourPct:   85.0, // Was 85% used
		FiveHourReset: &resetTime,
	}

	EstimateCodexSnapshot(snap, time.Now())

	if snap.FiveHourPct != 0 {
		t.Fatalf("fiveHourPct = %.0f, want 0 (recent reset should clear usage)", snap.FiveHourPct)
	}
}

func TestEstimateCodexSnapshot_FiveHourResetStale(t *testing.T) {
	resetTime := time.Now().Add(-3 * time.Hour) // Reset 3h ago
	snap := &store.CodexSnapshot{
		FiveHourPct:   85.0,
		FiveHourReset: &resetTime,
	}

	EstimateCodexSnapshot(snap, time.Now())

	if snap.FiveHourPct != 0 {
		t.Fatalf("fiveHourPct = %.0f, want 0 (within window, rolling recovery)", snap.FiveHourPct)
	}
}

func TestEstimateCodexSnapshot_FiveHourResetVeryStale(t *testing.T) {
	resetTime := time.Now().Add(-48 * time.Hour) // Reset 2 days ago
	snap := &store.CodexSnapshot{
		FiveHourPct:   85.0,
		FiveHourReset: &resetTime,
	}

	EstimateCodexSnapshot(snap, time.Now())

	// >24h — keep original value
	if snap.FiveHourPct != 85.0 {
		t.Fatalf("fiveHourPct = %.0f, want 85 (>24h, keep original)", snap.FiveHourPct)
	}
}

func TestEstimateCodexSnapshot_FiveHourNotElapsed(t *testing.T) {
	resetTime := time.Now().Add(2 * time.Hour) // Reset in 2h (future)
	snap := &store.CodexSnapshot{
		FiveHourPct:   50.0,
		FiveHourReset: &resetTime,
	}

	EstimateCodexSnapshot(snap, time.Now())

	// Not elapsed — keep original
	if snap.FiveHourPct != 50.0 {
		t.Fatalf("fiveHourPct = %.0f, want 50 (not yet reset)", snap.FiveHourPct)
	}
}

func TestEstimateCodexSnapshot_SevenDayReset(t *testing.T) {
	resetTime := time.Now().Add(-30 * time.Minute)
	sevenDayPct := 60.0
	snap := &store.CodexSnapshot{
		SevenDayPct:   &sevenDayPct,
		SevenDayReset: &resetTime,
	}

	EstimateCodexSnapshot(snap, time.Now())

	if snap.SevenDayPct == nil || *snap.SevenDayPct != 0 {
		t.Fatalf("sevenDayPct = %v, want 0 (recent weekly reset)", snap.SevenDayPct)
	}
}

// ═══════════════════════════════════════════════════════════════
//  Claude estimation tests
// ═══════════════════════════════════════════════════════════════

func TestEstimateClaudeSnapshot_NilSafe(t *testing.T) {
	EstimateClaudeSnapshot(nil, time.Now())
}

func TestEstimateClaudeSnapshot_FiveHourResetRecent(t *testing.T) {
	resetTime := time.Now().Add(-5 * time.Minute)
	snap := &store.ClaudeSnapshot{
		FiveHourPct:   90.0,
		FiveHourReset: &resetTime,
	}

	EstimateClaudeSnapshot(snap, time.Now())

	if snap.FiveHourPct != 0 {
		t.Fatalf("fiveHourPct = %.0f, want 0 (5h window fully restores on reset)", snap.FiveHourPct)
	}
}

func TestEstimateClaudeSnapshot_SevenDayResetIndependent(t *testing.T) {
	// 5h window is future (active), 7d window has elapsed
	fiveReset := time.Now().Add(2 * time.Hour)
	sevenReset := time.Now().Add(-30 * time.Minute)
	sevenDayPct := 75.0
	snap := &store.ClaudeSnapshot{
		FiveHourPct:   40.0,
		FiveHourReset: &fiveReset,
		SevenDayPct:   &sevenDayPct,
		SevenDayReset: &sevenReset,
	}

	EstimateClaudeSnapshot(snap, time.Now())

	// 5h should NOT be touched (future reset)
	if snap.FiveHourPct != 40.0 {
		t.Fatalf("fiveHourPct = %.0f, want 40 (not reset yet)", snap.FiveHourPct)
	}
	// 7d SHOULD be estimated to 0
	if snap.SevenDayPct == nil || *snap.SevenDayPct != 0 {
		t.Fatalf("sevenDayPct = %v, want 0 (weekly reset elapsed)", snap.SevenDayPct)
	}
}

// ═══════════════════════════════════════════════════════════════
//  Cursor estimation tests
// ═══════════════════════════════════════════════════════════════

func TestEstimateCursorSnapshot_NilSafe(t *testing.T) {
	EstimateCursorSnapshot(nil, time.Now())
}

func TestEstimateCursorSnapshot_CycleNotEnded(t *testing.T) {
	cycleEnd := time.Now().Add(10 * 24 * time.Hour).Format(time.RFC3339)
	snap := &store.CursorSnapshot{
		UsagePct:     65.0,
		RequestsUsed: 120,
		CycleEnd:     cycleEnd,
	}

	EstimateCursorSnapshot(snap, time.Now())

	if snap.UsagePct != 65.0 {
		t.Fatalf("usagePct = %.0f, want 65 (cycle not ended)", snap.UsagePct)
	}
}

func TestEstimateCursorSnapshot_CycleEnded(t *testing.T) {
	cycleEnd := time.Now().Add(-12 * time.Hour).Format(time.RFC3339)
	snap := &store.CursorSnapshot{
		UsagePct:     80.0,
		RequestsUsed: 200,
		UsedCents:    1500,
		AutoPct:      30.0,
		APIPct:       10.0,
		CycleEnd:     cycleEnd,
	}

	EstimateCursorSnapshot(snap, time.Now())

	if snap.UsagePct != 0 {
		t.Fatalf("usagePct = %.0f, want 0 (new billing cycle)", snap.UsagePct)
	}
	if snap.RequestsUsed != 0 {
		t.Fatalf("requestsUsed = %d, want 0", snap.RequestsUsed)
	}
	if snap.UsedCents != 0 {
		t.Fatalf("usedCents = %d, want 0", snap.UsedCents)
	}
}

func TestEstimateCursorSnapshot_CycleEndedTooLong(t *testing.T) {
	cycleEnd := time.Now().Add(-5 * 24 * time.Hour).Format(time.RFC3339)
	snap := &store.CursorSnapshot{
		UsagePct:     80.0,
		RequestsUsed: 200,
		CycleEnd:     cycleEnd,
	}

	EstimateCursorSnapshot(snap, time.Now())

	// >48h after cycle end — keep original
	if snap.UsagePct != 80.0 {
		t.Fatalf("usagePct = %.0f, want 80 (>48h, keep original)", snap.UsagePct)
	}
}

func TestEstimateCursorSnapshot_NoCycleEnd(t *testing.T) {
	snap := &store.CursorSnapshot{
		UsagePct: 50.0,
		CycleEnd: "",
	}

	EstimateCursorSnapshot(snap, time.Now())

	if snap.UsagePct != 50.0 {
		t.Fatalf("usagePct = %.0f, want 50 (no cycle end data)", snap.UsagePct)
	}
}

// ═══════════════════════════════════════════════════════════════
//  Copilot estimation tests
// ═══════════════════════════════════════════════════════════════

func TestEstimateCopilotSnapshot_NilSafe(t *testing.T) {
	EstimateCopilotSnapshot(nil, time.Now())
}

func TestEstimateCopilotSnapshot_SameMonth(t *testing.T) {
	snap := &store.CopilotSnapshot{
		PremiumPct: 45.0,
		ChatPct:    20.0,
		CapturedAt: time.Now().Add(-2 * time.Hour),
	}

	EstimateCopilotSnapshot(snap, time.Now())

	if snap.PremiumPct != 45.0 {
		t.Fatalf("premiumPct = %.0f, want 45 (same month)", snap.PremiumPct)
	}
}

func TestEstimateCopilotSnapshot_PreviousMonth(t *testing.T) {
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC) // May 3rd
	snap := &store.CopilotSnapshot{
		PremiumPct: 80.0,
		ChatPct:    60.0,
		CapturedAt: time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC), // April 28th
	}

	EstimateCopilotSnapshot(snap, now)

	// May 3rd is within first 7 days of month → estimate 0%
	if snap.PremiumPct != 0 {
		t.Fatalf("premiumPct = %.0f, want 0 (new month, early in cycle)", snap.PremiumPct)
	}
	if snap.ChatPct != 0 {
		t.Fatalf("chatPct = %.0f, want 0 (new month, early in cycle)", snap.ChatPct)
	}
}

func TestEstimateCopilotSnapshot_PreviousMonthLate(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC) // May 20th
	snap := &store.CopilotSnapshot{
		PremiumPct: 80.0,
		ChatPct:    60.0,
		CapturedAt: time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC), // April 15th
	}

	EstimateCopilotSnapshot(snap, now)

	// May 20th is past first week — keep original (too uncertain)
	if snap.PremiumPct != 80.0 {
		t.Fatalf("premiumPct = %.0f, want 80 (past first week, keep original)", snap.PremiumPct)
	}
}
