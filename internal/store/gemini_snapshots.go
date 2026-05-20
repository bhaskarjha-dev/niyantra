package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// GeminiSnapshot represents a stored Gemini CLI usage snapshot.
type GeminiSnapshot struct {
	ID            int64     `json:"id"`
	AccountID     int64     `json:"accountId"`
	Email         string    `json:"email,omitempty"`
	Tier          string    `json:"tier"`       // standard/enterprise/unknown
	OverallPct    float64   `json:"overallPct"` // arithmetic mean usage % across reported model buckets
	ModelsJSON    string    `json:"modelsJson"` // per-model quota breakdown
	ProjectID     string    `json:"projectId"`  // cloudaicompanionProject
	CapturedAt    time.Time `json:"capturedAt"`
	CaptureMethod string    `json:"captureMethod"` // auto/manual
	CaptureSource string    `json:"captureSource"` // server/ui
}

// InsertGeminiSnapshot stores a Gemini CLI usage snapshot.
func (s *Store) InsertGeminiSnapshot(snap *GeminiSnapshot) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO gemini_snapshots
			(account_id, email, tier, overall_pct, models_json, project_id,
			 captured_at, capture_method, capture_source)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'), ?, ?)`,
		nullableAccountID(snap.AccountID), snap.Email, snap.Tier, snap.OverallPct,
		snap.ModelsJSON, snap.ProjectID,
		snap.CaptureMethod, snap.CaptureSource,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// LatestGeminiSnapshots returns the most recent Gemini snapshot for each account.
func (s *Store) LatestGeminiSnapshots() ([]*GeminiSnapshot, error) {
	rows, err := s.db.Query(`
		SELECT g1.id, COALESCE(g1.account_id,0), COALESCE(g1.email,''), COALESCE(g1.tier,''), g1.overall_pct,
		       COALESCE(g1.models_json,'[]'), COALESCE(g1.project_id,''), g1.captured_at, g1.capture_method, g1.capture_source
		FROM gemini_snapshots g1
		INNER JOIN (
			SELECT COALESCE(NULLIF(email, ''), CAST(account_id AS TEXT)) as grouping_key, MAX(captured_at) as max_captured_at
			FROM gemini_snapshots
			GROUP BY COALESCE(NULLIF(email, ''), CAST(account_id AS TEXT))
		) g2 ON COALESCE(NULLIF(g1.email, ''), CAST(g1.account_id AS TEXT)) = g2.grouping_key AND g1.captured_at = g2.max_captured_at
		ORDER BY g1.captured_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snaps []*GeminiSnapshot
	for rows.Next() {
		snap := &GeminiSnapshot{}
		var capturedAt sql.NullString
		if err := rows.Scan(
			&snap.ID, &snap.AccountID, &snap.Email, &snap.Tier,
			&snap.OverallPct, &snap.ModelsJSON, &snap.ProjectID,
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
		snaps = append(snaps, snap)
	}

	return snaps, rows.Err()
}

// RecentGeminiSnapshots returns the last N Gemini snapshots.
func (s *Store) RecentGeminiSnapshots(limit int) ([]*GeminiSnapshot, error) {
	rows, err := s.db.Query(`
		SELECT id, COALESCE(account_id,0), COALESCE(email,''), COALESCE(tier,''),
		       overall_pct, COALESCE(models_json,'[]'), COALESCE(project_id,''),
		       captured_at, capture_method, capture_source
		FROM gemini_snapshots
		ORDER BY captured_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snaps []*GeminiSnapshot
	for rows.Next() {
		snap := &GeminiSnapshot{}
		var capturedAt sql.NullString
		if err := rows.Scan(
			&snap.ID, &snap.AccountID, &snap.Email, &snap.Tier,
			&snap.OverallPct, &snap.ModelsJSON, &snap.ProjectID,
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
		snaps = append(snaps, snap)
	}
	return snaps, rows.Err()
}

// RecentGeminiSnapshotsInWindow returns Gemini snapshots from the last `window` duration.
func (s *Store) RecentGeminiSnapshotsInWindow(window time.Duration) ([]*GeminiSnapshot, error) {
	cutoff := time.Now().Add(-window).UTC().Format("2006-01-02 15:04:05")
	rows, err := s.db.Query(`
		SELECT id, COALESCE(account_id,0), COALESCE(email,''), COALESCE(tier,''),
		       overall_pct, COALESCE(models_json,'[]'), COALESCE(project_id,''),
		       captured_at, capture_method, capture_source
		FROM gemini_snapshots
		WHERE captured_at >= ?
		ORDER BY captured_at ASC`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snaps []*GeminiSnapshot
	for rows.Next() {
		snap := &GeminiSnapshot{}
		var capturedAt sql.NullString
		if err := rows.Scan(
			&snap.ID, &snap.AccountID, &snap.Email, &snap.Tier,
			&snap.OverallPct, &snap.ModelsJSON, &snap.ProjectID,
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
		snaps = append(snaps, snap)
	}
	return snaps, rows.Err()
}

// FormatGeminiModelsJSON converts model quota data to JSON string for storage.
func FormatGeminiModelsJSON(models interface{}) string {
	data, err := json.Marshal(models)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// GeminiSnapshotCount returns the total number of Gemini snapshots.
func (s *Store) GeminiSnapshotCount() int {
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM gemini_snapshots`).Scan(&count)
	return count
}

// DeleteGeminiSnapshotsForAccount deletes all Gemini snapshots for an account.
func (s *Store) DeleteGeminiSnapshotsForAccount(accountID int64) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM gemini_snapshots WHERE account_id = ?`, accountID)
	if err != nil {
		return 0, fmt.Errorf("store: delete gemini snapshots: %w", err)
	}
	return res.RowsAffected()
}
