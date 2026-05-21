# ADR 005: Adopt OpenTelemetry (OTLP) for Claude Code Quota Tracking

## Status
Proposed

## Context
Niyantra currently monitors Anthropic's Claude Code CLI usage by polling and parsing local JSONL session files located in `~/.claude/projects/`. While this approach provides basic historical tracking, it suffers from several severe limitations that our community has highlighted:
1. **Mid-Session Blindness:** File polling is inherently delayed. Users are unaware of rapid quota depletion (often caused by prompt caching bugs or runaway context) until the session halts or the file is finally written.
2. **Fragility:** Parsing undocumented, internal JSONL files is brittle and liable to break if Anthropic alters their logging structure.
3. **Availability of Better Standards:** Anthropic officially documents and supports native OpenTelemetry (OTel) integration within Claude Code, allowing real-time emission of precise metrics.

To achieve state-of-the-art, "frontier" tracking capabilities, Niyantra needs to abandon reactive file scraping in favor of proactive, real-time telemetry ingestion.

## Decision
We will transition Niyantra's Claude Code tracking engine to a **Local OpenTelemetry (OTLP) Sink** architecture.

1. **Embedded OTLP Collector:** The Niyantra background agent (`internal/agent`) will instantiate a lightweight, local OTLP HTTP receiver (e.g., listening on `localhost:4318`).
2. **Environment Injection:** Niyantra will configure the local developer environment (or specifically processes spawned through Niyantra) with the necessary telemetry variables:
   ```bash
   export CLAUDE_CODE_ENABLE_TELEMETRY=1
   export OTEL_METRICS_EXPORTER=otlp
   export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4318"
   export OTEL_EXPORTER_OTLP_PROTOCOL="http/json"
   ```
3. **Metric Ingestion:** The Niyantra agent will ingest and process the `claude_code_token_usage` and `claude_code_cost_usage` metrics streamed from the CLI. Token usage includes attributes segmented by type (`input`, `output`, `cache_read`, `cache_creation`). Default temporality is **delta** (each export contains only the increment since last export).
4. **Real-Time UI Updates:** The telemetry data will be piped to the Niyantra dashboard via our existing WebSocket/SSE channels, enabling a live "Burn Rate" widget and proactive quota exhaustion alerts in the Quotas tab.

## Consequences

### Positive
*   **Real-Time Protection:** Users will receive mid-session alerts if token burn rates spike, protecting their wallets from runaway CLI processes.
*   **High Reliability:** OTLP is an industry-standard, stable contract supported officially by Anthropic, removing the fragility of regex/JSON parsing.
*   **Granular Metrics:** We gain access to high-fidelity cache hit/miss data exactly as measured by Anthropic's systems.

### Negative
*   **Increased Agent Complexity:** Niyantra's Go backend must now run an OTLP-compliant ingestion server and handle cumulative counter logic.
*   **Opt-In Dependency:** Relies on the user accepting the `CLAUDE_CODE_ENABLE_TELEMETRY` flag, which we must clearly communicate as a necessary step for live tracking.
