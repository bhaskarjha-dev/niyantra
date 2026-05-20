# Testing Guide: Niyantra

> Updated: 2026-05-19. Current schema target: v21.

This guide defines the test gates for code changes and the manual checks required before a release candidate. It intentionally separates automated coverage from browser/operator checks because some trust boundaries depend on real browser storage, downloads, file selection, and local provider state.

## Automated Verification

Run from the repository root:

```bash
# Backend unit and integration tests
go test -count=1 ./...

# Data-race coverage. Requires CGO and a C compiler.
go test -race -count=1 ./...

# Static checks
go vet ./...

# Frontend typecheck and development bundles
npm run check
npm run build
npm run css

# Production bundles
npm run build:prod
npm run css:prod

# Restore tracked development bundles after production bundle checks
npm run build
npm run css

# Dependency and documentation gates
npm audit --audit-level=high
go run ./scripts/checkdocs
```

On Windows, use a writable Go cache if the default cache is locked down:

```powershell
$env:GOCACHE="$PWD\.gocache"
$env:GOTELEMETRY="off"
go test -count=1 ./...
go vet ./...
```

`go test -race` requires `CGO_ENABLED=1` and a C compiler such as GCC. If the local machine cannot run the race suite, CI must run it on Linux before merge.

## CI Gates

The GitHub Actions workflow must pass:

- Backend build, vet, and tests on Linux, macOS, and Windows.
- Linux race tests.
- Security/integrity-focused package tests for CLI, web, store, plugin, advisor, and cost tracking.
- Frontend typecheck, development bundle build, generated asset drift check, production bundle build, and restored development bundle drift check.
- `npm audit --audit-level=high`.
- Documentation sanity checks via `scripts/checkdocs`.
- Docker image build.
- `govulncheck ./...`.
- `golangci-lint`.

Generated frontend assets are tracked. If `npm run build` or `npm run css` changes `internal/web/static/app.js` or `internal/web/static/style.css`, commit those generated files with the source change.

## Focused Test Commands

```bash
# Auth, middleware, API handlers, backup/export/import, browser-facing contracts
go test -count=1 ./internal/web

# Schema, migrations, FK behavior, import rollback, backup redaction, store contracts
go test -count=1 ./internal/store

# CLI backup/restore/token helpers
go test -count=1 ./cmd/niyantra

# Plugin discovery, manifest validation, symlink/path checks
go test -count=1 ./internal/plugin

# Advisor semantics
go test -count=1 ./internal/advisor

# Readiness and reset inference truthfulness
go test -count=1 ./internal/readiness

# Cost math and unavailable-price behavior
go test -count=1 ./internal/costtrack

# Notifications
go test -count=1 ./internal/notify

# Git attribution
go test -count=1 ./internal/gitcorr
```

## Coverage Expectations By Area

### Security

Automated tests must cover:

- Unauthenticated `/api/*` and HTTP `/mcp` requests return `401`.
- Bad bearer token returns `401`; valid bearer token reaches the handler.
- `/healthz` and static UI assets remain unauthenticated.
- Same-origin browser requests may use `Authorization`; cross-origin API/MCP requests are rejected.
- Mutating routes are rate limited: config, subscriptions, backup create, notification tests, web push, alerts, provider snaps, pricing, account/snapshot mutation, plugin config, and plugin run compatibility route.
- Request body limits are enforced before JSON parsing: default API JSON, plugin config, and import.
- Sensitive config values are masked in `/api/config`, `/api/export/json`, logs, and error responses.
- CLI and dashboard backups redact sensitive config values from the copied SQLite file.
- `POST /api/plugins/{id}/run` returns `410 Gone` and never spawns a process.
- Plugin discovery rejects oversized manifests, manifest symlinks, and entry points resolving outside the plugin directory.

### Data Integrity

Automated tests must cover:

- Fresh database creation reaches schema v21.
- Upgrade fixtures from older schema versions converge to the same v21 constraints.
- `activity_log.snapshot_id` uses `ON DELETE SET NULL`.
- Subscription/account and provider/account links allow `NULL`, not fake `0` references.
- Provider snapshot account links are FK-backed where applicable.
- Plugin snapshots are tied to `data_sources` through FK-backed `data_source_id`.
- Invalid legacy `0` and dangling IDs are repaired during migration.
- JSON import validates row shape and timestamps and rolls back transactionally on row errors.
- Backup/export/import round trips exclude secrets by default.
- Restore rejects invalid SQLite files and preserves the previous database on failure.

### Correctness And Product Truth

Automated tests must cover:

- Reset-time elapsed now produces optimistic 100% estimates (with graduated confidence) matching provider behavior, rather than keeping stale 0% data.
- Estimated or unknown reset state is represented with `isEstimated`, `basis`, `confidence`, and `unavailableReason`.
- Missing pricing produces unavailable cost output, not false zero-dollar spend.
- Cache/input/output token prices are applied to the correct token classes.
- Advisor `mode: "ranking"` never tells the user to switch from an unknown current account.
- Advisor with `currentAccountId` can return `stay`, `switch`, or `wait`.
- Timezone and month-boundary calculations for renewals and budget periods.
- Provider partial failures and stale snapshots surface as unavailable/stale/error states rather than fresh data.

### Frontend

Automated checks currently include TypeScript compilation and bundle generation. Add browser automation when changing UI-critical flows. Until then, run the manual release checklist below.

Frontend test cases should cover:

- Token bootstrap from `?token=...`, URL stripping, and reload from `sessionStorage`.
- Unauthorized API session displays a visible auth error.
- `apiFetch()` attaches bearer auth, applies timeout, and preserves external fetch behavior.
- Plugin manual run control is absent.
- Loading, empty, stale, partial, provider-error, and unavailable states render without overlap.
- Data-quality labels appear next to estimated or unavailable values.
- Keyboard navigation through tabs, command palette, forms, dialogs, and settings.
- Mobile width around 375 px.
- Dark/light contrast and focus visibility.

## Manual Release Checklist

Use a disposable database and a fresh browser profile or private window.

### Manual Test Setup And Evidence

Before starting manual verification, record:

- Commit SHA, build command, operating system, browser name/version, and database path.
- Whether the binary was run from `go run`, a locally built binary, Docker, or a release artifact.
- The exact `serve` command and whether `--mcp-http`, `--enable-plugins`, `--allow-remote`, `--auth`, or `--behind-https-proxy` were used.
- Screenshots or short screen recordings for every failed UI case.
- Network captures for auth, export, import, restore, plugin, and MCP checks with secret values redacted.

Use at least three disposable databases:

- Fresh empty database: validates first-run, empty states, and token creation.
- Demo database: validates normal dashboard rendering without requiring provider credentials.
- Imported/backup database: validates migration, import, restore, and data-integrity behavior.

Release-blocking manual failures:

- Any secret value appears in an API response, backup, export, console log, or visible error.
- Any `/api/*` or HTTP `/mcp` request works without a bearer token.
- Any manual HTTP plugin route spawns a process.
- Invalid import or restore mutates existing data.
- A stale, estimated, or unavailable number is rendered as fresh observed truth.
- Mobile layout hides destructive controls, auth errors, import errors, or provider failures.
- Keyboard-only navigation cannot reach core settings, dialogs, or destructive confirmation buttons.

### Startup And Auth

- Start `niyantra serve --db <disposable.db>`.
- Open only the printed tokenized dashboard URL.
- Confirm the URL token is removed after load.
- Refresh the page and confirm it still works from session storage.
- Visit `/healthz` without a token and confirm it returns only minimal liveness data.
- Visit `/api/status` without a token and confirm `401`.
- Call `/api/status` with `Authorization: Bearer <dashboard_api_token>` and confirm `200`.
- Rotate the token with `niyantra token rotate`; confirm the old browser/API session fails and the new token works.

### Dashboard Smoke

- Quotas tab loads with no console errors.
- Empty state is useful on a fresh database.
- Snap button failure states are visible when providers are not configured.
- Search, provider filter, status filter, and tag filter do not break layout.
- Expanding account rows keeps buttons reachable and labels readable.
- History chart shows a no-data state instead of a blank canvas.
- Activity heatmap renders an empty state or low-activity grid honestly.

### Advisor And Data Truth

- With no current account selected/supplied, advisor shows rankings only.
- With an explicit current account, advisor can show stay/switch/wait guidance.
- Estimated quota/cost values have visible labels.
- Missing pricing does not display `$0.00` as if observed.
- Stale snapshots show stale timing, not fresh status.
- Provider errors are visible and do not silently reuse old data as current.

### Plugins

- Start with plugins disabled and confirm plugin API routes return disabled-state errors.
- Start with `--enable-plugins` on a disposable DB.
- Confirm Settings has no manual "run/test plugin" action.
- Directly `POST /api/plugins/{id}/run` with a valid bearer token and confirm `410 Gone`.
- Confirm no plugin process is spawned from the HTTP route.
- Add a plugin with an oversized manifest and confirm discovery rejects it.
- Add a plugin with an entry-point symlink outside the plugin directory and confirm discovery rejects it.

### Backup, Export, Import, Restore

- Create a dashboard backup and open it with a disposable Niyantra instance.
- Confirm `dashboard_api_token`, provider secrets, notification secrets, and plugin secrets are blank/redacted in the backup.
- Create JSON export and confirm sensitive config values are masked as `configured`.
- Try invalid JSON import and confirm the UI shows validation errors and existing data remains unchanged.
- Import a valid JSON export into a disposable database and confirm deduplication.
- Restore an invalid database and confirm the existing database is preserved.
- Restore a valid disposable backup and confirm the app reopens cleanly.

### Provider Capture Matrix

Run each provider check with the provider disabled, misconfigured, and configured where credentials are available.

| Provider | Disabled state | Misconfigured state | Configured state |
|----------|----------------|---------------------|------------------|
| Antigravity | Snap explains missing LS or auth without crashing | Stale LS/cache data remains adjustable and labeled | Snap stores one local LS capture with provenance |
| Codex/ChatGPT | Status shows capture disabled | Missing/expired auth file gives visible error | 5h/7d/review windows render with timestamp and account identity |
| Claude Code | Bridge/deep tracking disabled states are clear | Missing settings/log path gives visible provider error | Statusline and token analytics render observed data only |
| Cursor | Capture disabled state is clear | Missing session token gives visible setup error | Request/credit usage renders with observed timestamp |
| GitHub Copilot | Capture disabled state is clear | Missing PAT gives visible setup error | Premium/chat usage renders with observed timestamp |
| Plugins | Disabled API returns disabled-state errors | Invalid manifest/symlink/oversize is rejected | Enabled polling stores snapshots; HTTP run still returns `410 Gone` |

For every configured provider, confirm the activity log records the capture source, method, account identity when available, and provider-specific errors without storing secrets.

### Subscription And Budget

- Add, edit, pause, cancel, and delete subscriptions across monthly, annual, lifetime, and pay-as-you-go cycles.
- Validate renewal dates around month boundaries, leap days, and year rollover.
- Confirm trial end warnings distinguish trial conversion from normal renewal.
- Confirm recurring budget headroom excludes unobserved pay-as-you-go usage and does not label estimates as spend.
- Export CSV and verify currency, cycle, renewal, status, notes, and URL fields round-trip correctly.
- Link and unlink an account-backed subscription, then delete the account and confirm v21 nullable relationships behave as documented.

### MCP And API Contract

- Start without `--mcp-http` and confirm `/mcp` is unavailable.
- Start with `--mcp-http` on loopback and confirm `/mcp` requires bearer auth.
- For non-loopback bind, confirm startup refuses HTTP MCP unless `--allow-remote`, `--auth`, and `--behind-https-proxy` are all present.
- Call `tools/list` over stdio and HTTP and confirm both expose the same 13 tools.
- Call `switch_recommendation` without current-account context and confirm it ranks only.
- Call `switch_recommendation` with explicit current-account context and confirm switch/stay/wait semantics are possible.
- Confirm MCP responses contain no dashboard token, provider credentials, notification secrets, or plugin secret config values.

### Data Quality And Product Truth

- For each card, table, chart, badge, and report number, identify whether the UI says observed, estimated, stale, unavailable, or provider-error.
- Confirm missing model pricing produces unavailable cost output, not `$0.00`.
- Confirm quota reset elapsed state produces a 100% estimate but is visibly labeled as "Reset Est." with appropriate confidence, not passed off as observed truth.
- Confirm stale snapshots still show their original `observedAt`/captured timestamp.
- Confirm Git commit costs show low-confidence heuristic attribution and are not described as accounting-grade spend.
- Confirm provider health cards do not collapse unrelated provider metrics into one false normalized score.

### Responsive And Accessibility

- Test at 375 px width: no clipped controls, overlapping text, broken tables, or unreachable buttons.
- Test at 768 px, 1024 px, and a wide desktop viewport; tables, charts, split buttons, and modals must remain reachable.
- Test keyboard-only navigation through tabs, settings forms, modals, and command palette.
- Confirm focus states are visible.
- Confirm destructive confirmations are reachable and cancellable by keyboard.
- Confirm loading, empty, unauthorized, provider-error, stale, and unavailable states are announced or visibly distinguishable.
- Test dark and light themes if supported.
- Confirm chart, button, and badge text passes visual contrast checks against the active theme.

### Operations

- Run with `--debug` and confirm logs are useful without leaking secrets.
- Confirm non-loopback bind is refused unless `--allow-remote`, `--auth`, and `--behind-https-proxy` are provided.
- Confirm HTTP MCP is unavailable unless `--mcp-http` is provided.
- Confirm HTTP MCP requires bearer auth.
- Leave the app idle, refresh, and verify stale/refresh states are honest.
- Rotate the token while the browser is open and confirm the next API call fails visibly.
- Run `niyantra backup` and `niyantra restore` from the CLI and compare behavior with dashboard backup/restore.
- Start Docker locally, read the printed tokenized URL from logs, and confirm the remote-bind safeguards are configured.

### Manual Performance And Scale

Use generated or imported disposable data, not a personal production database.

- 100 accounts: Quotas tab filters, expansion, advisor, and status calls remain usable.
- 10,000 snapshots: history chart, heatmap, retention cleanup, backup, and export remain bounded.
- 100,000 activity rows: activity view pagination/limits prevent browser lockups.
- Large invalid import: request is rejected within body-size limits and does not grow memory unboundedly.
- Provider timeout simulation: one slow provider does not block all provider status rendering indefinitely.
- Plugin timeout simulation: a hanging plugin is killed by timeout and recorded as a provider error.

### Manual Evidence Template

Record the final manual signoff in the release notes or PR:

```text
Commit:
Build:
OS/browser:
Database fixtures:
Automated checks run:
Manual sections completed:
Known skips with reason:
Failures found:
Fix commit(s):
Final verdict:
```

## Current Limits

- Browser end-to-end tests are not yet comprehensive enough to replace manual release checks.
- Race tests depend on a CGO-capable machine.
- Observed spend forecasting is intentionally absent until a trustworthy spend ledger exists.
- Plugin execution remains operator-trusted local-code execution through polling only; there is no sandbox.
