# Niyantra — Agent Briefing

## Project Identity
Niyantra is a local-first AI operations dashboard tracking 6 AI providers (Antigravity, Claude Code, Codex/ChatGPT, Cursor, GitHub Copilot, Plugins). It compiles to a single Go binary embedding a web dashboard, utilizing local SQLite storage. (Note: Gemini CLI provider is removed).

## Tech Stack & Commands
- **Backend:** Go 1.22+ (stdlib net/http, no framework)
- **Database:** SQLite via modernc.org/sqlite (pure Go, no CGo)
- **Frontend:** TypeScript (esbuild, IIFE bundle, vanilla CSS)
- **Key Commands:**
  - Build: `go build ./cmd/niyantra`
  - Test: `go test ./internal/... ./cmd/...` (Note: `go test ./...` includes scripts which build as main package and may fail)
  - Run/Dev: `go run ./cmd/niyantra serve`
  - Frontend Build: `node esbuild.config.mjs`

## Architecture Rules
1. **Explicit over implicit:** No magic, no hidden state or convention-dependent behavior.
2. **Files as context boundaries:** Each file must be understandable in isolation. Goal: no code file >300 lines, no test file >500 lines.
3. **Vertical slicing:** Code that changes together lives together.
4. **Contracts are architecture:** Use Go interfaces/types to enforce contracts rather than documentation/comments.
5. **Self-healing:** System detects and prevents its own corruption (e.g. foreign keys, registry validation).
6. **One source of truth:** Every fact is stated once; everything else is derived.
7. **Additive evolution:** New features ADD files, rather than modifying working files.
8. **Registry over conditionals:** Avoid switch/case chains that grow with every provider.
9. **Tests are invariants:** Write NEW tests, but never modify existing test assertions to make them pass.
10. **Observability:** Every operation logs intent, result, and timing.

## Forbidden Actions
- **NEVER** modify existing test assertions to make tests pass.
- **NEVER** add Go dependencies without documenting why and having an exit strategy.
- **NEVER** put business logic in HTTP handlers (must go in service layer).
- **NEVER** use `innerHTML` for complex rendering in frontend.
- **NEVER** commit directly to the main branch.
- **NEVER** read the full `knowledge_base.md` (900+ lines; use grep instead) or the full `roadmap.md` (700+ lines; read only the needed section).

## Key Files for Context
- `internal/store/store.go`: SQLite persistence & migrations.
- `internal/web/server.go`: HTTP server routing & handlers.
- `internal/web/src/main.ts`: Main frontend entrypoint.

## Git Workflow
- Create a feature branch: `feat/<name>` or `refactor/<name>`.
- Commit messages must follow the Conventional Commits format: `feat: ...`, `fix: ...`, `refactor: ...`, `test: ...`, `docs: ...`.
- Never push directly to `main` branch.
