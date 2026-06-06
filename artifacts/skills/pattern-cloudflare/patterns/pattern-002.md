# Pattern 002: Message Contract and Idempotency for Cloudflare Queues and Workflows

## Pattern header

- **Pattern ID:** 002
- **Name:** Message Contract and Idempotency for Cloudflare Queues and Workflows
- **Status:** Draft
- **Scope:** Any system that uses **Cloudflare Queues** and/or **Cloudflare Workflows** for async processing, including:
  - Producer Workers (HTTP endpoints, cron, workflows emitting downstream work)
  - Queue consumers (atomic processing)
  - Workflows (durable orchestration and step retries)
  - D1 writes triggered by async processing (progress tables, results tables, audit logs)
- **Primary goal:** Achieve **effectively-once side effects** (no double-writes, no double-charges, no duplicate notifications) over **at-least-once delivery** by standardizing message structure and enforcing idempotency rules end-to-end.
- **Non-goals:**
  - Designing the whole pipeline topology (covered by Pattern 001)
  - Designing the job "brain" record and lifecycle control plane (covered by Pattern 003)
  - Database schema normalization and query APIs (covered by Pattern 004)
- **Key Cloudflare products involved:** Workers, Queues, Workflows, D1 (plus optional R2 for large payload offload)
- **Primary constraints to respect (hard platform realities):**
  - **Queues** are **at-least-once**; duplicates are expected.
  - **Queues message size:** 128 KB per message; batches up to 100 messages and up to 256 KB total per `sendBatch`.
  - **Queue consumer duration:** up to **15 minutes wall-clock** per invocation; CPU time is configurable (Workers limits).
  - **Queue ordering:** best-effort only; do not assume FIFO.
  - **Workflows steps are retriable**; external calls must be idempotent.
  - **Workflows payload/state limits:** event payload max 1 MiB; persisted state per step max 1 MiB; max steps per workflow 1024.
  - **D1 write retries are recommended** on transient errors → write paths must be safe under retries.

---

## Executive summary

Cloudflare Queues prioritize reliability and latency by providing **at-least-once delivery**, which means **the same logical work may be processed more than once**.
Cloudflare Workflows prioritize durability by retrying failed steps and resuming execution; this similarly implies **re-execution is normal** and must be made safe.

This pattern defines:

1. A **standard message envelope** (stable IDs, schema versioning, causality/correlation)
2. A **stable idempotency model** (dedupe keys + durable checks)
3. **Ack/retry rules** for batch consumers (avoid "whole-batch replay" surprises)
4. A consistent approach to **DLQs**, retry delays, and observability so failures become diagnosable, not mysterious.

---

## Core architectural principle

**Assume duplicates and replays. Design so reprocessing produces the same final state as processing once.**

- For Queues, duplicates happen because delivery is **at-least-once**.
- For Workflows, steps can retry, so non-idempotent operations must be guarded.

---

## Context and forces

### What Queues guarantees (and what it does not)

- **Delivery is at-least-once**, so you must de-duplicate at the application level when duplicates would cause harm.
- **Message ordering is not guaranteed** (best-effort only).
- **Batch failure semantics matter:** if one message in a batch fails and you don't explicitly acknowledge successful ones, **the whole batch can be retried** and redelivered.

### What the consumer API provides that you should use

- Each message includes:
  - `id` (system-generated)
  - `attempts` (starts at 1)
  - `ack()` and `retry()` controls
- The batch includes `ackAll()` and `retryAll()`—powerful but easy to misuse.

### D1 reality: retries are normal

- Cloudflare recommends retrying **write queries** on transient errors, and D1 automatically retries read-only queries up to two more times. This pushes you toward idempotent write design.

### Queue and workflow limits shape message design

- Queue messages are capped at **128 KB** and include internal metadata overhead; keep headroom.
- Workflows payload/state limits are **1 MiB**, so "just pass the dataset" is not viable. Use pointers.

---

## Decision triggers

### When must you implement idempotency?

You must implement idempotency whenever **any side effect** would be harmful if repeated, including:

- database writes (insert/update progress, results, audit logs)
- external API calls (payments, emails, ticket creation)
- enqueuing downstream work that would cause duplication

This is non-negotiable because:

- queues are at-least-once
- workflow steps may retry

### When is a dedupe store required vs "unique constraint is enough"?

- **Unique constraint is enough** when the side effect is a single-row insert/upsert keyed by a stable business key (example: "this work item result row").
- **Dedicated dedupe/inbox table is required** when:
  - side effects span multiple rows/tables
  - side effects include external calls (email/payment) without strong idempotency support
  - you need a recorded "processed / failed / last error" lifecycle for triage and replay

### When to prefer Queues vs Workflows for a unit of work (tie-in rule-of-thumb)

- If a **single message** can be completed within a queue consumer invocation (≤ **15 minutes wall-clock**) use Queues + idempotent consumer.
- If the work needs **waiting, multi-step retries/timeouts, durable progress**, use Workflows and still enforce idempotency at each step boundary.

---

## Solution: Standard message contract

### 1) Envelope: required fields (word-defined, not code)

Every Queue message body (and every Workflow parameter payload) should include a small envelope with these fields:

- **`schema_version`**
  Integer. Enables safe evolution of message formats.
- **`message_type`**
  String. A namespaced command/event name (e.g., "file.process.request"). Used for routing and DLQ triage.
- **`correlation_id`**
  Stable ID that groups a whole job/request across components (often your "job_id" from Pattern 003).
- **`causation_id`**
  The upstream thing that caused this message (an HTTP request ID, an upstream message's idempotency key, or a workflow instance/step reference). This makes debugging chains possible.
- **`idempotency_key`**
  Stable, deterministic identifier for the _logical_ work item. This is the key you use for dedupe and "effectively-once side effects."
- **`subject`**
  The primary entity reference ("file_id", "rule_id", "repo + path", etc.). Keep it small and stable.
- **`created_at`**
  ISO timestamp for audit and DLQ reasoning.
- **`payload_ref` (preferred)**
  Pointer to larger data stored elsewhere (D1 row id, R2 key, etc.). Use this when the natural payload might grow.
- **`payload_inline` (allowed, bounded)**
  Only if it stays comfortably under message size limits and won't grow unpredictably.

**Why this envelope exists:** Cloudflare explicitly recommends generating a unique ID when writing a message and using it as the primary key on inserts and/or as an idempotency key to deduplicate processing (and even reusing it as an upstream API idempotency key).

### 2) Payload rules: "pointer-first" design

- Queue message bodies must remain under **128 KB**, with batch limits and internal overhead in mind.
- If the payload can grow with dataset size (lists of items, large text, blobs), store it externally and send **only a reference**.

### 3) Ordering rules: never rely on arrival order

Even within a batch, message ordering is "best effort — not guaranteed."
If sequencing matters, the envelope must include:

- a **sequence number** or **cursor**
- a rule for how to handle gaps (wait, retry later, or skip)

---

## Idempotency model: how to get "effectively-once" outcomes

### 1) Define the idempotency boundary explicitly

For each message type, define the exact side effect that must be "effectively-once," for example:

- "Create result row for work item X"
- "Store object at key K"
- "Charge customer invoice I"
- "Emit downstream message for shard S"

Your consumer/workflow must treat repeating the message as a no-op after success.

### 2) Generate idempotency keys that survive retries and replays

**Rule:** `idempotency_key` must be stable across:

- consumer retries (`message.attempts` increases)
- producer retries (HTTP retries, workflow retries)
- manual replays (DLQ re-drive, operator action)

**Avoid using**:

- Queue's system-generated `message.id` as the idempotency key (it is unique per sent message, not per logical intent).

**Prefer**:

- deterministic keys derived from business identity:
  - `job_id + shard_id + item_id + action`
  - `repo + path + commit_sha + operation`
  - `customer_id + invoice_id + operation`

### 3) Where to store dedupe state

Common options (choose based on side effects):

- **Database uniqueness / upsert**
  Use a unique key on the record you're creating (often the idempotency key itself). If the insert conflicts, treat as "already done."
- **Inbox / processed table (recommended when failures matter)**
  Store:
  - `idempotency_key`
  - `status` (processing/succeeded/failed)
  - timestamps
  - last error
  - correlation_id, message_type, subject
  This enables:
  - safe dedupe
  - DLQ triage
  - "re-drive failed only" workflows
- **Upstream idempotency (when available)**
  If an external service supports idempotency keys (payments, email APIs), reuse your `idempotency_key` there too—Cloudflare explicitly calls this out as a pattern.

### 4) Workflows step idempotency is mandatory

Workflow steps may retry, so Cloudflare explicitly advises ensuring API/binding calls are idempotent.
Practical rule:

- Any non-idempotent call must be guarded by "already done?" checks keyed by `idempotency_key`, and/or performed against an idempotent external API.

---

## Acknowledgement and retry rules for Queue consumers

### 1) Acknowledge only after durable success

In Cloudflare Queues:

- `ack()` marks a message as successfully delivered **regardless of whether the handler returns successfully**.
  So the safe pattern is:
- do the durable work first
- then acknowledge
- if you can't confirm durable success, do **not** ack

### 2) Avoid "whole-batch replay" by explicitly acknowledging per message

Cloudflare documents that when one message in a batch fails, **the entire batch is retried** unless you explicitly acknowledge the messages that succeeded.
Therefore:

- treat each message as an independent unit of work
- ack successes immediately (after durable commit)
- retry only the failed ones

### 3) Retries, max_retries, delays

- Default retry behavior is to retry delivery three times; you can configure `max_retries` (default 3) and Cloudflare recommends leaving it at default "in most cases."
- Messages that hit max retries are deleted, or written to a DLQ if configured.
- Delays are supported on send and retry; be aware Cloudflare docs show two caps:
  - limits page: `delaySeconds` up to **24 hours**
  - batching/retries guide and JS types: delay up to **12 hours**
    A conservative rule is to keep delays within 12 hours unless you've validated 24-hour delays for your environment.

### 4) DLQ is not optional in production

- A DLQ receives messages after `max_retries` is reached; without a DLQ, those messages are permanently deleted.
  Pattern rule:
- Every production queue consumer should configure a DLQ.
- DLQ processing should be a deliberate workflow: inspect → fix → re-drive (or mark terminal).

**Retention note:** Messages delivered to a DLQ without a consumer persist for four days before deletion.
This drives how fast your ops loop must be.

### 5) Async work inside consumers must be awaited; late work must use waitUntil

Cloudflare notes:

- Don't use iteration styles that don't await async work (it can cause unfinished processing).
- For tasks that must continue after the handler completes (logging/metrics), use `waitUntil()`; unresolved promises may not complete otherwise.

---

## D1 interaction rules (idempotency + retries)

### 1) Treat D1 writes as retryable

Cloudflare recommends retrying write queries on transient errors, and D1 automatically retries read-only queries up to two more times.
Implication:

- any write your consumer/workflow performs must be idempotent (unique keys, upserts, inbox table).

### 2) Backoff strategy belongs at the edge of the system

- Use exponential backoff + jitter for D1 write retries, as Cloudflare advises.
- For upstream 429s, prefer queue retry delay rather than hot-looping the same failing call.

---

## Failure modes and mitigations

1. **Duplicate side effects (double-write / double-charge / double-email)**
   - Mitigation: stable `idempotency_key` + unique constraint/inbox table + idempotent external API keys.

2. **Whole-batch replay amplifies work**
   - Mitigation: per-message ack after success; retry only failed messages.

3. **Out-of-order processing corrupts state**
   - Mitigation: do not rely on queue ordering; include sequencing/cursors when order matters.

4. **Poison messages churn retries and disappear**
   - Mitigation: configure DLQ and alert on DLQ depth; implement validation to fail fast and route to DLQ.

5. **Dedupe TTL too short → late retries cause duplicates**
   - Mitigation: store dedupe entries for at least queue retention + expected re-drive window (and consider that retention is configurable up to 14 days).

---

## Observability and operations

Minimum fields to log/attach to traces for every processing attempt:

- `correlation_id`
- `idempotency_key`
- Queue `message.id` and `attempts`
- `message_type`, `subject`
- outcome: acked / retried / sent to DLQ
- last error class + upstream status code (if any)

Operational alerts:

- DLQ depth > 0 sustained
- high retry rate (retries cost reads; Cloudflare notes each retry counts as an additional read)
- backlog nearing retention window (risk of expiry)

---

## Anti-patterns to avoid

- **Using queue `message.id` as the idempotency key**
  It's system-generated per send, not stable for the logical action.
- **Generating a new idempotency key for every retry attempt**
  That guarantees duplicates.
- **Acking before durable work completes**
  `ack()` marks delivered even if your handler fails later.
- **Relying on message ordering**
  Ordering is best-effort only.
- **No DLQ in production**
  Without DLQ, messages at retry limit are deleted permanently.
- **Large payload messages instead of pointers**
  128 KB cap per message; 256 KB cap per sendBatch.

---

## Acceptance checklist

- Every message has `schema_version`, `message_type`, `correlation_id`, `idempotency_key`, and a stable `subject`.
- Side effects are safe under duplicates (queues) and retries (workflows).
- Consumers ack only after durable success; batch partial failures do not cause whole-batch reprocessing of already-completed work.
- DLQs are configured for all production consumers; DLQ has an operational process.
- Message payloads stay within limits and use pointers for large data.
- D1 writes are idempotent and safe under recommended retry behavior.

---

## References

- Queues delivery guarantees and Cloudflare's guidance on idempotency keys
- Queues JavaScript APIs (MessageBatch/Message, ack/retry, attempts, ordering best-effort)
- Queues batching, retries, and batch failure behavior
- Queues DLQ behavior and deletion semantics without a DLQ
- Queues platform limits (message size, batch limits, consumer duration, retention, delaySeconds)
- Workflows rules (idempotency expectation due to retries)
- Workflows limits (payload/state caps, steps cap)
- D1 retry guidance (writes should be retried; reads auto-retried)
