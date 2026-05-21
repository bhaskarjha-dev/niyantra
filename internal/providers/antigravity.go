package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/client"
	"github.com/bhaskarjha-com/niyantra/internal/core"
)

// Antigravity implements the core.Provider interface for Antigravity.
type Antigravity struct{}

var _ core.Provider = (*Antigravity)(nil)

func (p *Antigravity) ID() string {
	return "antigravity"
}

func (p *Antigravity) Name() string {
	return "Antigravity"
}

func (p *Antigravity) Category() core.Category {
	return core.CategoryCoding
}

func (p *Antigravity) Capabilities() core.Cap {
	return core.CapQuota | core.CapReset | core.CapModels | core.CapMultiAcct
}

func (p *Antigravity) AuthMethods() []core.AuthMethod {
	return []core.AuthMethod{
		{
			Type:       core.AuthProcessRPC,
			Priority:   1,
			Label:      "Local Process RPC",
			ConfigKeys: []string{},
		},
	}
}

func (p *Antigravity) AutoDiscover() (*core.Credentials, error) {
	return &core.Credentials{}, nil
}

func (p *Antigravity) Fetch(ctx context.Context, creds *core.Credentials) (*core.Snapshot, error) {
	c := client.New(slog.Default())
	resps, err := c.FetchQuotas(ctx)
	if err != nil {
		return nil, fmt.Errorf("antigravity fetch: %w", err)
	}

	if len(resps) == 0 {
		return nil, fmt.Errorf("antigravity: no user status responses found")
	}

	now := time.Now().UTC()
	var email string
	var planTier string
	var allModels []any
	var allAICredits []any

	if len(resps) == 1 {
		legacySnap := resps[0].ToSnapshot(now)
		email = legacySnap.Email
		planTier = legacySnap.PlanName

		for _, m := range legacySnap.Models {
			allModels = append(allModels, m)
		}
		for _, ac := range legacySnap.AICredits {
			allAICredits = append(allAICredits, ac)
		}
	} else {
		emails := make([]string, 0, len(resps))
		plans := make([]string, 0, len(resps))
		for _, resp := range resps {
			legacySnap := resp.ToSnapshot(now)
			emails = append(emails, legacySnap.Email)
			plans = append(plans, legacySnap.PlanName)

			for _, m := range legacySnap.Models {
				m.Label = fmt.Sprintf("%s (%s)", m.Label, legacySnap.Email)
				allModels = append(allModels, m)
			}
			for _, ac := range legacySnap.AICredits {
				allAICredits = append(allAICredits, ac)
			}
		}
		email = strings.Join(emails, ", ")
		planTier = strings.Join(plans, ", ")
	}

	rawJSONBytes, _ := json.Marshal(resps)

	data := map[string]any{
		"prompt_credits":  0.0,
		"monthly_credits": 0,
		"raw_json":        string(rawJSONBytes),
		"ai_credits_json": "[]",
	}

	if len(resps) == 1 {
		legacySnap := resps[0].ToSnapshot(now)
		data["prompt_credits"] = legacySnap.PromptCredits
		data["monthly_credits"] = legacySnap.MonthlyCredits
		if b, err := json.Marshal(legacySnap.AICredits); err == nil {
			data["ai_credits_json"] = string(b)
		}
	} else {
		if b, err := json.Marshal(allAICredits); err == nil {
			data["ai_credits_json"] = string(b)
		}
	}

	return &core.Snapshot{
		Provider:      "antigravity",
		Email:         email,
		OverallPct:    0.0,
		PlanTier:      planTier,
		CostUSD:       0.0,
		Data:          data,
		Models:        allModels,
		CaptureMethod: "auto",
		CaptureSource: "ls_poll",
	}, nil
}

func (p *Antigravity) ConfigSchema() []core.ConfigField {
	return []core.ConfigField{}
}

func (p *Antigravity) Color() string {
	return "#3B82F6"
}

func (p *Antigravity) Icon() string {
	return "🌌"
}
