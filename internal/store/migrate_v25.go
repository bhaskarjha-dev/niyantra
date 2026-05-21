package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/core"
)

func (s *Store) migrateToV25() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("migrate v25: begin tx: %w", err)
	}
	defer tx.Rollback()

	formatTime := func(t time.Time) string {
		return t.UTC().Format(time.RFC3339)
	}

	formatTimePtr := func(t *time.Time) any {
		if t == nil {
			return nil
		}
		return t.UTC().Format(time.RFC3339)
	}

	// 1. Create snapshots_v2 table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS snapshots_v2 (
			id             TEXT PRIMARY KEY,
			provider       TEXT NOT NULL,
			account_id     INTEGER NOT NULL,
			captured_at    DATETIME NOT NULL,
			email          TEXT DEFAULT '',
			overall_pct    REAL DEFAULT 0,
			plan_tier      TEXT DEFAULT '',
			cost_usd       REAL DEFAULT 0,
			reset_at       DATETIME,
			reset_type     TEXT DEFAULT '',
			data_json      TEXT DEFAULT '{}',
			models_json    TEXT DEFAULT '[]',
			capture_method TEXT DEFAULT 'auto',
			capture_source TEXT DEFAULT '',
			machine_id     TEXT DEFAULT '',
			version        INTEGER DEFAULT 1,
			updated_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
			deleted_at     DATETIME,
			owner_id       TEXT DEFAULT '',
			synced_at      DATETIME,
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		return fmt.Errorf("migrate v25: create table snapshots_v2: %w", err)
	}

	// Create indexes
	for _, idx := range []string{
		`CREATE INDEX IF NOT EXISTS idx_sv2_provider ON snapshots_v2(provider, captured_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_sv2_account  ON snapshots_v2(account_id, captured_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_sv2_time     ON snapshots_v2(captured_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_sv2_sync     ON snapshots_v2(synced_at) WHERE synced_at IS NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_sv2_active   ON snapshots_v2(deleted_at) WHERE deleted_at IS NULL;`,
	} {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("migrate v25: create index: %w", err)
		}
	}

	// Helper for getting or creating accounts
	getOrCreateAccount := func(provider, email, planTier string) (int64, error) {
		if email == "" {
			email = provider + "@niyantra.local"
		}
		var id int64
		err := tx.QueryRow(`SELECT id FROM accounts WHERE email = ? AND provider = ?`, email, provider).Scan(&id)
		if err == sql.ErrNoRows {
			res, err := tx.Exec(`INSERT INTO accounts (email, provider, plan_tier) VALUES (?, ?, ?)`, email, provider, PlanTierFromName(planTier))
			if err != nil {
				return 0, fmt.Errorf("insert account (%s, %s): %w", email, provider, err)
			}
			id, err = res.LastInsertId()
			if err != nil {
				return 0, err
			}
		} else if err != nil {
			return 0, fmt.Errorf("query account (%s, %s): %w", email, provider, err)
		}
		return id, nil
	}

	// Now migrate data from the 5 old tables

	// 1. snapshots (Antigravity)
	rows, err := tx.Query(`
		SELECT account_id, captured_at, email, plan_name, prompt_credits, monthly_credits, models_json, raw_json, capture_method, capture_source, ai_credits_json
		FROM snapshots
	`)
	if err == nil {
		defer rows.Close()
		stmt, err := tx.Prepare(`
			INSERT INTO snapshots_v2 (id, provider, account_id, captured_at, email, overall_pct, plan_tier, data_json, models_json, capture_method, capture_source)
			VALUES (?, 'antigravity', ?, ?, ?, 0, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("prepare antigravity insert: %w", err)
		}
		defer stmt.Close()

		for rows.Next() {
			var accID int64
			var capturedAt time.Time
			var email, planName, modelsJSON, rawJSON, captureMethod, captureSource, aiCreditsJSON string
			var promptCredits float64
			var monthlyCredits int64
			if err := rows.Scan(&accID, &capturedAt, &email, &planName, &promptCredits, &monthlyCredits, &modelsJSON, &rawJSON, &captureMethod, &captureSource, &aiCreditsJSON); err != nil {
				return fmt.Errorf("scan antigravity: %w", err)
			}

			dataMap := map[string]any{
				"prompt_credits":  promptCredits,
				"monthly_credits": monthlyCredits,
				"raw_json":        rawJSON,
				"ai_credits_json": aiCreditsJSON,
			}
			dataBytes, _ := json.Marshal(dataMap)

			ulid := core.NewULIDAt(capturedAt)
			_, err = stmt.Exec(ulid, accID, formatTime(capturedAt), email, planName, string(dataBytes), modelsJSON, captureMethod, captureSource)
			if err != nil {
				return fmt.Errorf("exec antigravity insert: %w", err)
			}
		}
		rows.Close()
	}

	// 2. claude_snapshots
	rows, err = tx.Query(`
		SELECT five_hour_pct, seven_day_pct, five_hour_reset, seven_day_reset, captured_at, source
		FROM claude_snapshots
	`)
	if err == nil {
		defer rows.Close()
		stmt, err := tx.Prepare(`
			INSERT INTO snapshots_v2 (id, provider, account_id, captured_at, email, overall_pct, plan_tier, reset_at, reset_type, data_json, models_json, capture_method, capture_source)
			VALUES (?, 'claude', ?, ?, '', ?, 'pro', ?, '5h', ?, '[]', 'auto', ?)
		`)
		if err != nil {
			return fmt.Errorf("prepare claude insert: %w", err)
		}
		defer stmt.Close()

		var claudeAccID int64
		var hasClaudeAcc bool

		for rows.Next() {
			if !hasClaudeAcc {
				var err error
				claudeAccID, err = getOrCreateAccount("claude", "", "pro")
				if err != nil {
					return err
				}
				hasClaudeAcc = true
			}
			var fiveHourPct float64
			var sevenDayPct sql.NullFloat64
			var fiveHourReset, sevenDayReset sql.NullTime
			var capturedAt time.Time
			var source string
			if err := rows.Scan(&fiveHourPct, &sevenDayPct, &fiveHourReset, &sevenDayReset, &capturedAt, &source); err != nil {
				return fmt.Errorf("scan claude: %w", err)
			}

			var s7Pct *float64
			if sevenDayPct.Valid {
				s7Pct = &sevenDayPct.Float64
			}
			var fReset, sReset *time.Time
			if fiveHourReset.Valid {
				fReset = &fiveHourReset.Time
			}
			if sevenDayReset.Valid {
				sReset = &sevenDayReset.Time
			}

			dataMap := map[string]any{
				"five_hour_pct":   fiveHourPct,
				"seven_day_pct":   s7Pct,
				"five_hour_reset": fReset,
				"seven_day_reset": sReset,
			}
			dataBytes, _ := json.Marshal(dataMap)

			ulid := core.NewULIDAt(capturedAt)
			_, err = stmt.Exec(ulid, claudeAccID, formatTime(capturedAt), fiveHourPct, formatTimePtr(fReset), string(dataBytes), source)
			if err != nil {
				return fmt.Errorf("exec claude insert: %w", err)
			}
		}
		rows.Close()
	}

	// 3. codex_snapshots
	rows, err = tx.Query(`
		SELECT COALESCE(owner_account_id, 0), email, five_hour_pct, seven_day_pct, code_review_pct, five_hour_reset, seven_day_reset, plan_type, COALESCE(credits_balance, 0), captured_at, capture_method, capture_source
		FROM codex_snapshots
	`)
	if err == nil {
		defer rows.Close()
		stmt, err := tx.Prepare(`
			INSERT INTO snapshots_v2 (id, provider, account_id, captured_at, email, overall_pct, plan_tier, reset_at, reset_type, data_json, models_json, capture_method, capture_source)
			VALUES (?, 'codex', ?, ?, ?, ?, ?, ?, '5h', ?, '[]', ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("prepare codex insert: %w", err)
		}
		defer stmt.Close()

		for rows.Next() {
			var accID int64
			var email, planType, captureMethod, captureSource string
			var fiveHourPct, creditsBalance float64
			var sevenDayPct, codeReviewPct sql.NullFloat64
			var fiveHourReset, sevenDayReset sql.NullTime
			var capturedAt time.Time

			if err := rows.Scan(&accID, &email, &fiveHourPct, &sevenDayPct, &codeReviewPct, &fiveHourReset, &sevenDayReset, &planType, &creditsBalance, &capturedAt, &captureMethod, &captureSource); err != nil {
				return fmt.Errorf("scan codex: %w", err)
			}

			if accID == 0 {
				accID, err = getOrCreateAccount("codex", email, planType)
				if err != nil {
					return err
				}
			}

			var s7Pct *float64
			if sevenDayPct.Valid {
				s7Pct = &sevenDayPct.Float64
			}
			var crPct *float64
			if codeReviewPct.Valid {
				crPct = &codeReviewPct.Float64
			}
			var fReset, sReset *time.Time
			if fiveHourReset.Valid {
				fReset = &fiveHourReset.Time
			}
			if sevenDayReset.Valid {
				sReset = &sevenDayReset.Time
			}

			dataMap := map[string]any{
				"five_hour_pct":   fiveHourPct,
				"seven_day_pct":   s7Pct,
				"code_review_pct": crPct,
				"five_hour_reset": fReset,
				"seven_day_reset": sReset,
				"credits_balance": creditsBalance,
			}
			dataBytes, _ := json.Marshal(dataMap)

			ulid := core.NewULIDAt(capturedAt)
			_, err = stmt.Exec(ulid, accID, formatTime(capturedAt), email, fiveHourPct, planType, formatTimePtr(fReset), string(dataBytes), captureMethod, captureSource)
			if err != nil {
				return fmt.Errorf("exec codex insert: %w", err)
			}
		}
		rows.Close()
	}

	// 4. cursor_snapshots
	rows, err = tx.Query(`
		SELECT COALESCE(account_id, 0), email, premium_used, premium_limit, usage_pct, plan_type, start_of_month, models_json, captured_at, capture_method, capture_source
		FROM cursor_snapshots
	`)
	if err == nil {
		defer rows.Close()
		stmt, err := tx.Prepare(`
			INSERT INTO snapshots_v2 (id, provider, account_id, captured_at, email, overall_pct, plan_tier, data_json, models_json, capture_method, capture_source)
			VALUES (?, 'cursor', ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("prepare cursor insert: %w", err)
		}
		defer stmt.Close()

		for rows.Next() {
			var accID int64
			var email, planType, startOfMonth, modelsJSON, captureMethod, captureSource string
			var premiumUsed, premiumLimit int
			var usagePct float64
			var capturedAt time.Time

			if err := rows.Scan(&accID, &email, &premiumUsed, &premiumLimit, &usagePct, &planType, &startOfMonth, &modelsJSON, &capturedAt, &captureMethod, &captureSource); err != nil {
				return fmt.Errorf("scan cursor: %w", err)
			}

			if accID == 0 {
				accID, err = getOrCreateAccount("cursor", email, planType)
				if err != nil {
					return err
				}
			}

			overallPct := 0.0
			if premiumLimit > 0 {
				overallPct = (float64(premiumLimit-premiumUsed) / float64(premiumLimit)) * 100
			}

			dataMap := map[string]any{
				"premium_used":   premiumUsed,
				"premium_limit":  premiumLimit,
				"start_of_month": startOfMonth,
			}
			dataBytes, _ := json.Marshal(dataMap)

			ulid := core.NewULIDAt(capturedAt)
			_, err = stmt.Exec(ulid, accID, formatTime(capturedAt), email, overallPct, planType, string(dataBytes), modelsJSON, captureMethod, captureSource)
			if err != nil {
				return fmt.Errorf("exec cursor insert: %w", err)
			}
		}
		rows.Close()
	}

	// 5. copilot_snapshots
	rows, err = tx.Query(`
		SELECT COALESCE(account_id, 0), email, username, plan, premium_pct, chat_pct, models_json, captured_at, capture_method, capture_source
		FROM copilot_snapshots
	`)
	if err == nil {
		defer rows.Close()
		stmt, err := tx.Prepare(`
			INSERT INTO snapshots_v2 (id, provider, account_id, captured_at, email, overall_pct, plan_tier, data_json, models_json, capture_method, capture_source)
			VALUES (?, 'copilot', ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("prepare copilot insert: %w", err)
		}
		defer stmt.Close()

		for rows.Next() {
			var accID int64
			var email, username, plan, modelsJSON, captureMethod, captureSource string
			var premiumPct, chatPct float64
			var capturedAt time.Time

			if err := rows.Scan(&accID, &email, &username, &plan, &premiumPct, &chatPct, &modelsJSON, &capturedAt, &captureMethod, &captureSource); err != nil {
				return fmt.Errorf("scan copilot: %w", err)
			}

			if accID == 0 {
				accID, err = getOrCreateAccount("copilot", email, plan)
				if err != nil {
					return err
				}
			}

			dataMap := map[string]any{
				"premium_pct": premiumPct,
				"chat_pct":    chatPct,
				"username":    username,
			}
			dataBytes, _ := json.Marshal(dataMap)

			ulid := core.NewULIDAt(capturedAt)
			_, err = stmt.Exec(ulid, accID, formatTime(capturedAt), email, premiumPct, plan, string(dataBytes), modelsJSON, captureMethod, captureSource)
			if err != nil {
				return fmt.Errorf("exec copilot insert: %w", err)
			}
		}
		rows.Close()
	}

	// 6. plugin_snapshots
	rows, err = tx.Query(`
		SELECT plugin_id, provider, label, email, usage_pct, usage_display, plan, models_json, metadata_json, captured_at, capture_method
		FROM plugin_snapshots
	`)
	if err == nil {
		defer rows.Close()
		stmt, err := tx.Prepare(`
			INSERT INTO snapshots_v2 (id, provider, account_id, captured_at, email, overall_pct, plan_tier, data_json, models_json, capture_method)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("prepare plugin insert: %w", err)
		}
		defer stmt.Close()

		for rows.Next() {
			var pluginID, provider, label, email, usageDisplay, plan, modelsJSON, metadataJSON, captureMethod string
			var usagePct float64
			var capturedAt time.Time

			if err := rows.Scan(&pluginID, &provider, &label, &email, &usagePct, &usageDisplay, &plan, &modelsJSON, &metadataJSON, &capturedAt, &captureMethod); err != nil {
				return fmt.Errorf("scan plugin: %w", err)
			}

			fullProvider := "plugin_" + pluginID
			accID, err := getOrCreateAccount(fullProvider, email, plan)
			if err != nil {
				return err
			}

			dataMap := map[string]any{
				"plugin_id":     pluginID,
				"provider":      provider,
				"label":         label,
				"usage_display": usageDisplay,
				"metadata_json": metadataJSON,
			}
			dataBytes, _ := json.Marshal(dataMap)

			ulid := core.NewULIDAt(capturedAt)
			_, err = stmt.Exec(ulid, fullProvider, accID, formatTime(capturedAt), email, usagePct, plan, string(dataBytes), modelsJSON, captureMethod)
			if err != nil {
				return fmt.Errorf("exec plugin insert: %w", err)
			}
		}
		rows.Close()
	}

	return tx.Commit()
}
