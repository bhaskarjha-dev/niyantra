package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/client"
	"github.com/bhaskarjha-com/niyantra/internal/store"
	"github.com/bhaskarjha-com/niyantra/internal/tracker"
)

func TestHandleUsageIncludesAccountIdentityPerModel(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "usage_identity.db"), store.WithSecretBackend(store.NewMemorySecretBackend()))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer st.Close()

	now := time.Now().UTC()
	resetTime := now.Add(2 * time.Hour)

	accountA, err := st.GetOrCreateAccount("alpha@example.com", "Pro", "antigravity")
	if err != nil {
		t.Fatalf("GetOrCreateAccount A: %v", err)
	}
	accountB, err := st.GetOrCreateAccount("beta@example.com", "Pro", "antigravity")
	if err != nil {
		t.Fatalf("GetOrCreateAccount B: %v", err)
	}

	for _, snap := range []*client.Snapshot{
		{
			AccountID:     accountA,
			CapturedAt:    now,
			Email:         "alpha@example.com",
			PlanName:      "Pro",
			CaptureMethod: "manual",
			CaptureSource: "test",
			SourceID:      "antigravity",
			Models: []client.ModelQuota{
				{ModelID: "claude-sonnet", Label: "Claude Sonnet", RemainingFraction: 0.8, RemainingPercent: 80, ResetTime: &resetTime},
			},
		},
		{
			AccountID:     accountB,
			CapturedAt:    now.Add(30 * time.Second),
			Email:         "beta@example.com",
			PlanName:      "Pro",
			CaptureMethod: "manual",
			CaptureSource: "test",
			SourceID:      "antigravity",
			Models: []client.ModelQuota{
				{ModelID: "claude-sonnet", Label: "Claude Sonnet", RemainingFraction: 0.6, RemainingPercent: 60, ResetTime: &resetTime},
			},
		},
	} {
		if _, err := st.InsertSnapshot(snap); err != nil {
			t.Fatalf("InsertSnapshot(%s): %v", snap.Email, err)
		}
	}

	srv := &Server{
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		store:   st,
		tracker: tracker.New(st, slog.New(slog.NewTextHandler(io.Discard, nil))),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/usage", nil)
	rec := httptest.NewRecorder()
	srv.handleUsage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var payload struct {
		Models []struct {
			AccountID    int64  `json:"accountId"`
			AccountEmail string `json:"accountEmail"`
			Provider     string `json:"provider"`
			ModelID      string `json:"modelId"`
		} `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(payload.Models) != 2 {
		t.Fatalf("expected 2 usage rows, got %d", len(payload.Models))
	}

	got := map[int64]string{}
	for _, model := range payload.Models {
		got[model.AccountID] = model.AccountEmail
		if model.Provider != "antigravity" {
			t.Fatalf("provider = %q, want %q", model.Provider, "antigravity")
		}
		if model.ModelID != "claude-sonnet" {
			t.Fatalf("modelID = %q, want %q", model.ModelID, "claude-sonnet")
		}
	}

	if got[accountA] != "alpha@example.com" {
		t.Fatalf("account A email = %q, want %q", got[accountA], "alpha@example.com")
	}
	if got[accountB] != "beta@example.com" {
		t.Fatalf("account B email = %q, want %q", got[accountB], "beta@example.com")
	}
}
