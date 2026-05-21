package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/codex"
	"github.com/bhaskarjha-com/niyantra/internal/core"
)

// Codex implements the core.Provider interface for Codex/ChatGPT.
type Codex struct{}

var _ core.Provider = (*Codex)(nil)

func (p *Codex) ID() string {
	return "codex"
}

func (p *Codex) Name() string {
	return "Codex/ChatGPT"
}

func (p *Codex) Category() core.Category {
	return core.CategoryChat
}

func (p *Codex) Capabilities() core.Cap {
	return core.CapQuota | core.CapReset | core.CapBalance
}

func (p *Codex) AuthMethods() []core.AuthMethod {
	return []core.AuthMethod{
		{
			Type:       core.AuthOAuth,
			Priority:   1,
			Label:      "OAuth Token",
			ConfigKeys: []string{"codex_access_token", "codex_refresh_token"},
		},
	}
}

func (p *Codex) AutoDiscover() (*core.Credentials, error) {
	creds, err := codex.DetectCredentials(nil)
	if err != nil {
		return nil, fmt.Errorf("codex autodiscover: %w", err)
	}
	extra := make(map[string]string)
	if creds.AccountID != "" {
		extra["account_id"] = creds.AccountID
	}
	if creds.RefreshToken != "" {
		extra["refresh_token"] = creds.RefreshToken
	}
	if creds.Email != "" {
		extra["email"] = creds.Email
	}
	return &core.Credentials{
		AccessToken: creds.AccessToken,
		Extra:       extra,
	}, nil
}

func (p *Codex) Fetch(ctx context.Context, creds *core.Credentials) (*core.Snapshot, error) {
	if creds == nil || creds.AccessToken == "" {
		return nil, fmt.Errorf("codex: missing access token")
	}

	accountID := creds.Extra["account_id"]
	client := codex.NewClient(creds.AccessToken, accountID, nil)

	usage, err := client.FetchUsage(ctx)
	if err != nil {
		return nil, fmt.Errorf("codex fetch: %w", err)
	}

	var fiveHourPct float64
	var sevenDayPct *float64
	var codeReviewPct *float64
	var resetAt *time.Time

	for _, q := range usage.Quotas {
		switch q.Name {
		case "five_hour":
			fiveHourPct = q.Utilization
			resetAt = q.ResetsAt
		case "seven_day":
			val := q.Utilization
			sevenDayPct = &val
		case "code_review":
			val := q.Utilization
			codeReviewPct = &val
		}
	}

	email := creds.Extra["email"]
	if email == "" {
		email = accountID
	}
	if email == "" {
		email = "Codex Account"
	}

	data := map[string]any{
		"five_hour_pct":   fiveHourPct,
		"seven_day_pct":   sevenDayPct,
		"code_review_pct": codeReviewPct,
		"five_hour_reset": resetAt,
		"credits_balance": usage.CreditsBalance,
		"pct_5h":          fiveHourPct,
		"pct_7d":          sevenDayPct,
		"plan_type":       usage.PlanType,
	}

	return &core.Snapshot{
		Provider:      "codex",
		Email:         email,
		AccountID:     accountID,
		OverallPct:    fiveHourPct,
		PlanTier:      usage.PlanType,
		CostUSD:       0.0,
		ResetAt:       resetAt,
		ResetType:     "5h",
		Data:          data,
		CaptureMethod: "auto",
		CaptureSource: "oauth",
	}, nil
}

func (p *Codex) ConfigSchema() []core.ConfigField {
	return []core.ConfigField{
		{
			Key:      "codex_access_token",
			Label:    "Access Token",
			Type:     core.FieldSecret,
			Required: true,
			Hint:     "ChatGPT OAuth access token",
		},
		{
			Key:      "codex_refresh_token",
			Label:    "Refresh Token",
			Type:     core.FieldSecret,
			Required: false,
			Hint:     "ChatGPT OAuth refresh token (used for rotation)",
		},
	}
}

func (p *Codex) Color() string {
	return "#10a37f"
}

func (p *Codex) Icon() string {
	return "💬"
}
