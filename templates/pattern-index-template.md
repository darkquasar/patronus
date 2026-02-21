# <Project/Domain> Pattern Index

> **Purpose:** Load this file first. Use it to decide which full pattern file(s) to fetch for the task at hand. Each entry provides enough context to reason about applicability without reading the full document.
>
> **Pattern files location:** `<path>/pattern-NNN.md`

---

## Quick routing table

<!-- One row per pattern. Keep the "Load when..." column keyword-rich so an LLM can match tasks to patterns in a single scan. -->

| Pattern | Name | Load when the task involves... |
|---------|------|-------------------------------|
| 001 | <Name> | <comma-separated keywords/phrases> |
| 002 | <Name> | <comma-separated keywords/phrases> |
| ... | ... | ... |

---

## Dependency graph

<!-- Show which patterns must be co-loaded. Use ASCII arrows. Follow with a plain-English rule of thumb. -->

```text
001 (<short name>) ──requires──▶ 002 (<short name>)
003 (<short name>) ──uses──▶ 001, 002
...
```

**Rule of thumb:** <1-2 sentences describing the most common co-loading combinations>

---

## Platform limits cheat sheet

<!-- Consolidate all hard numeric limits that recur across patterns. An LLM can reference this table without loading any full pattern file. Only include limits that are load-bearing for design decisions. -->

| Resource | Limit |
|----------|-------|
| <resource> | <value + unit> |
| ... | ... |

---

## Pattern summaries

<!-- For each pattern, include exactly these sections. Keep each summary under ~15 lines. -->

### Pattern NNN — <Full Name>

**Core invariant:** <The one rule that must never be violated — one sentence.>

**What it solves:** <The problem this pattern addresses — one sentence.>

**Key structural elements:**
<!-- The main architectural components, shapes, or layers — as a compact list or short paragraph. This is the "shape of the solution" compressed. -->
- <element 1>
- <element 2>
- ...

**Critical rules:** <2-4 of the most important rules from the full pattern, condensed to one line each.>

---

<!-- Repeat the summary block for each pattern -->

---

## Decision triggers (condensed)

<!-- Map common task descriptions to the pattern(s) an LLM should load. Use natural-language phrases a user or LLM would actually say. -->

- **"<task description>"** → NNN + NNN
- **"<task description>"** → NNN
- ...
