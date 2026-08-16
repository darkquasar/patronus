# Code Provenance Guide

Provenance tracks which spec drove which code changes. Instead of embedding headers in source files (which pile up, cause merge conflicts, and drift), provenance lives in a single `provenance.md` alongside the spec that drove the work.

## Where Provenance Lives

Each research-effort folder gets a `provenance.md` alongside its other artifacts:

```
docs/specs/NN-slug/
  <slug>-research.md
  <stream>-spec.md
  <stream>-plan.md
  provenance.md   ← tracks what files this stream changed
```

## provenance.md Format

```markdown
# <Stream Name> — Provenance

| File | Tickets | Change Summary |
|------|---------|----------------|
| path/to/file.ts | pat-a1b2, pat-c3d4 | Description of what changed |
| path/to/other.sql | pat-e5f6 | Description of what changed |
```

- **File**: path relative to project root
- **Tickets**: ticket ids from the tk graph (e.g. `pat-a1b2`) that drove the change
- **Change Summary**: one-line description of what was modified

## Rules

1. **One provenance.md per stream** — lives in the research-effort folder, not in source code.
2. **Do NOT add provenance headers to source files.** No `@spec`, `@plan`, `@changed` comments in code.
3. **Every created or significantly modified file must appear in the table.** Trivial changes (typo, whitespace) can be skipped.
4. **Reverse lookup**: to find all streams that touched a file, use `grep -r "filename" docs/specs/**/provenance.md`.
5. **The Team Lead writes provenance.md** during Phase 6 (merge), not teammates during implementation.

## Why Not In-Code Headers?

- Headers pile up after multiple specs touch the same file — becomes a changelog nobody reads
- Parallel teammates editing the same header block causes merge conflicts
- Headers drift when code is refactored or moved — nobody maintains them
- `git blame` already provides per-line attribution to commits
