package providers

import (
	"testing"

	"github.com/bhaskarjha-com/niyantra/internal/core"
	"github.com/bhaskarjha-com/niyantra/internal/plugin"
)

func TestPluginProviderMetadata(t *testing.T) {
	// 1. Test placeholder/empty plugin provider
	pEmpty := &Plugin{}
	if pEmpty.ID() != "plugin" {
		t.Errorf("expected ID plugin, got %q", pEmpty.ID())
	}
	if pEmpty.Name() != "Plugin" {
		t.Errorf("expected Name Plugin, got %q", pEmpty.Name())
	}
	if pEmpty.Category() != core.CategoryCustom {
		t.Errorf("expected Category custom, got %q", pEmpty.Category())
	}
	if !pEmpty.Capabilities().Has(core.CapQuota) {
		t.Errorf("expected capability CapQuota")
	}
	if pEmpty.ConfigSchema() != nil {
		t.Errorf("expected nil ConfigSchema for empty plugin provider")
	}

	// 2. Test wrapping a concrete plugin
	pl := &plugin.Plugin{
		Manifest: plugin.Manifest{
			ID:          "test-plugin",
			Name:        "Test Plugin",
			Description: "A test plugin description",
			Config: map[string]plugin.ConfigField{
				"api_key": {
					Type:     "string",
					Label:    "API Key",
					Required: true,
					Secret:   true,
				},
				"debug": {
					Type:    "bool",
					Label:   "Debug Mode",
					Default: "false",
				},
			},
		},
	}
	p := NewPlugin(pl)

	if p.ID() != "plugin_test-plugin" {
		t.Errorf("expected ID plugin_test-plugin, got %q", p.ID())
	}
	if p.Name() != "Test Plugin" {
		t.Errorf("expected Name Test Plugin, got %q", p.Name())
	}

	schema := p.ConfigSchema()
	if len(schema) != 2 {
		t.Fatalf("expected 2 config fields, got %d", len(schema))
	}

	var foundAPIKey, foundDebug bool
	for _, field := range schema {
		switch field.Key {
		case "plugin_test-plugin_api_key":
			foundAPIKey = true
			if field.Type != core.FieldSecret {
				t.Errorf("expected secret type for api_key, got %q", field.Type)
			}
			if !field.Required {
				t.Errorf("expected api_key to be required")
			}
		case "plugin_test-plugin_debug":
			foundDebug = true
			if field.Type != core.FieldBool {
				t.Errorf("expected bool type for debug, got %q", field.Type)
			}
			if field.Default != "false" {
				t.Errorf("expected debug default 'false', got %q", field.Default)
			}
		}
	}

	if !foundAPIKey {
		t.Error("missing config field plugin_test-plugin_api_key")
	}
	if !foundDebug {
		t.Error("missing config field plugin_test-plugin_debug")
	}
}
