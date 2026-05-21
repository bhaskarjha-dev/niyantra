package web

import (
	"fmt"
	"net/http"

	"github.com/bhaskarjha-com/niyantra/internal/copilot"
	"github.com/bhaskarjha-com/niyantra/internal/store"
)

// handleCopilotStatus returns Copilot detection state and latest snapshot.
func (s *Server) handleCopilotStatus(w http.ResponseWriter, r *http.Request) {
	pat := s.store.GetConfig("copilot_pat")
	hasAutoToken := false
	source := "manual_pat"
	if pat == "" {
		detected, src, err := copilot.DetectCredentials(s.logger)
		if err == nil && detected != "" {
			hasAutoToken = true
			source = src
		}
	}

	result := map[string]interface{}{
		"configured":      pat != "" || hasAutoToken,
		"autoDetected":    hasAutoToken && pat == "",
		"detectionSource": source,
		"captureEnabled":  s.store.GetConfigBool("copilot_capture"),
	}

	// Latest snapshots
	snaps, err := s.store.LatestCopilotSnapshots()
	if err != nil {
		s.logger.Error("Failed to get Copilot snapshots", "error", err)
	}
	if len(snaps) > 0 {
		result["snapshots"] = snaps
	}

	writeJSON(w, result)
}

// handleCopilotSnap triggers a manual Copilot usage snapshot.
func (s *Server) handleCopilotSnap(w http.ResponseWriter, r *http.Request) {
	pat := s.store.GetConfig("copilot_pat")
	source := "manual_pat"
	if pat == "" {
		detected, src, err := copilot.DetectCredentials(s.logger)
		if err == nil && detected != "" {
			pat = detected
			source = src
		} else {
			jsonError(w, "GitHub Copilot PAT not configured and auto-detection failed. Set it in Settings → Copilot PAT or log in to Copilot in your IDE.", http.StatusBadRequest)
			return
		}
	}

	client := copilot.NewClient(pat, s.logger)
	snapshot, err := client.FetchSnapshot(r.Context())
	if err != nil {
		jsonError(w, fmt.Sprintf("Copilot API error: %v", err), http.StatusBadGateway)
		return
	}

	snap := &store.CopilotSnapshot{
		Email:         snapshot.Email,
		Username:      snapshot.Username,
		Plan:          snapshot.Plan,
		PremiumPct:    snapshot.PremiumPct,
		ChatPct:       snapshot.ChatPct,
		HasPremium:    snapshot.HasPremium,
		HasChat:       snapshot.HasChat,
		CaptureMethod: "manual",
		CaptureSource: source,
	}

	// Create/update account if we have identity info
	email := snapshot.Email
	if email == "" && snapshot.Username != "" {
		email = snapshot.Username + "@github.com"
	}
	if email != "" {
		accountID, err := s.store.GetOrCreateAccount(email, "Copilot "+snapshot.Plan, "copilot")
		if err == nil {
			snap.AccountID = accountID
		}
	}

	snapID, err := s.store.InsertCopilotSnapshot(snap)
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	s.store.UpdateSourceCapture("copilot")
	s.store.LogInfo("ui", "copilot_snap", email, map[string]interface{}{
		"plan": snapshot.Plan, "premiumPct": snapshot.PremiumPct,
		"chatPct": snapshot.ChatPct, "method": "manual",
		"username": snapshot.Username,
	})

	writeJSON(w, map[string]interface{}{
		"message":    "Copilot snapshot captured",
		"snapshotId": snapID,
		"plan":       snapshot.Plan,
		"premiumPct": snapshot.PremiumPct,
		"chatPct":    snapshot.ChatPct,
		"username":   snapshot.Username,
	})
}
