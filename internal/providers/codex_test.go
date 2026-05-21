package providers

import (
	"context"
	"testing"

	"github.com/bhaskarjha-com/niyantra/internal/core"
)

func TestCodexProviderMetadata(t *testing.T) {
	p := &Codex{}

	if p.ID() != "codex" {
		t.Errorf("expected ID codex, got %q", p.ID())
	}
	if p.Name() != "Codex/ChatGPT" {
		t.Errorf("expected Name Codex/ChatGPT, got %q", p.Name())
	}
	if p.Category() != core.CategoryChat {
		t.Errorf("expected Category chat, got %q", p.Category())
	}
	if !p.Capabilities().Has(core.CapQuota) {
		t.Errorf("expected capability CapQuota")
	}

	auths := p.AuthMethods()
	if len(auths) != 1 || auths[0].Type != core.AuthOAuth {
		t.Errorf("unexpected AuthMethods: %+v", auths)
	}

	schema := p.ConfigSchema()
	if len(schema) != 2 || schema[0].Key != "codex_access_token" {
		t.Errorf("unexpected ConfigSchema: %+v", schema)
	}
}

func TestCodexProviderFetch_MissingCreds(t *testing.T) {
	p := &Codex{}
	_, err := p.Fetch(context.Background(), nil)
	if err == nil {
		t.Error("expected error with nil credentials, got nil")
	}

	_, err = p.Fetch(context.Background(), &core.Credentials{})
	if err == nil {
		t.Error("expected error with empty AccessToken, got nil")
	}
}
