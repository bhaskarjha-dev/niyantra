// Package copilot provides GitHub Copilot credential detection and usage API polling.
//
// Key types:
//   - Client: Makes authenticated HTTPS calls to the GitHub Copilot API
//   - UsageResponse: Holds raw JSON data returned by the copilot_internal endpoint
//   - Snapshot: Parsed, normalized Copilot quota utilization information
//
// Dependencies:
//   - None: This package uses only standard library packages
//
// Files:
//   - copilot.go: Implements GitHub PAT discovery, CLI fallback, and usage API queries
package copilot
