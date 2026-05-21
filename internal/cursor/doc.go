// Package cursor provides Cursor IDE credential detection and usage API polling.
//
// Key types:
//   - Credentials: Holds Cursor user ID, access token, and Stripe membership details
//   - Client: Connects to legacy and modern period usage endpoints on cursor.sh/com
//   - Snapshot: Consolidated usage quotas and subscription plan tiers
//
// Dependencies:
//   - None: This package uses only standard library packages
//
// Files:
//   - cursor.go: Extracts credentials from SQLite global state DB and queries Cursor APIs
package cursor
