package core_test

import (
	"context"
	"testing"

	"github.com/bhaskarjha-com/niyantra/internal/core"
)

// mockProvider is a minimal Provider implementation for testing.
type mockProvider struct {
	id       string
	name     string
	category core.Category
	caps     core.Cap
	color    string
}

func (m *mockProvider) ID() string                { return m.id }
func (m *mockProvider) Name() string              { return m.name }
func (m *mockProvider) Category() core.Category   { return m.category }
func (m *mockProvider) Capabilities() core.Cap    { return m.caps }
func (m *mockProvider) AuthMethods() []core.AuthMethod { return nil }
func (m *mockProvider) AutoDiscover() (*core.Credentials, error) { return nil, nil }
func (m *mockProvider) Fetch(ctx context.Context, creds *core.Credentials) (*core.Snapshot, error) {
	return &core.Snapshot{Provider: m.id, OverallPct: 100}, nil
}
func (m *mockProvider) ConfigSchema() []core.ConfigField { return nil }
func (m *mockProvider) Color() string             { return m.color }
func (m *mockProvider) Icon() string              { return "🧪" }

func TestRegistryRegisterAndGet(t *testing.T) {
	core.Reset()
	defer core.Reset()

	mp := &mockProvider{id: "test", name: "Test Provider", category: core.CategoryAPI}
	core.Register(mp)

	got := core.Get("test")
	if got == nil {
		t.Fatal("expected to find provider 'test', got nil")
	}
	if got.Name() != "Test Provider" {
		t.Errorf("expected name 'Test Provider', got %q", got.Name())
	}
}

func TestRegistryGetUnknown(t *testing.T) {
	core.Reset()
	defer core.Reset()

	got := core.Get("nonexistent")
	if got != nil {
		t.Errorf("expected nil for unknown provider, got %v", got)
	}
}

func TestRegistryDuplicatePanics(t *testing.T) {
	core.Reset()
	defer core.Reset()

	mp1 := &mockProvider{id: "dup", name: "First"}
	mp2 := &mockProvider{id: "dup", name: "Second"}

	core.Register(mp1)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate registration, got none")
		}
	}()
	core.Register(mp2)
}

func TestRegistryAll(t *testing.T) {
	core.Reset()
	defer core.Reset()

	core.Register(&mockProvider{id: "bravo", name: "B"})
	core.Register(&mockProvider{id: "alpha", name: "A"})
	core.Register(&mockProvider{id: "charlie", name: "C"})

	all := core.All()
	if len(all) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(all))
	}
	// Should be sorted by ID
	if all[0].ID() != "alpha" || all[1].ID() != "bravo" || all[2].ID() != "charlie" {
		t.Errorf("providers not sorted: %s, %s, %s", all[0].ID(), all[1].ID(), all[2].ID())
	}
}

func TestCapHas(t *testing.T) {
	caps := core.CapQuota | core.CapReset | core.CapModels
	if !caps.Has(core.CapQuota) {
		t.Error("expected CapQuota to be set")
	}
	if caps.Has(core.CapBalance) {
		t.Error("expected CapBalance to NOT be set")
	}
}
