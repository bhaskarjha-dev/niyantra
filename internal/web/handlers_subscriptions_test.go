package web

import (
	"encoding/csv"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bhaskarjha-com/niyantra/internal/store"
)

func TestHandleOverviewDedupesQuickLinks(t *testing.T) {
	st := openTestStore(t)
	srv := &Server{
		logger:   slog.Default(),
		store:    st,
		agentMgr: nil,
	}

	insertTestSubscription(t, st, &store.Subscription{
		Platform:     "Antigravity",
		Category:     "coding",
		PlanName:     "Legacy",
		Status:       "cancelled",
		URL:          "https://old.example.com",
		CostCurrency: "USD",
		BillingCycle: "monthly",
	})
	insertTestSubscription(t, st, &store.Subscription{
		Platform:     "Antigravity",
		Category:     "coding",
		PlanName:     "Current",
		Status:       "active",
		URL:          "https://new.example.com",
		CostCurrency: "USD",
		BillingCycle: "monthly",
	})
	insertTestSubscription(t, st, &store.Subscription{
		Platform:     "Cursor",
		Category:     "coding",
		PlanName:     "Pro",
		Status:       "active",
		URL:          "https://cursor.example.com",
		CostCurrency: "USD",
		BillingCycle: "monthly",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/overview", nil)
	rec := httptest.NewRecorder()
	srv.handleOverview(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload struct {
		QuickLinks []struct {
			Platform string `json:"platform"`
			URL      string `json:"url"`
		} `json:"quickLinks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(payload.QuickLinks) != 2 {
		t.Fatalf("expected 2 quick links, got %d", len(payload.QuickLinks))
	}

	links := make(map[string]string, len(payload.QuickLinks))
	for _, link := range payload.QuickLinks {
		links[link.Platform] = link.URL
	}

	if links["Antigravity"] != "https://new.example.com" {
		t.Fatalf("Antigravity link = %q, want %q", links["Antigravity"], "https://new.example.com")
	}
	if links["Cursor"] != "https://cursor.example.com" {
		t.Fatalf("Cursor link = %q, want %q", links["Cursor"], "https://cursor.example.com")
	}
}

func TestHandleExportCSVIncludesQuotaLimitColumns(t *testing.T) {
	st := openTestStore(t)
	srv := &Server{
		logger: slog.Default(),
		store:  st,
	}

	insertTestSubscription(t, st, &store.Subscription{
		Platform:     "Cursor",
		Category:     "coding",
		PlanName:     "Ultra",
		Status:       "active",
		CostAmount:   20,
		CostCurrency: "USD",
		BillingCycle: "monthly",
		TokenLimit:   123,
		CreditLimit:  456,
		RequestLimit: 789,
		LimitPeriod:  "monthly",
		LimitNote:    "shared quota",
		URL:          "https://cursor.example.com",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/export/csv", nil)
	rec := httptest.NewRecorder()
	srv.handleExportCSV(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	rows, err := csv.NewReader(strings.NewReader(rec.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll CSV: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 CSV rows, got %d", len(rows))
	}

	header := strings.Join(rows[0], ",")
	for _, col := range []string{"Token Limit", "Credit Limit", "Request Limit", "Limit Period", "Limit Note"} {
		if !strings.Contains(header, col) {
			t.Fatalf("expected CSV header to contain %q, got %q", col, header)
		}
	}

	row := strings.Join(rows[1], ",")
	for _, value := range []string{"123", "456", "789", "monthly", "shared quota"} {
		if !strings.Contains(row, value) {
			t.Fatalf("expected CSV row to contain %q, got %q", value, row)
		}
	}
}

func insertTestSubscription(t *testing.T, st *store.Store, sub *store.Subscription) {
	t.Helper()
	if _, err := st.InsertSubscription(sub); err != nil {
		t.Fatalf("InsertSubscription(%s): %v", sub.Platform, err)
	}
}
