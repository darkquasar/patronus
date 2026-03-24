# Code Provenance Guide

Provenance tracks which spec drove which code changes. Instead of embedding headers in source files (which pile up, cause merge conflicts, and drift), provenance lives in a single `provenance.md` alongside the spec that drove the work.

## Where Provenance Lives

Each research domain directory gets a `provenance.md` alongside its other artifacts:

```
research/<domain>/<feature>/
  research.md
  spec.md
  plan.md
  tasks.md
  provenance.md   ← tracks what files this spec changed
```

## provenance.md Format

```markdown
# <Feature Name> — Provenance

| File | Tasks | Change Summary |
|------|-------|----------------|
| path/to/file.ts | A1, A2 | Description of what changed |
| path/to/other.sql | B1 | Description of what changed |
```

- **File**: path relative to project root
- **Tasks**: task IDs from `tasks.md` that drove the change
- **Change Summary**: one-line description of what was modified

## Rules

1. **One provenance.md per spec** — lives in the research domain directory, not in source code.
2. **Do NOT add provenance headers to source files.** No `@spec`, `@plan`, `@changed` comments in code.
3. **Every created or significantly modified file must appear in the table.** Trivial changes (typo, whitespace) can be skipped.
4. **Reverse lookup**: to find all specs that touched a file, use `grep -r "filename" research/**/provenance.md`.
5. **The Team Lead writes provenance.md** during Phase 6 (merge), not teammates during implementation.

## Why Not In-Code Headers?

- Headers pile up after multiple specs touch the same file — becomes a changelog nobody reads
- Parallel teammates editing the same header block causes merge conflicts
- Headers drift when code is refactored or moved — nobody maintains them
- `git blame` already provides per-line attribution to commits
