package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/claude"
	"github.com/bhaskarjha-com/niyantra/internal/client"
	"github.com/bhaskarjha-com/niyantra/internal/readiness"
	"github.com/bhaskarjha-com/niyantra/internal/store"
	"github.com/bhaskarjha-com/niyantra/internal/tracker"
)

// handleStatus returns readiness for all accounts.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	snapshots, err := s.store.LatestPerAccount()
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	accounts := readiness.Calculate(snapshots, 0.0)

	// F1: Enrich readiness results with account notes/tags/pinned_group/creditRenewalDay/planTier/overageCredits/hasClaimedBonus2026
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
		"observedAt":    latestObservedAt(snapshots),
		"basis":         "latest_provider_snapshots",
		"isEstimated":   false,
		"confidence":    "medium",
	}

	// C4: Include Codex snapshots (for homepage grid)
	codexSnaps, _ := s.store.LatestCodexSnapshots()
	if len(codexSnaps) > 0 {
		now := time.Now()
		for _, cs := range codexSnaps {
			readiness.EstimateCodexSnapshot(cs, now)
		}
		result["codexSnapshots"] = codexSnaps
	}

	// C4: Include Claude snapshot if available
	claudeSnap, _ := s.store.LatestClaudeSnapshot()
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

	// F15a: Include Cursor snapshots if available
	cursorSnaps, _ := s.store.LatestCursorSnapshots()
	if len(cursorSnaps) > 0 {
		now := time.Now()
		for _, cs := range cursorSnaps {
			readiness.EstimateCursorSnapshot(cs, now)
		}
		result["cursorSnapshots"] = cursorSnaps
	}

	// F15c: Include Copilot snapshots if available
	copilotSnaps, _ := s.store.LatestCopilotSnapshots()
	if len(copilotSnaps) > 0 {
		now := time.Now()
		for _, cs := range copilotSnaps {
			readiness.EstimateCopilotSnapshot(cs, now)
		}
		result["copilotSnapshots"] = copilotSnaps
	}

	pluginSnaps, _ := s.store.AllLatestPluginSnapshots()
	if len(pluginSnaps) > 0 {
		result["pluginSnapshots"] = pluginSnaps
	}

	// F7: Compute per-account forecasts using sliding-window rates
	forecastsByAccount := s.computeAccountForecasts(snapshots)
	if forecastsByAccount != nil {
		result["forecasts"] = forecastsByAccount
	}

	// F8: Compute per-account estimated costs using forecast rates + model pricing
	costsByAccount := s.computeAccountCosts(snapshots, forecastsByAccount)
	if costsByAccount != nil {
		result["estimatedCosts"] = costsByAccount
	}

	writeJSON(w, result)
}

// handleSnap triggers a snapshot capture.
func (s *Server) handleSnap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	s.client.Reset() // Clear cached connections to force fresh process/port detection (Antigravity v2.0)
	resps, err := s.client.FetchQuotas(ctx)
	if err != nil {
		s.logger.Error("snap failed", "error", err)
		s.store.LogError("ui", "snap_failed", "", map[string]interface{}{
			"error": err.Error(),
		})
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	type capturedInfo struct {
		Email      string `json:"email"`
		PlanName   string `json:"planName"`
		SnapshotID int64  `json:"snapshotId"`
		AccountID  int64  `json:"accountId"`
	}
	var captured []capturedInfo
	var lastDBErr error

	for _, resp := range resps {
		snap := resp.ToSnapshot(time.Now().UTC())

		// Tag provenance: captured via dashboard UI
		snap.CaptureMethod = "manual"
		snap.CaptureSource = "ui"
		snap.SourceID = "antigravity"

		accountID, err := s.store.GetOrCreateAccount(snap.Email, snap.PlanName, "antigravity")
		if err != nil {
			s.logger.Error("snap: database error creating account", "error", err, "email", snap.Email)
			lastDBErr = err
			continue
		}
		snap.AccountID = accountID

		snapID, err := s.store.InsertSnapshot(snap)
		if err != nil {
			s.logger.Error("snap: database error inserting snapshot", "error", err, "email", snap.Email)
			lastDBErr = err
			continue
		}

		// Log successful snap
		s.store.LogInfoSnap("ui", "snap", snap.Email, snapID, map[string]interface{}{
			"plan": snap.PlanName, "method": "manual", "source": "ui",
		})

		// Auto-link: create a subscription record if one doesn't exist for this account
		// Respects auto_link_subs config toggle (S2: was previously ignoring it)
		if s.store.GetConfig("auto_link_subs") != "false" {
			existing, _ := s.store.FindSubscriptionByAccountID(accountID)
			if existing == nil {
				autoSub := &store.Subscription{
					Platform:      "Antigravity",
					Category:      "coding",
					Email:         snap.Email,
					PlanName:      snap.PlanName,
					Status:        "active",
					CostCurrency:  "USD",
					BillingCycle:  "monthly",
					LimitPeriod:   "rolling_5h",
					Notes:         "Auto-created from quota snapshot. 5h sprint cycle quotas.",
					URL:           "https://antigravity.google",
					StatusPageURL: "https://status.google.com",
					AutoTracked:   true,
					AccountID:     accountID,
				}
				// Set cost based on plan name heuristic
				switch {
				case strings.Contains(strings.ToLower(snap.PlanName), "pro+"),
					strings.Contains(strings.ToLower(snap.PlanName), "ultimate"):
					autoSub.CostAmount = 60
				default:
					autoSub.CostAmount = 15
				}
				if _, err := s.store.InsertSubscription(autoSub); err != nil {
					s.logger.Warn("auto-link subscription failed", "error", err, "email", snap.Email)
				} else {
					s.logger.Info("auto-linked subscription", "email", snap.Email, "plan", snap.PlanName)
				}
			}
		}

		// Feed tracker for cycle intelligence (also works for manual snaps)
		if s.tracker != nil {
			if err := s.tracker.Process(snap, accountID); err != nil {
				s.logger.Warn("tracker error on manual snap", "error", err)
			}
		}

		captured = append(captured, capturedInfo{
			Email:      snap.Email,
			PlanName:   snap.PlanName,
			SnapshotID: snapID,
			AccountID:  accountID,
		})
	}

	if len(resps) > 0 && len(captured) == 0 && lastDBErr != nil {
		s.logger.Error("snap: database write failed for all detected quotas", "error", lastDBErr)
		s.store.LogError("ui", "snap_failed", "", map[string]interface{}{
			"error": "database write failed: " + lastDBErr.Error(),
		})
		jsonError(w, "database write failed: "+lastDBErr.Error(), http.StatusInternalServerError)
		return
	}

	// Update data source bookkeeping
	s.store.UpdateSourceCapture("antigravity")

	// Return updated accounts
	snapshots, _ := s.store.LatestPerAccount()
	accounts := readiness.Calculate(snapshots, 0.0)
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

	firstEmail := ""
	firstPlan := ""
	var firstSnapID int64
	var firstAccountID int64

	if len(captured) > 0 {
		firstEmail = captured[0].Email
		firstPlan = captured[0].PlanName
		firstSnapID = captured[0].SnapshotID
		firstAccountID = captured[0].AccountID
	}

	allAccounts, _ := s.store.AllAccounts()

	writeJSON(w, map[string]interface{}{
		"message":       "snapshot captured",
		"email":         firstEmail,
		"planName":      firstPlan,
		"snapshotId":    firstSnapID,
		"accountId":     firstAccountID,
		"captured":      captured,
		"accounts":      accounts,
		"allAccounts":   allAccounts,
		"accountCount":  s.store.AccountCount(),
		"snapshotCount": s.store.SnapshotCount(),
	})
}

// handleHistory returns snapshot history.
func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	var accountID int64
	if v := r.URL.Query().Get("account"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			accountID = id
		}
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 1000 {
		limit = 1000
	}

	providerFilter := r.URL.Query().Get("provider")
	sinceStr := r.URL.Query().Get("since")
	untilStr := r.URL.Query().Get("until")

	items, err := s.store.UnifiedHistory(accountID, limit, providerFilter, sinceStr, untilStr)
	if err != nil {
		jsonError(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"snapshots": items,
	})
}

// handleUsage returns per-model usage intelligence and budget forecast.
func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	var accountID int64
	if v := r.URL.Query().Get("account"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			accountID = id
		}
	}

	result := map[string]interface{}{
		"models":         nil,
		"budgetForecast": nil,
	}

	// Get latest snapshot(s) for the account(s)
	// N3b: When no account filter is specified, aggregate across all accounts
	snapshots, _ := s.store.LatestPerAccount()
	var allModels []*tracker.UsageSummary
	for _, snap := range snapshots {
		if accountID > 0 && snap.AccountID != accountID {
			continue
		}

		if s.tracker != nil {
			summaries, err := s.tracker.AllUsageSummaries(snap, snap.AccountID)
			if err != nil {
				s.logger.Warn("usage summary error", "error", err)
			}
			if summaries != nil {
				if accountID > 0 {
					// Single account filter — return directly
					result["models"] = summaries
					break
				}
				// Aggregate across all accounts
				allModels = append(allModels, summaries...)
			}
		}
	}
	if accountID == 0 && len(allModels) > 0 {
		result["models"] = allModels
	}

	// Budget forecast
	forecast := tracker.ComputeBudgetForecast(s.store)
	if forecast != nil {
		result["budgetForecast"] = forecast
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
