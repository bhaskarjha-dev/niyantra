package web

import (
	"net/http"
	"strconv"

	"github.com/bhaskarjha-com/niyantra/internal/tracker"
)

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
