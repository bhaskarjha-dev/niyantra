// Package providers contains concrete implementations of the core.Provider
// interface. Each file implements one provider.
//
// Adding a new provider:
//   1. Create <provider>.go in this package
//   2. Implement core.Provider interface
//   3. Add core.Register(&YourProvider{}) to RegisterAll()
//   4. Done — agent, API, MCP, and frontend will pick it up automatically.
//
// Existing provider files (one per provider):
//   - copilot.go: GitHub Copilot (PAT → REST API)
//   - codex.go: Codex/ChatGPT (OAuth → REST API)
//   - claude.go: Claude Code (JSONL file parse + status bridge)
//   - cursor.go: Cursor (session cookie → REST API)
//   - antigravity.go: Antigravity (local process RPC)
//   - plugin.go: Generic plugin (subprocess exec)
//
// Note: Gemini CLI was removed (Google sunsetting June 2026).
package providers
