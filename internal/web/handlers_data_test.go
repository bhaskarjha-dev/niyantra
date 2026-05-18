package web

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/client"
)

func TestHandleSnapAdjustTargetsModelID(t *testing.T) {
	st := openTestStore(t)

	accountID, err := st.GetOrCreateAccount("adjust@example.com", "Pro", "antigravity")
	if err != nil {
		t.Fatalf("GetOrCreateAccount: %v", err)
	}

	snapID, err := st.InsertSnapshot(&client.Snapshot{
		AccountID:  accountID,
		CapturedAt: time.Now().UTC(),
		Email:      "adjust@example.com",
		PlanName:   "Pro",
		Models: []client.ModelQuota{
			{ModelID: "model-a", Label: "Duplicate Label", RemainingFraction: 0.8, RemainingPercent: 80},
			{ModelID: "model-b", Label: "Duplicate Label", RemainingFraction: 0.6, RemainingPercent: 60},
		},
		CaptureMethod: "manual",
		CaptureSource: "ui",
		SourceID:      "antigravity",
	})
	if err != nil {
		t.Fatalf("InsertSnapshot: %v", err)
	}

	srv := &Server{
		logger: slog.Default(),
		store:  st,
	}

	body, err := json.Marshal(map[string]any{
		"snapshotId": snapID,
		"adjustments": []map[string]any{
			{"modelId": "model-b", "remainingPercent": 15},
		},
	})
	if err != nil {
		t.Fatalf("Marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/snap/adjust", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.handleSnapAdjust(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	updated, err := st.GetSnapshotByID(snapID)
	if err != nil {
		t.Fatalf("GetSnapshotByID: %v", err)
	}

	models := map[string]client.ModelQuota{}
	for _, m := range updated.Models {
		models[m.ModelID] = m
	}

	if got := models["model-a"].RemainingPercent; got != 80 {
		t.Fatalf("model-a remaining percent = %.0f, want 80", got)
	}
	if got := models["model-b"].RemainingPercent; got != 15 {
		t.Fatalf("model-b remaining percent = %.0f, want 15", got)
	}
}
