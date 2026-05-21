// Package readiness provides account and group readiness calculations from captured quota snapshots.
//
// Key types:
//   - AccountReadiness: Holds active status, email, staleness, credits, and groups for a monitored account
//   - GroupReadiness: Combines metrics for a specific quota category (e.g., claude_gpt, gemini_unified)
//   - ModelDetail: Captures per-model status, remaining percentages, and estimated availability details
//
// Dependencies:
//   - internal/client: Uses Snapshot structures, reset inferences, and group classification tools
//
// Files:
//   - readiness.go: Calculates readiness scores, processes model list upgrades, and formats staleness labels
package readiness
