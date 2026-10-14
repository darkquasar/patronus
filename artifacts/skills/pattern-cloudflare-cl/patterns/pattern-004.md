# Pattern 004: Drizzle ORM and Hybrid Query Architecture on Cloudflare D1

## Pattern header

- **Pattern ID:** 004
- **Name:** Drizzle ORM and Hybrid Query Architecture on Cloudflare D1
- **Status:** Draft
- **Scope:** Data access layer and query-facing API design for Workers using **D1 (SQLite semantics)** with **Drizzle ORM**; includes schema normalization, indexing strategy, pagination, aggregation endpoints, and "search mode" selection.
- **Primary goal:** Provide **type-safe**, **injection-resistant**, and **predictably fast** data access that scales with dataset growth and stays within Workers/D1 operational limits.
- **Non-goals:**
  - Designing the full async pipeline for long-running AI jobs (covered by Patterns 001/003)
  - Implementing a full semantic retrieval system (vector DB, embeddings lifecycle)
  - Replacing a dedicated analytics warehouse for heavy reporting workloads
- **Key Cloudflare products involved:** Workers, D1; optionally Cache API / KV for caching; optionally R2 for large content; optionally D1 FTS5 for keyword search.
- **Primary constraints to respect (hard limits):**
  - **Max DB size:** 10 GB per database (Paid).
  - **Max row/string/BLOB size:** 2 MB.
  - **Max SQL statement length:** 100 KB.
  - **Max bound parameters per query:** 100.
  - **Max SQL query duration:** 30 seconds.
  - **Queries per Worker invocation:** 1000 (Paid).
  - **Workers memory per isolate:** 128 MB.
  - D1 is **single-threaded per database**; throughput depends on query duration; "overloaded" errors occur if too many requests queue up.

---

## Executive summary

Cloudflare D1 is a managed database with **SQLite semantics** designed to scale by **keeping individual databases small and fast** (10 GB max) and scaling horizontally across many databases when needed.

This pattern establishes:

1. A **relational-first schema rule** (normalize what you filter/join/group on).
2. A **hybrid query API surface** with separate endpoints for:
   - **Record retrieval** (paginated lists of rows)
   - **Aggregations** (dashboards/statistics, no row lists)
   - **Discovery/search** (explicit mode: simple / full-text / AI-assisted)
3. A strict approach to **safe query composition**:
   - Parameterized queries only (no string concatenation)
   - Allowlisted fields for filters/sorts/group-bys (no user-provided identifiers)

It also treats D1's **rows_read/rows_written metrics** as first-class: query performance and cost are strongly driven by whether queries avoid scans via indexes.

---

## Core architectural principle

**Never query "relational" arrays by searching JSON text.**
If you need to filter/group/join on repeated values (e.g., techniques, tags, categories), model them as **normalized lookup/join tables** with indexes.

If you store JSON, query it using D1's **JSON functions/operators** (and consider generated columns for extracted fields), but don't treat JSON blobs as your primary relational access path.

---

## Context and forces

### D1 throughput is driven by query duration (and it's single-threaded per DB)

Each D1 database processes queries **one at a time**. If your average query is slow, overall QPS drops, and concurrent traffic can produce "overloaded" errors.

### D1 operations consume Workers limits

Query execution and result serialization run under Workers CPU/memory constraints, so unbounded results can become memory problems in the Worker.

### D1 has tight "shape" limits that affect endpoint design

- `IN (...)` filters and batched inserts must respect **100 bound parameters**.
- Large dynamic SQL can hit **100 KB statement length**.
- Very large "content in a column" approaches are capped at **2 MB per row/value** and tend to be painful operationally.

### D1 supports JSON functions and FTS5, but they're tools—not schema substitutes

D1 supports a subset of SQLite extensions including **JSON** and **FTS5**.
This enables:

- JSON extraction without round-tripping through application code
- Full-text keyword search with FTS5

### Drizzle is a supported fit for D1 on Workers

Drizzle ORM explicitly supports Cloudflare D1 and Workers, enabling type-safe query building while using D1's SQLite-like semantics.

---

## Decision triggers

### Trigger 1: Is the endpoint returning rows or metrics?

- Returning **individual records** → use **Record Retrieval API** (pagination is mandatory).
- Returning **statistics only** → use **Aggregation API** (no row lists).
- Returning **discovery results** → use **Search API** (explicit mode selection).

### Trigger 2: Are you filtering on a "repeated value" field?

If the user needs queries like "find rules where technique ∈ {…}":

- **Normalize** into a join table if it's frequently queried / grouped / filtered.
- Keep JSON only for display-only, rarely filtered, or highly variable structures.

### Trigger 3: Are filters user-controlled?

If users can provide filters/sorts/group-bys:

- Only allow **predefined fields and operations** (allowlist).
- Never accept raw SQL or raw column names from the client.

### Trigger 4: Do your queries use large `IN` lists?

If the client can send large lists (IDs, tags, techniques):

- Cap list size and/or chunk internally to stay within **100 bound parameters**.

---

## Solution: the hybrid API architecture (in words)

### Pattern A: Record retrieval

**Flow:** `POST /query → rows → paginated response`

**Contract requirements**

- A hard requirement for `limit`, with a server-enforced maximum.
- Deterministic ordering (stable sort + tie-breaker).
- Bounded column selection (avoid selecting huge JSON/blob columns unless explicitly needed).

**Why**

- Prevents Worker memory spikes and unbounded serialization costs.

### Pattern B: Statistical analysis (pure aggregation)

**Flow:** `POST /query/aggregate → COUNT/SUM/GROUP BY → metrics-only response`

**Contract requirements**

- No "row list" output (only aggregated results).
- Allowlisted aggregate functions and allowlisted group-by fields.
- Optional caching layer (KV/Cache API) for dashboards where exact real-time isn't required.

**Why**

- Aggregations can scan many rows; keeping them isolated makes it easier to cache and protect your main "browse" endpoints.
- D1 billing/health is driven by **rows read**; scans are visible and measurable.

### Pattern C: Discovery (search with explicit mode)

**Flow:** `POST /search → mode-selected execution → ranked results`

**Modes**

- **Simple (structured):** exact matches / joins on normalized tables
- **Full-text:** FTS5-backed keyword search (when appropriate)
- **AI-assisted:** query interpretation or reranking (only when explicitly requested; heavy versions should move to async patterns)

**Why**

- "Auto-escalating" to expensive AI logic makes costs and latency unpredictable.
- D1 supports FTS5 for keyword discovery use cases without leaving SQL.

---

## Invariants (must always hold)

### Query safety and injection resistance

- Never build SQL with string concatenation from user input.
- Treat **field names** and **sort/group-by identifiers** as untrusted: map request fields to an internal allowlist.

### Bounded work

- Record retrieval endpoints must have mandatory pagination and a hard maximum limit.
- No endpoint may return unbounded results.

### Performance correctness

- Any field used in high-traffic `WHERE` / `JOIN` / `ORDER BY` paths must be indexed.
- Avoid N+1 patterns: D1 allows **1000 queries per Worker invocation**; chatty query patterns burn this quickly.

### Normalize what you query

- Don't "search JSON text" to simulate relational membership.
- If JSON is stored: query via JSON operators/functions (and use generated columns for common extracted fields) rather than app-level parsing for filtering.

### Retry awareness implies idempotent writes

- D1 recommends retrying write queries on transient errors and automatically retries read-only queries up to two more times for retryable errors.
  Therefore, any write path must be safe under retries (often by using an idempotency key design—owned by Pattern 002).

---

## Implementation guidance (LLM-facing, word-first)

### Schema design

Do:

- Model primary entities as tables with stable primary keys.
- For "arrays of primitives you filter on", create join tables:
  - Composite primary key on `(entity_id, value)`
  - Index on `value` for fast membership queries
  - Index on `entity_id` if you commonly join back
- Use foreign keys for integrity; D1 enforces foreign keys by default.

Avoid:

- Storing repeated relational values as JSON strings and using wildcard text search.

**Because**

- Table scans increase latency and rows_read; indexes dramatically reduce rows scanned.

### JSON fields (when they're acceptable)

Do:

- Keep complex, variable, display-only metadata as JSON.
- If you frequently filter on a JSON subfield, extract it using D1 JSON functions and consider generated columns + indexes.

Avoid:

- Treating JSON as the "primary relational model".

### Query composition (Drizzle + guardrails)

Do:

- Use Drizzle as the primary query builder for type-safe, parameterized queries on D1.
- Centralize filter building in one module that:
  - Only permits allowlisted fields
  - Only permits allowlisted operators
  - Enforces caps on list sizes (due to 100 bound parameter limit).

Avoid:

- Letting clients pass `sort_by`, `group_by`, or raw filter expressions that are interpreted as column identifiers.

### Pagination strategy

Do:

- Prefer cursor/keyset pagination for deep paging or frequently-changing datasets.
- If using offset pagination, keep maximum offsets bounded and ensure stable ordering.

Avoid:

- "Unlimited list endpoints"
- "Return everything then filter client-side"

### Aggregations and dashboards

Do:

- Keep aggregation endpoints separate and cacheable.
- Track "rows read" from query metadata and set budgets per endpoint.

Avoid:

- Mixing record lists + multiple aggregates in one response (hard to cache and hard to protect under load).

### Migrations and foreign keys (D1-specific)

Do:

- Keep schema changes as migrations (Drizzle Kit or SQL migrations).
- If a migration temporarily violates foreign keys, use D1's supported mechanism (`defer_foreign_keys`) rather than trying to disable foreign key enforcement globally.

---

## Failure modes and mitigations

1. **D1 "overloaded" errors under concurrency**
   - Mitigate by reducing query duration, adding indexes, caching hot reads, and/or splitting workloads across multiple databases when appropriate.

2. **Slow queries due to scans**
   - Use D1's per-query metadata (rows_read) and D1 analytics to spot scans and fix them with indexes/query rewrites.

3. **Bound parameter / statement size limit errors**
   - Cap user-provided list filters; chunk internally when appropriate.
   - Keep generated SQL small (100 KB statement limit; 100 bound params).

4. **Hitting Worker memory limits from large result sets**
   - Enforce pagination and column selection; offload large content to R2 and store pointers in D1.

5. **Foreign key constraint failures during schema evolution**
   - Use correct migration ordering and D1-supported constraint deferral during migrations.

---

## Observability and operations

- Log/emit D1Result metadata for key queries:
  - `rows_read`, `rows_written`, `sql_duration_ms`, and retry attempts (`total_attempts`).
- Use D1 analytics to track query volume, latency, and storage.
- Alert on:
  - overloaded errors
  - rising rows_read per request
  - queries approaching 30s duration limit

---

## Anti-patterns (explicit "don't do this")

- "JSON-as-text membership search" (e.g., wildcard searches over JSON arrays)
- Unbounded list endpoints (no limit, no pagination)
- N+1 query loops (per-row queries inside a loop)
- User-defined sort/group-by identifiers without allowlisting
- Building dynamic SQL that risks exceeding D1 limits (100 KB statement, 100 bound params)

---

## Acceptance checklist

- All list/record endpoints enforce `limit` and server max limit; no unbounded responses.
- Filters/sorts/group-bys are allowlisted; no raw SQL/identifiers from the client.
- Frequently queried fields are indexed; join tables exist for repeated filterable values.
- No N+1 patterns; total query count stays comfortably under 1000 per invocation in worst case.
- Queries respect D1 size limits (2 MB row/value, 100 KB statement, 100 bound params).
- Observability includes rows_read/rows_written and query timings.
- Write paths are safe under retries (idempotency handled via Pattern 002).

---

## References

- Cloudflare D1 Limits (size, statement, params, duration, queries/invocation)
- Cloudflare D1 Pricing + rows_read / rows_written definitions
- Cloudflare D1 JSON functions + generated columns
- Cloudflare D1 supported extensions: JSON + FTS5
- Cloudflare D1 foreign key enforcement + defer_foreign_keys
- Drizzle ORM ↔ Cloudflare D1 support
- Workers memory limit (128 MB per isolate)
