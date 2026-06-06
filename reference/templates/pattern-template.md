# Pattern NNN: <Pattern Name>

## Pattern header
- **Pattern ID:** NNN
- **Name:** Clear, concrete, platform-scoped (e.g., “Queue-Decoupled Streaming Pipeline on Workers”)
- **Status:** Draft | Adopted | Deprecated
- **Scope:** What parts of the system this governs (e.g., “API handlers, Queues, Workflows, D1”)
- **Primary goal:** One sentence (e.g., “Process large datasets without exceeding Workers memory limits.”)
- **Non-goals:** What this pattern is *not* trying to solve
- **Key Cloudflare products involved:** Workers, Queues, Workflows, D1, R2, etc.
- **Primary constraints to respect:** Memory, CPU, payload size, delivery semantics, retention, ordering, timeouts
- **Related patterns**
  - **Requires:** Pattern 002 (Message Contract & Idempotency) for any async boundary (Queues/Workflows), unless explicitly stated otherwise
  - **See also:** Pattern 001 (Pipeline Topology), Pattern 003 (Job Orchestrator), Pattern 004 (Query Architecture) *(edit as appropriate)*

---

## Executive summary (5–8 lines)
A short “why it exists” summary:
- The core problem
- The key invariant(s)
- The recommended shape of the solution (in words and arrows)
- The main trade-off(s)

---

## Context and forces
Describe the pressures that shape the design (no solutions yet):
- **Platform limits that matter** (include numeric limits + links)
- **Data shape** (volume, size, fan-out, ordering needs)
- **Latency vs throughput** (interactive vs batch)
- **Correctness model** (at-least-once, idempotency, replay, partial failure)
- **Operational needs** (observability, cancellation, retries, DLQ)

---

## Decision triggers
A compact “if this, then that” section:
- If **X** is true → choose **Approach A**
- If **Y** is true → choose **Approach B**
- Explicit “don’t use this pattern if …”

---

## Solution
### Architecture in words
Use:
- **Component roles** (what each piece is responsible for)
- **Boundaries** (what each piece must *not* do)
- **Interfaces/contracts** (message fields, IDs, dedupe keys — described as bullet lists)

Represent flows as arrows, not code:
- Example: `HTTP endpoint → Queue → Consumer → Workflow`

---

## Invariants (must always hold)
These are the “hard rules” that the LLM must preserve when generating code:
- **Memory bounds:** What must never happen (e.g., no dataset-sized arrays)
- **Idempotency:** What must be safe to repeat (e.g., “at-least-once safe”)
- **Payload constraints:** What must be small / referenced indirectly
- **Side-effect boundaries:** Where side effects may occur (and where they may *not*)
- **Async boundary invariant (Pattern 002):**
  - Any transition across `HTTP → Queue`, `Queue → Consumer`, `Consumer → Workflow`, `Workflow → Queue`, or `Workflow step → side effect` MUST follow Pattern 002’s envelope and idempotency-key rules (unless this pattern explicitly documents an exception)

---

## Implementation guidance (LLM-facing)
Write as **Do / Avoid / Because**:
- ✅ Do …
- ❌ Avoid …
- **Because** …

When you must include “code”, use **non-executable pseudocode**:
- No real imports, no full functions, no ready-to-run config blocks
- Use placeholders like `<<DB_QUERY>>`, `<<QUEUE_SEND>>`, `<<WORKFLOW_STEP>>`

---

## Failure modes and mitigations
List the predictable ways this breaks in production:
- “Backlog grows until retention expires”
- “Duplicates cause double-writes”
- “Large payload causes OOM”
- “Workflow step returns too much state”
- …and how to mitigate each

---

## Observability and operations
- **Metrics:** queue backlog, retry counts, DLQ depth, workflow status, DB rows read/written, latency
- **Logs must contain:** correlation IDs, job IDs, message IDs, idempotency keys, attempt counts, outcomes (acked/retried/DLQ)
- **Alerts:** what thresholds matter and why

---

## Anti-patterns to avoid
Write these as recognizable smells:
- “Consumer fetches the full dataset”
- “Workflow returns large blobs”
- “Endpoint blocks on long work”
- “Assumes message order”
- “Ad-hoc messages without idempotency keys”

---

## Acceptance checklist
A short list the LLM (or reviewer) can validate:
- [ ] Memory stays **O(1)** with respect to dataset size
- [ ] All side effects are idempotent or deduped (safe under retries/replays)
- [ ] Message sizes stay under platform limits; large payloads are referenced indirectly
- [ ] Cancellation is possible and observable (where applicable)
- [ ] Retries won’t corrupt state
- [ ] **Pattern 002 compliance:** all async boundaries use the standard envelope + idempotency key rules (unless explicitly exempted)

---

## References
Prefer official Cloudflare docs, then reputable examples/blogs. Include only what’s load-bearing.
