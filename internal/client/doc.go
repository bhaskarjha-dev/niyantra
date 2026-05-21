// Package client provides the Antigravity language server API client.
//
// Key types:
//   - Client: Orchestrates auto-detection of processes and Connect RPC endpoint queries
//   - ConnInfo: Contains diagnostic information for established client connections
//
// Dependencies:
//   - None: This package uses only standard library packages
//
// Files:
//   - client.go: Core detection caching and quota-fetching orchestrations
//   - detect_unix.go: UNIX-based implementation for discovering process IDs
//   - detect_windows.go: Windows-based implementation for discovering process IDs
//   - helpers.go: IDE process signature parsing and path lookups
//   - probe.go: TCP port probing and CSRF token verification
//   - types.go: Connect RPC JSON payload mappings and response schemas
package client
