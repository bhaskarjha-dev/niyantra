package providers

import (
	"context"
	"testing"

	"github.com/bhaskarjha-com/niyantra/internal/core"
)

func TestCopilotProviderMetadata(t *testing.T) {
	p := &Copilot{}

	if p.ID() != "copilot" {
		t.Errorf("expected ID copilot, got %q", p.ID())
	}
	if p.Name() != "GitHub Copilot" {
		t.Errorf("expected Name GitHub Copilot, got %q", p.Name())
	}
	if p.Category() != core.CategoryCoding {
		t.Errorf("expected Category coding, got %q", p.Category())
	}
	if !p.Capabilities().Has(core.CapQuota) {
		t.Errorf("expected capability CapQuota")
	}

	auths := p.AuthMethods()
	if len(auths) != 1 || auths[0].Type != core.AuthAPIKey {
		t.Errorf("unexpected AuthMethods: %+v", auths)
	}

	schema := p.ConfigSchema()
	if len(schema) != 1 || schema[0].Key != "copilot_pat" {
		t.Errorf("unexpected ConfigSchema: %+v", schema)
	}
}

func TestCopilotProviderFetch_MissingCreds(t *testing.T) {
	p := &Copilot{}
	_, err := p.Fetch(context.Background(), nil)
	if err == nil {
		t.Error("expected error with nil credentials, got nil")
	}

	_, err = p.Fetch(context.Background(), &core.Credentials{})
	if err == nil {
		t.Error("expected error with empty API key, got nil")
	}
}
