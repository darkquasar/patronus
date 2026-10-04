# Code Provenance Guide

Provenance tracks which spec drove which code changes. Instead of embedding headers in source files (which pile up, cause merge conflicts, and drift), provenance lives in a stream-scoped `<stream>-provenance.md` alongside the spec that drove the work.

## Where Provenance Lives

Each research-effort folder gets a `<stream>-provenance.md` alongside its other artifacts:

```
docs/specs/NN-slug/
  <slug>-research.md
  <stream>-spec.md
  <stream>-plan.md
  <stream>-provenance.md   ← tracks what files this stream changed
```

## Stream provenance format

```markdown
# <Stream Name> — Provenance

| File | Workflow keys | Change Summary |
|------|---------|----------------|
| path/to/file.ts | writer-api, writer-ui | Description of what changed |
| path/to/other.sql | writer-storage | Description of what changed |
```

- **File**: path relative to project root
- **Workflow keys**: stable keys and exact run/mission receipt references that drove the change
- **Change Summary**: one-line description of what was modified

## Rules

1. **One `<stream>-provenance.md` per stream** — lives in the research-effort folder, not in source code.
2. **Do NOT add provenance headers to source files.** No `@spec`, `@plan`, `@changed` comments in code.
3. **Every created or significantly modified file must appear in the table.** Trivial changes (typo, whitespace) can be skipped.
4. **Reverse lookup**: to find all streams that touched a file, search the project's stream-provenance files for its relative path.
5. **Only the authorized coordinator writes shared provenance**, after consuming the exact lane handoffs. Leaves return proposed rows; integration and metadata writes require their own grants.

## Why Not In-Code Headers?

- Headers pile up after multiple specs touch the same file — becomes a changelog nobody reads
- Parallel teammates editing the same header block causes merge conflicts
- Headers drift when code is refactored or moved — nobody maintains them
- `git blame` already provides per-line attribution to commits
