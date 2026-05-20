package store

import (
	"database/sql"
	"time"
)

// CodexSnapshot represents a stored Codex usage snapshot.
// AccountID is the provider-native OpenAI org/account identifier. OwnerAccountID
// links the snapshot back to Niyantra's local accounts table when the codex
// account can be identified by email.
type CodexSnapshot struct {
	ID             int64      `json:"id"`
	AccountID      string     `json:"accountId"` // OpenAI org/account ID
	OwnerAccountID int64      `json:"ownerAccountId,omitempty"`
	Email          string     `json:"email,omitempty"` // User email from OIDC id_token
	FiveHourPct    float64    `json:"fiveHourPct"`
	SevenDayPct    *float64   `json:"sevenDayPct"`
	CodeReviewPct  *float64   `json:"codeReviewPct"`
	FiveHourReset  *time.Time `json:"fiveHourReset"`
	SevenDayReset  *time.Time `json:"sevenDayReset"`
	PlanType       string     `json:"planType"`
	CreditsBalance *float64   `json:"creditsBalance"`
	CapturedAt     time.Time  `json:"capturedAt"`
	CaptureMethod  string     `json:"captureMethod"`
	CaptureSource  string     `json:"captureSource"`
}

// InsertCodexSnapshot stores a Codex usage snapshot.
func (s *Store) InsertCodexSnapshot(snap *CodexSnapshot) (int64, error) {
	var fiveReset, sevenReset *string
	if snap.FiveHourReset != nil {
		t := snap.FiveHourReset.UTC().Format(time.RFC3339)
		fiveReset = &t
	}
	if snap.SevenDayReset != nil {
		t := snap.SevenDayReset.UTC().Format(time.RFC3339)
		sevenReset = &t
	}

	res, err := s.db.Exec(`
		INSERT INTO codex_snapshots
			(account_id, owner_account_id, email, five_hour_pct, seven_day_pct, code_review_pct,
			 five_hour_reset, seven_day_reset, plan_type, credits_balance,
			 captured_at, capture_method, capture_source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), ?, ?)`,
		snap.AccountID, nullableAccountID(snap.OwnerAccountID), snap.Email, snap.FiveHourPct, snap.SevenDayPct, snap.CodeReviewPct,
		fiveReset, sevenReset, snap.PlanType, snap.CreditsBalance,
		snap.CaptureMethod, snap.CaptureSource,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// LatestCodexSnapshots returns the most recent Codex snapshot for each account.
func (s *Store) LatestCodexSnapshots() ([]*CodexSnapshot, error) {
	rows, err := s.db.Query(`
		SELECT c1.id, c1.account_id, COALESCE(c1.owner_account_id,0), COALESCE(c1.email,''), c1.five_hour_pct, c1.seven_day_pct, c1.code_review_pct,
		       c1.five_hour_reset, c1.seven_day_reset, c1.plan_type, c1.credits_balance,
		       c1.captured_at, c1.capture_method, c1.capture_source
		FROM codex_snapshots c1
		INNER JOIN (
			SELECT COALESCE(NULLIF(email, ''), account_id) as grouping_key, MAX(captured_at) as max_captured_at
			FROM codex_snapshots
			GROUP BY COALESCE(NULLIF(email, ''), account_id)
		) c2 ON COALESCE(NULLIF(c1.email, ''), c1.account_id) = c2.grouping_key AND c1.captured_at = c2.max_captured_at
		ORDER BY c1.captured_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snaps []*CodexSnapshot
	for rows.Next() {
		snap := &CodexSnapshot{}
		var fiveReset, sevenReset, capturedAt sql.NullString
		if err := rows.Scan(
			&snap.ID, &snap.AccountID, &snap.OwnerAccountID, &snap.Email, &snap.FiveHourPct,
			&snap.SevenDayPct, &snap.CodeReviewPct,
			&fiveReset, &sevenReset,
			&snap.PlanType, &snap.CreditsBalance,
			&capturedAt, &snap.CaptureMethod, &snap.CaptureSource,
		); err != nil {
			return nil, err
		}

		if capturedAt.Valid {
			snap.CapturedAt, _ = time.Parse(time.RFC3339, capturedAt.String)
		}
		if fiveReset.Valid {
			t, _ := time.Parse(time.RFC3339, fiveReset.String)
			snap.FiveHourReset = &t
		}
		if sevenReset.Valid {
			t, _ := time.Parse(time.RFC3339, sevenReset.String)
			snap.SevenDayReset = &t
		}
		snaps = append(snaps, snap)
	}

	return snaps, rows.Err()
}

// CodexHistory returns Codex snapshots since the given time.
func (s *Store) CodexHistory(since time.Time) ([]*CodexSnapshot, error) {
	rows, err := s.db.Query(`
		SELECT id, account_id, COALESCE(owner_account_id,0), COALESCE(email,''), five_hour_pct, seven_day_pct, code_review_pct,
		       five_hour_reset, seven_day_reset, plan_type, credits_balance,
		       captured_at, capture_method, capture_source
		FROM codex_snapshots
		WHERE captured_at >= ?
		ORDER BY captured_at ASC`,
		since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snaps []*CodexSnapshot
	for rows.Next() {
		snap := &CodexSnapshot{}
		var fiveReset, sevenReset, capturedAt sql.NullString
		if err := rows.Scan(
			&snap.ID, &snap.AccountID, &snap.OwnerAccountID, &snap.Email, &snap.FiveHourPct,
			&snap.SevenDayPct, &snap.CodeReviewPct,
			&fiveReset, &sevenReset,
			&snap.PlanType, &snap.CreditsBalance,
			&capturedAt, &snap.CaptureMethod, &snap.CaptureSource,
		); err != nil {
			return nil, err
		}
		if capturedAt.Valid {
			snap.CapturedAt, _ = time.Parse(time.RFC3339, capturedAt.String)
		}
		if fiveReset.Valid {
			t, _ := time.Parse(time.RFC3339, fiveReset.String)
			snap.FiveHourReset = &t
		}
		if sevenReset.Valid {
			t, _ := time.Parse(time.RFC3339, sevenReset.String)
			snap.SevenDayReset = &t
		}
		snaps = append(snaps, snap)
	}
	return snaps, rows.Err()
}
