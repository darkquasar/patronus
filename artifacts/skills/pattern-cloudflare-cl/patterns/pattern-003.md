# Pattern 003: Long-Running Job Orchestrator with Cloudflare Workflows

## Executive summary

This pattern treats **Cloudflare Workflows** as the durable execution engine, while keeping **job state, progress, and results** in an external "single source of truth" (commonly D1). The workflow runs the logic; your system (DB + control-plane) manages lifecycle, observability, and user experience.

This separation matters because workflows have strong durability semantics and limits (payload/state sizes, step counts), and because operational needs (query status, audit logs, cancellation) are usually better served by a database and explicit control paths.

## Context and forces

### Workflows capabilities (relevant here)

- Workflows can persist state and run for long periods (minutes to weeks), using steps with retry and sleep mechanics.
- Workflow instances can be managed via the Workers API: **pause**, **resume**, **restart**, **terminate**.

### Workflows limits that push you toward an external "brain"

- **Max event payload:** 1 MiB
- **Max persisted state per step:** 1 MiB
- **Max steps per workflow:** 1024
- Event objects/payload should be treated as effectively immutable; state should be returned from steps or stored externally.

## Core philosophy (worded as invariants)

1. **Workflows are engines, not the system-of-record**
   - A workflow executes; it should not be the only place progress exists.

2. **The database is the brain**
   - D1 holds: status, counts, parameters, errors, timestamps, pointers to results (e.g., R2 keys).

3. **Control-plane actions are externalized**
   - Cancellation/pause/resume are initiated outside the workflow and enforced via the Workers API + durable flags.

4. **Cattle over pets**
   - A "job" may be executed by multiple workflow instances (sharded batches) for throughput and isolation.

## Recommended architecture (components and responsibilities)

### 1) Job record (the "brain")

**Responsibility:** Authoritative job state and progress that any component can read.

Suggested job fields (describe, don't prescribe SQL):

- `job_id` (stable external ID)
- `job_type`
- `status` (pending/running/paused/completed/cancelled/failed)
- `total_units`, `processed_units`, `succeeded_units`, `failed_units`
- `parameters` (small JSON config)
- `result_pointer` (R2 key / D1 table reference)
- `error_summary`
- timestamps (created/started/updated/completed)
- optional: `cancel_requested` flag + reason

**Why**

- Keeps status queryable without scraping workflow internals.
- Makes job recovery and audit feasible.

### 2) Initiator (public entry point)

**Responsibility:** Validate input, create the job record, and trigger workflow instance(s).

**Key rules**

- Return immediately with `job_id` and status endpoints.
- Decide whether to shard into multiple workflow instances based on:
  - units of work
  - payload size limits (1 MiB for workflow params)
  - expected execution time per unit

### 3) Workflow instance(s) (durable executors)

**Responsibility:** Execute business logic in retryable, idempotent steps.

**Hard rules from Cloudflare guidance**

- Steps should be granular and idempotent because they can be retried.
- Avoid side effects outside durable steps.
- Don't rely on mutated event state; treat event/payload as immutable, persist state via step returns or external storage.

**Progress reporting rule**

- After each durable unit (or small batch), write progress to the job record.

**Result handling rule**

- Store large results externally (R2/D1) and only store pointers in workflow outputs/state, due to per-step state limits.

### 4) Lifecycle manager (control plane)

**Responsibility:** Process lifecycle commands reliably (cancel, pause, resume), typically via a Queue consumer or a dedicated Worker route.

**Why queues work well here**

- Control actions must be durable and retryable.
- Queues provide retries and DLQ options for failures.

**What it does (in words)**

- On "cancel job" command:
  1. Update job record to `cancel_requested` / `cancelled`
  2. For each active workflow instance ID, call **terminate** (or **pause** if implementing resumability).
  3. Append an audit log entry

### 5) Optional: event-driven completion hooks

Cloudflare supports **event subscriptions** that can publish workflow lifecycle events (like instance completed) to a Queue, letting you trigger notifications or follow-up processing without polling.

## Cancellation and pause/resume (recommended model)

### Job-level cancellation: "flag + enforce"

- **Flag:** mark cancellation intent in the job record
- **Enforce:** terminate/pause workflow instances via the Workers API
- **Cooperate:** workflows should check the job record between steps/batches and stop creating new side effects when cancellation is requested

This avoids relying on any implied "workflow signal" concept and keeps cancellation observable and externally testable.

## Anti-patterns (what breaks durability or observability)

1. **Treating workflow status as the only truth**
   - You lose easy status queries, history, and analytics.

2. **Side effects outside durable steps**
   - Retries can double-charge, double-write, double-send. Cloudflare explicitly warns to avoid side effects outside step boundaries.

3. **Returning big results from steps**
   - Violates per-step persisted state constraints (1 MiB).

4. **Stuffing large configuration/data into workflow params**
   - Event payload max is 1 MiB; large work should be referenced by IDs/pointers.

5. **Non-deterministic step naming / non-idempotent design**
   - Makes retries confusing and can break "exactly-once effects" at the application boundary.

## Operational guidance (what to measure)

Minimum recommended signals:

- Job table: counts, last update time, terminal vs non-terminal states
- Workflow instance status for debugging (not as sole truth)
- Queue metrics: backlog depth, retry counts, DLQ depth
- Failure rate by step name (to find flaky upstreams)
- Cancellation latency: time from user cancel → job marked cancelled → workflow instances terminated

## Acceptance checklist

- Job state is queryable from D1 without inspecting workflow internals.
- Workflow steps are small, idempotent, and safe to retry.
- Payloads/results are sized to limits (1 MiB state/event).
- Cancellation works even under retries and partial failures (flag + terminate).
- The system tolerates duplicates where transport is at-least-once (dedupe keys).

---

## What I changed vs your originals (in spirit)

- Removed most code and replaced it with **roles, contracts, invariants, and decision triggers**.
- Replaced "`event.signal.aborted` in workflows" with a **documented, testable cancellation model** (DB flag + Workers API terminate/pause/resume).
- Anchored the patterns to **explicit Cloudflare limits** (Workers memory, Queues message size, Workflows step/payload/state limits).
- Elevated "at least once" and "no ordering guarantee" into first-class constraints that drive idempotency design.
