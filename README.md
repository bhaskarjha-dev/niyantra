# Niyantra

**Track every AI subscription. Monitor every quota. Control every dollar.**

[![Go 1.25+](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow?style=flat-square)](LICENSE)
[![CI](https://img.shields.io/github/actions/workflow/status/bhaskarjha-com/niyantra/ci.yml?style=flat-square&label=CI)](https://github.com/bhaskarjha-com/niyantra/actions)
[![Release](https://img.shields.io/github/v/release/bhaskarjha-com/niyantra?style=flat-square)](https://github.com/bhaskarjha-com/niyantra/releases)
[![Platform](https://img.shields.io/badge/macOS%20%7C%20Linux%20%7C%20Windows-grey?style=flat-square)](#install)

> Local-first. Zero daemon by default. Full provenance on every data point. MIT licensed.



---

## What Is Niyantra?

Developers in 2026 spend **$200-600/month** across 8+ AI tools. Each has its own quota dashboard, its own billing page, its own renewal date. Quota trackers show you API usage. Finance apps show you bank transactions. **Nothing shows you both.**

**Niyantra** is a local-first dashboard that **combines AI quota monitoring with subscription management, budget intelligence, and AI agent integration** in a single binary. No cloud. No accounts. All data on your machine.

## Install

### Quick Install (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/bhaskarjha-com/niyantra/main/install.sh | sh
```

### Quick Install (Windows PowerShell)

```powershell
# Download and extract latest release
$release = (Invoke-RestMethod "https://api.github.com/repos/bhaskarjha-com/niyantra/releases/latest").tag_name -replace '^v',''
$url = "https://github.com/bhaskarjha-com/niyantra/releases/latest/download/niyantra_${release}_windows_amd64.zip"
Invoke-WebRequest -Uri $url -OutFile "$env:TEMP\niyantra.zip"
Expand-Archive -Path "$env:TEMP\niyantra.zip" -DestinationPath "$env:LOCALAPPDATA\niyantra" -Force
# Add to PATH (run once)
$p = [Environment]::GetEnvironmentVariable("Path","User"); if ($p -notlike "*niyantra*") { [Environment]::SetEnvironmentVariable("Path","$p;$env:LOCALAPPDATA\niyantra","User") }
```

### Go Install (all platforms)

```bash
go install github.com/bhaskarjha-com/niyantra/cmd/niyantra@latest
```

### Download Binary

Pre-built binaries for macOS, Linux, and Windows are available on the [Releases page](https://github.com/bhaskarjha-com/niyantra/releases).

### Build from Source

```bash
git clone https://github.com/bhaskarjha-com/niyantra.git
cd niyantra

# macOS/Linux (with make)
make build

# Windows (or without make)
go build -o niyantra.exe ./cmd/niyantra
```

### Docker

```bash
# Quick start with docker-compose
git clone https://github.com/bhaskarjha-com/niyantra.git
cd niyantra
docker compose up -d
# Open the tokenized dashboard URL printed in the container logs.
```

Or run directly:

```bash
docker build -t niyantra:latest .
docker run -p 127.0.0.1:9222:9222 \
  -e NIYANTRA_BIND=0.0.0.0 \
  -e NIYANTRA_ALLOW_REMOTE=true \
  -e NIYANTRA_AUTH=admin:change-me \
  -e NIYANTRA_BEHIND_HTTPS_PROXY=true \
  -v ./niyantra-data:/data \
  niyantra:latest
```

For local Docker testing, publish the container only to host loopback as shown above. The server still sees a non-loopback container bind, so the explicit remote-bind safeguards are required. Use `docker compose logs` or container stdout to copy the generated `?token=...` dashboard URL.

Two image variants: **distroless** (default, ~15 MB, no shell) and **alpine** (`--target runtime-shell`, includes shell for `docker exec`).

## Try It Now

Don't have Antigravity running? No problem. The `demo` command seeds realistic sample data so you can explore immediately:

```bash
# macOS/Linux
make demo     # Seeds data + launches serve

# Windows
go run ./cmd/niyantra demo
go run ./cmd/niyantra serve
```

When you're ready to use real data:

```bash
niyantra snap     # Capture your Antigravity quotas (detects Antigravity 2.0 Main, IDE, and CLI)
niyantra serve    # Dashboard shows your real data
```

---

## Who Is This For?

- **Multi-account developers** juggling work + personal AI accounts
- **AI power users** paying $100-500/month across multiple subscriptions
- **Privacy-conscious developers** who want local-only data, no cloud dashboards
- **Freelancers & contractors** tracking AI costs per client or project
- **Anyone tired of checking 5 different dashboards** to answer "am I ready to code?"

---

## Features

### Know Your Quotas

Auto-capture the **Antigravity v2.0 suite** per-model quotas (Claude, Gemini, GPT) with rolling 5-hour reset detection. Track active sessions across **Antigravity 2.0 (Main)**, **Antigravity IDE**, and **Antigravity CLI** (with direct Google PA API fallback) concurrently. Track your native **Google AI Credits** balances continuously. Monitor **Codex/ChatGPT** via OAuth API, track **Claude Code** rate limits + deep JSONL token analytics, track **Cursor** usage via session token API, monitor **Gemini / Antigravity CLI** quotas via OAuth, and track **GitHub Copilot** premium interactions via PAT. **Quick Adjust** lets you fine-tune stale values with +/-5%/+/-10% buttons right on the dashboard. **7 providers** in one unified view. Optional plugins can collect custom local data through the polling agent; manual HTTP plugin execution is disabled.

### Control Your Spending

Track subscriptions across **26+ AI platforms** with renewals, spending breakdowns, and CSV export. Set a monthly budget and compare it against recurring commitments. Visual renewal calendar so nothing surprises you. **Estimated cost tracking** labels quota-derived values as estimates and reports unavailable costs when pricing is missing. **Git commit correlation** is heuristic activity attribution, not billing proof.

### Let AI Help You Code Smarter

**Advisor** ranks accounts by current readiness. It gives switch guidance only when a current account is explicitly supplied. **MCP Server** (13 tools over stdio + token-protected Streamable HTTP) lets your AI agent check quotas, analyze spending, query plugin data, and get routing recommendations. **Anomaly alerts** stay disabled until Niyantra has enough real daily spend history to analyze.

### Stay Informed

**Quad-channel notifications** alert you when quotas drop below threshold:
- **OS-native** (PowerShell / osascript / notify-send)
- **SMTP Email** (TLS/STARTTLS, HTML templates)
- **Webhooks** (Discord, Telegram, Slack, ntfy/Gotify)
- **WebPush** (browser push, works with tab closed)

### Your Data, Your Machine

SQLite database. No cloud. No telemetry. Full provenance audit trail on every snapshot -- you can always verify *how* and *when* data entered the system.

### Current Trust Limits

The dashboard prints a generated tokenized URL on `niyantra serve`; every `/api/*` and HTTP `/mcp` request must send `Authorization: Bearer <dashboard_api_token>`. Static UI assets and `/healthz` stay public. HTTP/manual plugin execution is disabled: `POST /api/plugins/{id}/run` returns `410 Gone`, and plugins may run only through the opt-in local polling path. Backups and JSON exports redact sensitive config values, including the dashboard token. Advisor output ranks accounts unless a current account is explicitly supplied.

---

## Dashboard

**4 tabs** -- Quotas, Subscriptions, Overview, Settings

| Tab | What it shows |
|-----|---------------|
| **Quotas** | Provider-sectioned layout (Antigravity/Codex/Claude/Cursor/Gemini/Copilot), per-model progress bars with reset timers, Quick Adjust (+/-5%/+/-10%), provider, status and tag filters, split-button snap, twin-axis history chart with event annotations, activity heatmap, AI Credits tracking |
| **Subscriptions** | Hybrid card + provider layout with spend summary, search, 26 platform presets, CSV export |
| **Overview** | Budget headroom against recurring subscriptions, advisor rankings/switch guidance, provider status signals, heuristic cost tracking, heuristic git attribution, observed token analytics, sessions timeline, renewal calendar, shareable PNG report, redacted JSON/CSV export + redacted DB backup |
| **Settings** | Auto-capture (7 providers), polling interval, notifications (4 channels + digest mode), plugin management, model pricing, Claude bridge, backup/restore, command palette (`Ctrl+K`) |

---

## CLI Reference

| Command | What it does |
|---------|-------------|
| `niyantra snap` | Capture current Antigravity account's quota |
| `niyantra status` | Show all accounts' readiness (offline) |
| `niyantra serve` | Launch web dashboard and print the tokenized first-open URL |
| `niyantra token show` | Print the dashboard API token |
| `niyantra token rotate` | Rotate the dashboard API token |
| `niyantra mcp` | Start MCP server (stdio) for AI agents |
| `niyantra demo` | Seed database with sample data |
| `niyantra backup` | Create timestamped database backup |
| `niyantra restore <file>` | Restore from backup |
| `niyantra healthcheck` | Docker health probe (GET /healthz) |
| `niyantra version` | Print version |

**Flags:** `--port 9222` `--bind 127.0.0.1` `--allow-remote` `--behind-https-proxy` `--mcp-http` `--db ~/.niyantra/niyantra.db` `--auth user:pass` `--insecure-plaintext-secrets` `--debug`

**Environment Variables:** `NIYANTRA_PORT` `NIYANTRA_BIND` `NIYANTRA_ALLOW_REMOTE` `NIYANTRA_BEHIND_HTTPS_PROXY` `NIYANTRA_MCP_HTTP` `NIYANTRA_DB` `NIYANTRA_AUTH` `NIYANTRA_INSECURE_PLAINTEXT_SECRETS` (CLI flags take precedence)

## MCP Integration

Niyantra exposes quota intelligence to AI coding agents via the [Model Context Protocol](https://modelcontextprotocol.io).

**13 tools:** `quota_status` `model_availability` `usage_intelligence` `budget_forecast` `best_model` `analyze_spending` `switch_recommendation` `codex_status` `quota_forecast` `token_usage_stats` `git_commit_costs` `copilot_status` `plugin_status`

**Stdio transport** (local agents):
```json
{
  "mcpServers": {
    "niyantra": {
      "command": "niyantra",
      "args": ["mcp"]
    }
  }
}
```

**Streamable HTTP transport** (opt-in): `POST /mcp` on the running dashboard server after starting `niyantra serve --mcp-http`. HTTP MCP requires `Authorization: Bearer <dashboard_api_token>`. For non-local binds, Niyantra refuses startup unless you also set `--allow-remote`, `--auth user:pass`, and `--behind-https-proxy` so Basic auth is not sent over plaintext HTTP. SSE streaming and session management are handled by the MCP SDK.

Then ask: *"What's my quota?"* or *"Which account should I use?"* or *"How much Claude activity landed near my recent commits?"*

---

## How Does Niyantra Compare?

| Feature | Niyantra | Quota trackers (onWatch, CodexBar) | Sub trackers (Wallos) | Account managers (Antigravity-Manager) |
|---------|----------|-------------------------------|---------------------------|-----------|
| AI quota monitoring | 7 providers (AG Suite [Main/IDE/CLI] + Codex + Claude + Cursor + Gemini + Copilot + Manual) | 9-16+ providers | — | 4+ providers |
| Multi-account per provider | ✅ 28+ (passive, safe) | ❌ Single-account | — | ✅ (active switching ⚠️) |
| Subscription management | 26 AI platforms, renewals, CSV | — | Generic subs | — |
| Budget forecasting | Monthly budget with projections | — | Basic budget | — |
| Advisor (account routing) | Multi-factor scoring; switch guidance only with explicit current account | — | — | — |
| MCP for AI agents | 13 tools over stdio + HTTP | — | — | — |
| Notifications | Quad-channel (OS + SMTP + Webhook + WebPush) | — | 10+ channels | — |
| Token analytics | Per-model cost estimation + git correlation | — | — | — |
| Renewal calendar | Visual month view | — | Calendar view | — |
| Activity audit trail | Full provenance on every data point | — | — | — |
| Zero-daemon default | Manual mode, opt-in auto | Daemon by default | N/A | Daemon |
| Account switching | ❌ (by design — safety) | ❌ | — | ✅ (⚠️ T&S risk) |
| Docker support | ✅ Distroless + Alpine | ✅ | ✅ | — |
| License | MIT | MIT / GPL-3 | GPL-3 | MIT |

> **Competitive position:** Niyantra scores **37+ features** in our competitive analysis — leading all 28 tools evaluated across 8 market categories. The closest competitor (onWatch) scores 12/37. See [VISION.md](docs/VISION.md) for full market positioning.

---

## FAQ / Troubleshooting

### "Antigravity not detected"

Niyantra dynamically auto-detects all running tools in the **Antigravity v2.0 suite** (Antigravity 2.0 Main and Antigravity IDE). If neither is active, Niyantra automatically falls back to "CLI Solo Mode" using your local **Antigravity / Gemini CLI** OAuth credentials (`~/.gemini/oauth_creds.json`) and queries Google's Cloud Code PA API directly.

If you are experiencing detection issues, make sure:
1. Either Antigravity 2.0 Main or Antigravity IDE is active (not just installed).
2. For the IDE, ensure a project/workspace file is open (as the local language server starts lazily).
3. For CLI Solo Mode, ensure you have logged in via the terminal (`antigravity auth login` or `gemini auth login`) so credentials are cached at `~/.gemini/oauth_creds.json`.
4. Try `niyantra snap --debug` for verbose process detection logs.

### "Can I use this without Antigravity?"

Yes! Niyantra's subscription manager, budget tracking, and renewal calendar work independently. Use `niyantra demo` to explore, or add subscriptions manually without any Antigravity integration.

### "Is my data sent anywhere?"

No. By default, Niyantra makes HTTP calls purely locally to `127.0.0.1` (your local Antigravity Language Server). There is no telemetry or analytics. **Optional provider polling** (Codex, Cursor, Gemini, Copilot) makes HTTPS calls to their respective APIs when explicitly enabled. **Optional notifications** (SMTP, Webhook, WebPush) make outbound calls to configured endpoints. See [SECURITY.md](docs/SECURITY.md) for the full threat model.

### "How do I update?"

Re-run the install command or download the latest from [Releases](https://github.com/bhaskarjha-com/niyantra/releases). Your data (`~/.niyantra/niyantra.db`) is preserved across updates.

---

## Dependencies

| Dependency | Purpose |
|-----------|---------|
| [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) | Pure-Go SQLite -- no CGo, true single binary |
| [`go-sdk/mcp`](https://github.com/modelcontextprotocol/go-sdk) | Official MCP Go SDK for AI agent integration |
| [`go-keyring`](https://github.com/zalando/go-keyring) | OS-native secret storage for sensitive config values |
| Go stdlib | Everything else -- HTTP, JSON, embed, crypto |

No web frameworks. No ORMs. Chart.js bundled locally from embedded assets.
Frontend: 40 TypeScript modules (strict mode) bundled by esbuild into a single IIFE.

## Documentation

| Document | Content |
|----------|---------|
| **[USER_GUIDE.md](docs/USER_GUIDE.md)** | **Complete feature guide — start here** |
| [VISION.md](docs/VISION.md) | Product vision, market position, roadmap (Phases 1-16), competitive analysis |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design, data flow, security model |
| [API_SPEC.md](docs/API_SPEC.md) | REST API reference (55+ endpoints) |
| [DATA_MODEL.md](docs/DATA_MODEL.md) | SQLite schema v21 (18 persistent tables) |
| [SECURITY.md](docs/SECURITY.md) | What data is accessed, network behavior, threat model |
| [TESTING.md](docs/TESTING.md) | Automated gates and comprehensive manual release checklist |
| [CONTRIBUTING.md](docs/CONTRIBUTING.md) | Development setup, code style, PR guidelines |
| [CHANGELOG.md](CHANGELOG.md) | Version history (v0.1.0 → v0.29.0) |

## Contributing

See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for development setup. Quick start:

```bash
git clone https://github.com/bhaskarjha-com/niyantra.git
cd niyantra
make build          # Build binary
make test           # Run tests
niyantra demo       # Seed sample data
niyantra serve      # Launch dashboard for development
```

## License

[MIT](LICENSE) -- (c) 2026 Bhaskar Jha
