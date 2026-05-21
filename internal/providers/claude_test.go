package providers

import (
	"context"
	"testing"

	"github.com/bhaskarjha-com/niyantra/internal/core"
)

func TestClaudeProviderMetadata(t *testing.T) {
	p := &Claude{}

	if p.ID() != "claude" {
		t.Errorf("expected ID claude, got %q", p.ID())
	}
	if p.Name() != "Claude Code" {
		t.Errorf("expected Name Claude Code, got %q", p.Name())
	}
	if p.Category() != core.CategoryChat {
		t.Errorf("expected Category chat, got %q", p.Category())
	}
	if !p.Capabilities().Has(core.CapQuota) {
		t.Errorf("expected capability CapQuota")
	}

	auths := p.AuthMethods()
	if len(auths) != 1 || auths[0].Type != core.AuthProcessRPC {
		t.Errorf("unexpected AuthMethods: %+v", auths)
	}

	schema := p.ConfigSchema()
	if len(schema) != 1 || schema[0].Key != "claude_bridge" {
		t.Errorf("unexpected ConfigSchema: %+v", schema)
	}
}

func TestClaudeProviderFetch_StaleOrMissing(t *testing.T) {
	p := &Claude{}
	// Since we are running in a clean environment, the statusline file is likely missing,
	// so Fetch should return an error.
	_, err := p.Fetch(context.Background(), &core.Credentials{})
	if err == nil {
		t.Error("expected error for stale/missing statusline data, got nil")
	}
}
