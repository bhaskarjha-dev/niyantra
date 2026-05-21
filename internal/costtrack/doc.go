// Package costtrack provides estimated cost calculations from quota fraction
// changes and configurable model pricing.
//
// Key types:
//   - GroupCeiling: Assumed token capacity details for a quota group cycle
//   - ModelPricing: Model pricing values mapped in USD per 1M tokens
//   - GroupCostEstimate: Detailed cost and burn-rate estimates for a single group
//   - AccountCostEstimate: Aggregated costs across all groups for a specific account
//
// Dependencies:
//   - None: This package uses only standard library packages
//
// Files:
//   - costtrack.go: Computes blended prices and calculates cost estimates per account/group
package costtrack
