package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"database/sql"

	"github.com/bhaskarjha-com/niyantra/internal/core"
)

// StoredSnapshot represents a database record in the snapshots_v2 table.
type StoredSnapshot struct {
	ID            string     `json:"id"`
	Provider      string     `json:"provider"`
	AccountID     string     `json:"accountId"`
	CapturedAt    time.Time  `json:"capturedAt"`
	Email         string     `json:"email"`
	OverallPct    float64    `json:"overallPct"`
	PlanTier      string     `json:"planTier"`
	CostUSD       float64    `json:"costUsd"`
	ResetAt       *time.Time `json:"resetAt,omitempty"`
	ResetType     string     `json:"resetType,omitempty"`
	DataJSON      string     `json:"dataJson"`
	ModelsJSON    string     `json:"modelsJson"`
	CaptureMethod string     `json:"captureMethod"`
	CaptureSource string     `json:"captureSource"`
	MachineID     string     `json:"machineId,omitempty"`
	Version       int        `json:"version"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
	OwnerID       string     `json:"ownerId,omitempty"`
	SyncedAt      *time.Time `json:"syncedAt,omitempty"`
}

// HistoryOpts provides filtering options for the history query.
type HistoryOpts struct {
	Provider  string
	AccountID string
	Since     *time.Time
	Until     *time.Time
	Limit     int
}

// SaveSnapshot stores a provider-agnostic snapshot.
func (s *Store) SaveSnapshot(ctx context.Context, snap *core.Snapshot) error {
	capturedAt := time.Now().UTC()
	ulid := core.NewULIDAt(capturedAt)

	dataJSON := "{}"
	if snap.Data != nil {
		if b, err := json.Marshal(snap.Data); err == nil {
			dataJSON = string(b)
		}
	}
	modelsJSON := "[]"
	if snap.Models != nil {
		if b, err := json.Marshal(snap.Models); err == nil {
			modelsJSON = string(b)
		}
	}

	var resetAtVal any
	if snap.ResetAt != nil {
		resetAtVal = snap.ResetAt.UTC().Format(time.RFC3339)
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO snapshots_v2 (
			id, provider, account_id, captured_at, email, overall_pct, plan_tier, cost_usd, reset_at, reset_type, data_json, models_json, capture_method, capture_source
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		ulid,
		snap.Provider,
		snap.AccountID,
		capturedAt.Format(time.RFC3339),
		snap.Email,
		snap.OverallPct,
		snap.PlanTier,
		snap.CostUSD,
		resetAtVal,
		snap.ResetType,
		dataJSON,
		modelsJSON,
		snap.CaptureMethod,
		snap.CaptureSource,
	)
	if err != nil {
		return fmt.Errorf("store: save snapshot v2: %w", err)
	}
	return nil
}

// LatestByProvider returns the most recent snapshot for a provider+account.
func (s *Store) LatestByProvider(ctx context.Context, provider, accountID string) (*StoredSnapshot, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, provider, account_id, captured_at, email, overall_pct, plan_tier, cost_usd, reset_at, reset_type, data_json, models_json, capture_method, capture_source, machine_id, version, updated_at, deleted_at, owner_id, synced_at
		FROM snapshots_v2
		WHERE provider = ? AND account_id = ? AND deleted_at IS NULL
		ORDER BY captured_at DESC, id DESC
		LIMIT 1
	`, provider, accountID)

	var snap StoredSnapshot
	var resetAt, deletedAt, syncedAt sql.NullTime

	err := row.Scan(
		&snap.ID, &snap.Provider, &snap.AccountID, &snap.CapturedAt, &snap.Email,
		&snap.OverallPct, &snap.PlanTier, &snap.CostUSD, &resetAt, &snap.ResetType,
		&snap.DataJSON, &snap.ModelsJSON, &snap.CaptureMethod, &snap.CaptureSource,
		&snap.MachineID, &snap.Version, &snap.UpdatedAt, &deletedAt, &snap.OwnerID, &syncedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: latest by provider: %w", err)
	}

	if resetAt.Valid {
		snap.ResetAt = &resetAt.Time
	}
	if deletedAt.Valid {
		snap.DeletedAt = &deletedAt.Time
	}
	if syncedAt.Valid {
		snap.SyncedAt = &syncedAt.Time
	}

	return &snap, nil
}

// LatestAll returns the most recent snapshot for every account across all providers.
func (s *Store) LatestAll(ctx context.Context) ([]StoredSnapshot, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.provider, s.account_id, s.captured_at, s.email, s.overall_pct, s.plan_tier, s.cost_usd, s.reset_at, s.reset_type, s.data_json, s.models_json, s.capture_method, s.capture_source, s.machine_id, s.version, s.updated_at, s.deleted_at, s.owner_id, s.synced_at
		FROM snapshots_v2 s
		WHERE s.deleted_at IS NULL AND s.id = (
			SELECT s2.id
			FROM snapshots_v2 s2
			WHERE s2.provider = s.provider AND s2.account_id = s.account_id AND s2.deleted_at IS NULL
			ORDER BY s2.captured_at DESC, s2.id DESC
			LIMIT 1
		)
		ORDER BY s.captured_at DESC, s.id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("store: query latest all v2: %w", err)
	}
	defer rows.Close()

	var snaps []StoredSnapshot
	for rows.Next() {
		var snap StoredSnapshot
		var resetAt, deletedAt, syncedAt sql.NullTime

		err := rows.Scan(
			&snap.ID, &snap.Provider, &snap.AccountID, &snap.CapturedAt, &snap.Email,
			&snap.OverallPct, &snap.PlanTier, &snap.CostUSD, &resetAt, &snap.ResetType,
			&snap.DataJSON, &snap.ModelsJSON, &snap.CaptureMethod, &snap.CaptureSource,
			&snap.MachineID, &snap.Version, &snap.UpdatedAt, &deletedAt, &snap.OwnerID, &syncedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: scan latest all v2: %w", err)
		}

		if resetAt.Valid {
			snap.ResetAt = &resetAt.Time
		}
		if deletedAt.Valid {
			snap.DeletedAt = &deletedAt.Time
		}
		if syncedAt.Valid {
			snap.SyncedAt = &syncedAt.Time
		}

		snaps = append(snaps, snap)
	}

	return snaps, nil
}

// History returns snapshots within a time range, with optional provider/account filters.
func (s *Store) History(ctx context.Context, opts HistoryOpts) ([]StoredSnapshot, error) {
	var conditions []string
	var args []any

	conditions = append(conditions, "deleted_at IS NULL")

	if opts.Provider != "" {
		conditions = append(conditions, "provider = ?")
		args = append(args, opts.Provider)
	}
	if opts.AccountID != "" {
		conditions = append(conditions, "account_id = ?")
		args = append(args, opts.AccountID)
	}
	if opts.Since != nil {
		conditions = append(conditions, "captured_at >= ?")
		args = append(args, opts.Since.UTC().Format(time.RFC3339))
	}
	if opts.Until != nil {
		conditions = append(conditions, "captured_at <= ?")
		args = append(args, opts.Until.UTC().Format(time.RFC3339))
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, provider, account_id, captured_at, email, overall_pct, plan_tier, cost_usd, reset_at, reset_type, data_json, models_json, capture_method, capture_source, machine_id, version, updated_at, deleted_at, owner_id, synced_at
		FROM snapshots_v2
		%s
		ORDER BY captured_at DESC, id DESC
		LIMIT ?
	`, whereClause)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query history v2: %w", err)
	}
	defer rows.Close()

	var snaps []StoredSnapshot
	for rows.Next() {
		var snap StoredSnapshot
		var resetAt, deletedAt, syncedAt sql.NullTime

		err := rows.Scan(
			&snap.ID, &snap.Provider, &snap.AccountID, &snap.CapturedAt, &snap.Email,
			&snap.OverallPct, &snap.PlanTier, &snap.CostUSD, &resetAt, &snap.ResetType,
			&snap.DataJSON, &snap.ModelsJSON, &snap.CaptureMethod, &snap.CaptureSource,
			&snap.MachineID, &snap.Version, &snap.UpdatedAt, &deletedAt, &snap.OwnerID, &syncedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("store: scan history v2: %w", err)
		}

		if resetAt.Valid {
			snap.ResetAt = &resetAt.Time
		}
		if deletedAt.Valid {
			snap.DeletedAt = &deletedAt.Time
		}
		if syncedAt.Valid {
			snap.SyncedAt = &syncedAt.Time
		}

		snaps = append(snaps, snap)
	}

	return snaps, nil
}

// HeatmapUnified returns daily snapshot counts for the heatmap chart.
func (s *Store) HeatmapUnified(ctx context.Context, days int) ([]HeatmapDay, error) {
	if days <= 0 {
		days = 365
	}

	cutoff := fmt.Sprintf("-%d days", days)

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			day,
			SUM(CASE WHEN provider = 'antigravity' THEN 1 ELSE 0 END) as ag,
			SUM(CASE WHEN provider = 'claude' THEN 1 ELSE 0 END) as cl,
			SUM(CASE WHEN provider = 'codex' THEN 1 ELSE 0 END) as cx,
			SUM(CASE WHEN provider = 'cursor' THEN 1 ELSE 0 END) as cr,
			SUM(CASE WHEN provider = 'copilot' THEN 1 ELSE 0 END) as cp,
			SUM(CASE WHEN provider LIKE 'plugin_%' THEN 1 ELSE 0 END) as pl
		FROM (
			SELECT date(captured_at) as day, provider
			FROM snapshots_v2
			WHERE captured_at >= datetime('now', ?) AND deleted_at IS NULL
		)
		GROUP BY day
		ORDER BY day ASC
	`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("store: heatmap unified query: %w", err)
	}
	defer rows.Close()

	var result []HeatmapDay
	for rows.Next() {
		var d HeatmapDay
		if err := rows.Scan(&d.Date, &d.Antigravity, &d.Claude, &d.Codex, &d.Cursor, &d.Copilot, &d.Plugin); err != nil {
			return nil, fmt.Errorf("store: scan heatmap unified row: %w", err)
		}
		d.Count = d.Antigravity + d.Claude + d.Codex + d.Cursor + d.Copilot + d.Plugin
		result = append(result, d)
	}
	return result, rows.Err()
}
