# Pattern 006: Standalone OpenAPI Specification on Cloudflare Workers

## Pattern header
- **Pattern ID:** 006
- **Name:** Standalone OpenAPI Spec — YAML-First API Documentation on Workers
- **Status:** Draft
- **Scope:** API documentation, spec authoring, spec serving, interactive UI, spec-to-code synchronization
- **Primary goal:** Maintain a hand-authored OpenAPI YAML file as the single source of truth for API documentation, served directly from a Worker with an interactive UI, without runtime YAML parsing.
- **Non-goals:**
  - Code-first / decorator-driven spec generation (e.g., `chanfana`, `zod-openapi`)
  - API gateway validation that blocks requests based on the spec at runtime
  - Client SDK generation (can be layered separately)
- **Key Cloudflare products involved:** Workers, Hono
- **Primary constraints to respect:**
  - Workers cannot `import` YAML natively — spec must be converted to a JS module at build time
  - No runtime YAML parsing — avoids unnecessary CPU cost and bundle-size bloat from YAML parsers
  - Spec and route handlers are maintained independently — synchronization must be validated, not assumed
- **Related patterns**
  - **See also:** Pattern 004 (Query Architecture) — endpoint shapes described in the spec should follow Pattern 004's API surface conventions
  - **See also:** Pattern 002 (Message Contract & Idempotency) — async endpoints documented in the spec should reference idempotency semantics

---

## Executive summary (5–8 lines)

This pattern separates API documentation from API validation into two independent layers: a hand-authored OpenAPI YAML spec (documentation layer) and Zod schemas in route handlers (runtime layer). The spec describes the API for consumers, Swagger UI, and AI agents; Zod enforces what the Worker actually accepts. Neither layer replaces the other. The YAML is converted to a JS template-literal module at build time (no runtime YAML parsing), served as `text/yaml` via a Worker route, and rendered interactively via Swagger UI. A build-time sync checker ensures the two layers don't drift apart. The key trade-off is manual synchronization, accepted in exchange for full editorial control over documentation quality and zero runtime overhead.

---

## Context and forces

### Platform constraints
- Workers cannot `import` `.yaml` files — the bundler does not handle YAML as a module type.
- Bundling `js-yaml` or similar parsers adds ~50 KB to the bundle and wastes CPU on every spec request.
- Workers have **128 MB memory** and **30s default CPU** — parsing a large YAML spec on every request is unnecessary overhead.

### Documentation quality
- Code-generated specs (from decorators or Zod schemas) produce technically correct but often unreadable documentation — missing examples, sparse descriptions, poor grouping.
- Hand-authored YAML allows full control over descriptions, examples, tags, and ordering — critical for external consumers and AI agent integration.
- A curated spec can include multiple request/response examples per endpoint, detailed markdown descriptions, and precise `operationId` values.

### Synchronization risk
- Decoupling spec from code means they can drift — a renamed field in Zod won't auto-update the YAML.
- This risk is acceptable when mitigated with a build-time sync check and CI enforcement.

### Serving constraints
- The spec must be served with `Content-Type: text/yaml` for Swagger UI and other tooling to consume it.
- Swagger UI is loaded via `@hono/swagger-ui`, which renders client-side and fetches the spec URL dynamically.

---

## Decision triggers

- If you want **full editorial control** over API docs (rich descriptions, curated examples, precise wording) → **use this pattern**.
- If you want **zero runtime overhead** for spec serving → **use this pattern**.
- If you want **automatic spec generation** that stays in sync with code changes with zero effort → **don't use this pattern** — use a code-first approach like `chanfana` or `zod-openapi` instead.
- If your API has **fewer than 3 endpoints** and no external consumers → this pattern may be overkill — a simple JSON response describing the API may suffice.
- If your spec will be consumed by **AI agents or MCP servers** → this pattern is strongly preferred because hand-authored descriptions are significantly better than auto-generated ones.

---

## Solution

### Two-layer architecture (the core idea)

This pattern produces two independent layers that describe the same API surface but serve different purposes:

```text
┌─────────────────────────────────────────────────────────┐
│  DOCUMENTATION LAYER  (what consumers see)              │
│                                                         │
│  OpenAPI YAML spec  →  Swagger UI, agent specs,         │
│                        client generators, MCP tooling   │
│                                                         │
│  Authored by hand. Optimized for human/LLM readability. │
│  Rich descriptions, curated examples, precise wording.  │
│  Has NO runtime effect — never validates a request.     │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│  RUNTIME LAYER  (what the Worker enforces)              │
│                                                         │
│  Zod schemas in route handlers  →  request validation,  │
│                                    type inference,      │
│                                    error responses      │
│                                                         │
│  Written in code. Optimized for correctness & safety.   │
│  This is what actually accepts or rejects requests.     │
└─────────────────────────────────────────────────────────┘

         ▲                              ▲
         │      SYNC CHECK (CI)         │
         └──────────────────────────────┘
         Validates that both layers describe
         the same fields, types, and constraints
```

**Neither layer replaces the other.** The spec documents; Zod enforces. A field described in the YAML but missing from the Zod schema means consumers will send data the handler silently ignores. A field validated by Zod but missing from the YAML means consumers won't know it exists. The sync checker and integration tests keep them aligned.

### Components

**Five components support this two-layer architecture:**

1. **YAML spec file** (`src/openapi/v1.yaml`) — the single source of truth. Developers edit only this file. Contains all paths, schemas, examples, security definitions, and server info.

2. **Build-time generator script** (`scripts/generate-openapi.js`) — reads the YAML, escapes special characters for JS template literals, and writes a `.js` module that exports the spec as a default string. Runs before `dev` and `deploy`.

3. **Spec-serving route** — a Worker route that dynamically imports the generated JS module and returns the string with `Content-Type: text/yaml`.

4. **Swagger UI route** — uses `@hono/swagger-ui` to render an interactive documentation page that points at the spec-serving route URL.

5. **Sync checker script** (`scripts/check-openapi-sync.js`) — compares the YAML source against the generated JS module to detect drift. Runs in CI and can run as a pre-commit hook.

### Flow

```text
Developer edits v1.yaml
        │
        ▼
npm run generate-openapi  (build-time)
        │
        ▼
Writes v1.js  (template-literal string export)
        │
        ▼
Worker imports v1.js on request
        │
        ├──▶  GET /openapi.yaml   →  serves raw YAML string
        │
        └──▶  GET /ui             →  Swagger UI (fetches /openapi.yaml client-side)
```

### File layout

```text
src/
├── openapi/
│   ├── v1.yaml          ← EDIT THIS (source of truth)
│   ├── v1.js            ← GENERATED (do not edit)
│   └── v1.d.ts          ← TypeScript declaration for the module
├── api/v1/<service>/
│   ├── ui.ts            ← Swagger UI route
│   └── *.ts             ← Route handlers with Zod validation
└── index.ts             ← Main app: registers spec route + UI route

scripts/
├── generate-openapi.js  ← YAML → JS converter
└── check-openapi-sync.js ← Drift detector
```

---

## Invariants (must always hold)

- **Two-layer separation:** The YAML spec is the documentation layer; Zod schemas in route handlers are the runtime layer. The spec never validates requests. Zod never generates documentation. Both must exist and describe the same API surface.
- **Single source of truth:** The YAML file is the only place spec content is authored. The `.js` module is always generated, never hand-edited.
- **No runtime YAML parsing:** The spec is served as a pre-built string. No YAML parser is loaded or invoked at runtime.
- **Build-before-serve:** The generator must run before `wrangler dev` and `wrangler deploy`. Wire it into `package.json` scripts as a prerequisite.
- **Sync validation in CI:** The sync checker must run in CI. A desync between YAML and JS must fail the build.
- **Correct Content-Type:** The spec route must return `Content-Type: text/yaml` (not `text/plain` or `application/json`).
- **Generated file marker:** The generated JS file must contain a `DO NOT EDIT` comment at the top to prevent accidental hand-edits.

---

## Implementation guidance (LLM-facing)

### Generator script

- ✅ Do read the YAML file as a UTF-8 string and embed it inside a JS template literal (`export default \`...\``).
- ✅ Do escape backticks (`\``), backslashes (`\\`), and template expressions (`${`) in the YAML content before embedding.
- ✅ Do include a `⚠️ DO NOT EDIT MANUALLY` header comment in the generated file.
- ❌ Avoid parsing the YAML into an object and then serializing it — this round-trip can alter formatting, ordering, and comments.
- **Because** the goal is an exact string reproduction of the YAML, not a structural transformation.

### Spec-serving route

- ✅ Do use dynamic `import()` to load the generated module — this allows tree-shaking and avoids bundling the spec string into the main entry chunk if the bundler supports code splitting.
- ✅ Do set the response header explicitly: `{ headers: { 'Content-Type': 'text/yaml' } }`.
- ❌ Avoid reading from R2 or KV at runtime to serve the spec — the generated JS module is simpler and has no external dependency.
- **Because** the spec is a static asset that changes only at deploy time — it belongs in the bundle, not in storage.

Pseudocode:
```
GET /api/v1/<<SERVICE>>/openapi.yaml
  module = dynamic_import('./openapi/v1.js')
  return text_response(module.default, content_type='text/yaml')
```

### Swagger UI route

- ✅ Do create a dedicated sub-router for the UI endpoint.
- ✅ Do point `swaggerUI({ url: ... })` at the spec-serving route's absolute path.
- ❌ Avoid inlining the spec content into the Swagger UI page — always reference it by URL so the UI fetches the latest deployed version.
- **Because** `@hono/swagger-ui` renders client-side and needs a fetchable URL.

Pseudocode:
```
GET /api/v1/<<SERVICE>>/ui
  return swagger_ui({ url: '/api/v1/<<SERVICE>>/openapi.yaml' })
```

### TypeScript declaration

- ✅ Do create a `v1.d.ts` file so TypeScript recognizes the generated module:

```
declare const spec: string;
export default spec;
```

### Sync checker

- ✅ Do extract the template-literal content from the JS file and compare it against the raw YAML after unescaping.
- ✅ Do normalize line endings (`\r\n` → `\n`) before comparison.
- ✅ Do exit with a non-zero code on desync so CI fails.
- ✅ Do print a remediation command (`npm run generate-openapi`) on failure.
- **Because** developers may edit the YAML and forget to regenerate — the sync check catches this before deploy.

### Package.json wiring

- ✅ Do wire the generator into `dev`, `build`, and `deploy` scripts as a prerequisite:

```
"generate-openapi": "node scripts/generate-openapi.js",
"check-openapi":    "node scripts/check-openapi-sync.js",
"dev":              "npm run generate-openapi && wrangler dev",
"deploy":           "npm run generate-openapi && wrangler deploy --minify",
"build":            "npm run generate-openapi && wrangler deploy --dry-run"
```

### YAML authoring conventions

- ✅ Do include multiple `examples` per request/response to show different usage patterns.
- ✅ Do write detailed `description` fields with markdown formatting — these become the primary documentation for consumers.
- ✅ Do use `operationId` on every endpoint — these are used by code generators and AI agent tooling.
- ✅ Do use `tags` to group related endpoints.
- ✅ Do define reusable schemas in `components/schemas/` and reference them with `$ref`.
- ❌ Avoid duplicating schema definitions inline when a `$ref` would work.
- **Because** hand-authored specs justify their maintenance cost only when they are significantly better than auto-generated ones — invest in quality.

### Agent-facing spec variant (optional)

- ✅ Do consider maintaining a separate curated spec (`agent-openapi-specs.yaml`) optimized for LLM context windows — fewer endpoints, more concise descriptions, focused on the endpoints an AI agent would actually call.
- ❌ Avoid auto-generating the agent spec from the main spec — the value is in manual curation.
- **Because** LLM context windows are limited and a full spec may be too large; a focused subset improves agent performance.

---

## Failure modes and mitigations

| Failure | Impact | Mitigation |
|---------|--------|------------|
| YAML edited but generator not run | Stale spec served to consumers | Wire generator into `dev`/`deploy` scripts; run sync check in CI |
| Generated JS hand-edited | Edits lost on next generation | `DO NOT EDIT` comment + sync check detects divergence |
| Spec describes fields that code doesn't validate | Consumer sends valid-per-spec data that the handler rejects | Build-time sync check (structural); integration tests that exercise spec examples against live handlers |
| Spec missing new endpoints | Consumers don't discover new functionality | Code review checklist; CI check that compares registered routes against spec paths |
| Large spec inflates bundle size | Slower cold starts | Acceptable for most APIs (<200 KB); for very large specs, consider serving from static assets or R2 |
| Swagger UI route exposed in production without auth | Unintended public documentation | Apply auth middleware to UI route or conditionally register it only in dev/staging |

---

## Observability and operations

- **Metrics:** Track request counts to `/openapi.yaml` and `/ui` to understand documentation usage.
- **Logs:** Log spec-serving errors (e.g., failed dynamic import) with request context.
- **CI checks:**
  - `npm run check-openapi` — validates YAML↔JS sync.
  - Optional: lint the YAML with `spectral` or `redocly` for OpenAPI best practices.
- **Deploy verification:** After deploy, `curl` the spec endpoint and verify it returns valid YAML with the expected `info.version`.

---

## Anti-patterns to avoid

- **"Parse YAML at runtime"** — bundling `js-yaml` or `yaml` just to serve a spec adds unnecessary weight and CPU cost. Convert at build time instead.
- **"Generate spec from decorators"** — produces spec that is technically correct but documentation-poor. If you want rich, curated docs, author the YAML by hand.
- **"Inline the spec in the handler"** — embedding a large YAML string directly in `index.ts` makes the file unreadable. Keep it in a dedicated file.
- **"Forget to regenerate after edits"** — the most common failure. Always wire generation into the dev/deploy pipeline.
- **"Ship Swagger UI to production without auth"** — interactive documentation can expose endpoint structure to unauthorized users. Gate it behind auth or restrict to non-production environments.
- **"One giant spec file with no $ref usage"** — a 3000+ line YAML file is manageable with `$ref` for reusable schemas; without it, duplication makes maintenance painful.
- **"Trust the spec as validation"** — the spec documents what the API *should* accept; Zod schemas in handlers enforce what it *actually* accepts. Both must exist.

---

## Acceptance checklist

- [ ] YAML file exists at `src/openapi/<version>.yaml` and is the only place spec content is authored
- [ ] Generator script converts YAML → JS module with proper escaping (backticks, backslashes, template expressions)
- [ ] Generated JS file contains a `DO NOT EDIT` header comment
- [ ] `npm run dev` and `npm run deploy` both run the generator before starting/deploying
- [ ] Spec route serves the content with `Content-Type: text/yaml`
- [ ] Swagger UI route renders and successfully fetches the spec
- [ ] Sync checker script exists and exits non-zero when YAML and JS diverge
- [ ] Sync checker runs in CI
- [ ] Route handlers validate requests with Zod independently of the spec
- [ ] Spec includes `operationId`, `tags`, `examples`, and detailed `description` fields for all endpoints
- [ ] Reusable schemas use `components/schemas/` with `$ref` (no inline duplication)
- [ ] TypeScript declaration (`v1.d.ts`) exists for the generated module

---

## References

- [Cloudflare Workers — Dynamic Imports](https://developers.cloudflare.com/workers/reference/apis/dynamic-imports/) — confirms dynamic `import()` works in Workers
- [@hono/swagger-ui](https://hono.dev/docs/middleware/builtin/swagger-ui) — Hono middleware for rendering Swagger UI
- [OpenAPI 3.0 Specification](https://spec.openapis.org/oas/v3.0.3) — the spec format itself
- [Spectral — OpenAPI Linter](https://github.com/stoplightio/spectral) — optional CI linting for spec quality
