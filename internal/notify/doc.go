// Package notify provides a quad-channel notification engine (OS-native, SMTP email, webhook, and WebPush).
//
// Key types:
//   - Engine: Tracks alert state, enforces anti-spam guards, and coordinates multi-channel delivery
//   - DigestBatcher: Batches multiple alerts within a sliding window to prevent notification fatigue
//
// Dependencies:
//   - None: This package uses only standard library packages
//
// Files:
//   - digest.go: Handles aggregation of alerts into batched notifications
//   - engine.go: Core alerting lifecycle, rate limit checks, and delivery dispatcher
//   - notify.go: Dispatcher for desktop notifications across operating systems
//   - notify_other.go: Fallback notifications for unsupported OS targets
//   - notify_windows.go: Implementation of Windows-native toast alerts via PowerShell
//   - smtp.go: Integrates SMTP mail client sending with HTML formatting
//   - webhook.go: Sends custom JSON payloads to third-party endpoints (e.g. Discord, Slack)
//   - webpush.go: Delivers encrypted push payloads to browser subscriptions
package notify
