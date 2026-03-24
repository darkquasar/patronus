# Deliverable Templates

The Team Lead synthesizes three deliverables from the raw `*-findings.md` files produced by researchers.

---

## Deliverable 1: `research.md`

The consolidated research document.

```markdown
# <Domain Name> — Research

**Date**: <date>
**Status**: Complete
**Authors**: Team <team-name> (AI-assisted research)

## Problem Statement

<what we set out to understand>

## Scope

**In scope**: <what was investigated>
**Out of scope**: <what was explicitly excluded>

## Key Findings

### <Finding Area 1>

<synthesized findings from relevant streams, with evidence>

### <Finding Area 2>

<synthesized findings>

...

## Constraints & Hard Limits

<all discovered constraints, consolidated and deduplicated>

## Trade-off Analysis

<comparison of approaches where multiple options exist, with recommendations>

## Open Questions

<unresolved items that need future investigation or user decisions>

## Appendix: Stream Findings

- [<stream-1>-findings.md](<relative-path>) — <one-line summary>
- [<stream-2>-findings.md](<relative-path>) — <one-line summary>
...
```

---

## Deliverable 2: `spec.md`

The technical specification derived from the research.

```markdown
# <Domain Name> — Specification

**Date**: <date>
**Status**: Draft
**Source**: research.md (this directory)

## Overview

<1-2 paragraph description of what will be built and why>

## Goals

1. <goal 1>
2. <goal 2>
...

## Non-Goals

- <what this does NOT do>

## Architecture

<how the solution fits into the existing system — diagrams, data flow, component relationships>

## Detailed Design

### <Component/Module 1>

<interface contracts, data schemas, behavior specifications>

### <Component/Module 2>

...

## Dependencies

- <what this depends on — existing code, external services, CF features>

## Constraints

- <hard limits from research that shape the design>

## Testing Strategy

- <how correctness will be verified>

## Migration / Rollout

- <if applicable, how to transition from current state to new state>

## Future Considerations

- <things deferred but worth noting for later phases>
```

---

## Deliverable 3: `plan.md`

The implementation plan derived from the spec.

```markdown
# <Domain Name> — Implementation Plan

**Date**: <date>
**Status**: Draft
**Source**: spec.md (this directory)

## Prerequisites

- <what must be true before implementation starts>

## Phase Breakdown

### Phase 1: <name>

**Goal**: <what this phase achieves>
**Estimated scope**: <small / medium / large>

- <high-level work item 1>
- <high-level work item 2>
...

**Verification**: <how to confirm this phase is complete>

### Phase 2: <name>

...

## Concern Boundaries

<how the work naturally splits into parallel streams for /team-implement>

| Boundary | Owns | Produces | Consumes |
|----------|------|----------|----------|
| <name>   | <files/modules> | <outputs> | <inputs from others> |
| ...      | ...  | ...      | ...      |

## Risk & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| ...  | ...    | ...        |

## Deploy Order

<if relevant, the order in which changes should be deployed>

## Definition of Done

- [ ] <criterion 1>
- [ ] <criterion 2>
...
```
