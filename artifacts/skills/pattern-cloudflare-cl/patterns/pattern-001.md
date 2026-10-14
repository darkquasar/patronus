# Pattern 001: Streaming Pipeline with Queues and Workflows on Cloudflare Workers

## Executive summary

This pattern describes how to process **large or unbounded datasets** on Cloudflare Workers without running out of memory or exceeding execution limits by combining **streaming/pagination**, **Queues**, and **Workflows**.

**Core invariant:** _Never make memory usage proportional to dataset size._
Workers isolates have **128 MB memory per isolate**, so designs that accumulate large arrays or buffer big payloads will eventually fail.

The pattern provides two proven decoupling shapes:

- **Pattern A — Single decoupling (common):**
  `HTTP endpoint → Queue → Consumer → Workflow`
- **Pattern B — Double decoupling (high fan-out / heavy coordination):**
  `HTTP endpoint → Queue₁ → Consumer₁ → Workflow (orchestrator) → Queue₂ → Consumer₂`

## Related Patterns

Requires: Pattern 002 (Message Contract & Idempotency) for any queue/workflow boundary

See also: Pattern 001 (Pipeline Topology), Pattern 003 (Job Orchestrator), Pattern 004 (Query Architecture)

## Context and forces (what drives this pattern)

### Cloudflare constraints you must design around

- **Workers memory:** 128 MB per isolate.
- **Workers CPU time (Paid):** default 30s active CPU per invocation, configurable up to 5 minutes.
- **Queue consumer duration:** up to 15 minutes wall-clock per consumer invocation.
- **Queues message size and batching:**
  - Max message size **128 KB**
  - Max consumer batch size **100**
  - `sendBatch` max **100 messages** (or **256 KB total**)
- **Queues delivery semantics:** at-least-once (duplicates are possible).
- **Queues ordering:** no guarantee of delivery order.
- **Workflows durability & limits:**
  - Max steps per workflow: **1024**
  - Event payload size: **1 MiB**
  - Persisted state per step: **1 MiB**
  - Steps should be designed for retries and idempotency

### Common real-world pressure

- Users want fast HTTP responses, but the "real work" may take minutes/hours.
- Data sources can be large (thousands to millions of rows/files).
- External APIs impose rate limits; you need buffering and backpressure.
- Failures happen; retries must not create duplicate side effects.

## The solution: three-layer pipeline (roles and boundaries)

### Layer 1 — API endpoints (request interface)

**Responsibility:** Accept user intent, validate it, and respond quickly.

**Do**

- Return quickly with a job/workflow ID and a status URL.
- For "list" endpoints, enforce pagination and reasonable limits so response size stays bounded.
- Delegate any heavy or long work to Queues/Workflows.

**Avoid**

- Scanning entire datasets during an HTTP request.
- Returning unbounded lists.
- Building large arrays in memory.

**Why**

- Memory is capped at 128 MB per isolate.
- Paid CPU defaults to 30s active CPU per invocation; compute-heavy work can exceed it.

### Layer 2 — Queue consumers (atomic "work item" processors)

**Responsibility:** Turn queued messages into small, repeatable operations.

**Do**

- Keep each message a _single unit of work_ (one file, one record, one page token).
- Expect duplicates and ensure processing is idempotent (dedupe keys, upserts, "already processed?" checks).
- Use DLQs for "poison messages" (after max retries).
- Let concurrency autoscale unless you're constrained by an upstream API; Cloudflare recommends leaving max concurrency unset in many cases.

**Avoid**

- "Consumer expands message into 5,000 items" (it defeats batching controls).
- Doing large DB scans inside the consumer.

**Why**

- Consumer invocations are bounded by platform limits; Queues are meant to control throughput by message shaping and batching.

### Layer 3 — Workflows (durable orchestration and long-running coordination)

**Responsibility:** Coordinate multi-step processes durably and safely retry steps.

**Do**

- Use Workflows when you need retries, sleeps, waiting for events, or long-running multi-step logic.
- Keep workflow steps granular, idempotent, and deterministic.
- Persist large artifacts externally (D1/R2), and keep workflow step outputs small (Workflows limit persisted state per step).

**Avoid**

- Treating a workflow as a place to load "all items" into memory.
- Returning large blobs from steps or stuffing large arrays into step state.

## Decoupling options

### Pattern A — Single decoupling (simple coordination, heavy processing)

**Shape:** `Endpoint → Queue → Consumer → Workflow`

**Use when**

- The endpoint can quickly validate and enqueue.
- The heavy work is in the durable workflow (AI calls, transformations, multi-step logic).

**Message contract rule**
Because queues messages max at **128 KB**, the queued payload should usually include **IDs and pointers**, not raw data.

Typical message fields (word description):

- Correlation/job ID
- A small batch of item identifiers (or a page cursor)
- Parameters (model name, flags) kept small

### Pattern B — Double decoupling (heavy orchestration + high fan-out)

**Shape:** `Endpoint → Queue₁ → Consumer₁ → Workflow (orchestrator) → Queue₂ → Consumer₂`

**Use when**

- You must crawl/stream a large external data source (repo scan, export, migration).
- You need an orchestrator to produce thousands of fine-grained work items.
- Per-item work is short/atomic and can be parallelized widely.

**Important rule**
Queues do **not** guarantee ordering. If order matters, build an ordering mechanism in your data model (sequence numbers + "only process next if previous done").

## Streaming and batching rules (memory-safe by design)

### Rule 1 — Stream from sources, don't accumulate

- When reading from D1 or an external API, read in **pages** and process each page independently.
- Never append pages into a growing in-memory array.

### Rule 2 — Batch boundaries must respect message limits

- Queue messages are capped at **128 KB**, and `sendBatch` caps at **256 KB total**.
  So "batching" often means batching **IDs**, not full objects.

### Rule 3 — Design for at-least-once delivery

- Messages can be delivered more than once; consumers must dedupe/idempotently write.
  Practical approaches:
- Use a message ID / dedupe key stored in D1.
- Use database uniqueness constraints (where possible) or "upsert with idempotency key".
- Make downstream APIs idempotent (pass a stable idempotency key).

## Cancellation model (two distinct kinds)

### A) HTTP request cancellation (client disconnect)

If a client disconnects, Workers will cancel tasks associated with that request. There is also a limited grace period via `waitUntil` (up to ~30 seconds) described in Workers limits.
Cloudflare also introduced support for handling request cancellation via `Request.signal` (with a compatibility flag).

**Pattern rule:** request cancellation should not be your job-cancellation mechanism. It only applies to the request lifecycle.

### B) Job/workflow cancellation (application-level)

For long-running background jobs:

- Store cancellation intent in durable state (e.g., D1 job record).
- Have orchestration logic check that flag between batches/steps.
- Use the Workflows API to terminate or pause/resume instances when appropriate (see Pattern 003 for full control-plane pattern).

## Anti-patterns to avoid (recognizable smells)

1. **Memory accumulation**
   - "Read pages → push into array → process later"
   - Fails under 128 MB isolate limit.

2. **Consumer expands scope**
   - "Message contains repo ID → consumer fetches all files and processes them"
   - Defeats queue batching controls and creates unpredictable load.

3. **Oversized messages**
   - "Queue message includes full document contents"
   - Violates the **128 KB** message limit.

4. **Assuming ordering**
   - "Process files in the exact order sent"
   - Queues do not guarantee delivery order.

## Acceptance checklist (what "done" looks like)

- Memory usage is bounded by `page_size` / `batch_size`, never dataset size.
- Queue messages contain pointers/IDs and stay under 128 KB.
- Consumer logic is idempotent against duplicates.
- DLQ exists for poison messages (or an explicit discard policy).
- Workflows steps are granular and safe to retry; no side effects outside durable steps.

## References
