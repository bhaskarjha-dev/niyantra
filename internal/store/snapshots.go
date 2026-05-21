package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/client"
)

// InsertSnapshot stores a snapshot in the database.
func (s *Store) InsertSnapshot(snap *client.Snapshot) (int64, error) {
	modelsJSON, err := json.Marshal(snap.Models)
	if err != nil {
		return 0, fmt.Errorf("store: marshal models: %w", err)
	}

	aiCreditsJSON := ""
	if len(snap.AICredits) > 0 {
		if b, err := json.Marshal(snap.AICredits); err == nil {
			aiCreditsJSON = string(b)
		}
	}

	result, err := s.db.Exec(`
		INSERT INTO snapshots (account_id, captured_at, email, plan_name,
			prompt_credits, monthly_credits, models_json, raw_json,
			capture_method, capture_source, source_id, ai_credits_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		snap.AccountID,
		snap.CapturedAt.UTC().Format(time.RFC3339),
		snap.Email,
		snap.PlanName,
		snap.PromptCredits,
		snap.MonthlyCredits,
		string(modelsJSON),
		snap.RawJSON,
		snap.CaptureMethod,
		snap.CaptureSource,
		snap.SourceID,
		aiCreditsJSON,
	)
	if err != nil {
		return 0, fmt.Errorf("store: insert snapshot: %w", err)
	}

	return result.LastInsertId()
}

// LatestPerAccount returns the latest snapshot for each account.
func (s *Store) LatestPerAccount() ([]*client.Snapshot, error) {
	rows, err := s.db.Query(`
		SELECT s.id, s.account_id, s.captured_at, s.email, s.plan_name,
			s.prompt_credits, s.monthly_credits, s.models_json, s.raw_json,
			COALESCE(s.capture_method,'manual'), COALESCE(s.capture_source,'cli'), COALESCE(s.source_id,'antigravity'),
			COALESCE(s.ai_credits_json,'')
		FROM snapshots s
		WHERE s.id = (
			SELECT s2.id
			FROM snapshots s2
			WHERE s2.account_id = s.account_id
			ORDER BY s2.captured_at DESC, s2.id DESC
			LIMIT 1
		)
		ORDER BY s.captured_at DESC, s.id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("store: query latest snapshots: %w", err)
	}
	defer rows.Close()

	return scanSnapshots(rows)
}

// HistoryLegacy returns recent snapshots, optionally filtered by account.
func (s *Store) HistoryLegacy(accountID int64, limit int) ([]*client.Snapshot, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	var query string
	var args []interface{}

	if accountID > 0 {
		query = `SELECT id, account_id, captured_at, email, plan_name,
			prompt_credits, monthly_credits, models_json, raw_json,
			COALESCE(capture_method,'manual'), COALESCE(capture_source,'cli'), COALESCE(source_id,'antigravity'),
			COALESCE(ai_credits_json,'')
			FROM snapshots WHERE account_id = ?
			ORDER BY captured_at DESC LIMIT ?`
		args = []interface{}{accountID, limit}
	} else {
		query = `SELECT id, account_id, captured_at, email, plan_name,
			prompt_credits, monthly_credits, models_json, raw_json,
			COALESCE(capture_method,'manual'), COALESCE(capture_source,'cli'), COALESCE(source_id,'antigravity'),
			COALESCE(ai_credits_json,'')
			FROM snapshots ORDER BY captured_at DESC LIMIT ?`
		args = []interface{}{limit}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query history: %w", err)
	}
	defer rows.Close()

	return scanSnapshots(rows)
}

// GetSnapshotByID returns a single snapshot by its primary key.
func (s *Store) GetSnapshotByID(id int64) (*client.Snapshot, error) {
	row := s.db.QueryRow(`
		SELECT id, account_id, captured_at, email, plan_name,
			prompt_credits, monthly_credits, models_json, raw_json,
			COALESCE(capture_method,'manual'), COALESCE(capture_source,'cli'),
			COALESCE(source_id,'antigravity'), COALESCE(ai_credits_json,'')
		FROM snapshots WHERE id = ?`, id)

	var snap client.Snapshot
	var capturedAt, modelsJSON, aiCreditsJSON string
	if err := row.Scan(
		&snap.ID, &snap.AccountID, &capturedAt, &snap.Email,
		&snap.PlanName, &snap.PromptCredits, &snap.MonthlyCredits,
		&modelsJSON, &snap.RawJSON, &snap.CaptureMethod, &snap.CaptureSource,
		&snap.SourceID, &aiCreditsJSON,
	); err != nil {
		return nil, fmt.Errorf("store: get snapshot %d: %w", id, err)
	}

	if t, err := time.Parse(time.RFC3339, capturedAt); err == nil {
		snap.CapturedAt = t
	}
	json.Unmarshal([]byte(modelsJSON), &snap.Models)
	if aiCreditsJSON != "" {
		json.Unmarshal([]byte(aiCreditsJSON), &snap.AICredits)
	}

	return &snap, nil
}

// SnapshotCount returns the total number of snapshots.
func (s *Store) SnapshotCount() int {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM snapshots").Scan(&count)
	return count
}

// HeatmapDay represents a single day's snapshot activity for the heatmap.
type HeatmapDay struct {
	Date        string `json:"date"`        // YYYY-MM-DD
	Count       int    `json:"count"`       // total snapshots
	Antigravity int    `json:"antigravity"` // Antigravity snapshots
	Claude      int    `json:"claude"`      // Claude Code snapshots
	Codex       int    `json:"codex"`       // Codex snapshots
	Cursor      int    `json:"cursor"`      // Cursor snapshots
	Copilot     int    `json:"copilot"`     // GitHub Copilot snapshots
	Plugin      int    `json:"plugin"`      // External plugin snapshots
}

// HeatmapData returns daily snapshot counts across all providers for the last N days.
// Used by the F6 Activity Heatmap to render a GitHub-style contribution calendar.
func (s *Store) HeatmapData(days int) ([]HeatmapDay, error) {
	if days <= 0 {
		days = 365
	}

	cutoff := fmt.Sprintf("-%d days", days)

	rows, err := s.db.Query(`
		SELECT
			day,
			SUM(ag) as antigravity,
			SUM(cl) as claude,
			SUM(cx) as codex,
			SUM(cr) as cursor_cnt,
			SUM(cp) as copilot_cnt,
			SUM(pl) as plugin_cnt
		FROM (
			SELECT date(captured_at) as day, 1 as ag, 0 as cl, 0 as cx, 0 as cr, 0 as cp, 0 as pl
			FROM snapshots WHERE captured_at >= datetime('now', ?)
			UNION ALL
			SELECT date(captured_at) as day, 0 as ag, 1 as cl, 0 as cx, 0 as cr, 0 as cp, 0 as pl
			FROM claude_snapshots WHERE captured_at >= datetime('now', ?)
			UNION ALL
			SELECT date(captured_at) as day, 0 as ag, 0 as cl, 1 as cx, 0 as cr, 0 as cp, 0 as pl
			FROM codex_snapshots WHERE captured_at >= datetime('now', ?)
			UNION ALL
			SELECT date(captured_at) as day, 0 as ag, 0 as cl, 0 as cx, 1 as cr, 0 as cp, 0 as pl
			FROM cursor_snapshots WHERE captured_at >= datetime('now', ?)
			UNION ALL
			SELECT date(captured_at) as day, 0 as ag, 0 as cl, 0 as cx, 0 as cr, 1 as cp, 0 as pl
			FROM copilot_snapshots WHERE captured_at >= datetime('now', ?)
			UNION ALL
			SELECT date(captured_at) as day, 0 as ag, 0 as cl, 0 as cx, 0 as cr, 0 as cp, 1 as pl
			FROM plugin_snapshots WHERE captured_at >= datetime('now', ?)
		)
		GROUP BY day
		ORDER BY day ASC`, cutoff, cutoff, cutoff, cutoff, cutoff, cutoff)
	if err != nil {
		return nil, fmt.Errorf("store: heatmap query: %w", err)
	}
	defer rows.Close()

	var result []HeatmapDay
	for rows.Next() {
		var d HeatmapDay
		if err := rows.Scan(&d.Date, &d.Antigravity, &d.Claude, &d.Codex, &d.Cursor, &d.Copilot, &d.Plugin); err != nil {
			return nil, fmt.Errorf("store: scan heatmap row: %w", err)
		}
		d.Count = d.Antigravity + d.Claude + d.Codex + d.Cursor + d.Copilot + d.Plugin
		result = append(result, d)
	}
	return result, rows.Err()
}

// DeleteSnapshotsOlderThan removes snapshots older than the given number of days.
// Also cleans up old Claude and Codex snapshots with the same retention policy.
// Returns the total number of deleted rows.
func (s *Store) DeleteSnapshotsOlderThan(days int) (int64, error) {
	cutoff := fmt.Sprintf("-%d days", days)

	result, err := s.db.Exec(
		`DELETE FROM snapshots WHERE captured_at < datetime('now', ?)`, cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("store: delete old snapshots: %w", err)
	}
	deleted, _ := result.RowsAffected()

	// Also clean up old Claude snapshots
	result2, err := s.db.Exec(
		`DELETE FROM claude_snapshots WHERE captured_at < datetime('now', ?)`, cutoff,
	)
	if err == nil {
		d2, _ := result2.RowsAffected()
		deleted += d2
	}

	// N13: Also clean up old Codex snapshots (previously unbounded)
	result3, err := s.db.Exec(
		`DELETE FROM codex_snapshots WHERE captured_at < datetime('now', ?)`, cutoff,
	)
	if err == nil {
		d3, _ := result3.RowsAffected()
		deleted += d3
	}

	// Also clean up old Cursor snapshots
	result4, err := s.db.Exec(
		`DELETE FROM cursor_snapshots WHERE captured_at < datetime('now', ?)`, cutoff,
	)
	if err == nil {
		d4, _ := result4.RowsAffected()
		deleted += d4
	}


	// Also clean up old Copilot snapshots
	result6, err := s.db.Exec(
		`DELETE FROM copilot_snapshots WHERE captured_at < datetime('now', ?)`, cutoff,
	)
	if err == nil {
		d6, _ := result6.RowsAffected()
		deleted += d6
	}

	// Also clean up old plugin snapshots (F18)
	result7, err := s.db.Exec(
		`DELETE FROM plugin_snapshots WHERE captured_at < datetime('now', ?)`, cutoff,
	)
	if err == nil {
		d7, _ := result7.RowsAffected()
		deleted += d7
	}

	// Also clean up old token usage records
	result8, err := s.db.Exec(
		`DELETE FROM token_usage WHERE date < date('now', ?)`, cutoff,
	)
	if err == nil {
		d8, _ := result8.RowsAffected()
		deleted += d8
	}

	return deleted, nil
}

// scanSnapshots reads snapshot rows into Snapshot structs.
func scanSnapshots(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
}) ([]*client.Snapshot, error) {
	var snapshots []*client.Snapshot

	type scanner interface {
		Next() bool
		Scan(dest ...interface{}) error
	}
	r := rows.(scanner)

	for r.Next() {
		var snap client.Snapshot
		var capturedAt string
		var modelsJSON string
		var aiCreditsJSON string

		if err := r.Scan(
			&snap.ID, &snap.AccountID, &capturedAt, &snap.Email,
			&snap.PlanName, &snap.PromptCredits, &snap.MonthlyCredits,
			&modelsJSON, &snap.RawJSON, &snap.CaptureMethod, &snap.CaptureSource, &snap.SourceID,
			&aiCreditsJSON,
		); err != nil {
			return nil, fmt.Errorf("store: scan snapshot: %w", err)
		}

		if t, err := time.Parse(time.RFC3339, capturedAt); err == nil {
			snap.CapturedAt = t
		}

		if err := json.Unmarshal([]byte(modelsJSON), &snap.Models); err != nil {
			snap.Models = nil // graceful degradation
		}

		if aiCreditsJSON != "" {
			json.Unmarshal([]byte(aiCreditsJSON), &snap.AICredits)
		}

		snapshots = append(snapshots, &snap)
	}

	return snapshots, nil
}

// UpdateSnapshotModels updates the models_json for a snapshot.
// Used by Quick Adjust to let users fine-tune quota percentages.
func (s *Store) UpdateSnapshotModels(snapshotID int64, models []client.ModelQuota) error {
	modelsJSON, err := json.Marshal(models)
	if err != nil {
		return fmt.Errorf("store: marshal models: %w", err)
	}

	result, err := s.db.Exec(
		`UPDATE snapshots SET models_json = ? WHERE id = ?`,
		string(modelsJSON), snapshotID,
	)
	if err != nil {
		return fmt.Errorf("store: update snapshot models: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("store: snapshot %d not found", snapshotID)
	}

	return nil
}

// RecentModelData is a lightweight snapshot for rate computation.
// Contains only the fields needed for burn rate calculation.
type RecentModelData struct {
	CapturedAt time.Time
	ModelsJSON string
}

// RecentModelSnapshots returns lightweight model data for an account
// captured within the given window. Results are ordered chronologically (ASC).
// Used by the forecast package for sliding-window rate computation.
func (s *Store) RecentModelSnapshots(accountID int64, window time.Duration) ([]RecentModelData, error) {
	since := time.Now().UTC().Add(-window).Format(time.RFC3339)

	rows, err := s.db.Query(`
		SELECT captured_at, models_json
		FROM snapshots
		WHERE account_id = ? AND captured_at >= ?
		ORDER BY captured_at ASC`,
		accountID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("store: recent model snapshots: %w", err)
	}
	defer rows.Close()

	var results []RecentModelData
	for rows.Next() {
		var capturedAtStr, modelsJSON string
		if err := rows.Scan(&capturedAtStr, &modelsJSON); err != nil {
			return nil, err
		}
		capturedAt, _ := time.Parse(time.RFC3339, capturedAtStr)
		results = append(results, RecentModelData{
			CapturedAt: capturedAt,
			ModelsJSON: modelsJSON,
		})
	}
	return results, rows.Err()
}

// RecentClaudeSnapshots returns recent Claude Code snapshots within the given
// window. Results are ordered chronologically (ASC).
// Used by the forecast package for Claude TTX computation.
func (s *Store) RecentClaudeSnapshots(window time.Duration) ([]ClaudeSnapshot, error) {
	since := time.Now().UTC().Add(-window).Format(time.RFC3339)

	return s.ClaudeSnapshotsSince(since)
}

// ClaudeSnapshotsSince returns Claude snapshots captured on or after the given
// RFC3339 timestamp, ordered chronologically (ASC).
func (s *Store) ClaudeSnapshotsSince(since string) ([]ClaudeSnapshot, error) {
	rows, err := s.db.Query(`
		SELECT id, five_hour_pct, seven_day_pct, five_hour_reset, seven_day_reset, captured_at, source
		FROM claude_snapshots
		WHERE captured_at >= ?
		ORDER BY captured_at ASC`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanClaudeRows(rows)
}

// scanClaudeRows scans rows into ClaudeSnapshot structs.
func scanClaudeRows(rows *sql.Rows) ([]ClaudeSnapshot, error) {
	var snaps []ClaudeSnapshot
	for rows.Next() {
		snap := ClaudeSnapshot{}
		var sevenPct sql.NullFloat64
		var fiveReset, sevenReset sql.NullTime

		if err := rows.Scan(&snap.ID, &snap.FiveHourPct, &sevenPct, &fiveReset, &sevenReset,
			&snap.CapturedAt, &snap.Source); err != nil {
			return nil, err
		}

		if sevenPct.Valid {
			snap.SevenDayPct = &sevenPct.Float64
		}
		if fiveReset.Valid {
			snap.FiveHourReset = &fiveReset.Time
		}
		if sevenReset.Valid {
			snap.SevenDayReset = &sevenReset.Time
		}

		snaps = append(snaps, snap)
	}
	return snaps, nil
}

// RecentCodexSnapshots returns recent Codex snapshots within the given window,
// ordered chronologically (ASC). Used by the forecast package for Codex TTX.
func (s *Store) RecentCodexSnapshots(window time.Duration) ([]*CodexSnapshot, error) {
	since := time.Now().UTC().Add(-window)
	return s.CodexHistory(since)
}

// UnifiedHistory returns merged chronological history from all snapshot tables.
func (s *Store) UnifiedHistory(accountID int64, limit int, providerFilter string, sinceStr string, untilStr string) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	var items []map[string]interface{}

	// Determine provider if accountID > 0
	var provider string
	if accountID > 0 {
		err := s.db.QueryRow("SELECT provider FROM accounts WHERE id = ?", accountID).Scan(&provider)
		if err == sql.ErrNoRows {
			return items, nil
		} else if err != nil {
			return nil, fmt.Errorf("store: query provider: %w", err)
		}
	}

	// 1. Antigravity Snapshots
	if (accountID == 0 && (providerFilter == "" || providerFilter == "all" || providerFilter == "antigravity")) || (accountID > 0 && provider == "antigravity") {
		var query string
		var args []interface{}
		var conditions []string

		if accountID > 0 {
			conditions = append(conditions, "account_id = ?")
			args = append(args, accountID)
		}
		if sinceStr != "" {
			conditions = append(conditions, "datetime(captured_at) >= datetime(?)")
			args = append(args, sinceStr)
		}
		if untilStr != "" {
			conditions = append(conditions, "datetime(captured_at) <= datetime(?)")
			args = append(args, untilStr)
		}

		whereClause := ""
		if len(conditions) > 0 {
			whereClause = "WHERE " + strings.Join(conditions, " AND ")
		}

		query = fmt.Sprintf(`SELECT id, account_id, captured_at, email, plan_name,
			models_json, COALESCE(capture_method,'manual'), COALESCE(capture_source,'cli'), COALESCE(ai_credits_json,'')
			FROM snapshots %s
			ORDER BY captured_at DESC LIMIT ?`, whereClause)
		args = append(args, limit)

		rows, err := s.db.Query(query, args...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, accID int64
				var capturedAtStr, email, planName, modelsJSON, captureMethod, captureSource, aiCreditsJSON string
				if err := rows.Scan(&id, &accID, &capturedAtStr, &email, &planName, &modelsJSON, &captureMethod, &captureSource, &aiCreditsJSON); err == nil {
					capturedAt, _ := time.Parse(time.RFC3339, capturedAtStr)
					
					// Parse models
					var models []struct {
						ModelID           string  `json:"modelId"`
						Label             string  `json:"label"`
						RemainingFraction float64 `json:"remainingFraction"`
						RemainingPercent  float64 `json:"remainingPercent"`
						IsExhausted       bool    `json:"isExhausted"`
					}
					json.Unmarshal([]byte(modelsJSON), &models)

					// Group models
					var claudeGPTFraction, geminiUnifiedFraction float64
					var claudeGPTCount, geminiUnifiedCount int
					
					for _, m := range models {
						text := strings.ToLower(m.ModelID + " " + m.Label)
						if strings.Contains(text, "gemini") ||
							strings.Contains(text, "model_placeholder_m133") ||
							strings.Contains(text, "model_placeholder_m20") ||
							strings.Contains(text, "model_placeholder_m16") ||
							strings.Contains(text, "model_placeholder_m36") {
							geminiUnifiedFraction += m.RemainingFraction
							geminiUnifiedCount++
						} else if strings.Contains(text, "claude") || strings.Contains(text, "anthropic") || strings.Contains(text, "gpt") || strings.Contains(text, "openai") {
							claudeGPTFraction += m.RemainingFraction
							claudeGPTCount++
						}
					}

					var groups []map[string]interface{}
					if claudeGPTCount > 0 {
						rem := claudeGPTFraction / float64(claudeGPTCount)
						groups = append(groups, map[string]interface{}{
							"groupKey": "claude_gpt",
							"displayName": "Claude + GPT",
							"remainingPercent": math.Round(rem * 100),
							"isExhausted": rem <= 0,
							"color": "#D97757",
						})
					}
					if geminiUnifiedCount > 0 {
						rem := geminiUnifiedFraction / float64(geminiUnifiedCount)
						groups = append(groups, map[string]interface{}{
							"groupKey": "gemini_unified",
							"displayName": "Gemini Pool",
							"remainingPercent": math.Round(rem * 100),
							"isExhausted": rem <= 0,
							"color": "#3B82F6",
						})
					}

					var aiCredits []map[string]interface{}
					if aiCreditsJSON != "" {
						json.Unmarshal([]byte(aiCreditsJSON), &aiCredits)
					}

					items = append(items, map[string]interface{}{
						"id": id,
						"provider": "antigravity",
						"accountId": accID,
						"email": email,
						"capturedAt": capturedAt,
						"planName": planName,
						"groups": groups,
						"captureMethod": captureMethod,
						"captureSource": captureSource,
						"aiCredits": aiCredits,
					})
				}
			}
		}
	}

	// 2. Cursor Snapshots
	if (accountID == 0 && (providerFilter == "" || providerFilter == "all" || providerFilter == "cursor")) || (accountID > 0 && provider == "cursor") {
		var query string
		var args []interface{}
		var conditions []string

		if accountID > 0 {
			conditions = append(conditions, "account_id = ?")
			args = append(args, accountID)
		}
		if sinceStr != "" {
			conditions = append(conditions, "datetime(captured_at) >= datetime(?)")
			args = append(args, sinceStr)
		}
		if untilStr != "" {
			conditions = append(conditions, "datetime(captured_at) <= datetime(?)")
			args = append(args, untilStr)
		}

		whereClause := ""
		if len(conditions) > 0 {
			whereClause = "WHERE " + strings.Join(conditions, " AND ")
		}

		query = fmt.Sprintf(`SELECT id, COALESCE(account_id,0), COALESCE(email,''), usage_pct, plan_type, captured_at, capture_method, capture_source
			FROM cursor_snapshots %s
			ORDER BY captured_at DESC LIMIT ?`, whereClause)
		args = append(args, limit)

		rows, err := s.db.Query(query, args...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, accID int64
				var email, planType, capturedAtStr, captureMethod, captureSource string
				var usagePct float64
				if err := rows.Scan(&id, &accID, &email, &usagePct, &planType, &capturedAtStr, &captureMethod, &captureSource); err == nil {
					capturedAt, _ := time.Parse(time.RFC3339, capturedAtStr)
					remainingPct := 100.0 - usagePct
					if remainingPct < 0 {
						remainingPct = 0
					}

					groups := []map[string]interface{}{
						{
							"groupKey": "cursor",
							"displayName": "Cursor Quota",
							"remainingPercent": math.Round(remainingPct),
							"isExhausted": remainingPct <= 0,
							"color": "#00E6FF",
						},
					}

					items = append(items, map[string]interface{}{
						"id": id,
						"provider": "cursor",
						"accountId": accID,
						"email": email,
						"capturedAt": capturedAt,
						"planName": "Cursor " + planType,
						"groups": groups,
						"captureMethod": captureMethod,
						"captureSource": captureSource,
					})
				}
			}
		}
	}

	// 3. Codex Snapshots
	if (accountID == 0 && (providerFilter == "" || providerFilter == "all" || providerFilter == "codex")) || (accountID > 0 && provider == "codex") {
		var query string
		var args []interface{}
		var conditions []string

		if accountID > 0 {
			conditions = append(conditions, "owner_account_id = ?")
			args = append(args, accountID)
		}
		if sinceStr != "" {
			conditions = append(conditions, "datetime(captured_at) >= datetime(?)")
			args = append(args, sinceStr)
		}
		if untilStr != "" {
			conditions = append(conditions, "datetime(captured_at) <= datetime(?)")
			args = append(args, untilStr)
		}

		whereClause := ""
		if len(conditions) > 0 {
			whereClause = "WHERE " + strings.Join(conditions, " AND ")
		}

		query = fmt.Sprintf(`SELECT id, COALESCE(owner_account_id,0), COALESCE(email,''), five_hour_pct, seven_day_pct, plan_type, captured_at, capture_method, capture_source
			FROM codex_snapshots %s
			ORDER BY captured_at DESC LIMIT ?`, whereClause)
		args = append(args, limit)

		rows, err := s.db.Query(query, args...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, accID int64
				var email, planType, capturedAtStr, captureMethod, captureSource string
				var fiveHourPct float64
				var sevenDayPct sql.NullFloat64
				if err := rows.Scan(&id, &accID, &email, &fiveHourPct, &sevenDayPct, &planType, &capturedAtStr, &captureMethod, &captureSource); err == nil {
					capturedAt, _ := time.Parse(time.RFC3339, capturedAtStr)
					
					fiveHourRemaining := 100.0 - fiveHourPct
					if fiveHourRemaining < 0 {
						fiveHourRemaining = 0
					}

					groups := []map[string]interface{}{
						{
							"groupKey": "codex_5h",
							"displayName": "Codex 5-Hour",
							"remainingPercent": math.Round(fiveHourRemaining),
							"isExhausted": fiveHourRemaining <= 0,
							"color": "#9B51E0",
						},
					}

					if sevenDayPct.Valid {
						sevenDayRemaining := 100.0 - sevenDayPct.Float64
						if sevenDayRemaining < 0 {
							sevenDayRemaining = 0
						}
						groups = append(groups, map[string]interface{}{
							"groupKey": "codex_7d",
							"displayName": "Codex 7-Day",
							"remainingPercent": math.Round(sevenDayRemaining),
							"isExhausted": sevenDayRemaining <= 0,
							"color": "#BB6BD9",
						})
					}

					items = append(items, map[string]interface{}{
						"id": id,
						"provider": "codex",
						"accountId": accID,
						"email": email,
						"capturedAt": capturedAt,
						"planName": planType,
						"groups": groups,
						"captureMethod": captureMethod,
						"captureSource": captureSource,
					})
				}
			}
		}
	}

	// 4. Copilot Snapshots
	if (accountID == 0 && (providerFilter == "" || providerFilter == "all" || providerFilter == "copilot")) || (accountID > 0 && provider == "copilot") {
		var query string
		var args []interface{}
		var conditions []string

		if accountID > 0 {
			conditions = append(conditions, "account_id = ?")
			args = append(args, accountID)
		}
		if sinceStr != "" {
			conditions = append(conditions, "datetime(captured_at) >= datetime(?)")
			args = append(args, sinceStr)
		}
		if untilStr != "" {
			conditions = append(conditions, "datetime(captured_at) <= datetime(?)")
			args = append(args, untilStr)
		}

		whereClause := ""
		if len(conditions) > 0 {
			whereClause = "WHERE " + strings.Join(conditions, " AND ")
		}

		query = fmt.Sprintf(`SELECT id, COALESCE(account_id,0), COALESCE(email,''), plan, premium_pct, captured_at, capture_method, capture_source
			FROM copilot_snapshots %s
			ORDER BY captured_at DESC LIMIT ?`, whereClause)
		args = append(args, limit)

		rows, err := s.db.Query(query, args...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, accID int64
				var email, plan, capturedAtStr, captureMethod, captureSource string
				var premiumPct float64
				if err := rows.Scan(&id, &accID, &email, &plan, &premiumPct, &capturedAtStr, &captureMethod, &captureSource); err == nil {
					capturedAt, _ := time.Parse(time.RFC3339, capturedAtStr)
					remainingPct := 100.0 - premiumPct
					if remainingPct < 0 {
						remainingPct = 0
					}

					groups := []map[string]interface{}{
						{
							"groupKey": "copilot",
							"displayName": "Copilot Premium",
							"remainingPercent": math.Round(remainingPct),
							"isExhausted": remainingPct <= 0,
							"color": "#2EA44F",
						},
					}

					items = append(items, map[string]interface{}{
						"id": id,
						"provider": "copilot",
						"accountId": accID,
						"email": email,
						"capturedAt": capturedAt,
						"planName": "Copilot " + plan,
						"groups": groups,
						"captureMethod": captureMethod,
						"captureSource": captureSource,
					})
				}
			}
		}
	}

	// Sort chronological items: newest first
	sort.Slice(items, func(i, j int) bool {
		tI := items[i]["capturedAt"].(time.Time)
		tJ := items[j]["capturedAt"].(time.Time)
		return tI.After(tJ)
	})

	// Slice to limit
	if len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}
