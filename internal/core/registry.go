package core

import (
	"fmt"
	"sort"
	"sync"
)

// Registry is a thread-safe container for registered providers.
// Providers are registered explicitly via RegisterAll() in internal/providers/,
// which is called from main(). This avoids init() side effects (Law 1:
// Explicit Over Implicit).
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// global is the package-level default registry.
var global = &Registry{providers: make(map[string]Provider)}

// Register adds a provider to the global registry.
// Called from providers.RegisterAll(), NOT from init().
// Panics if a provider with the same ID is already registered
// (indicates a programming error, not a runtime condition).
func Register(p Provider) {
	global.mu.Lock()
	defer global.mu.Unlock()
	id := p.ID()
	if _, exists := global.providers[id]; exists {
		panic(fmt.Sprintf("core.Register: duplicate provider ID %q", id))
	}
	global.providers[id] = p
}

// Get returns the provider with the given ID, or nil if not found.
func Get(id string) Provider {
	global.mu.RLock()
	defer global.mu.RUnlock()
	return global.providers[id]
}

// MustGet returns the provider with the given ID.
// Panics if not found (use Get for optional lookups).
func MustGet(id string) Provider {
	p := Get(id)
	if p == nil {
		panic(fmt.Sprintf("core.MustGet: unknown provider %q", id))
	}
	return p
}

// All returns all registered providers, sorted by ID.
func All() []Provider {
	global.mu.RLock()
	defer global.mu.RUnlock()
	result := make([]Provider, 0, len(global.providers))
	for _, p := range global.providers {
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID() < result[j].ID()
	})
	return result
}

// IDs returns all registered provider IDs, sorted.
func IDs() []string {
	all := All()
	ids := make([]string, len(all))
	for i, p := range all {
		ids[i] = p.ID()
	}
	return ids
}

// Count returns the number of registered providers.
func Count() int {
	global.mu.RLock()
	defer global.mu.RUnlock()
	return len(global.providers)
}

// Reset clears all registered providers. Used only in tests.
func Reset() {
	global.mu.Lock()
	defer global.mu.Unlock()
	global.providers = make(map[string]Provider)
}
