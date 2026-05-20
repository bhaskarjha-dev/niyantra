package store

import (
	"database/sql"
	"encoding/json"
	"time"
)

// CopilotSnapshot represents a stored GitHub Copilot usage snapshot.
type CopilotSnapshot struct {
	ID            int64     `json:"id"`
	AccountID     int64     `json:"accountId"`
	Email         string    `json:"email,omitempty"`
	Username      string    `json:"username,omitempty"`
	Plan          string    `json:"plan"`          // Pro/Pro+/Free/Business/Enterprise
	PremiumPct    float64   `json:"premiumPct"`    // premium interactions: % used (0-100)
	ChatPct       float64   `json:"chatPct"`       // chat: % used (0-100)
	HasPremium    bool      `json:"hasPremium"`     // whether premium data available
	HasChat       bool      `json:"hasChat"`        // whether chat data available
	CapturedAt    time.Time `json:"capturedAt"`
	CaptureMethod string    `json:"captureMethod"`
	CaptureSource string    `json:"captureSource"`
}

// UsagePct returns the primary usage percentage.
// Premium interactions is primary; falls back to chat.
func (s *CopilotSnapshot) UsagePct() float64 {
	if s.HasPremium {
		return s.PremiumPct
	}
	if s.HasChat {
		return s.ChatPct
	}
	return 0
}

// InsertCopilotSnapshot stores a GitHub Copilot usage snapshot.
func (s *Store) InsertCopilotSnapshot(snap *CopilotSnapshot) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO copilot_snapshots
			(account_id, email, username, plan, premium_pct, chat_pct,
			 models_json, captured_at, capture_method, capture_source)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), ?, ?)`,
		nullableAccountID(snap.AccountID), snap.Email, snap.Username, snap.Plan,
		snap.PremiumPct, snap.ChatPct,
		formatCopilotModelsJSON(snap),
		snap.CaptureMethod, snap.CaptureSource,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// formatCopilotModelsJSON builds a JSON string with extended billing info.
func formatCopilotModelsJSON(snap *CopilotSnapshot) string {
	obj := map[string]interface{}{
		"hasPremium": snap.HasPremium,
		"hasChat":    snap.HasChat,
	}
	data, _ := json.Marshal(obj)
	return string(data)
}

// LatestCopilotSnapshots returns the most recent Copilot snapshot for each account.
func (s *Store) LatestCopilotSnapshots() ([]*CopilotSnapshot, error) {
	rows, err := s.db.Query(`
		SELECT c1.id, COALESCE(c1.account_id,0), COALESCE(c1.email,''), COALESCE(c1.username,''),
		       COALESCE(c1.plan,''), c1.premium_pct, c1.chat_pct, COALESCE(c1.models_json,'{}'),
		       c1.captured_at, c1.capture_method, c1.capture_source
		FROM copilot_snapshots c1
		INNER JOIN (
			SELECT COALESCE(NULLIF(email, ''), CAST(account_id AS TEXT)) as grouping_key, MAX(captured_at) as max_captured_at
			FROM copilot_snapshots
			GROUP BY COALESCE(NULLIF(email, ''), CAST(account_id AS TEXT))
		) c2 ON COALESCE(NULLIF(c1.email, ''), CAST(c1.account_id AS TEXT)) = c2.grouping_key AND c1.captured_at = c2.max_captured_at
		ORDER BY c1.captured_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snaps []*CopilotSnapshot
	for rows.Next() {
		snap := &CopilotSnapshot{}
		var capturedAt sql.NullString
		var modelsJSON string
		if err := rows.Scan(
			&snap.ID, &snap.AccountID, &snap.Email, &snap.Username,
			&snap.Plan, &snap.PremiumPct, &snap.ChatPct, &modelsJSON,
			&capturedAt, &snap.CaptureMethod, &snap.CaptureSource,
		); err != nil {
			return nil, err
		}
		if capturedAt.Valid {
			snap.CapturedAt, _ = time.Parse(time.RFC3339, capturedAt.String)
			if snap.CapturedAt.IsZero() {
				snap.CapturedAt, _ = time.Parse("2006-01-02 15:04:05", capturedAt.String)
			}
		}
		parseCopilotModelsJSON(snap, modelsJSON)
		snaps = append(snaps, snap)
	}

	return snaps, rows.Err()
}

// parseCopilotModelsJSON restores extended fields from the stored JSON.
func parseCopilotModelsJSON(snap *CopilotSnapshot, raw string) {
	var obj map[string]interface{}
	if json.Unmarshal([]byte(raw), &obj) != nil {
		return
	}
	if v, ok := obj["hasPremium"].(bool); ok {
		snap.HasPremium = v
	}
	if v, ok := obj["hasChat"].(bool); ok {
		snap.HasChat = v
	}
}

// RecentCopilotSnapshots returns the last N copilot snapshots.
func (s *Store) RecentCopilotSnapshots(limit int) ([]*CopilotSnapshot, error) {
	rows, err := s.db.Query(`
		SELECT id, COALESCE(account_id,0), COALESCE(email,''), COALESCE(username,''),
		       COALESCE(plan,''), premium_pct, chat_pct,
		       COALESCE(models_json,'{}'),
		       captured_at, capture_method, capture_source
		FROM copilot_snapshots
		ORDER BY captured_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snaps []*CopilotSnapshot
	for rows.Next() {
		snap := &CopilotSnapshot{}
		var capturedAt sql.NullString
		var modelsJSON string
		if err := rows.Scan(
			&snap.ID, &snap.AccountID, &snap.Email, &snap.Username,
			&snap.Plan, &snap.PremiumPct, &snap.ChatPct,
			&modelsJSON,
			&capturedAt, &snap.CaptureMethod, &snap.CaptureSource,
		); err != nil {
			return nil, err
		}
		if capturedAt.Valid {
			snap.CapturedAt, _ = time.Parse(time.RFC3339, capturedAt.String)
		}
		parseCopilotModelsJSON(snap, modelsJSON)
		snaps = append(snaps, snap)
	}
	return snaps, rows.Err()
}

// CopilotSnapshotCount returns the total number of Copilot snapshots.
func (s *Store) CopilotSnapshotCount() int {
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM copilot_snapshots`).Scan(&n)
	return n
}
