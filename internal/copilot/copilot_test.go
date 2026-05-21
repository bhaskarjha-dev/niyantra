package copilot

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectCredentials(t *testing.T) {
	// Create a temp directory to simulate the config directory
	tempDir, err := os.MkdirTemp("", "github-copilot-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("valid hosts.json with github.com", func(t *testing.T) {
		hostsPath := filepath.Join(tempDir, "hosts.json")
		content := `{
			"github.com": {
				"user": "test-user",
				"oauth_token": "ghu_test_token_123"
			}
		}`
		if err := os.WriteFile(hostsPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write mock hosts.json: %v", err)
		}

		token, err := extractTokenFromFile(hostsPath, slog.Default())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "ghu_test_token_123" {
			t.Errorf("expected ghu_test_token_123, got %s", token)
		}
	})

	t.Run("valid hosts.json with other host fallback", func(t *testing.T) {
		hostsPath := filepath.Join(tempDir, "hosts.json")
		content := `{
			"github.enterprise.com": {
				"user": "enterprise-user",
				"oauth_token": "ghu_enterprise_token"
			}
		}`
		if err := os.WriteFile(hostsPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write mock hosts.json: %v", err)
		}

		token, err := extractTokenFromFile(hostsPath, slog.Default())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "ghu_enterprise_token" {
			t.Errorf("expected ghu_enterprise_token, got %s", token)
		}
	})

	t.Run("missing oauth_token", func(t *testing.T) {
		hostsPath := filepath.Join(tempDir, "hosts.json")
		content := `{
			"github.com": {
				"user": "test-user"
			}
		}`
		if err := os.WriteFile(hostsPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write mock hosts.json: %v", err)
		}

		_, err := extractTokenFromFile(hostsPath, slog.Default())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		hostsPath := filepath.Join(tempDir, "hosts.json")
		content := `{invalid json}`
		if err := os.WriteFile(hostsPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write mock hosts.json: %v", err)
		}

		_, err := extractTokenFromFile(hostsPath, slog.Default())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := extractTokenFromFile(filepath.Join(tempDir, "non_existent.json"), slog.Default())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFetchSnapshot_LimitedUser(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/copilot_internal/user" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"login": "test-user-limited",
				"access_type_sku": "free_limited_copilot",
				"copilot_plan": "individual",
				"limited_user_quotas": {
					"chat": 400,
					"completions": 3000
				},
				"monthly_quotas": {
					"chat": 500,
					"completions": 4000
				}
			}`))
			return
		}
		if r.URL.Path == "/user" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"login": "test-user-limited",
				"email": "test@example.com"
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Override endpoints
	oldUsageURL := usageURL
	oldIdentityURL := identityURL
	usageURL = server.URL + "/copilot_internal/user"
	identityURL = server.URL + "/user"
	defer func() {
		usageURL = oldUsageURL
		identityURL = oldIdentityURL
	}()

	client := NewClient("mock-pat", slog.Default())
	snap, err := client.FetchSnapshot(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if snap.Plan != "Free" {
		t.Errorf("expected plan Free, got %s", snap.Plan)
	}
	if snap.Username != "test-user-limited" {
		t.Errorf("expected username test-user-limited, got %s", snap.Username)
	}
	if snap.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", snap.Email)
	}

	// Completions: total 4000, remaining 3000 -> used 1000 -> 25% used
	if snap.PremiumPct != 25.0 {
		t.Errorf("expected completions premium pct 25.0, got %f", snap.PremiumPct)
	}

	// Chat: total 500, remaining 400 -> used 100 -> 20% used
	if snap.ChatPct != 20.0 {
		t.Errorf("expected chat pct 20.0, got %f", snap.ChatPct)
	}
}
