// Package plugin provides an extensible plugin system for custom data sources.
//
// Key types:
//   - Manifest: Represents a parsed plugin.json file containing metadata, entryPoint, and config specs
//   - Plugin: Represents a discovered and validated plugin ready to execute
//   - CaptureResult: Holds stdout response from a plugin process
//
// Dependencies:
//   - None: This package uses only standard library packages
//
// Files:
//   - plugin.go: Implements manifest validation, directory canonicalization, and plugin discovery
//   - runner.go: Handles subprocess execution, JSON requests/responses, timeouts, and logging
package plugin
