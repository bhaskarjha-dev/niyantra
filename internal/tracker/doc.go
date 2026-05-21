// Package tracker manages reset cycle detection and usage calculation for Antigravity models.
//
// Key types:
//   - Tracker: Orchestrates model reset detection (time shift, remaining fraction increase, time-based check) and active cycle data updates
//
// Dependencies:
//   - internal/client: Uses model quota structures and snapshots
//   - internal/store: Reads and updates cycle metadata in SQLite database
//
// Files:
//   - tracker.go: Implements base load, reset detection logic, and cycle update operations
package tracker
