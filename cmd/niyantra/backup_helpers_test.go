package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/client"
	"github.com/bhaskarjha-com/niyantra/internal/store"
)

func openCommandTestStore(t *testing.T, dbPath string) *store.Store {
	t.Helper()

	s, err := store.Open(dbPath, store.WithSecretBackend(store.NewMemorySecretBackend()))
	if err != nil {
		t.Fatalf("store.Open(%q): %v", dbPath, err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateDatabaseBackupProducesValidSQLiteCopy(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "niyantra.db")
	s := openCommandTestStore(t, dbPath)

	accountID, err := s.GetOrCreateAccount("backup@example.com", "Pro", "antigravity")
	if err != nil {
		t.Fatalf("GetOrCreateAccount: %v", err)
	}
	if _, err := s.InsertSnapshot(&client.Snapshot{
		AccountID:     accountID,
		CapturedAt:    time.Now().UTC(),
		Email:         "backup@example.com",
		PlanName:      "Pro",
		Models:        []client.ModelQuota{{ModelID: "claude-sonnet-4.6", RemainingFraction: 0.75, RemainingPercent: 75}},
		CaptureMethod: "manual",
		CaptureSource: "cli",
		SourceID:      "antigravity",
	}); err != nil {
		t.Fatalf("InsertSnapshot: %v", err)
	}
	if _, _, err := s.EnsureDashboardToken(); err != nil {
		t.Fatalf("EnsureDashboardToken: %v", err)
	}
	if _, err := s.SetConfig("copilot_pat", "ghp_secret"); err != nil {
		t.Fatalf("SetConfig copilot_pat: %v", err)
	}
	s.Close()

	backupPath := filepath.Join(t.TempDir(), "backup.db")
	if _, err := createDatabaseBackup(dbPath, backupPath, false); err != nil {
		t.Fatalf("createDatabaseBackup: %v", err)
	}

	backup, err := store.Open(backupPath, store.WithSecretBackend(store.NewMemorySecretBackend()))
	if err != nil {
		t.Fatalf("store.Open(backup): %v", err)
	}
	defer backup.Close()

	if got := backup.AccountCount(); got != 1 {
		t.Fatalf("backup account count = %d, want 1", got)
	}
	if got := backup.SnapshotCount(); got != 1 {
		t.Fatalf("backup snapshot count = %d, want 1", got)
	}
	if got := backup.GetConfig(store.DashboardTokenConfigKey); got != "" {
		t.Fatal("dashboard token was present in CLI backup")
	}
	if got := backup.GetConfig("copilot_pat"); got != "" {
		t.Fatal("copilot_pat was present in CLI backup")
	}
}

func TestRestoreDatabaseFromBackupReplacesTargetAndClearsSidecars(t *testing.T) {
	dir := t.TempDir()
	sourceDB := filepath.Join(dir, "source.db")
	targetDB := filepath.Join(dir, "target.db")

	source := openCommandTestStore(t, sourceDB)
	if _, err := source.SetConfig("currency", "EUR"); err != nil {
		t.Fatalf("SetConfig source: %v", err)
	}
	source.Close()

	backupPath := filepath.Join(dir, "restore.db")
	if _, err := createDatabaseBackup(sourceDB, backupPath, false); err != nil {
		t.Fatalf("createDatabaseBackup: %v", err)
	}

	target := openCommandTestStore(t, targetDB)
	if _, err := target.SetConfig("currency", "USD"); err != nil {
		t.Fatalf("SetConfig target: %v", err)
	}
	target.Close()

	if err := os.WriteFile(targetDB+"-wal", []byte("stale wal"), 0o644); err != nil {
		t.Fatalf("WriteFile wal: %v", err)
	}
	if err := os.WriteFile(targetDB+"-shm", []byte("stale shm"), 0o644); err != nil {
		t.Fatalf("WriteFile shm: %v", err)
	}

	if _, err := restoreDatabaseFromBackup(targetDB, backupPath, false); err != nil {
		t.Fatalf("restoreDatabaseFromBackup: %v", err)
	}
	if _, err := os.Stat(targetDB + "-wal"); !os.IsNotExist(err) {
		t.Fatalf("expected stale WAL to be removed before reopen, stat err = %v", err)
	}
	if _, err := os.Stat(targetDB + "-shm"); !os.IsNotExist(err) {
		t.Fatalf("expected stale SHM to be removed before reopen, stat err = %v", err)
	}

	restored, err := store.Open(targetDB, store.WithSecretBackend(store.NewMemorySecretBackend()))
	if err != nil {
		t.Fatalf("store.Open(restored): %v", err)
	}
	defer restored.Close()

	if got := restored.GetConfig("currency"); got != "EUR" {
		t.Fatalf("restored currency = %q, want %q", got, "EUR")
	}
}
