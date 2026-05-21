// Package web implements the HTTP dashboard web server, rate limiters, and routing middleware.
//
// Key types:
//   - Server: Holds HTTP routes, connects tracker and notification components, and handles daemon lifecycles
//
// Dependencies:
//   - internal/agent: Starts and stops auto-capture polling jobs
//   - internal/claude: Interfaces with Claude Code statusline events
//   - internal/client: Triggers remote status pulls and captures credentials
//   - internal/mcpserver: Serves Streamable HTTP MCP sessions
//   - internal/notify: Feeds low-quota alerts into OS/WebPush/Webhook delivery channels
//   - internal/plugin: Controls discovered plugin states and triggers runs
//   - internal/store: Reads and updates the primary SQLite database
//   - internal/tracker: Seeds baselines and maps cycle updates
//
// Files:
//   - auth.go: Enforces rate-limiting middleware, basic authentication, and session cookie validation
//   - mcp.go: Handles MCP client status checks and snapshot pulls
//   - plugin.go: Handles REST routes for scanning and executing custom plugins
//   - server.go: Mounts REST endpoints, loads embeds, and listens on the configured host/port
package web
