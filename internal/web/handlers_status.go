package web

import (
	"net/http"
	"strings"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/claude"
	"github.com/bhaskarjha-com/niyantra/internal/client"
	"github.com/bhaskarjha-com/niyantra/internal/readiness"
	"github.com/bhaskarjha-com/niyantra/internal/store"
)

// handleStatus returns readiness for all accounts.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	storedSnaps, err := s.store.LatestAll(r.Context())
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var antigravitySnaps []*client.Snapshot
	var codexSnaps []*store.CodexSnapshot
	var claudeSnap *store.ClaudeSnapshot
	var cursorSnaps []*store.CursorSnapshot
	var copilotSnaps []*store.CopilotSnapshot
	var pluginSnaps []*store.PluginSnapshot

	for _, snap := range storedSnaps {
		switch snap.Provider {
		case "antigravity":
			antigravitySnaps = append(antigravitySnaps, storedToClientSnapshot(snap))
		case "codex":
			codexSnaps = append(codexSnaps, storedToCodexSnapshot(snap))
		case "claude":
			if claudeSnap == nil {
				claudeSnap = storedToClaudeSnapshot(snap)
			}
		case "cursor":
			cursorSnaps = append(cursorSnaps, storedToCursorSnapshot(snap))
		case "copilot":
			copilotSnaps = append(copilotSnaps, storedToCopilotSnapshot(snap))
		default:
			if strings.HasPrefix(snap.Provider, "plugin_") {
				pluginSnaps = append(pluginSnaps, storedToPluginSnapshot(snap))
			}
		}
	}

	accounts := readiness.Calculate(antigravitySnaps, 0.0)

	// Enrich readiness results with account details
	for i := range accounts {
		acc, err := s.store.GetAccountByID(accounts[i].AccountID)
		if err == nil {
			accounts[i].Notes = acc.Notes
			accounts[i].Tags = acc.Tags
			accounts[i].PinnedGroup = acc.PinnedGroup
			accounts[i].CreditRenewalDay = acc.CreditRenewalDay
			accounts[i].PlanTier = acc.PlanTier
			accounts[i].OverageCredits = acc.OverageCredits
			accounts[i].HasClaimedBonus2026 = acc.HasClaimedBonus2026
			accounts[i].Provider = acc.Provider
		}
	}

	allAccounts, err := s.store.AllAccounts()
	if err != nil {
		allAccounts = nil
	}

	result := map[string]interface{}{
		"accounts":      accounts,
		"allAccounts":   allAccounts,
		"snapshotCount": s.store.SnapshotCount(),
		"accountCount":  s.store.AccountCount(),
		"observedAt":    latestObservedAt(antigravitySnaps),
		"basis":         "latest_provider_snapshots",
		"isEstimated":   false,
		"confidence":    "medium",
	}

	// Include Codex snapshots (for homepage grid)
	if len(codexSnaps) > 0 {
		now := time.Now()
		for _, cs := range codexSnaps {
			readiness.EstimateCodexSnapshot(cs, now)
		}
		result["codexSnapshots"] = codexSnaps
	}

	// Include Claude snapshot if available
	if claudeSnap != nil {
		readiness.EstimateClaudeSnapshot(claudeSnap, time.Now())
		result["claudeSnapshot"] = claudeSnap
	}

	// Always include Claude install/bridge status for contextual UI
	result["claudeStatus"] = map[string]interface{}{
		"installed":     claude.IsClaudeCodeInstalled(),
		"bridgeEnabled": s.store.GetConfigBool("claude_bridge"),
		"bridgeFresh":   claude.IsFresh(claude.DefaultStaleness),
	}

	// Include Cursor snapshots if available
	if len(cursorSnaps) > 0 {
		now := time.Now()
		for _, cs := range cursorSnaps {
			readiness.EstimateCursorSnapshot(cs, now)
		}
		result["cursorSnapshots"] = cursorSnaps
	}

	// Include Copilot snapshots if available
	if len(copilotSnaps) > 0 {
		now := time.Now()
		for _, cs := range copilotSnaps {
			readiness.EstimateCopilotSnapshot(cs, now)
		}
		result["copilotSnapshots"] = copilotSnaps
	}

	if len(pluginSnaps) > 0 {
		result["pluginSnapshots"] = pluginSnaps
	}

	// Compute per-account forecasts using sliding-window rates
	forecastsByAccount := s.computeAccountForecasts(antigravitySnaps)
	if forecastsByAccount != nil {
		result["forecasts"] = forecastsByAccount
	}

	// Compute per-account estimated costs using forecast rates + model pricing
	costsByAccount := s.computeAccountCosts(antigravitySnaps, forecastsByAccount)
	if costsByAccount != nil {
		result["estimatedCosts"] = costsByAccount
	}

	writeJSON(w, result)
}

func latestObservedAt(snapshots []*client.Snapshot) string {
	var latest time.Time
	for _, snap := range snapshots {
		if snap == nil {
			continue
		}
		if snap.CapturedAt.After(latest) {
			latest = snap.CapturedAt
		}
	}
	if latest.IsZero() {
		return ""
	}
	return latest.UTC().Format(time.RFC3339)
}
