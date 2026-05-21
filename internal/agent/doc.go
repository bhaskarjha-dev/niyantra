// Package agent provides background polling for auto-capture.
//
// Key types:
//   - PollingAgent: Background manager orchestration loops to query provider metrics
//
// Dependencies:
//   - internal/client: Connects to local client instance to fetch quotas
//   - internal/codex: Communicates with Codex/ChatGPT APIs
//   - internal/notify: Triggers notifications based on quota changes and alerts
//   - internal/plugin: Integrates external monitoring plugins
//   - internal/store: Reads configuration settings and persists snapshots
//   - internal/tracker: Tracks active user sessions and burn rate summaries
//
// Files:
//   - agent.go: Core polling orchestrator loop and retention manager
//   - antigravity.go: Background polling logic for Antigravity provider
//   - claude.go: Background polling logic for Claude Code provider
//   - codex.go: Background polling logic for Codex/ChatGPT provider
//   - copilot.go: Background polling logic for Copilot provider
//   - cursor.go: Background polling logic for Cursor provider
//   - manager.go: Provider manager utility logic
//   - plugins.go: Background polling logic for external plugins
package agent
