package providers

import (
	"context"
	"testing"

	"github.com/bhaskarjha-com/niyantra/internal/core"
)

func TestCursorProviderMetadata(t *testing.T) {
	p := &Cursor{}

	if p.ID() != "cursor" {
		t.Errorf("expected ID cursor, got %q", p.ID())
	}
	if p.Name() != "Cursor" {
		t.Errorf("expected Name Cursor, got %q", p.Name())
	}
	if p.Category() != core.CategoryCoding {
		t.Errorf("expected Category coding, got %q", p.Category())
	}
	if !p.Capabilities().Has(core.CapQuota) {
		t.Errorf("expected capability CapQuota")
	}

	auths := p.AuthMethods()
	if len(auths) != 1 || auths[0].Type != core.AuthCookie {
		t.Errorf("unexpected AuthMethods: %+v", auths)
	}

	schema := p.ConfigSchema()
	if len(schema) != 1 || schema[0].Key != "cursor_session_token" {
		t.Errorf("unexpected ConfigSchema: %+v", schema)
	}
}

func TestCursorProviderFetch_MissingCreds(t *testing.T) {
	p := &Cursor{}
	_, err := p.Fetch(context.Background(), nil)
	if err == nil {
		t.Error("expected error with nil credentials, got nil")
	}

	_, err = p.Fetch(context.Background(), &core.Credentials{})
	if err == nil {
		t.Error("expected error with empty credentials, got nil")
	}
}
