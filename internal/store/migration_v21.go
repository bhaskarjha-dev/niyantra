package store

import "fmt"

func (s *Store) applyV21IntegrityMigration() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: v21 begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		UPDATE activity_log
		SET snapshot_id = NULL
		WHERE COALESCE(snapshot_id, 0) = 0
		   OR NOT EXISTS (SELECT 1 FROM snapshots WHERE snapshots.id = activity_log.snapshot_id);

		UPDATE subscriptions
		SET account_id = NULL
		WHERE COALESCE(account_id, 0) = 0
		   OR NOT EXISTS (SELECT 1 FROM accounts WHERE accounts.id = subscriptions.account_id);

		UPDATE codex_snapshots
		SET owner_account_id = NULL
		WHERE COALESCE(owner_account_id, 0) = 0
		   OR NOT EXISTS (SELECT 1 FROM accounts WHERE accounts.id = codex_snapshots.owner_account_id);

		UPDATE cursor_snapshots
		SET account_id = NULL
		WHERE COALESCE(account_id, 0) = 0
		   OR NOT EXISTS (SELECT 1 FROM accounts WHERE accounts.id = cursor_snapshots.account_id);

		UPDATE gemini_snapshots
		SET account_id = NULL
		WHERE COALESCE(account_id, 0) = 0
		   OR NOT EXISTS (SELECT 1 FROM accounts WHERE accounts.id = gemini_snapshots.account_id);

		UPDATE copilot_snapshots
		SET account_id = NULL
		WHERE COALESCE(account_id, 0) = 0
		   OR NOT EXISTS (SELECT 1 FROM accounts WHERE accounts.id = copilot_snapshots.account_id);
	`); err != nil {
		return fmt.Errorf("store: v21 cleanup invalid references: %w", err)
	}

	if _, err := tx.Exec(`
		CREATE TABLE activity_log_new (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp       DATETIME NOT NULL DEFAULT (datetime('now')),
			level           TEXT     NOT NULL DEFAULT 'info',
			source          TEXT     NOT NULL DEFAULT 'system',
			event_type      TEXT     NOT NULL,
			account_email   TEXT     DEFAULT '',
			snapshot_id     INTEGER  DEFAULT NULL,
			details         TEXT     DEFAULT '{}',
			FOREIGN KEY (snapshot_id) REFERENCES snapshots(id) ON DELETE SET NULL
		);
		INSERT INTO activity_log_new (id, timestamp, level, source, event_type, account_email, snapshot_id, details)
		SELECT id, timestamp, level, source, event_type, account_email, snapshot_id, details FROM activity_log;
		DROP TABLE activity_log;
		ALTER TABLE activity_log_new RENAME TO activity_log;
		CREATE INDEX IF NOT EXISTS idx_activity_log_time ON activity_log(timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_activity_log_type ON activity_log(event_type);

		CREATE TABLE subscriptions_new (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			platform        TEXT    NOT NULL,
			category        TEXT    DEFAULT 'other',
			icon_key        TEXT    DEFAULT '',
			email           TEXT    DEFAULT '',
			plan_name       TEXT    DEFAULT '',
			status          TEXT    DEFAULT 'active',
			cost_amount     REAL    DEFAULT 0,
			cost_currency   TEXT    DEFAULT 'USD',
			billing_cycle   TEXT    DEFAULT 'monthly',
			token_limit     INTEGER DEFAULT 0,
			credit_limit    INTEGER DEFAULT 0,
			request_limit   INTEGER DEFAULT 0,
			limit_period    TEXT    DEFAULT 'monthly',
			limit_note      TEXT    DEFAULT '',
			next_renewal    TEXT    DEFAULT '',
			started_at      TEXT    DEFAULT '',
			trial_ends_at   TEXT    DEFAULT '',
			notes           TEXT    DEFAULT '',
			url             TEXT    DEFAULT '',
			status_page_url TEXT    DEFAULT '',
			auto_tracked    INTEGER DEFAULT 0,
			account_id      INTEGER DEFAULT NULL,
			created_at      DATETIME DEFAULT (datetime('now')),
			updated_at      DATETIME DEFAULT (datetime('now')),
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE SET NULL
		);
		INSERT INTO subscriptions_new (
			id, platform, category, icon_key, email, plan_name, status,
			cost_amount, cost_currency, billing_cycle, token_limit, credit_limit,
			request_limit, limit_period, limit_note, next_renewal, started_at,
			trial_ends_at, notes, url, status_page_url, auto_tracked,
			account_id, created_at, updated_at
		)
		SELECT id, platform, category, icon_key, email, plan_name, status,
			cost_amount, cost_currency, billing_cycle, token_limit, credit_limit,
			request_limit, limit_period, limit_note, next_renewal, started_at,
			trial_ends_at, notes, url, status_page_url, auto_tracked,
			account_id, created_at, updated_at
		FROM subscriptions;
		DROP TABLE subscriptions;
		ALTER TABLE subscriptions_new RENAME TO subscriptions;
		CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);
		CREATE INDEX IF NOT EXISTS idx_subscriptions_renewal ON subscriptions(next_renewal);
		CREATE INDEX IF NOT EXISTS idx_subscriptions_category ON subscriptions(category);
		CREATE INDEX IF NOT EXISTS idx_subscriptions_account ON subscriptions(account_id);

		CREATE TABLE codex_snapshots_new (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id       TEXT    DEFAULT '',
			owner_account_id INTEGER DEFAULT NULL,
			five_hour_pct    REAL    NOT NULL,
			seven_day_pct    REAL,
			code_review_pct  REAL,
			five_hour_reset  DATETIME,
			seven_day_reset  DATETIME,
			plan_type        TEXT    DEFAULT '',
			credits_balance  REAL,
			captured_at      DATETIME DEFAULT (datetime('now')),
			capture_method   TEXT    DEFAULT 'manual',
			capture_source   TEXT    DEFAULT 'ui',
			email            TEXT    DEFAULT '',
			FOREIGN KEY (owner_account_id) REFERENCES accounts(id) ON DELETE SET NULL
		);
		INSERT INTO codex_snapshots_new (
			id, account_id, owner_account_id, five_hour_pct, seven_day_pct,
			code_review_pct, five_hour_reset, seven_day_reset, plan_type,
			credits_balance, captured_at, capture_method, capture_source, email
		)
		SELECT id, account_id, owner_account_id, five_hour_pct, seven_day_pct,
			code_review_pct, five_hour_reset, seven_day_reset, plan_type,
			credits_balance, captured_at, capture_method, capture_source, COALESCE(email,'')
		FROM codex_snapshots;
		DROP TABLE codex_snapshots;
		ALTER TABLE codex_snapshots_new RENAME TO codex_snapshots;
		CREATE INDEX IF NOT EXISTS idx_codex_snapshots_time ON codex_snapshots(captured_at DESC);
		CREATE INDEX IF NOT EXISTS idx_codex_snapshots_account ON codex_snapshots(account_id, captured_at DESC);
		CREATE INDEX IF NOT EXISTS idx_codex_snapshots_owner_account ON codex_snapshots(owner_account_id, captured_at DESC);

		CREATE TABLE cursor_snapshots_new (
			id              INTEGER  PRIMARY KEY AUTOINCREMENT,
			account_id      INTEGER  DEFAULT NULL,
			email           TEXT     DEFAULT '',
			premium_used    INTEGER  DEFAULT 0,
			premium_limit   INTEGER  DEFAULT 0,
			usage_pct       REAL     DEFAULT 0,
			plan_type       TEXT     DEFAULT '',
			start_of_month  TEXT     DEFAULT '',
			models_json     TEXT     DEFAULT '{}',
			captured_at     DATETIME DEFAULT (datetime('now')),
			capture_method  TEXT     DEFAULT 'manual',
			capture_source  TEXT     DEFAULT 'ui',
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE SET NULL
		);
		INSERT INTO cursor_snapshots_new
		SELECT id, account_id, email, premium_used, premium_limit, usage_pct,
			plan_type, start_of_month, models_json, captured_at, capture_method, capture_source
		FROM cursor_snapshots;
		DROP TABLE cursor_snapshots;
		ALTER TABLE cursor_snapshots_new RENAME TO cursor_snapshots;
		CREATE INDEX IF NOT EXISTS idx_cursor_snapshots_time ON cursor_snapshots(captured_at DESC);
		CREATE INDEX IF NOT EXISTS idx_cursor_snapshots_account ON cursor_snapshots(account_id, captured_at DESC);

		CREATE TABLE gemini_snapshots_new (
			id              INTEGER  PRIMARY KEY AUTOINCREMENT,
			account_id      INTEGER  DEFAULT NULL,
			email           TEXT     DEFAULT '',
			tier            TEXT     DEFAULT '',
			overall_pct     REAL     DEFAULT 0,
			models_json     TEXT     DEFAULT '[]',
			project_id      TEXT     DEFAULT '',
			captured_at     DATETIME DEFAULT (datetime('now')),
			capture_method  TEXT     DEFAULT 'manual',
			capture_source  TEXT     DEFAULT 'ui',
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE SET NULL
		);
		INSERT INTO gemini_snapshots_new
		SELECT id, account_id, email, tier, overall_pct, models_json, project_id,
			captured_at, capture_method, capture_source
		FROM gemini_snapshots;
		DROP TABLE gemini_snapshots;
		ALTER TABLE gemini_snapshots_new RENAME TO gemini_snapshots;
		CREATE INDEX IF NOT EXISTS idx_gemini_snapshots_time ON gemini_snapshots(captured_at DESC);
		CREATE INDEX IF NOT EXISTS idx_gemini_snapshots_account ON gemini_snapshots(account_id, captured_at DESC);

		CREATE TABLE copilot_snapshots_new (
			id              INTEGER  PRIMARY KEY AUTOINCREMENT,
			account_id      INTEGER  DEFAULT NULL,
			email           TEXT     DEFAULT '',
			username        TEXT     DEFAULT '',
			plan            TEXT     DEFAULT '',
			premium_pct     REAL     DEFAULT 0,
			chat_pct        REAL     DEFAULT 0,
			models_json     TEXT     DEFAULT '{}',
			captured_at     DATETIME DEFAULT (datetime('now')),
			capture_method  TEXT     DEFAULT 'manual',
			capture_source  TEXT     DEFAULT 'ui',
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE SET NULL
		);
		INSERT INTO copilot_snapshots_new
		SELECT id, account_id, email, username, plan, premium_pct, chat_pct,
			models_json, captured_at, capture_method, capture_source
		FROM copilot_snapshots;
		DROP TABLE copilot_snapshots;
		ALTER TABLE copilot_snapshots_new RENAME TO copilot_snapshots;
		CREATE INDEX IF NOT EXISTS idx_copilot_snapshots_time ON copilot_snapshots(captured_at DESC);
		CREATE INDEX IF NOT EXISTS idx_copilot_snapshots_account ON copilot_snapshots(account_id, captured_at DESC);

		CREATE TABLE plugin_snapshots_new (
			id              INTEGER  PRIMARY KEY AUTOINCREMENT,
			plugin_id       TEXT     NOT NULL,
			data_source_id  TEXT     DEFAULT NULL,
			provider        TEXT     DEFAULT '',
			label           TEXT     DEFAULT '',
			email           TEXT     DEFAULT '',
			usage_pct       REAL     DEFAULT 0,
			usage_display   TEXT     DEFAULT '',
			plan            TEXT     DEFAULT '',
			models_json     TEXT     DEFAULT '[]',
			metadata_json   TEXT     DEFAULT '{}',
			captured_at     DATETIME DEFAULT (datetime('now')),
			capture_method  TEXT     DEFAULT 'plugin',
			FOREIGN KEY (data_source_id) REFERENCES data_sources(id) ON DELETE SET NULL
		);
		INSERT INTO plugin_snapshots_new (
			id, plugin_id, data_source_id, provider, label, email, usage_pct,
			usage_display, plan, models_json, metadata_json, captured_at, capture_method
		)
		SELECT ps.id, ps.plugin_id,
			CASE WHEN ds.id IS NULL THEN NULL ELSE 'plugin_' || ps.plugin_id END,
			ps.provider, ps.label, ps.email, ps.usage_pct, ps.usage_display,
			ps.plan, ps.models_json, ps.metadata_json, ps.captured_at, ps.capture_method
		FROM plugin_snapshots ps
		LEFT JOIN data_sources ds ON ds.id = 'plugin_' || ps.plugin_id;
		DROP TABLE plugin_snapshots;
		ALTER TABLE plugin_snapshots_new RENAME TO plugin_snapshots;
		CREATE INDEX IF NOT EXISTS idx_plugin_snapshots_time ON plugin_snapshots(captured_at DESC);
		CREATE INDEX IF NOT EXISTS idx_plugin_snapshots_plugin ON plugin_snapshots(plugin_id, captured_at DESC);
		CREATE INDEX IF NOT EXISTS idx_plugin_snapshots_source ON plugin_snapshots(data_source_id, captured_at DESC);
	`); err != nil {
		return fmt.Errorf("store: v21 rebuild constrained tables: %w", err)
	}

	return tx.Commit()
}
