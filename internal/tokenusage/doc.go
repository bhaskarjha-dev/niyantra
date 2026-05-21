// Package tokenusage aggregates token usage analytics across monitored AI provider channels.
//
// Key types:
//   - Summary: Consolidated report of tokens, model metrics, daily breakdowns, and KPI stats
//   - Totals: Total sums of input, output, cache, and costs
//   - ModelBreakdown: Model-level usage percentages and totals
//   - DailyBreakdown: Day-by-day token counts and spend
//   - KPIs: High-level analytics metrics (days active, average tokens/costs per day, hit rates)
//
// Dependencies:
//   - internal/claude: Retrieves token counts from local Claude Code JSONL logs
//   - internal/store: Reads token_usage table rows for other providers
//
// Files:
//   - tokenusage.go: Implements aggregation, merging of summaries, and KPI calculations
package tokenusage
