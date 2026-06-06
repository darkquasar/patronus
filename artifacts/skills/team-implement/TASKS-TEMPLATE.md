# tasks.md Template

When `tasks.md` does not already exist in the research directory, the Team Lead creates it using this structure.

```markdown
# <Domain Name> — Implementation Tasks

**Source**: spec.md, plan.md (this directory)
**Created**: <date>
**Status**: Not started

---

## Concern: <boundary-name> (e.g., "Storage Layer", "API Routes", "UI Components")

### Tasks

- [ ] `<task-id>` — <clear description of what to build>
  - Files: <expected files to create or modify>
  - Acceptance: <how to verify this is done>
  - Refs: <which section of spec.md or plan.md defines this>

- [ ] `<task-id>` — <next task>
  ...

## Concern: <next-boundary-name>

### Tasks

- [ ] ...
```

## Task ID conventions

- Use short IDs like `A1`, `B3`, `C2` — letter = concern boundary, number = sequence
- Each task must have:
  - Clear description of WHAT to build (not HOW)
  - Expected files to create or modify
  - Acceptance criteria (testable/verifiable)
  - Reference back to the spec/plan section that defines it
