package providers

import (
	"context"
	"strings"
	"testing"

	"github.com/bhaskarjha-com/niyantra/internal/core"
)

func TestAntigravityProviderMetadata(t *testing.T) {
	p := &Antigravity{}

	if p.ID() != "antigravity" {
		t.Errorf("expected ID antigravity, got %q", p.ID())
	}
	if p.Name() != "Antigravity" {
		t.Errorf("expected Name Antigravity, got %q", p.Name())
	}
	if p.Category() != core.CategoryCoding {
		t.Errorf("expected Category coding, got %q", p.Category())
	}
	if !p.Capabilities().Has(core.CapQuota) {
		t.Errorf("expected capability CapQuota")
	}
	if !p.Capabilities().Has(core.CapReset) {
		t.Errorf("expected capability CapReset")
	}
	if !p.Capabilities().Has(core.CapModels) {
		t.Errorf("expected capability CapModels")
	}
	if !p.Capabilities().Has(core.CapMultiAcct) {
		t.Errorf("expected capability CapMultiAcct")
	}

	auths := p.AuthMethods()
	if len(auths) != 1 || auths[0].Type != core.AuthProcessRPC {
		t.Errorf("unexpected AuthMethods: %+v", auths)
	}

	schema := p.ConfigSchema()
	if len(schema) != 0 {
		t.Errorf("unexpected ConfigSchema: %+v", schema)
	}

	if p.Color() != "#3B82F6" {
		t.Errorf("unexpected Color: %q", p.Color())
	}

	if p.Icon() != "🌌" {
		t.Errorf("unexpected Icon: %q", p.Icon())
	}
}

func TestAntigravityProviderFetch_Smoke(t *testing.T) {
	p := &Antigravity{}
	snap, err := p.Fetch(context.Background(), &core.Credentials{})
	if err != nil {
		errStr := err.Error()
		// If it failed because process or port wasn't found or connection was refused,
		// that's expected if the language server isn't running.
		isExpectedEnvError := strings.Contains(errStr, "not found") ||
			strings.Contains(errStr, "failed") ||
			strings.Contains(errStr, "refused") ||
			strings.Contains(errStr, "connection")
		if !isExpectedEnvError {
			t.Errorf("unexpected error on fetch: %v", err)
		}
		return
	}

	// If it succeeded (because a local language server is running), verify basic snapshot properties
	if snap.Provider != "antigravity" {
		t.Errorf("expected provider antigravity, got %q", snap.Provider)
	}
	if snap.CaptureSource != "ls_poll" {
		t.Errorf("expected capture source ls_poll, got %q", snap.CaptureSource)
	}
}
