// Package claude provides zero-dependency integration with Claude Code.
// It handles statusline bridge rate-limit monitoring and JSONL session log parsing.
//
// Key types:
//   - RateLimits: Rate limit window constraints decoded from statusline JSON
//   - TokenUsage: Individual turn token metrics containing input, output, and cache details
//   - UsageSummary: Aggregated daily historical token usage, costs, and cache hit metrics
//
// Dependencies:
//   - None: This package has zero external internal dependencies
//
// Files:
//   - bridge.go: Manages injection, cleanup, and validation of the statusline bridge hook
//   - deep.go: Discovers, parses, and aggregates token usage records from project JSONL logs
package claude
