package store

import (
	"path/filepath"
	"testing"
)

func TestDashboardTokenLifecycle(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "token.db"), WithSecretBackend(NewMemorySecretBackend()))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	token, created, err := s.EnsureDashboardToken()
	if err != nil {
		t.Fatalf("EnsureDashboardToken: %v", err)
	}
	if !created {
		t.Fatal("expected first EnsureDashboardToken call to create a token")
	}
	if len(token) < 32 {
		t.Fatalf("token length = %d, want at least 32", len(token))
	}

	again, created, err := s.EnsureDashboardToken()
	if err != nil {
		t.Fatalf("EnsureDashboardToken again: %v", err)
	}
	if created {
		t.Fatal("expected second EnsureDashboardToken call to reuse existing token")
	}
	if again != token {
		t.Fatal("EnsureDashboardToken returned a different token without rotation")
	}

	rotated, err := s.RotateDashboardToken()
	if err != nil {
		t.Fatalf("RotateDashboardToken: %v", err)
	}
	if rotated == token {
		t.Fatal("RotateDashboardToken did not change the token")
	}
}
