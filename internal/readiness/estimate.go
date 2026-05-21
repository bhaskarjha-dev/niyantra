package readiness

import (
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/store"
)

// ── Provider-specific post-reset estimation ──
//
// Each provider has its own quota model and reset mechanics. This file
// contains estimation functions that enrich stale snapshots with computed
// values so the dashboard shows the most accurate picture between polls.
//
// Provider reset mechanics (from research):
//
//   Codex/ChatGPT: Continuous rolling window. Usage "drains" as the oldest
//     requests age out of the 5h/7d window. NOT a hard reset to 0% used.
//     After reset time, the portion of usage that was oldest is reclaimed.
//
//   Claude Code: Same as Codex — 5h and 7d rolling windows. The 5h window
//     fully restores when it elapses (hard reset to 0% used), but weekly
//     cap persists independently.
//
//   Cursor: Monthly billing cycle. Usage resets to 0% at CycleEnd date.
//     This is a hard reset — all counters go to zero on the new cycle.
//
//   Copilot: Monthly calendar reset. Usage counters reset on the 1st of
//     each month at 00:00 UTC. Hard reset to 0%.

// EstimateCodexSnapshot enriches a Codex snapshot with post-reset estimation
// if the snapshot's reset times have elapsed. Codex uses continuous rolling
// windows: the 5h window recovers the used portion as oldest requests expire.
func EstimateCodexSnapshot(snap *store.CodexSnapshot, now time.Time) {
	if snap == nil {
		return
	}

	// 5-hour window estimation
	if snap.FiveHourReset != nil && !snap.FiveHourReset.After(now) {
		timeSinceReset := now.Sub(*snap.FiveHourReset)
		switch {
		case timeSinceReset < 30*time.Minute:
			// Just reset — rolling window means all old usage has expired.
			// 5h window resets fully. Set usage to 0% (= 100% remaining).
			snap.FiveHourPct = 0
		case timeSinceReset < 6*time.Hour:
			// Within one window. Assume fully recovered since the entire
			// old window has expired and no new usage is known.
			snap.FiveHourPct = 0
		case timeSinceReset < 24*time.Hour:
			// Stale but still reasonable to assume recovered.
			snap.FiveHourPct = 0
		default:
			// Very stale (>24h) — keep original value; too uncertain
		}
	}

	// 7-day window estimation
	if snap.SevenDayPct != nil && snap.SevenDayReset != nil && !snap.SevenDayReset.After(now) {
		timeSinceReset := now.Sub(*snap.SevenDayReset)
		switch {
		case timeSinceReset < 1*time.Hour:
			v := 0.0
			snap.SevenDayPct = &v
		case timeSinceReset < 24*time.Hour:
			v := 0.0
			snap.SevenDayPct = &v
		default:
			// >24h since weekly reset — keep original
		}
	}
}

// EstimateClaudeSnapshot enriches a Claude Code snapshot with post-reset
// estimation. Claude uses the same 5h+7d dual-window model as Codex.
// The 5h window resets fully; the 7d weekly cap is independent.
func EstimateClaudeSnapshot(snap *store.ClaudeSnapshot, now time.Time) {
	if snap == nil {
		return
	}

	// 5-hour window estimation
	if snap.FiveHourReset != nil && !snap.FiveHourReset.After(now) {
		timeSinceReset := now.Sub(*snap.FiveHourReset)
		switch {
		case timeSinceReset < 30*time.Minute:
			snap.FiveHourPct = 0 // Fully restored
		case timeSinceReset < 6*time.Hour:
			snap.FiveHourPct = 0
		case timeSinceReset < 24*time.Hour:
			snap.FiveHourPct = 0
		default:
			// Very stale — keep original
		}
	}

	// 7-day window estimation
	if snap.SevenDayPct != nil && snap.SevenDayReset != nil && !snap.SevenDayReset.After(now) {
		timeSinceReset := now.Sub(*snap.SevenDayReset)
		switch {
		case timeSinceReset < 1*time.Hour:
			v := 0.0
			snap.SevenDayPct = &v
		case timeSinceReset < 24*time.Hour:
			v := 0.0
			snap.SevenDayPct = &v
		default:
			// >24h since weekly reset — keep original
		}
	}
}

// EstimateCursorSnapshot enriches a Cursor snapshot with post-cycle estimation.
// Cursor uses monthly billing cycles. Usage resets to 0% at CycleEnd.
func EstimateCursorSnapshot(snap *store.CursorSnapshot, now time.Time) {
	if snap == nil {
		return
	}

	// Cursor has CycleEnd — if it has elapsed, usage resets to 0%
	if snap.CycleEnd == "" {
		return
	}

	// CycleEnd is stored as an ISO date string, try parsing it
	cycleEnd, err := time.Parse(time.RFC3339, snap.CycleEnd)
	if err != nil {
		// Try date-only format
		cycleEnd, err = time.Parse("2006-01-02", snap.CycleEnd)
		if err != nil {
			return
		}
	}

	if cycleEnd.After(now) {
		return // Cycle hasn't ended yet
	}

	// Cycle has ended — usage resets to 0% on new billing cycle
	timeSinceReset := now.Sub(cycleEnd)
	switch {
	case timeSinceReset < 48*time.Hour:
		// Within 2 days of cycle boundary — high confidence reset
		snap.UsagePct = 0
		snap.RequestsUsed = 0
		snap.UsedCents = 0
		snap.AutoPct = 0
		snap.APIPct = 0
	default:
		// Over 2 days since cycle end — keep original, too uncertain
		// (user may have consumed a lot in the new cycle)
	}
}

// EstimateCopilotSnapshot enriches a Copilot snapshot with post-reset estimation.
// GitHub Copilot resets premium request counters on the 1st of each month at 00:00 UTC.
func EstimateCopilotSnapshot(snap *store.CopilotSnapshot, now time.Time) {
	if snap == nil {
		return
	}

	// Copilot resets on the 1st of each month.
	// If the snapshot was captured in a previous month, usage has reset.
	snapMonth := snap.CapturedAt.UTC().Month()
	snapYear := snap.CapturedAt.UTC().Year()
	nowMonth := now.UTC().Month()
	nowYear := now.UTC().Year()

	if nowYear == snapYear && nowMonth == snapMonth {
		return // Same month — values are current
	}

	// Different month — usage counters have reset
	// Calculate how far into the new cycle we are
	monthStart := time.Date(nowYear, nowMonth, 1, 0, 0, 0, 0, time.UTC)
	timeSinceReset := now.Sub(monthStart)

	switch {
	case timeSinceReset < 48*time.Hour:
		// Within first 2 days of month — high confidence reset to 0%
		snap.PremiumPct = 0
		snap.ChatPct = 0
	case timeSinceReset < 7*24*time.Hour:
		// Within first week — reasonable estimate, low usage likely
		snap.PremiumPct = 0
		snap.ChatPct = 0
	default:
		// Past first week of month — keep original value
		// (user has likely consumed an unknown amount)
	}
}
