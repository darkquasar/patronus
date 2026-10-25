# Source and invocation inventory

Codex adaptation at source revision `46247c13c3de38008c59995c5b1b697f78d3f9e5`.

- `artifacts/skills/plan-execute/patronus.yaml`: SHA-256 `6945b45fb390fedbc2f1451aba90ea7d00ad3fb56852e7d0161806016cbaf7d7`.
- `artifacts/skills/plan-execute/SKILL.md`: SHA-256 `7cd82fcea28f20b3e3221a1450a67104b6e8580a4ae999803772770771021eac`.

Bodies are behaviorally authored for Codex, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Parallel-ready section (version 1.4.0)

The "Parallel-ready tasks" section and the outer serial/parallel-ready choice fold
in the retired `plan-execute-parallel-cx/SKILL.md` (lines 18-116, SHA-256
`d8d9471a3a9a33ace6df5270e8b5d414bd21d2c7bf94be021119b376e966ea82`)
as Patronus-authored routing.
Its allocator, allocation reference and fixture move to `using-git-worktrees`;
its scout route moves to `dispatching-parallel-subagents`.
The serial arm's mode assessment is unchanged.

## Active reads

- `dispatching-parallel-subagents/SKILL.md`: shared dispatcher, parallel-ready arm only; verify installed/readable before use.
- `using-git-worktrees/SKILL.md`: shared isolated-root preparation, parallel-ready arm only; verify installed/readable before use.
- `isolated-writer-supervisor/SKILL.md`: optional explicit-user CLI mode only, not read for native execution.
- `receiving-code-review-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `requesting-code-review-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `verification-before-completion-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-cx owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
