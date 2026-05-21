// Package mcpserver implements a Model Context Protocol (MCP) server
// that exposes Niyantra's quota intelligence to AI coding agents.
//
// Key types:
//   - MCPServer: Wrapper that initializes the MCP server, registers tools, and manages stdio/HTTP transport execution
//
// Dependencies:
//   - internal/store: Reads quota configurations, subscription metrics, and active settings
//   - internal/tracker: Retrieves real-time token tracking data and usage logs
//
// Files:
//   - mcpserver.go: Initializes the server, configures the transports, and defines tool routes
//   - tools_analytics.go: Handles token usage statistics and git commit cost correlation requests
//   - tools_forecast.go: Handles time-to-exhaustion (TTX) prediction tools
//   - tools_ops.go: Handles subscription analysis, account switching scoring, and Codex status tools
//   - tools_quota.go: Handles live status checking and model availability checks
package mcpserver
