# Cloudflare Workers Pattern Index

> **Purpose:** Load this file first. Use it to decide which full pattern file(s) to fetch for the task at hand. Each entry provides enough context to reason about applicability without reading the full document.
>
> **Pattern files location:** `cloudflare/pattern-NNN.md`

---

## Quick routing table

| Pattern | Name | Load when the task involves... |
|---------|------|-------------------------------|
| 001 | Streaming Pipeline | Processing large/unbounded data, designing queue/workflow pipelines, memory-safe batching |
| 002 | Message Contract & Idempotency | Queue message design, deduplication, at-least-once handling, ack/retry logic, DLQ setup |
| 003 | Job Orchestrator | Long-running jobs, workflow lifecycle, cancellation/pause/resume, progress tracking |
| 004 | Query Architecture | D1 schema design, Drizzle ORM, API pagination, aggregation endpoints, search modes |
| 005 | Testing Architecture | Writing tests for Workers/Queues/Workflows, Vitest pool setup, test isolation |
| 006 | Standalone OpenAPI Spec | API documentation, serving OpenAPI YAML, Swagger UI, spec-to-code sync |

---

## Dependency graph

```text
001 (Pipeline) ──requires──▶ 002 (Message Contract)
003 (Orchestrator) ──uses──▶ 001 (Pipeline topology)
003 (Orchestrator) ──uses──▶ 002 (Idempotency at boundaries)
004 (Query Arch) ──uses──▶ 002 (Idempotent writes under retries)
005 (Testing) ──validates──▶ 001 (Async boundaries)
005 (Testing) ──validates──▶ 002 (Message contracts & dedupe)
006 (OpenAPI Spec) ──documents──▶ 004 (Query Architecture endpoints)
```

**Rule of thumb:** If you load 001, always also load 002. If you load 003, load 001 + 002. Pattern 004 is mostly standalone but references 002 for write idempotency. Pattern 005 is standalone but cross-references 001 and 002 for what to test. Pattern 006 is standalone but documents endpoints that should follow 004's API surface conventions.

---

## Platform limits cheat sheet

These hard numbers recur across all patterns. Refer here instead of re-reading each file.

| Resource | Limit |
|----------|-------|
| Workers memory per isolate | 128 MB |
| Workers CPU time (Paid) | 30s default, up to 5 min |
| Queue message size | 128 KB |
| Queue `sendBatch` | 100 messages / 256 KB total |
| Queue consumer duration | 15 min wall-clock |
| Queue delivery | At-least-once (no ordering guarantee) |
| Queue `max_retries` default | 3 |
| Workflow max steps | 1024 |
| Workflow event payload | 1 MiB |
| Workflow persisted state per step | 1 MiB |
| D1 max DB size (Paid) | 10 GB |
| D1 max row/value size | 2 MB |
| D1 max SQL statement | 100 KB |
| D1 max bound parameters | 100 |
| D1 max query duration | 30s |
| D1 queries per Worker invocation (Paid) | 1000 |

---

## Pattern summaries

### Pattern 001 — Streaming Pipeline with Queues and Workflows

**Core invariant:** Never make memory usage proportional to dataset size.

**What it solves:** Processing large or unbounded datasets on Workers without exceeding memory/CPU limits, by combining streaming/pagination, Queues, and Workflows.

**Two pipeline shapes:**
- **A — Single decoupling:** `Endpoint → Queue → Consumer → Workflow` (common case)
- **B — Double decoupling:** `Endpoint → Queue₁ → Consumer₁ → Workflow → Queue₂ → Consumer₂` (high fan-out)

**Three-layer architecture:**
1. **API endpoints** — accept intent, return job ID, respond fast
2. **Queue consumers** — process one message = one unit of work, idempotently
3. **Workflows** — durable multi-step orchestration with retries

**Key rules:** Stream in pages (never accumulate), batch IDs not objects, design for at-least-once, use DLQs.

**Cancellation model:** HTTP cancellation is request-scoped only. Job cancellation uses a durable flag in D1 + Workflows API terminate/pause.

---

### Pattern 002 — Message Contract and Idempotency

**Core invariant:** Assume duplicates and replays. Reprocessing must produce the same final state as processing once.

**What it solves:** Achieving effectively-once side effects over at-least-once delivery by standardizing message structure and enforcing idempotency end-to-end.

**Standard message envelope fields:** `schema_version`, `message_type`, `correlation_id`, `causation_id`, `idempotency_key`, `subject`, `created_at`, `payload_ref` or `payload_inline`.

**Idempotency key rules:** Must be deterministic and stable across retries, replays, and re-drives. Derive from business identity (e.g., `job_id + item_id + action`), never from system-generated `message.id`.

**Ack/retry rules:** Ack only after durable success. Ack per-message (not whole batch) to prevent replay amplification. Configure DLQ for every production consumer.

**Dedupe storage options:** Database unique constraint/upsert, dedicated inbox table (for complex side effects), or upstream API idempotency keys.

---

### Pattern 003 — Long-Running Job Orchestrator

**Core invariant:** Workflows are engines, not the system-of-record. The database is the brain.

**What it solves:** Coordinating long-running jobs where workflow state/progress must be queryable, cancellable, and auditable externally.

**Architecture (5 components):**
1. **Job record** — D1 table with status, progress counters, parameters, result pointers, timestamps
2. **Initiator** — validates input, creates job record, launches workflow instance(s), returns immediately
3. **Workflow instance(s)** — executes business logic in idempotent steps, reports progress to job record
4. **Lifecycle manager** — processes cancel/pause/resume commands via Queue consumer or Worker route
5. **Completion hooks** (optional) — event subscriptions to trigger follow-up without polling

**Cancellation model:** Flag intent in job record → terminate/pause via Workers API → workflows cooperate by checking flag between steps.

---

### Pattern 004 — Drizzle ORM and Hybrid Query Architecture on D1

**Core invariant:** Never query relational arrays by searching JSON text. Normalize what you filter/join/group on.

**What it solves:** Type-safe, injection-resistant, predictably fast data access on D1 with Drizzle ORM, including API endpoint design.

**Hybrid API surface (3 endpoint types):**
- **Record retrieval** — paginated rows with mandatory `limit`, deterministic sort, bounded columns
- **Aggregation** — COUNT/SUM/GROUP BY only, no row lists, cacheable
- **Search/discovery** — explicit mode selection: simple (structured), full-text (FTS5), or AI-assisted (async for heavy)

**Query safety rules:** Parameterized queries only, allowlisted fields for filters/sorts/group-bys, cap `IN` list sizes to 100 params.

**Schema rules:** Normalize repeated filterable values into join tables with indexes. Use JSON only for display-only/variable metadata. Use D1 JSON functions + generated columns when filtering on JSON subfields.

---

### Pattern 005 — Testing Architecture (Workers Vitest Integration)

**Core invariant:** Test behavior at the same boundary you rely on in production. Only mock what you don't control (external HTTP), not the platform.

**What it solves:** High-confidence testing of Workers systems (handlers, Queues, Workflows, storage bindings) using the Workers Vitest pool for runtime fidelity.

**Test tiers:**
- **Tier 1 — Unit:** Direct function calls for pure logic; handler calls with execution context for Worker-shaped logic
- **Tier 2 — Integration:** `SELF.fetch()` for HTTP boundary; `MessageBatch` for queue delivery; workflow introspection for step behavior

**Key constraints:** Tests run in workerd (not Node); isolated storage is default (one file at a time, no `.concurrent`); coverage requires Istanbul instrumentation; must drain `waitUntil()` explicitly.

---

### Pattern 006 — Standalone OpenAPI Spec (YAML-First API Documentation)

**Core invariant:** The YAML file is the single source of truth for API documentation. No runtime YAML parsing — convert to a JS module at build time.

**What it solves:** Serving hand-authored, high-quality OpenAPI documentation from a Worker without bundling a YAML parser, while keeping spec and code in sync.

**Five components:**
1. **YAML spec file** — hand-authored source of truth (`src/openapi/v1.yaml`)
2. **Generator script** — converts YAML → JS template-literal module at build time
3. **Spec-serving route** — `GET /openapi.yaml` returns the string with `Content-Type: text/yaml`
4. **Swagger UI route** — `GET /ui` renders interactive docs via `@hono/swagger-ui`
5. **Sync checker** — build-time/CI script that detects YAML↔JS drift

**Key trade-off:** Manual synchronization between spec and route handlers (mitigated by CI sync checks), in exchange for full editorial control over documentation quality.

**Optional:** A curated agent-facing spec subset optimized for LLM context windows.

---

## Decision triggers (condensed)

Use these to quickly determine which patterns apply:

- **"We need to process a large dataset"** → 001 + 002
- **"We're adding a Queue or Workflow"** → 002 (always), then 001 or 003 depending on pipeline vs orchestration
- **"We need job status, progress, or cancellation"** → 003 + 002
- **"We're designing API endpoints that query D1"** → 004
- **"We need to write or review tests"** → 005, plus 001/002 for what contracts to validate
- **"We're getting duplicate side effects"** → 002
- **"We're hitting memory limits"** → 001
- **"We're hitting D1 performance issues"** → 004
- **"We need API documentation or Swagger UI"** → 006
- **"We want a hand-authored OpenAPI spec"** → 006
- **"We're building an API for AI agents or MCP"** → 006 (curated agent spec)
