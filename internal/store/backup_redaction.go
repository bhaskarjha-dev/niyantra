package store

import (
	"database/sql"
	"fmt"
)

// RedactSensitiveConfigFile clears sensitive config values from a copied
// SQLite database before it leaves the process as a backup artifact.
func RedactSensitiveConfigFile(path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("store: open backup for redaction: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT key FROM config`)
	if err != nil {
		return fmt.Errorf("store: query backup config keys: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return fmt.Errorf("store: scan backup config key: %w", err)
		}
		if IsSensitiveConfigKey(key) {
			keys = append(keys, key)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: iterate backup config keys: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin backup redaction: %w", err)
	}
	defer tx.Rollback()

	for _, key := range keys {
		if _, err := tx.Exec(`UPDATE config SET value = '', updated_at = datetime('now') WHERE key = ?`, key); err != nil {
			return fmt.Errorf("store: redact backup config %s: %w", key, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit backup redaction: %w", err)
	}
	return nil
}
