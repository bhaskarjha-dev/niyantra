package core

import "context"

// Provider is the universal interface for all AI tool data sources.
//
// Every AI tool tracked by Niyantra implements this interface.
// Adding a new provider = one file implementing Provider in internal/providers/.
//
// The interface is intentionally small. Providers only need to implement
// what they can do — the registry and generic pipeline handle the rest.
type Provider interface {
	// Identity — what is this provider?
	ID() string         // Lowercase slug: "deepseek", "openrouter", "claude"
	Name() string       // Display name: "DeepSeek", "OpenRouter", "Claude Code"
	Category() Category // Grouping for UI: CategoryCoding, CategoryChat, etc.

	// Capabilities — what can this provider report?
	Capabilities() Cap  // Bitmask: CapQuota | CapBalance | CapReset | ...

	// Authentication — how to get credentials.
	// Returns methods in priority order (try first method first).
	AuthMethods() []AuthMethod

	// AutoDiscover attempts to find credentials on the local machine.
	// It tries each AuthMethod.Discover in priority order, returning
	// the first successful credentials. Returns error if all fail.
	AutoDiscover() (*Credentials, error)

	// Fetch queries the provider and returns current status.
	// This is the core operation — called by the agent on each poll cycle.
	// ctx carries timeout/cancellation from the caller.
	// creds are pre-validated credentials for this provider.
	Fetch(ctx context.Context, creds *Credentials) (*Snapshot, error)

	// ConfigSchema returns the configuration fields this provider needs.
	// Used to auto-generate the Settings UI and validate config values.
	ConfigSchema() []ConfigField

	// Presentation — how to display this provider in the UI.
	Color() string // CSS color for provider accent (e.g., "#E86C3A")
	Icon() string  // SVG string or emoji for the provider icon
}
