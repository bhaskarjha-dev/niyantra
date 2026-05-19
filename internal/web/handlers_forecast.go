package web

import (
	"net/http"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/advisor"
	"github.com/bhaskarjha-com/niyantra/internal/costtrack"
	"github.com/bhaskarjha-com/niyantra/internal/forecast"
	"github.com/bhaskarjha-com/niyantra/internal/tracker"
)

// ── Phase 14: Cost Tracking (F8) ─────────────────────────────────

// handleCost returns estimated costs for all tracked accounts.
func (s *Server) handleCost(w http.ResponseWriter, r *http.Request) {
	snapshots, err := s.store.LatestPerAccount()
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	// Compute forecasts first (needed for burn rates)
	forecasts := s.computeAccountForecasts(snapshots)

	// Compute costs
	costs := s.computeAccountCosts(snapshots, forecasts)

	// Aggregate total
	var totalCost float64
	var accounts []costtrack.AccountCostEstimate
	for _, snap := range snapshots {
		if snap == nil {
			continue
		}
		if est, ok := costs[snap.AccountID]; ok {
			accounts = append(accounts, est)
			totalCost += est.TotalCost
		}
	}

	result := map[string]interface{}{
		"accounts":                 accounts,
		"totalCost":                totalCost,
		"totalLabel":               costtrack.FormatCost(totalCost),
		"basis":                    "estimated_quota_fraction",
		"confidence":               "low",
		"observedSpendAvailable":   false,
		"notAccountingGradeReason": "Costs are estimated from quota fraction deltas, configurable token ceilings, and blended model prices; they are not provider invoices or observed billing ledger entries.",
	}

	// Include ceilings for transparency
	ceilings, _ := costtrack.ParseCeilings(s.store.GetQuotaCeilings())
	result["quotaCeilings"] = ceilings

	writeJSON(w, result)
}

// ── Phase 14: Forecast Handlers ──────────────────────────────────

// handleForecast returns detailed TTX forecasts for all tracked providers.
func (s *Server) handleForecast(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{}

	// Antigravity account forecasts
	snapshots, err := s.store.LatestPerAccount()
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	forecasts := s.computeAccountForecasts(snapshots)

	// Build enriched forecast with account context
	type accountForecast struct {
		AccountID int64                    `json:"accountId"`
		Email     string                   `json:"email"`
		PlanName  string                   `json:"planName"`
		Groups    []forecast.GroupForecast `json:"groups"`
	}

	var antigravityForecasts []accountForecast
	for _, snap := range snapshots {
		if snap == nil {
			continue
		}
		gf := forecasts[snap.AccountID]
		if gf == nil {
			continue
		}
		antigravityForecasts = append(antigravityForecasts, accountForecast{
			AccountID: snap.AccountID,
			Email:     snap.Email,
			PlanName:  snap.PlanName,
			Groups:    gf,
		})
	}
	if len(antigravityForecasts) > 0 {
		result["antigravity"] = antigravityForecasts
	}

	// Claude Code forecast
	claudeForecasts := s.computeClaudeForecasts()
	if claudeForecasts != nil {
		result["claude"] = claudeForecasts
	}

	// Codex forecast
	codexForecasts := s.computeCodexForecasts()
	if codexForecasts != nil {
		result["codex"] = codexForecasts
	}

	// Advisor: best account recommendation with TTX context
	if len(antigravityForecasts) > 1 {
		summariesByAccount := make(map[int64][]*tracker.UsageSummary)
		if s.tracker != nil {
			for _, snap := range snapshots {
				summaries, _ := s.tracker.AllUsageSummaries(snap, snap.AccountID)
				if len(summaries) > 0 {
					summariesByAccount[snap.AccountID] = summaries
				}
			}
		}
		rec := advisor.Recommend(snapshots, summariesByAccount)
		if rec != nil {
			result["advisor"] = rec
		}
	}

	result["dataQuality"] = "heuristic"
	result["methodology"] = "sliding_window_burn_rate"
	result["disclaimer"] = "Forecasts are extrapolated from recent snapshot deltas (60-min window). They reflect current usage patterns, not guaranteed exhaustion times."

	writeJSON(w, result)
}

// ── F5: Anomaly Detection ────────────────────────────────────────

// handleAnomalies returns detected cost anomalies using Z-score analysis.
func (s *Server) handleAnomalies(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"anomalies": []forecast.Anomaly{},
		"disabled":  true,
		"reason":    "Insufficient historical daily spend data. Anomaly detection is disabled until real daily history is available.",
	})
}

// dayOfMonth returns the current day of the month (1-31).
func (s *Server) dayOfMonth() int {
	d := time.Now().Day()
	if d == 0 {
		d = 1
	}
	return d
}
