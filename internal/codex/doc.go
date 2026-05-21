// Package codex provides the Codex/ChatGPT OAuth API client.
//
// Key types:
//   - Credentials: Holds token and user details loaded from the Codex configuration
//   - Client: Connects to the ChatGPT backend usage API
//   - UsageResponse: Normalized rate limit and quota information
//
// Dependencies:
//   - None: This package uses only standard library packages
//
// Files:
//   - codex.go: Implements credentials discovery, token refreshing, and usage querying
package codex
