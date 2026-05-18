# Testing Guide: Niyantra

> Updated: 2026-05-18 · Schema v20

## Automated Verification

Run these commands from the repository root:

```bash
# Backend unit and integration tests
go test ./... -count=1

# Data-race coverage (CI also runs this on Linux)
go test ./... -race -count=1

# Frontend typecheck and bundle generation
npm run check
npm run build
npm run css

# Documentation truth checks
go run ./scripts/checkdocs
```

## What The Suite Covers

### Backend and data integrity
- Store migrations, schema upgrades, retention cleanup, import/export, snapshot selection, secret indirection, and account deletion cascades.
- Composite account identity across providers, including same-email multi-provider cases.
- Codex ownership backfill through the `owner_account_id` migration path.
- SQLite-safe backup and restore flows through the CLI helpers.

### HTTP and UI contracts
- Remote-bind validation, auth requirements, rate limiting, repo-path restrictions, and masked config/export responses.
- Usage API identity fields so duplicate model rows stay attributable to account and provider.
- Quick Adjust targeting by `modelId` instead of label matching.

### Analytics correctness
- Git-to-Claude attribution keeps each usage event assigned at most once.
- Budget output is treated as recurring subscription headroom rather than observed burn forecasting.
- Insight generation emits truthful server-side types: `renewal_imminent`, `trial_expiring`, `unused_subscription`, `category_overlap`, and `budget_exceeded`.

### Notifications
- Guard behavior, reset handling, digest batching, webhook delivery, SMTP validation, WebPush crypto, and permanent WebPush invalidation classification.

### Frontend build safety
- TypeScript compile checks.
- Production bundle generation for `internal/web/static/app.js`.
- CSS bundle generation for `internal/web/static/style.css`.

### Docs and release gates
- `scripts/checkdocs` blocks stale marketing/security/API/testing claims and mojibake in the checked docs set.
- CI builds the Docker image, runs backend tests on Linux/macOS/Windows, runs `-race` on Linux, runs frontend build/check steps, and runs the docs checker.

## Focused Test Commands

```bash
# Store-heavy work
go test ./internal/store/... -count=1

# Web/API handlers
go test ./internal/web/... -count=1

# Notification delivery logic
go test ./internal/notify/... -count=1

# Git attribution logic
go test ./internal/gitcorr/... -count=1

# CLI backup/restore helpers
go test ./cmd/niyantra/... -count=1
```

## Manual Spot Checks

These flows are still worth checking before a release candidate:

### Secure-default startup
- `niyantra serve` binds to loopback by default.
- Non-loopback binds fail unless explicit remote exposure is allowed and auth is configured.
- HTTP MCP is not exposed unless explicitly enabled.

### Backup and restore
- Dashboard backup downloads a valid SQLite snapshot.
- CLI backup opens as a valid SQLite database.
- Restore rejects invalid files and reopens the restored database cleanly.

### Data honesty
- Overview labels recurring subscription totals as recurring, heuristic, or observed exactly as the backend describes them.
- Token analytics empty state does not claim provider coverage that does not exist.
- Git attribution UI is visibly labeled as heuristic.

### Secret handling
- Sensitive config values never come back in plaintext through `/api/config` or `/api/export/json`.
- Keychain-backed environments store secret references in SQLite rather than raw secret values.
- `--insecure-plaintext-secrets` is required before plaintext secret fallback is allowed.

### Notifications
- Invalid WebPush subscriptions are pruned after permanent push-service failures.
- Failed configured delivery channels do not falsely mark an alert as successfully sent.

## Current Limits

- Browser end-to-end tests are not yet comprehensive enough to replace manual release smoke checks.
- MCP semantic coverage is improving, but the highest-value validation still comes from backend/store contract tests.
- Observed spend forecasting is intentionally absent until a trustworthy spend ledger exists.
