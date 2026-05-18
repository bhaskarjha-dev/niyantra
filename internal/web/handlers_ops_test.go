package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/bhaskarjha-com/niyantra/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "niyantra-test.db")
	s, err := store.Open(dbPath)
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
		Config []*store.ConfigEntry `json:"config"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &export); err != nil {
		t.Fatalf("unmarshal export JSON: %v", err)
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
