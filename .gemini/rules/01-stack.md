Backend: Go 1.22+ (stdlib net/http, no framework)
Database: SQLite via modernc.org/sqlite (pure Go, no CGo)
Frontend: TypeScript (40 modules, esbuild, IIFE bundle)
CSS: Vanilla (22 style files)
Build: go build ./cmd/niyantra
Test: go test ./...
Dev: go run ./cmd/niyantra serve
Frontend build: node esbuild.config.mjs
Note: Gemini CLI removed (June 2026). 6 providers active.
