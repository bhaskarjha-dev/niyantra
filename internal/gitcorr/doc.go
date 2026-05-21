// Package gitcorr correlates git commit activity with AI token consumption.
//
// Key types:
//   - CommitCost: Represents a single git commit with correlated token counts and estimated costs
//   - BranchCost: Aggregated commits, tokens, and costs grouped by branch name
//   - Summary: The top-level response containing branches, commits, totals, and metadata
//
// Dependencies:
//   - internal/claude: Discovers and parses Claude Code session logs to fetch token metrics
//
// Files:
//   - gitcorr.go: Implements log parsing, branch extraction, and time-based correlation algorithms
package gitcorr
