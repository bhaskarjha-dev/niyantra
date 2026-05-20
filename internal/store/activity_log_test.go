package store

import (
	"testing"
)

// TestEnforceRetentionDeletesOldEntries verifies that activity log entries
// older than the retention window are pruned.
func TestEnforceRetentionDeletesOldEntries(t *testing.T) {
	s := openTestDB(t)

	// Insert entries at different ages using direct SQL for precise timestamps
	s.db.Exec(`INSERT INTO activity_log (timestamp, level, source, event_type, account_email, snapshot_id, details)
		VALUES (datetime('now', '-400 days'), 'info', 'test', 'old_event', '', NULL, '{}')`)
	s.db.Exec(`INSERT INTO activity_log (timestamp, level, source, event_type, account_email, snapshot_id, details)
		VALUES (datetime('now', '-100 days'), 'info', 'test', 'mid_event', '', NULL, '{}')`)
	s.db.Exec(`INSERT INTO activity_log (timestamp, level, source, event_type, account_email, snapshot_id, details)
		VALUES (datetime('now'), 'info', 'test', 'new_event', '', NULL, '{}')`)

	// Enforce 365-day retention — should delete the 400-day-old entry
	deleted, err := s.EnforceRetention(365)
	if err != nil {
		t.Fatalf("EnforceRetention: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleted)
	}

	// Verify remaining entries
	entries, err := s.RecentActivity(100, "")
	if err != nil {
		t.Fatalf("RecentActivity: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 remaining entries, got %d", len(entries))
	}
}

// TestEnforceRetentionZeroDays verifies no-op when retention is 0.
func TestEnforceRetentionZeroDays(t *testing.T) {
	s := openTestDB(t)

	s.db.Exec(`INSERT INTO activity_log (timestamp, level, source, event_type, account_email, snapshot_id, details)
		VALUES (datetime('now', '-1000 days'), 'info', 'test', 'ancient', '', NULL, '{}')`)

	deleted, err := s.EnforceRetention(0)
	if err != nil {
		t.Fatalf("EnforceRetention: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("expected 0 deleted with retention=0, got %d", deleted)
	}
}

// TestLogActivityRedactsFreeTextFields verifies that long free-text fields
// are truncated in activity log entries.
func TestLogActivityRedactsFreeTextFields(t *testing.T) {
	s := openTestDB(t)

	longNotes := "This is a very long note that contains potentially sensitive information and should be truncated beyond 50 characters"

	// Test LogActivity with the redaction fix
	err := s.LogActivity("info", "test", "meta_update", "test@example.com", 0, map[string]interface{}{
		"notes":     longNotes,
		"accountId": 42,
	})
	if err != nil {
		t.Fatalf("LogActivity: %v", err)
	}

	// Verify entry was persisted with redacted data
	entries, err := s.RecentActivity(1, "meta_update")
	if err != nil {
		t.Fatalf("RecentActivity: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	// The details should contain the truncation marker
	if !containsSubstring(entries[0].Details, "\u2026") {
		t.Errorf("expected truncation marker in details: %s", entries[0].Details)
	}

	// accountId should be preserved verbatim
	if !containsSubstring(entries[0].Details, "42") {
		t.Errorf("expected accountId to be preserved: %s", entries[0].Details)
	}
}

// TestUpsertDataSource verifies the new parameterized method.
func TestUpsertDataSource(t *testing.T) {
	s := openTestDB(t)

	// Insert
	err := s.UpsertDataSource("test_plugin", "Test Plugin", "plugin", true)
	if err != nil {
		t.Fatalf("UpsertDataSource (insert): %v", err)
	}

	// Update
	err = s.UpsertDataSource("test_plugin", "Updated Plugin", "plugin", false)
	if err != nil {
		t.Fatalf("UpsertDataSource (update): %v", err)
	}

	// Verify
	sources, err := s.AllDataSources()
	if err != nil {
		t.Fatalf("AllDataSources: %v", err)
	}

	found := false
	for _, ds := range sources {
		if ds.ID == "test_plugin" {
			found = true
			if ds.Name != "Updated Plugin" {
				t.Errorf("expected name 'Updated Plugin', got %q", ds.Name)
			}
			if ds.Enabled {
				t.Error("expected disabled after update")
			}
		}
	}
	if !found {
		t.Error("test_plugin data source not found")
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && findSubstring(s, sub))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
