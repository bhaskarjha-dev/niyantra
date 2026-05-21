// Package core defines the universal provider interface and shared types
// for the Niyantra AI operations dashboard.
//
// Key types:
//   - Provider: interface that all AI tool providers implement
//   - Registry: thread-safe provider registration and lookup
//   - Snapshot: universal data structure from any provider fetch
//   - Credentials: authentication credentials for a provider
//   - ConfigField: declarative configuration schema for a provider
//
// This package has ZERO dependencies on other internal packages.
// It is the leaf of the dependency tree — everything depends on core,
// core depends on nothing.
package core
