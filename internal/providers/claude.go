package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/claude"
	"github.com/bhaskarjha-com/niyantra/internal/core"
)

// Claude implements the core.Provider interface for Claude Code.
type Claude struct{}

var _ core.Provider = (*Claude)(nil)

func (p *Claude) ID() string {
	return "claude"
}

func (p *Claude) Name() string {
	return "Claude Code"
}

func (p *Claude) Category() core.Category {
	return core.CategoryChat
}

func (p *Claude) Capabilities() core.Cap {
	return core.CapQuota | core.CapReset
}

func (p *Claude) AuthMethods() []core.AuthMethod {
	return []core.AuthMethod{
		{
			Type:       core.AuthProcessRPC,
			Priority:   1,
			Label:      "Statusline Bridge",
			ConfigKeys: []string{"claude_bridge"},
		},
	}
}

func (p *Claude) AutoDiscover() (*core.Credentials, error) {
	// Claude statusline data is collected locally, so discover is always successful if available.
	return &core.Credentials{}, nil
}

func (p *Claude) Fetch(ctx context.Context, creds *core.Credentials) (*core.Snapshot, error) {
	if !claude.IsFresh(claude.DefaultStaleness) {
		return nil, fmt.Errorf("claude statusline data is stale or missing")
	}

	rl, err := claude.ReadData()
	if err != nil {
		return nil, fmt.Errorf("claude read data: %w", err)
	}
	if !claude.IsValid(rl) {
		return nil, fmt.Errorf("claude statusline data is invalid")
	}

	var fiveHourPct float64
	var sevenDayPct *float64
	var fiveReset, sevenReset *time.Time
	var fiveResetStr, sevenResetStr string

	if rl.FiveHour != nil {
		fiveHourPct = rl.FiveHour.UsedPercentage
		if rl.FiveHour.ResetsAt > 0 {
			t := time.Unix(rl.FiveHour.ResetsAt, 0).UTC()
			fiveReset = &t
			fiveResetStr = t.Format(time.RFC3339)
		}
	}
	if rl.SevenDay != nil {
		v := rl.SevenDay.UsedPercentage
		sevenDayPct = &v
		if rl.SevenDay.ResetsAt > 0 {
			t := time.Unix(rl.SevenDay.ResetsAt, 0).UTC()
			sevenReset = &t
			sevenResetStr = t.Format(time.RFC3339)
		}
	}

	data := map[string]any{
		"five_hour_pct":   fiveHourPct,
		"seven_day_pct":   sevenDayPct,
		"five_hour_reset": fiveReset,
		"seven_day_reset": sevenReset,
		"pct_5h":          fiveHourPct,
		"pct_7d":          sevenDayPct,
		"reset_5h":        fiveResetStr,
		"reset_7d":        sevenResetStr,
		"status_text":     "",
	}

	return &core.Snapshot{
		Provider:      "claude",
		Email:         "",
		AccountID:     "claude_local",
		OverallPct:    fiveHourPct,
		PlanTier:      "pro",
		CostUSD:       0.0,
		ResetAt:       fiveReset,
		ResetType:     "5h",
		Data:          data,
		CaptureMethod: "auto",
		CaptureSource: "statusline",
	}, nil
}

func (p *Claude) ConfigSchema() []core.ConfigField {
	return []core.ConfigField{
		{
			Key:      "claude_bridge",
			Label:    "Enable Claude Statusline Bridge",
			Type:     core.FieldBool,
			Required: false,
			Default:  "true",
			Hint:     "Capture Claude Code statusline data from stdout interceptor",
		},
	}
}

func (p *Claude) Color() string {
	return "#D97757"
}

func (p *Claude) Icon() string {
	return "🗣️"
}
