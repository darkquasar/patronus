# Pattern 005: The Tester & Validator — Cloudflare Workers Testing Architecture

## Pattern header

- **Pattern ID:** 005
- **Name:** Tester & Validator — Workers Vitest Integration Test Architecture
- **Status:** Draft
- **Scope:** Worker handlers (fetch/scheduled/queue), Workflows, Durable Objects, storage bindings (D1/KV/R2/Cache), outbound HTTP, and CI test execution
- **Primary goal:** Achieve **high-confidence validation** of Workers systems with **fast feedback** while preserving **runtime fidelity**.
- **Non-goals:**
  - Load/soak testing, chaos testing, or performance benchmarking at production scale
  - Browser/UI end-to-end testing (can be layered separately)
  - Security scanning/static SAST (handled by separate pipelines)
- **Key Cloudflare products involved:** Workers, Queues, Workflows, Durable Objects, D1, KV, R2, Cache API
- **Primary constraints to respect:**
  - Tests must run in the **Workers runtime** when validating platform behavior (not "Node-only" assumptions).
  - **Isolation vs concurrency** trade-offs (isolated storage defaults and `.concurrent` limitations).
  - Code coverage must use **instrumented coverage (Istanbul)** (native V8 coverage not supported in the pool).
  - Known limitations: fake timers + storage, dynamic `import()` with `SELF`, DO alarms persistence, and DO WebSockets + isolated storage.

## Cross-references

- **See also:**
  - **Pattern 002** (message envelopes/contracts for Queue/Workflow boundaries) — tests should enforce these contracts
  - **Pattern 001** (Queue ↔ Workflow decoupling) — integration tests should validate the async boundaries and idempotency rules
- **Related Cloudflare testing primitives:** `cloudflare:test` APIs (SELF, env, queue/workflow helpers, D1 migrations helpers).

---

## Executive summary

Cloudflare Workers systems are event-driven and depend heavily on runtime-provided bindings (Queues, Workflows, D1, KV, R2). Cloudflare recommends the **Workers Vitest integration** so tests can run inside the **Workers runtime** (via a custom Vitest pool), which dramatically reduces environment drift.
This pattern splits tests into **fast "logic validation"** (unit) and **high-fidelity "system validation"** (integration), while preserving deterministic outcomes through isolation, durable execution context handling, and workflow/queue test helpers.
The core invariant is: **test behavior at the same boundary you rely on in production** (HTTP, queue delivery, workflow durability), and only mock what you do not control (external HTTP), not the platform itself.

---

## Context and forces

### Platform limits that matter

- **Two runtimes exist during test execution:**
  - Vitest **configuration + `globalSetup`** execute in **Node.js**
  - `setupFiles` + test files execute in **workerd** (Workers runtime)
- **Isolation and concurrency are coupled:**
  - `isolatedStorage` is **enabled by default**
  - With isolated storage, the current implementation runs **one test file at a time per workerd process**, and **does not support `.concurrent` tests**
- **Coverage constraint:** instrumented coverage via **Istanbul** is required.

### Data shape / system shape

- Workers apps typically span: HTTP endpoints, queue consumers, workflow steps, and storage side effects.
- Tests must prove correctness under **retries**, **at-least-once delivery**, and **partial failures** at async boundaries (Queues/Workflows). (Cross-reference Pattern 002.)

### Latency vs throughput (feedback loop)

- You need a tight loop for logic changes (unit) and a slightly slower loop for confidence at boundaries (integration).
- Where possible, use workflow/queue test utilities to avoid "waiting minutes" for sleeps/timeouts.

### Correctness model

- At-least-once semantics imply duplicates are possible → tests should validate **idempotency** and **dedupe keys** (Pattern 002).
- `waitUntil()` work must be explicitly awaited in tests to avoid false positives.

### Operational needs

- Tests must be deterministic and isolated (avoid cross-test storage bleed). `isolatedStorage` provides automatic rollback/undo per test, but requires disciplined async handling.

---

## Decision triggers

Use these "if this, then that" rules:

- **If you are validating pure business logic** (parsing, validation, transformations) → **Unit Test** (direct calls, no bindings).
- **If you are validating Worker behavior** (request routing, auth, headers, response shape, `waitUntil()` side effects) → **Handler Unit Test** (directly call `worker.fetch(...)` with an execution context and wait for it).
- **If you are validating the HTTP boundary as clients see it** → **Integration Test** using `SELF.fetch(...)` (service binding to your main Worker).
- **If you are validating queue delivery semantics / ack-retry outcomes** → **Queue Integration Test** using a `MessageBatch` + queue-result inspection utilities.
- **If you are validating workflow step behavior, waits, retries, sleeps** → **Workflow Test** using workflow introspection utilities (disable sleeps, mock step outcomes, await completion).

Don't use this pattern if:

- You only need quick smoke checks in a deployed preview environment (that's a different pattern: environment validation).
- You are trying to do high-volume performance/load testing (use dedicated load tools + staging).

---

## Solution (architecture in words)

### Component roles

- **Test runner (Vitest)**
  - Coordinates test discovery and reporting
  - Runs config + global setup in Node, runs test code in workerd
- **Workers Vitest pool (`@cloudflare/vitest-pool-workers`)**
  - Runs test code inside the Workers runtime for fidelity
- **`cloudflare:test` helper surface**
  - Provides `env` (typed bindings), `SELF` (integration fetcher), execution context helpers, queue/workflow helpers, and outbound fetch mocking.

### Boundaries (what each piece must not do)

- Unit tests must not depend on real storage or network.
- Integration tests must not rely on Node-only facilities (filesystem, Node globals) inside workerd test bodies.
- Tests must not "fire and forget" async work; they must drain `waitUntil()` and consume response bodies where applicable to avoid isolation issues and flakiness.

### Interfaces / contracts (described, not coded)

- **Correlation IDs:** each test scenario should define a stable `<<test_run_id>>` used in job IDs / message IDs / workflow instance IDs (so failures are debuggable).
- **Queue message contract (Pattern 002):**
  - Required fields: `<<message_id>>`, `<<job_id>>`, `<<attempt>>`, `<<dedupe_key>>`, `<<payload_ref_or_small_payload>>`
- **Workflow instance contract:**
  - Required fields: `<<instance_id>>`, `<<job_id>>`, `<<step_name>>`, `<<status>>`, `<<progress_marker>>`

### Flows as arrows (no code)

```text
Tier 1 — Logic validation (unit):
test → direct function call → assert output

Tier 1b — Handler validation (still "unit", but Worker-shaped):
test → create execution context → call handler directly → wait for waitUntil() → assert

Tier 2 — HTTP integration (production-shaped):
test → SELF.fetch → Worker fetch handler → real bindings (isolated) → assert response + persisted effects

Tier 2 — Queue integration:
test → create message batch → call queue handler → inspect ack/retry outcome → assert storage side effects

Tier 2 — Workflow validation:
test → introspect workflow instance → disable sleeps / mock steps → create instance → await status/output → assert
```
