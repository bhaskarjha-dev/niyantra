package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAnomaliesDisabledWithoutHistory(t *testing.T) {
	srv := &Server{
		logger: slog.Default(),
		store:  openTestStore(t),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/anomalies", nil)
	rec := httptest.NewRecorder()
	srv.handleAnomalies(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload struct {
		Anomalies []any  `json:"anomalies"`
		Disabled  bool   `json:"disabled"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if !payload.Disabled {
		t.Fatal("expected anomaly detection to be disabled")
	}
	if len(payload.Anomalies) != 0 {
		t.Fatalf("expected no anomalies, got %d", len(payload.Anomalies))
	}
	if payload.Reason == "" {
		t.Fatal("expected disabled reason to be populated")
	}
}

func TestHandleCostIncludesTruthMetadata(t *testing.T) {
	srv := &Server{
		logger: slog.Default(),
		store:  openTestStore(t),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/cost", nil)
	rec := httptest.NewRecorder()
	srv.handleCost(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload struct {
		Basis                  string `json:"basis"`
		Confidence             string `json:"confidence"`
		ObservedSpendAvailable bool   `json:"observedSpendAvailable"`
		Reason                 string `json:"notAccountingGradeReason"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Basis != "estimated_quota_fraction" {
		t.Fatalf("basis = %q, want estimated_quota_fraction", payload.Basis)
	}
	if payload.Confidence == "" {
		t.Fatal("expected confidence metadata")
	}
	if payload.ObservedSpendAvailable {
		t.Fatal("observedSpendAvailable should be false for quota-derived cost estimates")
	}
	if payload.Reason == "" {
		t.Fatal("expected notAccountingGradeReason")
	}
}
