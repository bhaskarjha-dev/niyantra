package notify

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Engine tracks notification state and prevents spam.
// Fires at most one notification per model per reset cycle.
// Supports quad-channel delivery: OS-native + SMTP email + Webhook + WebPush.
// Supports digest batching to reduce notification fatigue.
type Engine struct {
	mu        sync.Mutex
	enabled   bool
	threshold float64              // alert when remaining% drops below this (default 10)
	guard     map[string]time.Time // guard key -> time of last notification
	guardTTL  time.Duration        // how long to suppress re-notifications (default 6h)
	logger    *slog.Logger
	smtp      SMTPConfig
	webhook   WebhookConfig
	webpush   WebPushConfig
	digest    *DigestBatcher

	getSubscriptions   func() []WebPushSubscription
	deleteSubscription func(endpoint string)
	onNotify           func(model string, remainingPct float64)
}

// NewEngine creates a notification engine with default settings.
func NewEngine(logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	return &Engine{
		enabled:   false,
		threshold: 10,
		guard:     make(map[string]time.Time),
		guardTTL:  6 * time.Hour,
		logger:    logger,
	}
}

// Configure updates the engine's settings.
func (e *Engine) Configure(enabled bool, threshold float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = enabled
	if threshold > 0 {
		e.threshold = threshold
	}
}

// ConfigureSMTP updates the SMTP delivery settings.
func (e *Engine) ConfigureSMTP(cfg SMTPConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.smtp = cfg
	e.logger.Info("SMTP notification channel configured",
		"enabled", cfg.Enabled,
		"host", cfg.Host,
		"port", cfg.Port,
		"tls", cfg.TLSMode)
}

// SMTPEnabled returns whether SMTP delivery is active.
func (e *Engine) SMTPEnabled() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.smtp.IsConfigured()
}

// ConfigureWebhook updates the webhook delivery settings.
func (e *Engine) ConfigureWebhook(cfg WebhookConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.webhook = cfg
	e.logger.Info("Webhook notification channel configured",
		"enabled", cfg.Enabled,
		"type", cfg.Type,
		"url_set", cfg.URL != "")
}

// WebhookEnabled returns whether webhook delivery is active.
func (e *Engine) WebhookEnabled() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.webhook.IsConfigured()
}

// ConfigureWebPush updates the WebPush delivery settings.
func (e *Engine) ConfigureWebPush(cfg WebPushConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.webpush = cfg
	e.logger.Info("WebPush notification channel configured",
		"enabled", cfg.Enabled,
		"has_keys", cfg.PublicKey != "")
}

// WebPushEnabled returns whether WebPush delivery is active.
func (e *Engine) WebPushEnabled() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.webpush.IsConfigured()
}

// SetGetSubscriptions registers a callback to fetch stored WebPush subscriptions.
func (e *Engine) SetGetSubscriptions(fn func() []WebPushSubscription) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.getSubscriptions = fn
}

// SetDeleteSubscription registers a callback to prune permanently invalid
// WebPush subscriptions after 404/410 responses from push services.
func (e *Engine) SetDeleteSubscription(fn func(endpoint string)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.deleteSubscription = fn
}

// Enabled returns whether notifications are enabled.
func (e *Engine) Enabled() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.enabled
}

// ConfigureDigest sets up the digest batcher.
// enabled: whether to batch alerts; windowSec: batch window in seconds (0 = default 5 min).
func (e *Engine) ConfigureDigest(enabled bool, windowSec int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.digest == nil {
		e.digest = NewDigestBatcher(5*time.Minute, func(title, body string) {
			e.deliver(title, body)
		})
	}

	if windowSec > 0 {
		e.digest.SetWindow(time.Duration(windowSec) * time.Second)
	}
	e.digest.SetEnabled(enabled)

	e.logger.Info("Digest notification mode configured",
		"enabled", enabled,
		"window_sec", windowSec)
}

// DigestEnabled returns whether digest mode is active.
func (e *Engine) DigestEnabled() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.digest != nil && e.digest.IsEnabled()
}

// Threshold returns the current threshold.
func (e *Engine) Threshold() float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.threshold
}

// SetOnNotify registers a callback invoked after a notification is successfully sent.
func (e *Engine) SetOnNotify(fn func(model string, remainingPct float64)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onNotify = fn
}

// CheckQuota fires a notification if remaining% drops below threshold.
func (e *Engine) CheckQuota(model string, remainingPct float64) {
	e.checkQuotaAlert(model, model, remainingPct)
}

// CheckUsedQuota converts a used percentage into a remaining percentage and
// sends a provider-specific alert keyed by guardKey.
func (e *Engine) CheckUsedQuota(guardKey, label string, usedPct float64) {
	e.checkQuotaAlert(guardKey, label, 100.0-usedPct)
}

func (e *Engine) checkQuotaAlert(guardKey, label string, remainingPct float64) {
	e.mu.Lock()
	enabled := e.enabled
	threshold := e.threshold
	lastSent, exists := e.guard[guardKey]
	alreadySent := exists && time.Since(lastSent) < e.guardTTL
	digest := e.digest
	smtpCfg := e.smtp
	webhookCfg := e.webhook
	webpushCfg := e.webpush
	getSubs := e.getSubscriptions
	deleteSub := e.deleteSubscription
	e.mu.Unlock()

	if !enabled || alreadySent {
		return
	}
	if remainingPct > threshold {
		return
	}

	if digest != nil {
		batched := digest.Add(DigestAlert{
			Model:        label,
			RemainingPct: remainingPct,
			Timestamp:    time.Now(),
		})
		if batched {
			e.mu.Lock()
			e.guard[guardKey] = time.Now()
			cb := e.onNotify
			e.mu.Unlock()
			if cb != nil {
				cb(label, remainingPct)
			}
			return
		}
	}

	title := fmt.Sprintf("Alert: %s quota low", label)
	body := fmt.Sprintf("%.1f%% remaining - consider switching models", remainingPct)

	e.logger.Info("Sending quota alert notification",
		"model", label,
		"guard_key", guardKey,
		"remaining_pct", remainingPct,
		"threshold", threshold)

	delivered := false
	if err := Send(title, body); err != nil {
		e.logger.Error("Failed to send OS notification", "error", err, "model", label)
	} else {
		delivered = true
	}

	configuredAsync := 0
	if smtpCfg.IsConfigured() {
		configuredAsync++
	}
	if webhookCfg.IsConfigured() {
		configuredAsync++
	}

	var subs []WebPushSubscription
	if webpushCfg.IsConfigured() && getSubs != nil {
		subs = getSubs()
		if len(subs) > 0 {
			configuredAsync++
		}
	}

	if smtpCfg.IsConfigured() {
		subject := fmt.Sprintf("Niyantra Alert: %s quota low (%.1f%%)", label, remainingPct)
		htmlBody := FormatQuotaAlertHTML(label, remainingPct, threshold)
		if err := SendEmail(&smtpCfg, subject, htmlBody); err != nil {
			e.logger.Error("Failed to send SMTP notification", "error", err, "model", label)
		} else {
			delivered = true
			e.logger.Info("SMTP quota alert sent", "model", label, "to", smtpCfg.To)
		}
	}

	if webhookCfg.IsConfigured() {
		whTitle := fmt.Sprintf("Alert: %s quota low", label)
		whMsg := fmt.Sprintf("%.1f%% remaining (threshold: %.0f%%) - consider switching models", remainingPct, threshold)
		if err := SendWebhook(&webhookCfg, whTitle, whMsg, remainingPct); err != nil {
			e.logger.Error("Failed to send webhook notification", "error", err, "model", label)
		} else {
			delivered = true
			e.logger.Info("Webhook quota alert sent", "model", label, "type", webhookCfg.Type)
		}
	}

	if len(subs) > 0 {
		payload := FormatQuotaAlertPush(label, remainingPct, threshold)
		for _, sub := range subs {
			if err := SendWebPush(&webpushCfg, &sub, payload); err != nil {
				e.logger.Error("Failed to send WebPush notification", "error", err, "model", label)
				if deleteSub != nil && IsPermanentWebPushError(err) {
					deleteSub(sub.Endpoint)
					e.logger.Info("Pruned invalid WebPush subscription", "model", label)
				}
				continue
			}
			delivered = true
			epSnippet := sub.Endpoint
			if len(epSnippet) > 50 {
				epSnippet = epSnippet[:50]
			}
			e.logger.Info("WebPush quota alert sent", "model", label, "endpoint", epSnippet)
		}
	}

	if configuredAsync > 0 && !delivered {
		e.logger.Warn("No configured notification channel delivered; leaving alert guard unset", "model", label)
		return
	}

	e.mu.Lock()
	e.guard[guardKey] = time.Now()
	cb := e.onNotify
	e.mu.Unlock()

	if cb != nil {
		cb(label, remainingPct)
	}
}

// CheckClaudeQuota fires a notification for Claude Code rate limits.
// usedPct is the used percentage (0-100).
func (e *Engine) CheckClaudeQuota(window string, usedPct float64) {
	e.CheckUsedQuota("claude_"+window, claudeQuotaLabel(window), usedPct)
}

func claudeQuotaLabel(window string) string {
	switch window {
	case "five_hour":
		return "Claude Code 5-hour"
	case "seven_day":
		return "Claude Code 7-day"
	default:
		return "Claude Code"
	}
}

// OnReset clears the guard for a model (cycle detected -> can notify again).
func (e *Engine) OnReset(model string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.guard, model)
	e.logger.Debug("Notification guard cleared for model", "model", model)
}

// ResetGuard clears the notification suppression for a specific guard key.
func (e *Engine) ResetGuard(guardKey string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.guard, guardKey)
	e.logger.Debug("Notification guard manually reset", "key", guardKey)
}

// ResetAllGuards clears all notification suppression timers.
func (e *Engine) ResetAllGuards() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.guard = make(map[string]time.Time)
	e.logger.Info("All notification guards reset")
}

// SendTest sends a test notification to verify the platform works.
func (e *Engine) SendTest() error {
	return Send(
		"Niyantra - Test Notification",
		fmt.Sprintf("Notifications are working! Threshold: %.0f%%. Time: %s",
			e.Threshold(), time.Now().Format("15:04:05")),
	)
}

// SendTestEmail sends a test email to verify SMTP configuration.
func (e *Engine) SendTestEmail() error {
	e.mu.Lock()
	cfg := e.smtp
	e.mu.Unlock()

	if !cfg.IsConfigured() {
		return fmt.Errorf("SMTP is not configured")
	}

	return SendEmail(&cfg, "Niyantra - SMTP Test", FormatTestEmailHTML())
}

// SendTestWebhookFromEngine sends a test webhook to verify configuration.
func (e *Engine) SendTestWebhookFromEngine() error {
	e.mu.Lock()
	cfg := e.webhook
	e.mu.Unlock()

	if !cfg.IsConfigured() {
		return fmt.Errorf("webhook is not configured")
	}

	return SendTestWebhook(&cfg)
}

// SendTestWebPushFromEngine sends a test push to all subscriptions.
func (e *Engine) SendTestWebPushFromEngine() error {
	e.mu.Lock()
	cfg := e.webpush
	getSubs := e.getSubscriptions
	e.mu.Unlock()

	if !cfg.IsConfigured() {
		return fmt.Errorf("WebPush is not configured")
	}

	var subs []WebPushSubscription
	if getSubs != nil {
		subs = getSubs()
	}

	return SendTestWebPush(&cfg, subs)
}

// deliver sends a formatted notification to all configured channels.
// Used by the digest batcher to flush batched alerts.
func (e *Engine) deliver(title, body string) {
	e.logger.Info("Delivering notification", "title", title)

	if err := Send(title, body); err != nil {
		e.logger.Error("Failed to send OS notification", "error", err)
	}

	e.mu.Lock()
	smtpCfg := e.smtp
	webhookCfg := e.webhook
	webpushCfg := e.webpush
	getSubs := e.getSubscriptions
	deleteSub := e.deleteSubscription
	e.mu.Unlock()

	if smtpCfg.IsConfigured() {
		htmlBody := "<h3>" + title + "</h3><p>" + body + "</p>"
		if err := SendEmail(&smtpCfg, "Niyantra: "+title, htmlBody); err != nil {
			e.logger.Error("Failed to send digest SMTP", "error", err)
		}
	}

	if webhookCfg.IsConfigured() {
		if err := SendWebhook(&webhookCfg, title, body, 0); err != nil {
			e.logger.Error("Failed to send digest webhook", "error", err)
		}
	}

	if webpushCfg.IsConfigured() && getSubs != nil {
		subs := getSubs()
		payload := FormatDigestPush(title, body)
		for _, sub := range subs {
			if err := SendWebPush(&webpushCfg, &sub, payload); err != nil {
				e.logger.Error("Failed to send digest WebPush", "error", err)
				if deleteSub != nil && IsPermanentWebPushError(err) {
					deleteSub(sub.Endpoint)
				}
			}
		}
	}
}
