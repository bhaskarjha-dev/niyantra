package agent

import (
	"errors"
	"strings"

	"github.com/bhaskarjha-com/niyantra/internal/client"
	"github.com/bhaskarjha-com/niyantra/internal/core"
	"github.com/bhaskarjha-com/niyantra/internal/store"
)

// isProviderEnabled checks if the given provider is enabled in settings.
func (a *PollingAgent) isProviderEnabled(p core.Provider) bool {
	id := p.ID()
	switch id {
	case "antigravity":
		return true
	case "claude":
		return a.store.GetConfigBool("claude_bridge")
	case "codex":
		return a.store.GetConfigBool("codex_capture")
	case "cursor":
		return a.store.GetConfigBool("cursor_capture")
	case "copilot":
		return a.store.GetConfigBool("copilot_capture")
	default:
		if strings.HasPrefix(id, "plugin_") {
			return a.store.GetConfigBool(id + "_enabled")
		}
		return false
	}
}

// getCredentials fetches configured credentials from settings, falling back to auto-discovery if needed.
func (a *PollingAgent) getCredentials(p core.Provider) (*core.Credentials, error) {
	creds := &core.Credentials{
		Extra: make(map[string]string),
	}
	for _, field := range p.ConfigSchema() {
		val := a.store.GetConfig(field.Key)
		if val == "" && field.Default != "" {
			val = field.Default
		}
		if val != "" {
			creds.Extra[field.Key] = val
		}
	}

	for _, m := range p.AuthMethods() {
		if len(m.ConfigKeys) > 0 {
			firstKey := m.ConfigKeys[0]
			val := creds.Extra[firstKey]
			if val != "" {
				switch m.Type {
				case core.AuthAPIKey:
					creds.APIKey = val
				case core.AuthOAuth, core.AuthCLIToken:
					creds.AccessToken = val
				case core.AuthCookie:
					creds.Cookie = val
				}
			}
		}
	}

	hasCreds := false
	if p.ID() == "antigravity" || p.ID() == "claude" || strings.HasPrefix(p.ID(), "plugin_") {
		hasCreds = true
	} else {
		if creds.APIKey != "" || creds.AccessToken != "" || creds.Cookie != "" {
			hasCreds = true
		}
	}

	if !hasCreds {
		disc, err := p.AutoDiscover()
		if err == nil && disc != nil {
			if disc.APIKey != "" {
				creds.APIKey = disc.APIKey
			}
			if disc.AccessToken != "" {
				creds.AccessToken = disc.AccessToken
			}
			if disc.Cookie != "" {
				creds.Cookie = disc.Cookie
			}
			for k, v := range disc.Extra {
				creds.Extra[k] = v
			}
			hasCreds = true
		}
	}

	if !hasCreds {
		return nil, errors.New("no credentials configured or discovered")
	}

	return creds, nil
}

// checkProviderNotifications checks model or usage quotas against configured notifier alert thresholds.
func (a *PollingAgent) checkProviderNotifications(p core.Provider, snap *core.Snapshot) {
	if a.notifier == nil {
		return
	}
	id := p.ID()
	switch id {
	case "antigravity":
		if models, ok := snap.Models.([]any); ok {
			for _, mAny := range models {
				if mq, ok := mAny.(client.ModelQuota); ok {
					a.notifier.CheckQuota(mq.ModelID, mq.RemainingPercent)
				}
			}
		} else if models, ok := snap.Models.([]client.ModelQuota); ok {
			for _, mq := range models {
				a.notifier.CheckQuota(mq.ModelID, mq.RemainingPercent)
			}
		}
	case "claude":
		if data, ok := snap.Data.(map[string]any); ok {
			if pct, ok := data["five_hour_pct"].(float64); ok {
				a.notifier.CheckClaudeQuota("five_hour", pct)
			}
			if pctPtr, ok := data["seven_day_pct"].(*float64); ok && pctPtr != nil {
				a.notifier.CheckClaudeQuota("seven_day", *pctPtr)
			} else if pct, ok := data["seven_day_pct"].(float64); ok {
				a.notifier.CheckClaudeQuota("seven_day", pct)
			}
		}
	case "codex":
		if data, ok := snap.Data.(map[string]any); ok {
			if pct, ok := data["five_hour_pct"].(float64); ok {
				a.notifier.CheckUsedQuota("codex_five_hour", "Codex 5-hour", pct)
			}
			if pctPtr, ok := data["seven_day_pct"].(*float64); ok && pctPtr != nil {
				a.notifier.CheckUsedQuota("codex_seven_day", "Codex 7-day", *pctPtr)
			} else if pct, ok := data["seven_day_pct"].(float64); ok {
				a.notifier.CheckUsedQuota("codex_seven_day", "Codex 7-day", pct)
			}
		}
	case "cursor":
		a.notifier.CheckUsedQuota("cursor_usage", "Cursor", snap.OverallPct)
	case "copilot":
		a.notifier.CheckUsedQuota("copilot_usage", "GitHub Copilot", snap.OverallPct)
	default:
		if strings.HasPrefix(id, "plugin_") {
			if snap.OverallPct > 0 {
				a.notifier.CheckUsedQuota(id, "Plugin "+p.Name(), snap.OverallPct)
			}
		}
	}
}

// reportProviderSession feeds metrics to the respective session managers for idle/active session detection.
func (a *PollingAgent) reportProviderSession(p core.Provider, snap *core.Snapshot) {
	id := p.ID()
	switch id {
	case "antigravity":
		if a.antigravitySM != nil {
			var vals []float64
			if models, ok := snap.Models.([]any); ok {
				for _, mAny := range models {
					if mq, ok := mAny.(client.ModelQuota); ok {
						vals = append(vals, mq.RemainingFraction)
					}
				}
			} else if models, ok := snap.Models.([]client.ModelQuota); ok {
				for _, mq := range models {
					vals = append(vals, mq.RemainingFraction)
				}
			}
			if len(vals) > 0 {
				a.antigravitySM.ReportPoll(vals)
			}
		}
	case "claude":
		if a.claudeSM != nil {
			var vals []float64
			if data, ok := snap.Data.(map[string]any); ok {
				if pct, ok := data["five_hour_pct"].(float64); ok {
					vals = append(vals, pct)
				}
				if pctPtr, ok := data["seven_day_pct"].(*float64); ok && pctPtr != nil {
					vals = append(vals, *pctPtr)
				} else if pct, ok := data["seven_day_pct"].(float64); ok {
					vals = append(vals, pct)
				}
			}
			if len(vals) > 0 {
				a.claudeSM.ReportPoll(vals)
			}
		}
	case "codex":
		if a.codexSM != nil {
			var vals []float64
			if data, ok := snap.Data.(map[string]any); ok {
				if pct, ok := data["five_hour_pct"].(float64); ok {
					vals = append(vals, pct)
				}
				if pctPtr, ok := data["seven_day_pct"].(*float64); ok && pctPtr != nil {
					vals = append(vals, *pctPtr)
				} else if pct, ok := data["seven_day_pct"].(float64); ok {
					vals = append(vals, pct)
				}
			}
			if len(vals) > 0 {
				a.codexSM.ReportPoll(vals)
			}
		}
	}
}

// autoLink creates a subscription record if one doesn't exist for this account.
func (a *PollingAgent) autoLink(snap *core.Snapshot, accountID int64) {
	autoLinkEnabled := a.store.GetConfigBool("auto_link_subs")
	if !autoLinkEnabled {
		return
	}

	existing, _ := a.store.FindSubscriptionByAccountID(accountID)
	if existing != nil {
		return
	}

	autoSub := &store.Subscription{
		Platform:      strings.Title(snap.Provider),
		Category:      "coding",
		Email:         snap.Email,
		PlanName:      snap.PlanTier,
		Status:        "active",
		CostCurrency:  "USD",
		BillingCycle:  "monthly",
		LimitPeriod:   "rolling_5h",
		Notes:         "Auto-created from auto-capture. Quota tracking.",
		URL:           "https://" + snap.Provider + ".com",
		StatusPageURL: "https://status." + snap.Provider + ".com",
		AutoTracked:   true,
		AccountID:     accountID,
	}
	if snap.Provider == "antigravity" {
		switch {
		case strings.Contains(strings.ToLower(snap.PlanTier), "pro+"),
			strings.Contains(strings.ToLower(snap.PlanTier), "ultimate"):
			autoSub.CostAmount = 60
		default:
			autoSub.CostAmount = 15
		}
	} else {
		autoSub.CostAmount = 20
	}

	if _, err := a.store.InsertSubscription(autoSub); err != nil {
		a.logger.Warn("Auto-link subscription failed", "error", err, "email", snap.Email)
	} else {
		a.store.LogInfo("server", "auto_link", snap.Email, map[string]interface{}{
			"platform": snap.Provider, "plan": snap.PlanTier,
		})
		a.logger.Info("Auto-linked subscription", "email", snap.Email, "plan", snap.PlanTier)
	}
}

// cleanupOldSnapshots enforces the retention_days config by deleting old data.
func (a *PollingAgent) cleanupOldSnapshots() {
	retentionDays := a.store.GetConfigInt("retention_days", 365)
	if retentionDays <= 0 {
		return // disabled
	}

	deleted, err := a.store.DeleteSnapshotsOlderThan(retentionDays)
	if err != nil {
		a.logger.Warn("Retention cleanup failed", "error", err)
		return
	}
	if deleted > 0 {
		a.logger.Info("Retention cleanup", "deleted", deleted, "retentionDays", retentionDays)
		a.store.LogInfo("server", "retention_cleanup", "", map[string]interface{}{
			"deleted": deleted, "retentionDays": retentionDays,
		})
	}

	// Also clean up expired/dismissed alerts
	a.store.CleanupExpiredAlerts()
}
