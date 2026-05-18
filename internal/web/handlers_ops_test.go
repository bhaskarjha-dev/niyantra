package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/bhaskarjha-com/niyantra/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "niyantra-test.db")
	s, err := store.Open(dbPath, store.WithSecretBackend(store.NewMemorySecretBackend()))
	if err != nil {
		t.Fatalf("store.Open(%q): %v", dbPath, err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestHandleExportJSONMasksSensitiveConfig(t *testing.T) {
	st := openTestStore(t)
	if _, err := st.SetConfig("smtp_pass", "super-secret-pass"); err != nil {
		t.Fatalf("SetConfig smtp_pass: %v", err)
	}
	if _, err := st.SetConfig("copilot_pat", "ghp_secret"); err != nil {
		t.Fatalf("SetConfig copilot_pat: %v", err)
	}
	if _, err := st.SetConfig("budget_monthly", "200"); err != nil {
		t.Fatalf("SetConfig budget_monthly: %v", err)
	}

	srv := &Server{
		logger:  slog.Default(),
		store:   st,
		Version: "test",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/export/json", nil)
	rec := httptest.NewRecorder()
	srv.handleExportJSON(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var export struct {
		RedactedSecrets bool                 `json:"redactedSecrets"`
		FullBackupPath  string               `json:"fullBackupPath"`
		Config          []*store.ConfigEntry `json:"config"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &export); err != nil {
		t.Fatalf("unmarshal export JSON: %v", err)
	}

	if !export.RedactedSecrets {
		t.Fatal("expected export to mark secrets as redacted")
	}
	if export.FullBackupPath != "/api/backup" {
		t.Fatalf("fullBackupPath = %q, want %q", export.FullBackupPath, "/api/backup")
	}

	values := make(map[string]string, len(export.Config))
	for _, entry := range export.Config {
		values[entry.Key] = entry.Value
	}

	if values["smtp_pass"] != "configured" {
		t.Fatalf("smtp_pass = %q, want %q", values["smtp_pass"], "configured")
	}
	if values["copilot_pat"] != "configured" {
		t.Fatalf("copilot_pat = %q, want %q", values["copilot_pat"], "configured")
	}
	if values["budget_monthly"] != "200" {
		t.Fatalf("budget_monthly = %q, want %q", values["budget_monthly"], "200")
	}
}

func TestGitRepoPathFromRequest(t *testing.T) {
	baseDir := t.TempDir()
	insideDir := filepath.Join(baseDir, "subdir")

	t.Run("defaults to current directory", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/git-costs?days=30", nil)
		got, err := gitRepoPathFromRequest(baseDir, req)
		if err != nil {
			t.Fatalf("gitRepoPathFromRequest() unexpected error: %v", err)
		}
		if got != filepath.Clean(baseDir) {
			t.Fatalf("gitRepoPathFromRequest() = %q, want %q", got, filepath.Clean(baseDir))
		}
	})

	t.Run("uses explicit repo query within base dir", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/git-costs?repo="+url.QueryEscape(insideDir), nil)
		got, err := gitRepoPathFromRequest(baseDir, req)
		if err != nil {
			t.Fatalf("gitRepoPathFromRequest() unexpected error: %v", err)
		}
		want := filepath.Clean(insideDir)
		if got != want {
			t.Fatalf("gitRepoPathFromRequest() = %q, want %q", got, want)
		}
	})

	t.Run("rejects paths outside base dir", func(t *testing.T) {
		outsideDir := filepath.Join(filepath.Dir(baseDir), "other-repo")
		req := httptest.NewRequest(http.MethodGet, "/api/git-costs?repo="+url.QueryEscape(outsideDir), nil)
		if _, err := gitRepoPathFromRequest(baseDir, req); err == nil {
			t.Fatal("expected outside path to be rejected")
		}
	})
}
