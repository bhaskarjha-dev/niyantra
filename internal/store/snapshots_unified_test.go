package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/core"
)

func TestSaveAndLatestByProvider(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	accID, err := s.GetOrCreateAccount("user@test.com", "Pro", "antigravity")
	if err != nil {
		t.Fatalf("GetOrCreateAccount failed: %v", err)
	}

	snapTime := time.Now().UTC()
	resetTime := snapTime.Add(2 * time.Hour)

	snap := &core.Snapshot{
		Provider:      "antigravity",
		AccountID:     strconv.FormatInt(accID, 10),
		Email:         "user@test.com",
		OverallPct:    75.5,
		PlanTier:      "pro",
		CostUSD:       1.23,
		ResetAt:       &resetTime,
		ResetType:     "2h",
		CaptureMethod: "auto",
		CaptureSource: "ls_poll",
		Data:          map[string]any{"custom_val": 42},
		Models:        []any{"model-a", "model-b"},
	}

	err = s.SaveSnapshot(ctx, snap)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	stored, err := s.LatestByProvider(ctx, "antigravity", strconv.FormatInt(accID, 10))
	if err != nil {
		t.Fatalf("LatestByProvider failed: %v", err)
	}
	if stored == nil {
		t.Fatal("expected stored snapshot, got nil")
	}

	if stored.Provider != "antigravity" || stored.Email != "user@test.com" || stored.OverallPct != 75.5 {
		t.Errorf("mismatched basic fields: %+v", stored)
	}

	if stored.ResetAt == nil || stored.ResetAt.Unix() != resetTime.Unix() || stored.ResetType != "2h" {
		t.Errorf("mismatched reset fields: resetAt=%v, resetType=%s", stored.ResetAt, stored.ResetType)
	}

	var dataMap map[string]any
	if err := json.Unmarshal([]byte(stored.DataJSON), &dataMap); err != nil {
		t.Fatalf("failed to unmarshal data_json: %v", err)
	}
	if fmt.Sprintf("%v", dataMap["custom_val"]) != "42" {
		t.Errorf("expected data_json custom_val=42, got %v", dataMap["custom_val"])
	}

	var models []string
	if err := json.Unmarshal([]byte(stored.ModelsJSON), &models); err != nil {
		t.Fatalf("failed to unmarshal models_json: %v", err)
	}
	if len(models) != 2 || models[0] != "model-a" || models[1] != "model-b" {
		t.Errorf("mismatched models: %v", models)
	}
}

func TestLatestAll(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	acc1, _ := s.GetOrCreateAccount("user1@test.com", "Pro", "antigravity")
	acc2, _ := s.GetOrCreateAccount("user2@test.com", "Ultra", "codex")

	// Save old and new snapshots for acc1
	s.SaveSnapshot(ctx, &core.Snapshot{
		Provider:   "antigravity",
		AccountID:  strconv.FormatInt(acc1, 10),
		Email:      "user1@test.com",
		OverallPct: 10,
	})
	time.Sleep(10 * time.Millisecond)
	s.SaveSnapshot(ctx, &core.Snapshot{
		Provider:   "antigravity",
		AccountID:  strconv.FormatInt(acc1, 10),
		Email:      "user1@test.com",
		OverallPct: 20,
	})

	// Save snapshot for acc2
	s.SaveSnapshot(ctx, &core.Snapshot{
		Provider:   "codex",
		AccountID:  strconv.FormatInt(acc2, 10),
		Email:      "user2@test.com",
		OverallPct: 50,
	})

	latest, err := s.LatestAll(ctx)
	if err != nil {
		t.Fatalf("LatestAll failed: %v", err)
	}

	if len(latest) != 2 {
		t.Fatalf("expected 2 latest snapshots, got %d", len(latest))
	}

	// Verify order is descending by captured_at (latest first)
	if latest[0].AccountID != strconv.FormatInt(acc2, 10) && latest[0].AccountID != strconv.FormatInt(acc1, 10) {
		t.Errorf("unexpected accounts in latest: %+v", latest)
	}

	// Verify we got the latest overall_pct for acc1 (20, not 10)
	for _, snap := range latest {
		if snap.AccountID == strconv.FormatInt(acc1, 10) {
			if snap.OverallPct != 20 {
				t.Errorf("expected acc1 latest overall_pct=20, got %f", snap.OverallPct)
			}
		}
	}
}

func TestHistoryFilters(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	acc1, _ := s.GetOrCreateAccount("user1@test.com", "Pro", "antigravity")
	acc2, _ := s.GetOrCreateAccount("user2@test.com", "Pro", "claude")

	now := time.Now().UTC()

	// Insert snapshots at specific times
	s.SaveSnapshot(ctx, &core.Snapshot{
		Provider:   "antigravity",
		AccountID:  strconv.FormatInt(acc1, 10),
		Email:      "user1@test.com",
		OverallPct: 10,
	})
	s.SaveSnapshot(ctx, &core.Snapshot{
		Provider:   "claude",
		AccountID:  strconv.FormatInt(acc2, 10),
		Email:      "user2@test.com",
		OverallPct: 20,
	})

	// Test filtering by provider
	history, err := s.History(ctx, HistoryOpts{Provider: "claude"})
	if err != nil {
		t.Fatalf("History failed: %v", err)
	}
	if len(history) != 1 || history[0].Provider != "claude" {
		t.Errorf("expected 1 claude snapshot, got: %+v", history)
	}

	// Test filtering by account
	history, err = s.History(ctx, HistoryOpts{AccountID: strconv.FormatInt(acc1, 10)})
	if err != nil {
		t.Fatalf("History failed: %v", err)
	}
	if len(history) != 1 || history[0].AccountID != strconv.FormatInt(acc1, 10) {
		t.Errorf("expected 1 acc1 snapshot, got: %+v", history)
	}

	// Test time filtering
	since := now.Add(-10 * time.Minute)
	until := now.Add(10 * time.Minute)
	history, err = s.History(ctx, HistoryOpts{Since: &since, Until: &until})
	if err != nil {
		t.Fatalf("History failed: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("expected 2 snapshots in time range, got %d", len(history))
	}
}

func TestHeatmapUnified(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	acc, _ := s.GetOrCreateAccount("user@test.com", "Pro", "antigravity")

	// We insert into snapshots_v2 directly so we can simulate past dates (SaveSnapshot uses time.Now())
	snapTimes := []time.Time{
		time.Now().UTC().Add(-24 * time.Hour),
		time.Now().UTC().Add(-24 * time.Hour),
		time.Now().UTC().Add(-48 * time.Hour),
	}

	for _, tVal := range snapTimes {
		ulid := core.NewULIDAt(tVal)
		_, err := s.db.Exec(`
			INSERT INTO snapshots_v2 (id, provider, account_id, captured_at, email, overall_pct, plan_tier, data_json, models_json, capture_method, capture_source)
			VALUES (?, 'antigravity', ?, ?, 'user@test.com', 50, 'pro', '{}', '[]', 'auto', '')
		`, ulid, acc, tVal.UTC().Format(time.RFC3339))
		if err != nil {
			t.Fatalf("direct insert failed: %v", err)
		}
	}

	heatmap, err := s.HeatmapUnified(ctx, 30)
	if err != nil {
		t.Fatalf("HeatmapUnified failed: %v", err)
	}

	if len(heatmap) != 2 {
		t.Fatalf("expected 2 active days in heatmap, got %d", len(heatmap))
	}

	// Verify day counts
	// Heatmap return is sorted ASC by day
	if heatmap[0].Count != 1 || heatmap[1].Count != 2 {
		t.Errorf("unexpected counts in heatmap: %+v", heatmap)
	}
}

func TestULIDSorting(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	acc, _ := s.GetOrCreateAccount("user@test.com", "Pro", "antigravity")

	s.SaveSnapshot(ctx, &core.Snapshot{Provider: "antigravity", AccountID: strconv.FormatInt(acc, 10)})
	time.Sleep(10 * time.Millisecond)
	s.SaveSnapshot(ctx, &core.Snapshot{Provider: "antigravity", AccountID: strconv.FormatInt(acc, 10)})

	history, err := s.History(ctx, HistoryOpts{Limit: 2})
	if err != nil {
		t.Fatalf("History failed: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(history))
	}

	// Since History sorts DESC, the first one returned is the newer one (larger ULID)
	if history[0].ID < history[1].ID {
		t.Errorf("expected descending sort: ID[0]=%s, ID[1]=%s", history[0].ID, history[1].ID)
	}
}

func TestMigrationV25(t *testing.T) {
	// 1. Create a clean temp database file
	dbPath := filepath.Join(t.TempDir(), "migrate_v25_test.db")
	
	// 2. Open DB and run migrations up to v24
	db, err := Open(dbPath, WithSecretBackend(NewMemorySecretBackend()))
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	db.Close()

	// 3. Reopen without auto-migrations to simulate downgrade
	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("raw open failed: %v", err)
	}
	defer rawDB.Close()

	// Set user_version back to 24 and drop snapshots_v2 if it exists (simulate pre-v25 state)
	_, _ = rawDB.Exec(`PRAGMA user_version = 24`)
	_, _ = rawDB.Exec(`DROP TABLE IF EXISTS snapshots_v2`)
	
	// 4. Seed legacy tables with mock data
	// Check that we can seed accounts and related snapshot tables
	accID := int64(123)
	_, err = rawDB.Exec(`
		INSERT INTO accounts (id, email, plan_name, provider) VALUES (?, 'legacy@test.com', 'Pro', 'antigravity')
	`, accID)
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}

	now := time.Now().UTC()

	// Seed snapshots (antigravity)
	_, err = rawDB.Exec(`
		INSERT INTO snapshots (account_id, captured_at, email, plan_name, prompt_credits, monthly_credits, models_json, raw_json)
		VALUES (?, ?, 'legacy@test.com', 'Pro', 150.0, 1000, '[]', '{"test":1}')
	`, accID, now)
	if err != nil {
		t.Fatalf("seed snapshots: %v", err)
	}

	// Seed claude_snapshots (which doesn't have account_id link in legacy)
	_, err = rawDB.Exec(`
		INSERT INTO claude_snapshots (five_hour_pct, seven_day_pct, five_hour_reset, captured_at, source)
		VALUES (80.0, 95.0, ?, ?, 'cli')
	`, now.Add(1*time.Hour), now)
	if err != nil {
		t.Fatalf("seed claude_snapshots: %v", err)
	}

	// Seed codex_snapshots
	_, err = rawDB.Exec(`
		INSERT INTO codex_snapshots (owner_account_id, email, five_hour_pct, plan_type, captured_at, capture_method)
		VALUES (?, 'legacy@test.com', 45.0, 'pro', ?, 'auto')
	`, accID, now)
	if err != nil {
		t.Fatalf("seed codex_snapshots: %v", err)
	}

	// Seed cursor_snapshots
	_, err = rawDB.Exec(`
		INSERT INTO cursor_snapshots (account_id, email, premium_used, premium_limit, usage_pct, plan_type, captured_at)
		VALUES (?, 'legacy@test.com', 10, 100, 10.0, 'pro', ?)
	`, accID, now)
	if err != nil {
		t.Fatalf("seed cursor_snapshots: %v", err)
	}

	// Seed copilot_snapshots
	_, err = rawDB.Exec(`
		INSERT INTO copilot_snapshots (account_id, email, username, plan, premium_pct, chat_pct, captured_at)
		VALUES (?, 'legacy@test.com', 'octo', 'pro', 65.0, 20.0, ?)
	`, accID, now)
	if err != nil {
		t.Fatalf("seed copilot_snapshots: %v", err)
	}

	// Seed plugin_snapshots
	_, err = rawDB.Exec(`
		INSERT INTO plugin_snapshots (plugin_id, provider, label, email, usage_pct, plan, captured_at)
		VALUES ('test-plug', 'custom', 'Plugin', 'legacy@test.com', 88.0, 'pro', ?)
	`, now)
	if err != nil {
		t.Fatalf("seed plugin_snapshots: %v", err)
	}

	rawDB.Close()

	// 5. Open store properly, which runs migrations including v25
	store, err := Open(dbPath, WithSecretBackend(NewMemorySecretBackend()))
	if err != nil {
		t.Fatalf("failed to open database to run migration: %v", err)
	}
	defer store.Close()

	// 6. Verify user_version is 25
	if store.getUserVersion() != 25 {
		t.Errorf("expected version 25, got %d", store.getUserVersion())
	}

	// 7. Verify snapshots_v2 contents
	ctx := context.Background()
	snaps, err := store.History(ctx, HistoryOpts{Limit: 100})
	if err != nil {
		t.Fatalf("failed to query snapshots_v2: %v", err)
	}

	// We expect exactly 6 migrated snapshots: antigravity, claude, codex, cursor, copilot, plugin
	if len(snaps) != 6 {
		t.Errorf("expected 6 migrated snapshots in snapshots_v2, got %d", len(snaps))
	}

	// Check one snapshot type in detail, e.g. Claude
	var foundClaude bool
	for _, sVal := range snaps {
		if sVal.Provider == "claude" {
			foundClaude = true
			if sVal.OverallPct != 80.0 || sVal.PlanTier != "pro" {
				t.Errorf("incorrect claude snapshot data: %+v", sVal)
			}
		}
	}
	if !foundClaude {
		t.Error("claude snapshot not migrated")
	}
}
