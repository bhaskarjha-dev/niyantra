package store

import (
	"encoding/json"
	"fmt"
)

// ActivityEntry represents a structured event in the activity log.
type ActivityEntry struct {
	ID           int64  `json:"id"`
	Timestamp    string `json:"timestamp"`
	Level        string `json:"level"`
	Source       string `json:"source"`
	EventType    string `json:"eventType"`
	AccountEmail string `json:"accountEmail"`
	SnapshotID   int64  `json:"snapshotId"`
	Details      string `json:"details"`
}

// LogActivity writes a structured event to the activity log.
// Free-text fields are truncated to prevent sensitive data persistence.
func (s *Store) LogActivity(level, source, eventType, email string, snapID int64, details map[string]interface{}) error {
	detailsJSON := "{}"
	if details != nil {
		// Redact free-text fields that could contain sensitive user data.
		// These values are useful for debugging but should not persist verbatim
		// in activity logs that are included in exports and backups.
		redacted := make(map[string]interface{}, len(details))
		for k, v := range details {
			switch k {
			case "notes", "tags", "pinnedGroup":
				if str, ok := v.(string); ok && len(str) > 50 {
					redacted[k] = str[:50] + "…"
				} else {
					redacted[k] = v
				}
			default:
				redacted[k] = v
			}
		}
		if b, err := json.Marshal(redacted); err == nil {
			detailsJSON = string(b)
		}
	}

	// N2: Use NULL for snapshot_id when no snapshot is associated (0).
	// The FK constraint references snapshots(id) and ID 0 doesn't exist.
	var snapIDVal any
	if snapID > 0 {
		snapIDVal = snapID
	}

	_, err := s.db.Exec(`
		INSERT INTO activity_log (level, source, event_type, account_email, snapshot_id, details)
		VALUES (?, ?, ?, ?, ?, ?)
	`, level, source, eventType, email, snapIDVal, detailsJSON)
	if err != nil {
		return fmt.Errorf("store: log activity: %w", err)
	}
	return nil
}

// LogInfo is a convenience wrapper for info-level events.
func (s *Store) LogInfo(source, eventType, email string, details map[string]interface{}) {
	s.LogActivity("info", source, eventType, email, 0, details)
}

// LogInfoSnap logs an info event associated with a snapshot.
func (s *Store) LogInfoSnap(source, eventType, email string, snapID int64, details map[string]interface{}) {
	s.LogActivity("info", source, eventType, email, snapID, details)
}

// LogError is a convenience wrapper for error-level events.
func (s *Store) LogError(source, eventType, email string, details map[string]interface{}) {
	s.LogActivity("error", source, eventType, email, 0, details)
}

// RecentActivity returns the most recent activity log entries.
func (s *Store) RecentActivity(limit int, eventType string) ([]*ActivityEntry, error) {
	query := `SELECT id, timestamp, level, source, event_type, 
		COALESCE(account_email,''), COALESCE(snapshot_id, 0), COALESCE(details,'{}')
		FROM activity_log`
	args := []interface{}{}

	if eventType != "" {
		query += ` WHERE event_type = ?`
		args = append(args, eventType)
	}
	query += ` ORDER BY timestamp DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query activity: %w", err)
	}
	defer rows.Close()

	var entries []*ActivityEntry
	for rows.Next() {
		e := &ActivityEntry{}
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Level, &e.Source, &e.EventType,
			&e.AccountEmail, &e.SnapshotID, &e.Details); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// EnforceRetention deletes activity log entries older than retentionDays.
// Should be called on startup and periodically to prevent unbounded growth.
func (s *Store) EnforceRetention(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	result, err := s.db.Exec(`
		DELETE FROM activity_log
		WHERE timestamp < datetime('now', '-' || ? || ' days')
	`, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("store: enforce retention: %w", err)
	}
	return result.RowsAffected()
}
