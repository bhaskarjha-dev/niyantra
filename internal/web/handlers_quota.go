package web

import (
	"net/http"
	"strings"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/readiness"
	"github.com/bhaskarjha-com/niyantra/internal/store"
)



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

		if err := s.store.SaveSnapshot(ctx, clientToCoreSnapshot(snap, accountID)); err != nil {
			s.logger.Error("snap: database error inserting unified snapshot", "error", err, "email", snap.Email)
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




