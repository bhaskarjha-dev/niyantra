# ADR-0003: Public Website Architecture

| Field        | Value                                      |
| ------------ | ------------------------------------------ |
| **Status**   | Accepted                                   |
| **Date**     | 2026-05-18                                 |
| **Authors**  | Bhaskar Jha                                |
| **Feature**  | Public Web Presence (niyantra.bhaskarjha.dev) |
| **Deciders** | Project maintainer                         |

## Context

Niyantra is a local-first, single-binary AI operations dashboard built in Go. With the cloud sync phase (F17) approaching, Niyantra needs a public web presence at `niyantra.bhaskarjha.dev` serving four distinct purposes on a single origin:

1. **Marketing landing page** — features, pricing (Free vs Pro), install instructions, screenshots
2. **Documentation hub** — 10 markdown files (217KB+) from the repo's `docs/` folder rendered as a searchable, navigable documentation site
3. **Legal pages** — privacy policy and terms of service (required by Google OAuth consent screen)
4. **SEO/AI discovery** — `llms.txt`, `sitemap.xml`, JSON-LD structured data, Open Graph tags

The cloud dashboard SPA (`/app/`) and PocketBase REST API (`/api/`) are separate concerns already handled by the existing esbuild pipeline and PocketBase respectively. This ADR covers only the marketing + documentation website.

### Constraints

- **Same-origin requirement** — the website must be served from `niyantra.bhaskarjha.dev`, the same origin as PocketBase, to satisfy OAuth redirect URI requirements and avoid CORS complexity
- **Deploy target** — all static output goes to PocketBase's `pb_public/` directory on the Oracle Cloud VM
- **Documentation volume** — 10 markdown files totaling 217KB+ with code blocks, tables, diagrams, and deep technical content (the `API_SPEC.md` alone is 58KB)
- **Single maintainer** — the chosen tool must be maintainable by a solo developer long-term
- **Niyantra's identity** — `VISION.md` states: "Go compiles to a single static binary... No runtime dependencies, no package managers, no containers, no node_modules" and "Trivially portable, trivially deployable, trivially auditable"
- **Existing toolchain** — the main repo contains `package.json` with esbuild + TypeScript for the dashboard frontend (embedded in the Go binary via `embed.FS`)

### Prior Art: GitSetu Web

The maintainer's other project, GitSetu, uses a separate repository (`gitsetu-web`) built with Astro, hosted on Cloudflare Pages. A `sync_docs.sh` script pulls documentation from the main GitSetu repo before each build. This pattern is proven and operational.

## Decision

**We will build the public website using Astro in a separate repository (`niyantra-web`), with documentation synced from `niyantra/docs/` via a build-time script, deploying static output to the Oracle Cloud VM's `pb_public/` directory.**

### Two Decisions Made

**Decision 1 — Repository: Separate (`niyantra-web`)**

The website will live in its own repository, not inside the main `niyantra` repo.

**Decision 2 — Framework: Astro (custom, not Starlight)**

The website will be built with Astro using custom layouts and components, not the opinionated Starlight documentation theme.

## Decision 1: Why Separate Repository

### The Problem With Same Repo

Initially, same-repo (`website/` subfolder) was considered because Niyantra already has `node_modules/` (esbuild + TypeScript). However, deeper analysis revealed critical issues:

**Clone bloat:** Anyone cloning the repo to use the tool (for `go install`, Docker build, or development) downloads the entire website source — Astro components, marketing copy, pricing page templates — none of which are needed to build or use the tool.

**Philosophy violation:** Niyantra's `VISION.md` explicitly states "no node_modules" and "trivially auditable." The existing `node_modules` (esbuild + TypeScript) are build tools for the core product — they compile the dashboard that ships inside the binary. Astro's `node_modules` would serve marketing, a categorically different purpose. Adding them contradicts the tool's identity.

**Commit history pollution:** Go developers doing `git log` would see "Update pricing table copy" and "Fix hero section responsive breakpoint" mixed with "Fix Claude provider polling timeout." These are completely different work streams.

**Stack coupling:** If the website framework is ever changed (Astro → Hugo, or a future tool), the main repo carries dead code in its git history permanently.

**CI complexity:** Same-repo CI requires `paths:` filters to avoid running Go tests on website-only changes. Separate repos have simple, focused pipelines.

### What Major Go Projects Do

| Go Project | Website Location |
|-----------|-----------------|
| Kubernetes | `kubernetes/website` (separate) |
| Docker | `docker/docs` (separate) |
| Hugo | `gohugoio/hugoDocs` (separate) |
| PocketBase | `pocketbase/site` (separate) |
| Prometheus | `prometheus/docs` (separate) |
| Caddy | `caddyserver/website` (separate) |

Every major Go project keeps its website in a separate repository. The "monorepo" advice commonly found in web development literature applies to JavaScript projects where frontend and backend share types and deploy together — not to a Go CLI tool + a marketing website.

### How Documentation Stays In Sync

The `docs/` folder remains in the main `niyantra` repository as the single source of truth. The website repo syncs documentation before each build using `sync_docs.sh` (sparse git checkout, ~3 seconds). Automation via GitHub Actions `repository_dispatch` triggers a website rebuild when docs change in the main repo.

```
Developer pushes docs change to niyantra/docs/
  → GitHub Actions in niyantra detects docs/ path change
  → Sends repository_dispatch to niyantra-web
  → niyantra-web CI: sync_docs.sh → npm run build → rsync to pb_public/
  → Website shows updated docs within ~3 minutes
```

This pattern is proven in production by GitSetu.

## Decision 2: Why Astro

### How the Repo Decision Affects Framework Choice

With separate repo, the framework evaluation changes fundamentally:

- **Astro's weakness (Node.js in Go repo) disappears** — the website repo is a website project; JavaScript is natural for websites
- **Hugo's strength (Go-native in same repo) disappears** — in a separate repo, Hugo's Go-nativeness doesn't benefit the main tool's identity
- The comparison becomes purely about **website-building capability**, where Astro dominates

### Alternatives Considered

#### 1. Hugo (Separate Repo)

**Score: 31/45** (re-evaluated for separate repo context)

Hugo is a Go-native SSG — the fastest static site generator available. It renders markdown natively via Goldmark and requires zero npm dependencies.

**Why rejected:**
- ❌ **Weaker marketing page DX** — Go template syntax is verbose and less composable than Astro's `.astro` component format for complex marketing layouts with animations, hover effects, and interactive elements
- ❌ **Limited interactive element support** — terminal demos, animated feature showcases, and Islands Architecture require manual vanilla JavaScript wiring with no framework assistance
- ❌ **New tool to learn** — Go template partials, Hugo pipes, and content management patterns are unfamiliar; Astro knowledge already exists from GitSetu
- ✅ Zero JavaScript dependencies in the website repo
- ✅ Fastest possible build times (<100ms)

**When it would be right:** If the landing page were primarily text + screenshots with minimal interactivity, or if zero JavaScript in the website repo were a hard requirement.

#### 2. Hand-Crafted HTML

**Why rejected:**
- ❌ No templating — shared elements copy-pasted across 15+ pages
- ❌ 10 doc files (217KB) can't be auto-rendered from markdown
- ❌ No search without building from scratch

#### 3. Docusaurus

**Why rejected:**
- ❌ React ecosystem — heavy runtime, overkill for this scale
- ❌ Docs-focused — landing page is a second-class citizen

#### 4. VitePress

**Why rejected:**
- ❌ Vue ecosystem — unfamiliar framework
- ❌ Documentation-focused — weak on custom marketing pages

#### 5. Astro Starlight

**Why rejected in favor of custom Astro:**
- ❌ Opinionated documentation layout constrains landing page design
- ❌ Hero, pricing table, and feature showcase can't easily break out of Starlight's template
- ✅ Same Astro engine — could migrate to Starlight later if docs become the primary focus

### Why Astro Wins (41/45)

1. **Proven pattern** — `gitsetu-web` is the exact same architecture: Astro, separate repo, `sync_docs.sh`. 69 commits of production experience.
2. **Reusable foundation** — fork `gitsetu-web` as template; sync script, llms.txt generator, Orama/Pagefind search, JSON-LD, View Transitions all carry over (~70% reusable)
3. **Islands Architecture** — interactive components hydrate independently; static pages ship zero JavaScript
4. **Content Collections** — type-safe markdown handling with schema validation for the 10 doc files
5. **Marketing page DX** — `.astro` components are clean, composable, with scoped styles
6. **View Transitions** — SPA-like navigation without a client-side router
7. **Framework-agnostic Islands** — embed React, Svelte, or vanilla JS components without committing to any framework

## Architecture

### Repository Structure

```
github.com/bhaskarjha-com/niyantra       ← Go tool (pure, focused)
  ├── cmd/niyantra/                        ← Go entrypoint
  ├── internal/web/                        ← Dashboard backend + frontend
  │   ├── src/                             ← 34 TypeScript modules
  │   └── static/                          ← Embedded assets (app.js, style.css)
  ├── docs/                                ← Documentation SOURCE OF TRUTH
  ├── package.json                         ← esbuild + TS only (product build)
  └── go.mod

github.com/bhaskarjha-com/niyantra-web   ← Marketing + docs website
  ├── scripts/
  │   ├── sync_docs.sh                     ← Pulls docs from niyantra/docs/
  │   └── generate_llms.js                 ← AI agent discovery file
  ├── src/
  │   ├── components/                      ← Hero, Features, Pricing, etc.
  │   ├── content/docs/                    ← Synced markdown (gitignored)
  │   ├── layouts/                         ← Base, Marketing, Docs
  │   ├── pages/
  │   │   ├── index.astro                  ← Landing page (/)
  │   │   ├── privacy.astro                ← Privacy policy
  │   │   ├── terms.astro                  ← Terms of service
  │   │   └── docs/[...slug].astro         ← Dynamic doc routing
  │   └── styles/                          ← Niyantra-themed CSS
  ├── public/                              ← og-image, favicon, llms.txt
  ├── astro.config.mjs
  ├── package.json
  └── tsconfig.json
```

### Deploy Pipeline

Two independent deploy streams to the same `pb_public/` on Oracle VM:

```
niyantra-web CI:
  sync_docs.sh → npm run build → rsync dist/ → pb_public/ (exclude /app/)
  Serves: /, /docs/, /privacy, /terms, /llms.txt, /_assets/

niyantra CI:
  npm run build:prod → rsync static/ → pb_public/app/
  Serves: /app/ (cloud dashboard SPA)

PocketBase:
  Serves: /api/ (REST API), /_/ (admin dashboard)
```

### Documentation Sync

```bash
# scripts/sync_docs.sh
# Sparse checkout of niyantra/docs/ → src/content/docs/
# Adds Astro frontmatter to files that lack it
# Runs before every dev server start and production build
```

Automated trigger: when `docs/` files change in the main repo, GitHub Actions `repository_dispatch` triggers a niyantra-web rebuild.

### Multi-Project Pattern

```
[project] repo     + [project]-web repo    = Consistent pattern
gitsetu            + gitsetu-web           = ✅ In production
niyantra           + niyantra-web          = 🔜 This ADR
[future-project]   + [future-project]-web  = Same pattern
```

## Consequences

### Positive

- **Main repo stays pure** — `niyantra` contains only Go code, dashboard TypeScript, and documentation. No marketing website noise, no Astro dependencies, no pricing page components.
- **Clean commit history** — Go development and website updates are completely separate git histories. Contributors see only relevant commits.
- **Stack freedom** — the website framework can be changed at any time without touching the main repo. Archive `niyantra-web`, create a new one with a different tool.
- **Proven pattern** — identical to the working `gitsetu` + `gitsetu-web` architecture
- **Philosophy preserved** — Niyantra's identity as a "trivially auditable, no node_modules" tool remains intact. The website is a separate project.
- **Independent release cycles** — fix a typo on the pricing page without a Go binary release. Deploy in 30 seconds.
- **~70% reusable** — fork `gitsetu-web` as template; sync script, search, SEO, transitions all carry over
- **Focused CI** — each repo has simple, targeted pipelines. No path filters needed.

### Negative

- **Documentation drift risk** — docs can go out of sync if the sync automation fails. Mitigated by: (1) `repository_dispatch` auto-trigger, (2) sync runs on every CI build, (3) the sync script is 20 lines and battle-tested.
- **Two repositories to maintain** — more repos means more places to check. Mitigated by: the website is a relatively static project (updates weekly, not daily).
- **Deploy coordination** — two separate deploys target the same `pb_public/`. Mitigated by: `rsync --exclude='app/'` prevents website deploys from overwriting the cloud SPA.

### Neutral

- The `docs/` folder remains in the main niyantra repo regardless — it's the source of truth. The website only consumes it.
- Search (Pagefind) and SEO (llms.txt, JSON-LD) are framework-independent; they work identically with any SSG.
- The website CSS shares brand colors with the local dashboard but is otherwise an independent design system.

## References

- [Astro documentation](https://docs.astro.build/)
- [Astro Content Collections](https://docs.astro.build/en/guides/content-collections/)
- [Astro Islands Architecture](https://docs.astro.build/en/concepts/islands/)
- [GitSetu-web repository](https://github.com/bhaskarjha-com/gitsetu-web) (proven Astro + separate repo pattern)
- [Pagefind static search](https://pagefind.app/)
- [Hugo static site generator](https://gohugo.io/) (alternative considered)
- [Kubernetes website](https://github.com/kubernetes/website) (separate repo precedent)
- [PocketBase site](https://github.com/pocketbase/site) (separate repo precedent)
- [PocketBase pb_public](https://pocketbase.io/docs/going-to-production/)
- [GitHub Actions repository_dispatch](https://docs.github.com/en/rest/repos/repos#create-a-repository-dispatch-event)
- [llms.txt specification](https://llmstxt.org/)
- [JSON-LD SoftwareApplication schema](https://schema.org/SoftwareApplication)
