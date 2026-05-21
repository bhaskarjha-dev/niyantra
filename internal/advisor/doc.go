// Package advisor provides account switching recommendations based on
// multi-factor scoring of quota status, burn rates, and reset timers.
//
// Key types:
//   - Recommendation: Swapping guidance containing action, best account, alternatives, and rationale
//   - AccountScore: Detailed scoring information for a specific account
//
// Dependencies:
//   - internal/client: Uses the Snapshot struct for input data
//   - internal/readiness: Computes readiness metrics for scored accounts
//   - internal/tracker: Retrieves usage summaries and intelligence
//
// Files:
//   - advisor.go: Stateless switching recommendation logic
package advisor
