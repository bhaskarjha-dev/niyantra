package providers

import (
	"github.com/bhaskarjha-com/niyantra/internal/core"
	"github.com/bhaskarjha-com/niyantra/internal/plugin"
)

// RegisterAll registers all active providers with the core registry.
// This is called explicitly from main().
func RegisterAll() {
	core.Register(&Copilot{})
	core.Register(&Codex{})
	core.Register(&Claude{})
	core.Register(&Cursor{})
	core.Register(&Antigravity{})
	core.Register(&Plugin{}) // register placeholder

	// Dynamically discover and register plugins as providers
	plugins, _ := plugin.Discover("")
	for _, pl := range plugins {
		core.Register(&Plugin{pl: pl})
	}
}

