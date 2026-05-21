package providers

import (
	"context"
	"fmt"

	"github.com/bhaskarjha-com/niyantra/internal/copilot"
	"github.com/bhaskarjha-com/niyantra/internal/core"
)

// Copilot implements the core.Provider interface for GitHub Copilot.
type Copilot struct{}

var _ core.Provider = (*Copilot)(nil)

func (p *Copilot) ID() string {
	return "copilot"
}

func (p *Copilot) Name() string {
	return "GitHub Copilot"
}

func (p *Copilot) Category() core.Category {
	return core.CategoryCoding
}

func (p *Copilot) Capabilities() core.Cap {
	return core.CapQuota
}

func (p *Copilot) AuthMethods() []core.AuthMethod {
	return []core.AuthMethod{
		{
			Type:       core.AuthAPIKey,
			Priority:   1,
			Label:      "Personal Access Token",
			ConfigKeys: []string{"copilot_pat"},
		},
	}
}

func (p *Copilot) AutoDiscover() (*core.Credentials, error) {
	token, _, err := copilot.DetectCredentials(nil)
	if err != nil {
		return nil, fmt.Errorf("copilot autodiscover: %w", err)
	}
	return &core.Credentials{APIKey: token}, nil
}

func (p *Copilot) Fetch(ctx context.Context, creds *core.Credentials) (*core.Snapshot, error) {
	if creds == nil || creds.APIKey == "" {
		return nil, fmt.Errorf("copilot: missing PAT api key")
	}

	client := copilot.NewClient(creds.APIKey, nil)
	snap, err := client.FetchSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("copilot fetch: %w", err)
	}

	overallPct := snap.PremiumPct
	if !snap.HasPremium && snap.HasChat {
		overallPct = snap.ChatPct
	}

	data := map[string]any{
		"premium_pct": snap.PremiumPct,
		"chat_pct":    snap.ChatPct,
		"username":    snap.Username,
	}

	models := map[string]any{
		"hasPremium": snap.HasPremium,
		"hasChat":    snap.HasChat,
	}

	return &core.Snapshot{
		Provider:      "copilot",
		Email:         snap.Email,
		OverallPct:    overallPct,
		PlanTier:      snap.Plan,
		CostUSD:       0.0,
		Data:          data,
		Models:        models,
		CaptureMethod: "auto",
		CaptureSource: "api_key",
	}, nil
}

func (p *Copilot) ConfigSchema() []core.ConfigField {
	return []core.ConfigField{
		{
			Key:      "copilot_pat",
			Label:    "Personal Access Token",
			Type:     core.FieldSecret,
			Required: true,
			EnvVar:   "GITHUB_TOKEN",
			Hint:     "GitHub Personal Access Token (PAT) with read:user scope",
			Syncable: false,
		},
	}
}

func (p *Copilot) Color() string {
	return "#24292e"
}

func (p *Copilot) Icon() string {
	return "🤖"
}
