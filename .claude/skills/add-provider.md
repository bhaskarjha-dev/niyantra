# How to Add a New Provider

## Current Process (Pre-Overhaul)
1. Create internal/<provider>/<provider>.go — API client
2. Add snapshot table — migration in store.go
3. Add store methods — internal/store/<provider>_snapshots.go
4. Add HTTP handlers — internal/web/handlers_<provider>.go
5. Register routes — internal/web/server.go
6. Add agent polling — internal/agent/<provider>.go
7. Add MCP tool — internal/mcpserver/tools_quota.go
8. Add frontend rendering — internal/web/src/quotas/render.ts
9. Add settings UI — internal/web/src/settings/settings.ts
10. Add CSS — internal/web/src/styles/<provider>.css
11. Seed config keys — migration in store.go
12. Add tests

## Post-Overhaul Process (after M3)
1. Create internal/providers/<provider>.go — implements core.Provider
2. Add test — internal/providers/<provider>_test.go
Done. Everything else is generic.
