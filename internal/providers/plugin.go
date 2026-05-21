package providers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bhaskarjha-com/niyantra/internal/core"
	"github.com/bhaskarjha-com/niyantra/internal/plugin"
)

// Plugin implements core.Provider for a dynamic external plugin.
type Plugin struct {
	pl *plugin.Plugin
}

var _ core.Provider = (*Plugin)(nil)

// NewPlugin creates a new Plugin provider wrapping a specific dynamic plugin.
func NewPlugin(pl *plugin.Plugin) *Plugin {
	return &Plugin{pl: pl}
}

func (p *Plugin) ID() string {
	if p.pl == nil {
		return "plugin"
	}
	return "plugin_" + p.pl.Manifest.ID
}

func (p *Plugin) Name() string {
	if p.pl == nil {
		return "Plugin"
	}
	return p.pl.Manifest.Name
}

func (p *Plugin) Category() core.Category {
	return core.CategoryCustom
}

func (p *Plugin) Capabilities() core.Cap {
	return core.CapQuota
}

func (p *Plugin) AuthMethods() []core.AuthMethod {
	return nil
}

func (p *Plugin) AutoDiscover() (*core.Credentials, error) {
	return &core.Credentials{}, nil
}

func (p *Plugin) Fetch(ctx context.Context, creds *core.Credentials) (*core.Snapshot, error) {
	if p.pl == nil {
		return nil, fmt.Errorf("plugin provider: manifest plugin is nil")
	}

	// Reconstruct the plugin's configuration map from credentials extra keys.
	prefix := "plugin_" + p.pl.Manifest.ID + "_"
	p.pl.Config = make(map[string]string)
	if creds != nil && creds.Extra != nil {
		for k, v := range creds.Extra {
			if strings.HasPrefix(k, prefix) {
				originalKey := strings.TrimPrefix(k, prefix)
				p.pl.Config[originalKey] = v
			}
		}
	}

	// Fill defaults for any missing config keys.
	for key, field := range p.pl.Manifest.Config {
		if _, exists := p.pl.Config[key]; !exists {
			p.pl.Config[key] = field.Default
		}
	}

	// Run the plugin subprocess
	result, err := p.pl.Run(ctx, slog.Default())
	if err != nil {
		return nil, fmt.Errorf("plugin %s run: %w", p.pl.Manifest.ID, err)
	}

	if result.Status != "ok" {
		return nil, fmt.Errorf("plugin %s returned error: %s", p.pl.Manifest.ID, result.Error)
	}

	// Map overall percent, using fallback metadata keys if usage_pct is empty.
	overallPct := result.Data.UsagePct
	if overallPct == 0 {
		for k, v := range result.Data.Metadata {
			kLower := strings.ToLower(k)
			if strings.Contains(kLower, "pct") || strings.Contains(kLower, "percent") {
				if f, ok := v.(float64); ok {
					overallPct = f
					break
				}
			}
		}
	}

	data := map[string]any{
		"usage_display": result.Data.UsageDisplay,
		"metadata":      result.Data.Metadata,
	}

	// Flatten metadata fields directly to the top-level data JSON for convenience.
	for k, v := range result.Data.Metadata {
		data[k] = v
	}

	return &core.Snapshot{
		Provider:      "plugin_" + p.pl.Manifest.ID,
		Email:         result.Data.Email,
		OverallPct:    overallPct,
		PlanTier:      result.Data.Plan,
		CostUSD:       0.0,
		Data:          data,
		Models:        result.Data.Models,
		CaptureMethod: "auto",
		CaptureSource: "plugin",
	}, nil
}

func (p *Plugin) ConfigSchema() []core.ConfigField {
	if p.pl == nil {
		return nil
	}
	var fields []core.ConfigField
	for key, f := range p.pl.Manifest.Config {
		fieldType := core.FieldString
		switch f.Type {
		case "bool":
			fieldType = core.FieldBool
		case "int":
			fieldType = core.FieldInt
		default:
			if f.Secret {
				fieldType = core.FieldSecret
			}
		}
		fields = append(fields, core.ConfigField{
			Key:      "plugin_" + p.pl.Manifest.ID + "_" + key,
			Label:    f.Label,
			Type:     fieldType,
			Required: f.Required,
			Default:  f.Default,
		})
	}
	return fields
}

func (p *Plugin) Color() string {
	return "#64748B"
}

func (p *Plugin) Icon() string {
	return "🔌"
}
