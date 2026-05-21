package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/core"
	"github.com/bhaskarjha-com/niyantra/internal/cursor"
)

// Cursor implements the core.Provider interface for Cursor.
type Cursor struct{}

var _ core.Provider = (*Cursor)(nil)

func (p *Cursor) ID() string {
	return "cursor"
}

func (p *Cursor) Name() string {
	return "Cursor"
}

func (p *Cursor) Category() core.Category {
	return core.CategoryCoding
}

func (p *Cursor) Capabilities() core.Cap {
	return core.CapQuota | core.CapReset
}

func (p *Cursor) AuthMethods() []core.AuthMethod {
	return []core.AuthMethod{
		{
			Type:       core.AuthCookie,
			Priority:   1,
			Label:      "Session Cookie",
			ConfigKeys: []string{"cursor_session_token"},
		},
	}
}

func (p *Cursor) AutoDiscover() (*core.Credentials, error) {
	creds, err := cursor.DetectCredentials(nil, "")
	if err != nil {
		return nil, fmt.Errorf("cursor autodiscover: %w", err)
	}
	return &core.Credentials{
		Cookie: creds.SessionCookie(),
		Extra: map[string]string{
			"email":          creds.Email,
			"user_id":        creds.UserID,
			"access_token":   creds.AccessToken,
			"source":         creds.Source,
			"membershipType": creds.StripeMembershipType,
		},
	}, nil
}

func (p *Cursor) Fetch(ctx context.Context, creds *core.Credentials) (*core.Snapshot, error) {
	if creds == nil || (creds.Cookie == "" && creds.AccessToken == "") {
		return nil, fmt.Errorf("cursor: missing credentials")
	}

	// Reconstruct cursor.Credentials from core.Credentials
	cursorCreds := &cursor.Credentials{
		UserID:      creds.Extra["user_id"],
		AccessToken: creds.Extra["access_token"],
		Email:       creds.Extra["email"],
		Source:      creds.Extra["source"],
	}

	// If no user_id/access_token in extra but cookie is provided, we try to parse WorkosCursorSessionToken
	if (cursorCreds.UserID == "" || cursorCreds.AccessToken == "") && creds.Cookie != "" {
		// WorkosCursorSessionToken=userId%3A%3AaccessToken
		// Let's parse it manually or try DetectCredentials with the token
		token := creds.Cookie
		if idx := len("WorkosCursorSessionToken="); len(token) > idx && token[:idx] == "WorkosCursorSessionToken=" {
			token = token[idx:]
		}
		// Try parsing to see if we can extract userid and access_token
		parts := cSplitCookie(token)
		if len(parts) == 2 {
			cursorCreds.UserID = parts[0]
			cursorCreds.AccessToken = parts[1]
		} else {
			// fallback: just set AccessToken to token
			cursorCreds.AccessToken = token
		}
	}

	if cursorCreds.AccessToken == "" {
		// fallback to APIKey if present
		cursorCreds.AccessToken = creds.APIKey
	}

	client := cursor.NewClient(cursorCreds, nil)
	snapshot, err := client.FetchSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("cursor fetch: %w", err)
	}

	overallPct := 0.0
	if snapshot.BillingModel == "request_count" {
		if snapshot.RequestsMax > 0 {
			overallPct = (float64(snapshot.RequestsMax-snapshot.RequestsUsed) / float64(snapshot.RequestsMax)) * 100
		}
	} else if snapshot.BillingModel == "usd_credit" {
		if snapshot.LimitCents > 0 {
			overallPct = (float64(snapshot.LimitCents-snapshot.UsedCents) / float64(snapshot.LimitCents)) * 100
		}
	}

	email := cursorCreds.Email
	if email == "" {
		email = cursorCreds.UserID
	}
	if email == "" {
		email = "Cursor Account"
	}

	data := map[string]any{
		"premium_used":   snapshot.RequestsUsed,
		"premium_limit":  snapshot.RequestsMax,
		"start_of_month": snapshot.CycleStart,
		// adding extra mapping as required by task
		"basic_remaining": 0,
		"plan":            snapshot.PlanTier,
	}

	// Store billing details as models_json format (matches formatCursorModelsJSON)
	modelsMap := map[string]any{
		"billingModel": snapshot.BillingModel,
		"usedCents":    snapshot.UsedCents,
		"limitCents":   snapshot.LimitCents,
		"autoPct":      snapshot.AutoPercentUsed,
		"apiPct":       snapshot.APIPercentUsed,
		"cycleEnd":     snapshot.CycleEnd,
	}

	var resetAt *time.Time
	if snapshot.CycleEnd != "" {
		if t, err := time.Parse(time.RFC3339, snapshot.CycleEnd); err == nil {
			resetAt = &t
		}
	}

	return &core.Snapshot{
		Provider:      "cursor",
		Email:         email,
		AccountID:     cursorCreds.UserID,
		OverallPct:    overallPct,
		PlanTier:      snapshot.PlanTier,
		CostUSD:       0.0,
		ResetAt:       resetAt,
		ResetType:     "monthly",
		Data:          data,
		Models:        modelsMap,
		CaptureMethod: "auto",
		CaptureSource: "cookie",
	}, nil
}

func cSplitCookie(cookie string) []string {
	// split by "::" (which might be url-encoded as %3A%3A)
	if strings.Contains(cookie, "::") {
		return strings.SplitN(cookie, "::", 2)
	}
	if strings.Contains(cookie, "%3A%3A") {
		parts := strings.SplitN(cookie, "%3A%3A", 2)
		// Decode each part
		return parts
	}
	return nil
}

func (p *Cursor) ConfigSchema() []core.ConfigField {
	return []core.ConfigField{
		{
			Key:      "cursor_session_token",
			Label:    "Session Cookie Token",
			Type:     core.FieldSecret,
			Required: true,
			Hint:     "Cursor session token from WorkosCursorSessionToken cookie value",
		},
	}
}

func (p *Cursor) Color() string {
	return "#007acc"
}

func (p *Cursor) Icon() string {
	return "💻"
}
