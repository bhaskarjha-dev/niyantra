# Niyantra — Builder's Constitution

> **Read before every coding session. 10 minutes.**
> **Full analysis:** See artifacts 01-03 in the planning directory.

---

## What Niyantra IS

An **AI Operations Command Center** — the single pane of glass for a developer's
entire AI tool ecosystem. Quota states, subscription costs, usage intelligence,
budget health, and agentic context — all in one local-first binary.

**The three pillars:**
1. **VISIBILITY** — Every provider, every account, every dollar, one place.
2. **INTELLIGENCE** — Not just data display. Trends, anomalies, forecasts, advisor.
3. **INTEGRATION** — CLI, dashboard, tray, MCP, webhook, push. Always accessible.

---

## The Agentic Foundation

This project is developed entirely by AI agents. Every architectural
decision optimizes for this reality.

### The Core Inversion

> Code is cheap to write and expensive to understand.
> Organization, clarity, and navigability > cleverness or brevity.

### What This Means In Practice

| Rule | Implementation |
|------|---------------|
| **Files are context boundaries** | Every file understandable without reading any other file |
| **300-line limit** | No code file exceeds 300 lines. No test file exceeds 500 lines. |
| **Vertical slicing** | Code that changes together lives together |
| **Explicit over implicit** | No magic, no hidden state, no convention-dependent behavior |
| **Contracts over docs** | Types > tests > linter rules > docs > comments |
| **Additive evolution** | New features ADD files, they don't MODIFY working files |
| **Registry over conditionals** | No switch/case chains that grow with every provider |
| **Tests are invariants** | Agents may write NEW tests. Never modify existing tests to pass. |

---

## The Seven Laws

### 1. The Provider Contract

> Every provider implements ONE interface. Adding a provider touches ONE file.

```go
type Provider interface {
    ID() string
    Name() string
    Category() Category
    Capabilities() Cap
    AuthMethods() []AuthMethod
    AutoDiscover() (*Creds, error)
    Fetch(ctx context.Context, creds *Creds) (*Snapshot, error)
    ConfigSchema() []ConfigField
    Color() string
    Icon() string
}
```

New provider = one file in `internal/providers/`. Zero schema migrations.
Zero route registrations. Zero handler files.

### 2. Unified Snapshot Schema

> One table for ALL provider snapshots. `data_json` for specifics.

```sql
CREATE TABLE snapshots (
    id          TEXT PRIMARY KEY,  -- ULID
    provider    TEXT NOT NULL,     -- "antigravity", "claude", "deepseek"
    account_id  TEXT NOT NULL,
    captured_at DATETIME NOT NULL,
    overall_pct REAL DEFAULT 0,   -- universal health signal (0-100)
    data_json   TEXT DEFAULT '{}', -- full provider-specific data
    ...
);
```

No UNION ALL. No per-provider tables. New provider = zero tables.

### 3. Service Layer

> Handlers don't contain business logic. Services do.

```
HTTP request → handler (validate) → service (logic) → store (SQL) → response
MCP tool call → tool (validate) → service (logic) → store (SQL) → response
CLI command  → command (parse)  → service (logic) → store (SQL) → output
```

ONE service, THREE consumers. Zero duplication.

### 4. Component Frontend

> Every UI element is a self-contained component.

Svelte components with typed props. No 70KB render files.
No `innerHTML` string concatenation. No file over 300 lines.

### 5. Every Surface Answers "Am I Ready?"

> Dashboard, CLI, tray, MCP — all share the same status pipeline.

```
QuotaService.GetStatus() → tray | CLI | dashboard | MCP | webhook
```

### 6. Config Schema Drives Everything

> Providers declare their config. Settings UI auto-generates.

```go
func (d *DeepSeek) ConfigSchema() []ConfigField {
    return []ConfigField{
        {Key: "deepseek_api_key", Label: "API Key", Type: FieldSecret,
         EnvVar: "DEEPSEEK_API_KEY", Hint: "platform.deepseek.com/api_keys"},
    }
}
```

### 7. Self-Healing Architecture

> Invariants are enforced by the system, not by developer discipline.

- Foreign keys prevent orphaned data
- Schema validation rejects malformed input
- Registry rejects unknown providers
- Config validation rejects invalid values
- Idempotent operations are safe to retry

---

## Project Structure

```
niyantra/
├── AGENTS.md                    — Agent briefing (< 200 lines)
├── internal/
│   ├── core/                    — Provider interface + registry + types
│   ├── providers/               — ONE FILE per provider
│   │   ├── antigravity.go       — Antigravity LS detection + RPC
│   │   ├── claude.go            — Claude Code JSONL + bridge
│   │   ├── codex.go             — Codex/ChatGPT OAuth API
│   │   ├── cursor.go            — Cursor session token API
│   │   ├── copilot.go           — GitHub Copilot PAT API
│   │   └── ...                  — Future providers (DeepSeek, OpenRouter, etc.)
│   ├── service/                 — Business logic (quota, budget, analytics)
│   ├── store/                   — SQLite persistence (interface + impl)
│   ├── agent/                   — Background polling loop
│   ├── notify/                  — Notification channels (OS, SMTP, webhook, push)
│   ├── web/                     — HTTP server (thin handlers)
│   ├── mcp/                     — MCP server (calls services)
│   ├── tray/                    — System tray
│   └── plugin/                  — Plugin system
├── ui/                          — Svelte frontend
└── docs/
```


---

## Hard Lines

### Safety
1. **Read-only.** Never programmatically switch accounts or modify provider state.
2. **Credentials stay local.** API keys, tokens, cookies — never synced to cloud.
3. **No telemetry.** Zero phone-home. Works in air-gapped networks.

### Architecture
1. **Single binary.** No external databases, no Redis, no message queues.
2. **Pure Go.** No CGo. Cross-compilation must work.
3. **Local-first always.** Cloud sync is additive, never required.
4. **300-line file limit.** Enforced. No exceptions.

### Dependencies
Every non-stdlib dependency must have an exit strategy.
"If this library dies tomorrow, how many files change?"
If the answer is "all of them" — it's too deeply coupled.

### Out of Scope
| Out | Why | Use Instead |
|-----|-----|-------------|
| Account switching | T&S risk | Manual switch, advisor recommends |
| Proxy/gateway | Different product | LiteLLM, Portkey |
| Full observability | Different product | Langfuse, Helicone |
| Native mobile app | PWA sufficient | Cloud PWA |

---

## The Checklist

Before shipping anything:

```
1. Does it help the user understand their AI usage better or faster?
2. Does it follow the Provider Contract?
3. Can the same logic be consumed by CLI, dashboard, MCP, and tray?
4. Does it work without cloud (local-first)?
5. Would adding a new provider require touching this code?
   (YES → wrong layer. Fix first.)
6. Is every file under 300 lines?
7. Can an agent reading just this file understand what it does?

All YES → Ship it.
```
