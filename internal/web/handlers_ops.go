package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/advisor"
	"github.com/bhaskarjha-com/niyantra/internal/claude"
	"github.com/bhaskarjha-com/niyantra/internal/gitcorr"
	"github.com/bhaskarjha-com/niyantra/internal/notify"
	"github.com/bhaskarjha-com/niyantra/internal/store"
	"github.com/bhaskarjha-com/niyantra/internal/tokenusage"
	"github.com/bhaskarjha-com/niyantra/internal/tracker"
)

// ── Operational Endpoints ─────────────────────────────────────────

// handleHealthz returns basic health/liveness data for monitoring.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"status": "ok",
	})
}

// handleHealthDetail returns detailed authenticated health data for the UI and
// local diagnostics. The public /healthz endpoint intentionally stays minimal
// so remote/container probes do not leak schema or usage inventory.
func (s *Server) handleHealthDetail(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"status":        "ok",
		"version":       s.Version,
		"uptime":        time.Since(s.startTime).Truncate(time.Second).String(),
		"schemaVersion": s.store.SchemaVersion(),
		"accounts":      s.store.AccountCount(),
		"snapshots":     s.store.SnapshotCount(),
	})
}

// ── Phase 9 Handlers ─────────────────────────────────────────────

// handleClaudeStatus returns the current Claude Code rate limit data.
func (s *Server) handleClaudeStatus(w http.ResponseWriter, r *http.Request) {
	bridgeEnabled := s.store.GetConfigBool("claude_bridge")
	installed := claude.IsClaudeCodeInstalled()
	fresh := claude.IsFresh(claude.DefaultStaleness)

	result := map[string]interface{}{
		"installed":     installed,
		"bridgeEnabled": bridgeEnabled,
		"bridgeFresh":   fresh,
		"supported":     notify.IsSupported(),
	}

	// Get latest snapshot from DB
	snap, err := s.store.LatestClaudeSnapshot()
	if err != nil {
		s.logger.Error("Failed to get Claude snapshot", "error", err)
	}
	if snap != nil {
		snapMap := map[string]interface{}{
			"fiveHourPct": snap.FiveHourPct,
			"capturedAt":  snap.CapturedAt.Format(time.RFC3339),
			"source":      snap.Source,
		}
		if snap.SevenDayPct != nil {
			snapMap["sevenDayPct"] = *snap.SevenDayPct
		}
		if snap.FiveHourReset != nil {
			snapMap["fiveHourReset"] = snap.FiveHourReset.Format(time.RFC3339)
		}
		if snap.SevenDayReset != nil {
			snapMap["sevenDayReset"] = snap.SevenDayReset.Format(time.RFC3339)
		}
		result["snapshot"] = snapMap
	}

	writeJSON(w, result)
}

// handleClaudeSnap manually captures a Claude Code rate limit snapshot
// by reading the statusline bridge data file. Provides parity with the
// Codex/Cursor/Copilot manual snap buttons.
func (s *Server) handleClaudeSnap(w http.ResponseWriter, r *http.Request) {
	if !claude.IsClaudeCodeInstalled() {
		jsonError(w, "Claude Code is not installed", http.StatusBadRequest)
		return
	}

	if !claude.IsFresh(claude.DefaultStaleness) {
		jsonError(w, "No fresh statusline data. Start a Claude Code session to generate data.", http.StatusNotFound)
		return
	}

	rl, err := claude.ReadData()
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to read statusline data: %v", err), http.StatusInternalServerError)
		return
	}
	if !claude.IsValid(rl) {
		jsonError(w, "statusline data is invalid or empty", http.StatusBadRequest)
		return
	}

	var fiveHourPct float64
	var sevenDayPct *float64
	var fiveReset, sevenReset *time.Time

	if rl.FiveHour != nil {
		fiveHourPct = rl.FiveHour.UsedPercentage
		if rl.FiveHour.ResetsAt > 0 {
			t := time.Unix(rl.FiveHour.ResetsAt, 0).UTC()
			fiveReset = &t
		}
	}
	if rl.SevenDay != nil {
		v := rl.SevenDay.UsedPercentage
		sevenDayPct = &v
		if rl.SevenDay.ResetsAt > 0 {
			t := time.Unix(rl.SevenDay.ResetsAt, 0).UTC()
			sevenReset = &t
		}
	}

	id, err := s.store.InsertClaudeSnapshot(fiveHourPct, sevenDayPct, fiveReset, sevenReset, "manual", nil)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to store snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	s.store.UpdateSourceCapture("claude_code")
	s.store.LogInfo("ui", "claude_snap", "", map[string]interface{}{
		"fiveHourPct": fiveHourPct,
		"source":      "manual",
	})

	writeJSON(w, map[string]interface{}{
		"status":      "captured",
		"id":          id,
		"fiveHourPct": fiveHourPct,
		"sevenDayPct": sevenDayPct,
	})
}

// handleBackupDeprecated rejects the legacy GET backup route. A full SQLite
// backup is a sensitive state export and must use the protected POST flow.
func (s *Server) handleBackupDeprecated(w http.ResponseWriter, r *http.Request) {
	jsonError(w, "GET /api/backup is disabled; use POST /api/backup/create", http.StatusGone)
}

// handleBackupCreate serves a consistent database backup as a download.
// Uses VACUUM INTO for WAL-safe snapshot instead of raw file copy.
func (s *Server) handleBackupCreate(w http.ResponseWriter, r *http.Request) {
	issues, err := s.store.IntegrityCheck()
	if err != nil {
		s.logger.Error("Backup integrity check failed", "error", err)
		jsonError(w, "database integrity check failed", http.StatusInternalServerError)
		return
	}
	if len(issues) > 0 {
		s.logger.Error("Backup refused due to database integrity issues", "issues", len(issues))
		jsonError(w, "database integrity check failed", http.StatusInternalServerError)
		return
	}

	// Create temp file for VACUUM INTO
	backupPath := s.store.Path() + ".backup-" + time.Now().Format("20060102-150405")
	if err := s.store.VacuumInto(backupPath); err != nil {
		s.logger.Error("Backup VACUUM INTO failed", "error", err)
		jsonError(w, "backup failed", http.StatusInternalServerError)
		return
	}
	defer os.Remove(backupPath)

	if err := store.RedactSensitiveConfigFile(backupPath); err != nil {
		s.logger.Error("Backup redaction failed", "error", err)
		jsonError(w, "backup redaction failed", http.StatusInternalServerError)
		return
	}

	f, err := os.Open(backupPath)
	if err != nil {
		jsonError(w, "cannot open backup", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		jsonError(w, "cannot stat backup", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("niyantra-%s.db", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	io.Copy(w, f)
}

// handleNotifyTest sends a test notification.
func (s *Server) handleNotifyTest(w http.ResponseWriter, r *http.Request) {
	if !notify.IsSupported() {
		jsonError(w, "notifications not supported on this platform", http.StatusBadRequest)
		return
	}

	if err := s.notifier.SendTest(); err != nil {
		jsonError(w, fmt.Sprintf("notification failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "sent"})
}

// handleNotifyTestEmail sends a test email to verify SMTP configuration (F11).
func (s *Server) handleNotifyTestEmail(w http.ResponseWriter, r *http.Request) {
	if err := s.notifier.SendTestEmail(); err != nil {
		jsonError(w, fmt.Sprintf("email failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "sent"})
}

// handleNotifyTestWebhook sends a test webhook to verify configuration (F22).
func (s *Server) handleNotifyTestWebhook(w http.ResponseWriter, r *http.Request) {
	if err := s.notifier.SendTestWebhookFromEngine(); err != nil {
		jsonError(w, fmt.Sprintf("webhook failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "sent"})
}

// handleWebPushVAPIDKey returns (and auto-generates) the VAPID public key (F19).
func (s *Server) handleWebPushVAPIDKey(w http.ResponseWriter, r *http.Request) {
	pubKey := s.store.GetConfig("webpush_vapid_public")
	if pubKey == "" {
		// Auto-generate VAPID keys on first request
		pub, priv, err := notify.GenerateVAPIDKeys()
		if err != nil {
			jsonError(w, fmt.Sprintf("Failed to generate VAPID keys: %v", err), http.StatusInternalServerError)
			return
		}
		s.store.SetConfig("webpush_vapid_public", pub)
		s.store.SetConfig("webpush_vapid_private", priv)
		pubKey = pub

		// Reload WebPush config
		s.notifier.ConfigureWebPush(s.loadWebPushConfig())
	}

	writeJSON(w, map[string]string{"publicKey": pubKey})
}

// handleWebPushSubscribe stores a browser push subscription (F19).
func (s *Server) handleWebPushSubscribe(w http.ResponseWriter, r *http.Request) {
	var sub struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			Auth   string `json:"auth"`
			P256dh string `json:"p256dh"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		jsonError(w, "invalid subscription body", http.StatusBadRequest)
		return
	}
	if sub.Endpoint == "" || sub.Keys.P256dh == "" || sub.Keys.Auth == "" {
		jsonError(w, "missing required fields: endpoint, keys.auth, keys.p256dh", http.StatusBadRequest)
		return
	}

	if err := s.store.SaveWebPushSubscription(sub.Endpoint, sub.Keys.P256dh, sub.Keys.Auth); err != nil {
		jsonError(w, fmt.Sprintf("save subscription: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "subscribed"})
}

// handleWebPushUnsubscribe removes a browser push subscription (F19).
func (s *Server) handleWebPushUnsubscribe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Endpoint == "" {
		jsonError(w, "missing endpoint", http.StatusBadRequest)
		return
	}

	if err := s.store.DeleteWebPushSubscription(body.Endpoint); err != nil {
		jsonError(w, fmt.Sprintf("delete subscription: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "unsubscribed"})
}

// handleNotifyTestWebPush sends a test push to all subscriptions (F19).
func (s *Server) handleNotifyTestWebPush(w http.ResponseWriter, r *http.Request) {
	if err := s.notifier.SendTestWebPushFromEngine(); err != nil {
		jsonError(w, fmt.Sprintf("webpush failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "sent"})
}

// handleWebPushStatus returns the number of subscriptions and enabled state (F19).
func (s *Server) handleWebPushStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"enabled":       s.store.GetConfigBool("webpush_enabled"),
		"subscriptions": s.store.WebPushSubscriptionCount(),
		"has_vapid":     s.store.GetConfig("webpush_vapid_public") != "",
	})
}

// ── Phase 10 Handlers ────────────────────────────────────────────

// handleExportJSON exports a redacted JSON report for sharing and import.
// Use the database backup endpoint for a full-fidelity local backup.
func (s *Server) handleExportJSON(w http.ResponseWriter, r *http.Request) {
	export := map[string]interface{}{
		"version":          "1.0",
		"exportedAt":       time.Now().UTC().Format(time.RFC3339),
		"niyantraVersion":  s.Version,
		"redactedSecrets":  true,
		"historyScope":     "recent",
		"fullBackupPath":   "/api/backup/create",
		"exportDisclaimer": "Secrets are redacted and snapshot history is limited. Use POST /api/backup/create for a full local database backup.",
	}

	// Accounts
	accounts, _ := s.store.AllAccounts()
	if accounts == nil {
		accounts = []*store.Account{}
	}
	export["accounts"] = accounts

	// Subscriptions
	subs, _ := s.store.ListSubscriptions("", "")
	if subs == nil {
		subs = []*store.Subscription{}
	}
	export["subscriptions"] = subs

	// Recent snapshots (last 1000)
	snapshots, _ := s.store.HistoryLegacy(0, 1000)
	export["snapshots"] = snapshots

	// Claude snapshots (last 500)
	claudeSnaps, _ := s.store.ClaudeSnapshotHistory(500)
	export["claudeSnapshots"] = claudeSnaps

	// Codex snapshots (last 500)
	codexSnaps, _ := s.store.RecentCodexSnapshots(24 * 365 * time.Hour)
	export["codexSnapshots"] = codexSnaps

	// Cursor snapshots (last 500)
	cursorSnaps, _ := s.store.RecentCursorSnapshots(500)
	export["cursorSnapshots"] = cursorSnaps

	// Copilot snapshots (last 500)
	copilotSnaps, _ := s.store.RecentCopilotSnapshots(500)
	export["copilotSnapshots"] = copilotSnaps

	// Plugin snapshots (all latest per plugin)
	pluginSnaps, _ := s.store.AllLatestPluginSnapshots()
	export["pluginSnapshots"] = pluginSnaps

	// Config
	config, _ := s.store.AllConfig("")
	maskConfigEntries(config)
	export["config"] = config

	// Activity log (last 500)
	activity, _ := s.store.RecentActivity(500, "")
	export["activityLog"] = activity

	// Log the export event
	s.store.LogInfo("ui", "export", "", map[string]interface{}{
		"format": "json",
	})

	filename := fmt.Sprintf("niyantra-export-%s.json", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(export)
}

// handleImportJSON handles JSON data import with merge strategy.
func (s *Server) handleImportJSON(w http.ResponseWriter, r *http.Request) {
	// Read request body (limit to 50MB)
	body, err := io.ReadAll(io.LimitReader(r.Body, 50<<20))
	if err != nil {
		jsonError(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		jsonError(w, "empty request body", http.StatusBadRequest)
		return
	}

	result, err := s.store.ImportJSON(body)
	if err != nil {
		if result != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":  fmt.Sprintf("import failed: %v", err),
				"result": result,
			})
			return
		}
		jsonError(w, fmt.Sprintf("import failed: %v", err), http.StatusBadRequest)
		return
	}

	// Log the import
	s.store.LogInfo("ui", "import", "", map[string]interface{}{
		"accountsCreated":   result.AccountsCreated,
		"accountsSkipped":   result.AccountsSkipped,
		"subsCreated":       result.SubsCreated,
		"subsSkipped":       result.SubsSkipped,
		"snapshotsImported": result.SnapshotsImported,
		"snapshotsDuped":    result.SnapshotsDuped,
		"errors":            len(result.Errors),
	})

	writeJSON(w, result)
}

// handleAlerts returns active system alerts.
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := s.store.ActiveAlerts()
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}
	if alerts == nil {
		alerts = []*store.SystemAlert{}
	}

	writeJSON(w, map[string]interface{}{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

// handleDismissAlert dismisses a system alert by ID.
func (s *Server) handleDismissAlert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.ID <= 0 {
		jsonError(w, "alert ID required", http.StatusBadRequest)
		return
	}

	if err := s.store.DismissAlert(req.ID); err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"message": "dismissed"})
}

// handleAdvisor returns account switching recommendation.
func (s *Server) handleAdvisor(w http.ResponseWriter, r *http.Request) {
	snapshots, err := s.store.LatestPerAccount()
	if err != nil {
		jsonError(w, "database error", http.StatusInternalServerError)
		return
	}

	// Build per-account usage summaries for burn rate intelligence
	summariesByAccount := make(map[int64][]*tracker.UsageSummary)
	if s.tracker != nil {
		for _, snap := range snapshots {
			summaries, err := s.tracker.AllUsageSummaries(snap, snap.AccountID)
			if err == nil && len(summaries) > 0 {
				summariesByAccount[snap.AccountID] = summaries
			}
		}
	}

	var currentAccountID int64
	if raw := r.URL.Query().Get("currentAccountId"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			currentAccountID = parsed
		}
	}
	rec := advisor.RecommendWithCurrent(snapshots, summariesByAccount, currentAccountID)
	writeJSON(w, rec)
}

// ── Phase 14: Claude Code Deep Tracking (F15d) ──────────────────

// handleClaudeUsage returns deep token usage analytics from Claude Code's
// local JSONL session files. Zero network calls — pure filesystem parsing.
func (s *Server) handleClaudeUsage(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 && v <= 365 {
			days = v
		}
	}

	// Wire model pricing callback to F5 pricing config
	priceFn := func(modelID string) (float64, float64, float64, bool) {
		p := s.store.GetModelPrice(modelID)
		if p == nil {
			return 0, 0, 0, false
		}
		return p.InputPer1M, p.OutputPer1M, p.CachePer1M, true
	}

	summary, err := claude.AggregateUsage(days, priceFn)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to aggregate usage: %v", err), http.StatusInternalServerError)
		return
	}

	if summary == nil {
		summary = &claude.UsageSummary{
			Days: []claude.DailyUsage{},
		}
	}

	writeJSON(w, summary)
}

// ── Phase 15: Token Usage Analytics (F13) ────────────────────────

// handleTokenUsage returns observed token usage analytics from the sources that
// currently exist: Claude Code JSONL session files plus any persisted
// token_usage rows already written into the local store.
func (s *Server) handleTokenUsage(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 && v <= 365 {
			days = v
		}
	}

	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "all"
	}

	// Wire model pricing callback to F5 pricing config
	priceFn := func(modelID string) (float64, float64, float64, bool) {
		p := s.store.GetModelPrice(modelID)
		if p == nil {
			return 0, 0, 0, false
		}
		return p.InputPer1M, p.OutputPer1M, p.CachePer1M, true
	}

	var result *tokenusage.Summary

	if provider == "all" || provider == "claude" {
		// Claude Code: granular JSONL parsing (primary data source)
		claudeSummary, err := tokenusage.AggregateFromClaude(days, priceFn)
		if err != nil {
			s.logger.Warn("Token usage: Claude aggregation failed", "error", err)
			claudeSummary = nil
		}

		if provider == "claude" {
			result = claudeSummary
		} else {
			// Merge Claude data with persisted store data from other providers
			storeSummary, err := tokenusage.AggregateFromStore(s.store, days, "all")
			if err != nil {
				s.logger.Warn("Token usage: store aggregation failed", "error", err)
			}
			result = tokenusage.Merge(claudeSummary, storeSummary)
		}
	} else {
		// Specific non-Claude provider: query store only
		var err error
		result, err = tokenusage.AggregateFromStore(s.store, days, provider)
		if err != nil {
			jsonError(w, fmt.Sprintf("failed to aggregate usage: %v", err), http.StatusInternalServerError)
			return
		}
	}

	if result == nil {
		result = &tokenusage.Summary{
			ByModel: []tokenusage.ModelBreakdown{},
			ByDay:   []tokenusage.DailyBreakdown{},
			Period:  tokenusage.Period{Days: days},
		}
	}

	writeJSON(w, result)
}

// ── Phase 15: Git Commit Correlation (F16) ───────────────────────

// handleGitCosts correlates git commits with AI token consumption.
func (s *Server) handleGitCosts(w http.ResponseWriter, r *http.Request) {
	cwd, err := os.Getwd()
	if err != nil {
		jsonError(w, "cannot determine working directory", http.StatusInternalServerError)
		return
	}
	repoPath, err := gitRepoPathFromRequest(cwd, r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 && v <= 365 {
			days = v
		}
	}

	// Wire model pricing callback
	priceFn := func(modelID string) (float64, float64, float64, bool) {
		p := s.store.GetModelPrice(modelID)
		if p == nil {
			return 0, 0, 0, false
		}
		return p.InputPer1M, p.OutputPer1M, p.CachePer1M, true
	}

	result, err := gitcorr.Analyze(repoPath, days, gitcorr.DefaultWindowMinutes, priceFn)
	if err != nil {
		s.logger.Warn("Git costs analysis failed", "error", err, "repo", repoPath)
		jsonError(w, fmt.Sprintf("git analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, result)
}

func gitRepoPathFromRequest(baseDir string, r *http.Request) (string, error) {
	repoPath := strings.TrimSpace(r.URL.Query().Get("repo"))
	if repoPath == "" {
		return filepath.Clean(baseDir), nil
	}
	return resolveGitRepoPath(baseDir, repoPath)
}

func resolveGitRepoPath(baseDir, requested string) (string, error) {
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("cannot resolve working directory: %w", err)
	}
	requestedPath := strings.TrimSpace(requested)
	if !filepath.IsAbs(requestedPath) {
		requestedPath = filepath.Join(baseAbs, requestedPath)
	}
	requestedAbs, err := filepath.Abs(requestedPath)
	if err != nil {
		return "", fmt.Errorf("invalid repo path: %w", err)
	}
	rel, err := filepath.Rel(baseAbs, requestedAbs)
	if err != nil {
		return "", fmt.Errorf("invalid repo path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("repo path must stay within %s", baseAbs)
	}
	return requestedAbs, nil
}
