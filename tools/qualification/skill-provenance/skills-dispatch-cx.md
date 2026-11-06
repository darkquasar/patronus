# Source and invocation inventory

Codex adaptation at source revision `46247c13c3de38008c59995c5b1b697f78d3f9e5`.

- `artifacts/skills/skills-dispatch/patronus.yaml`: SHA-256 `1ee2d2e08cf81851898ee2deedfe5f45968fd422f9b9eda39c280cfc0d3019aa`.
- `artifacts/skills/skills-dispatch/SKILL.md`: SHA-256 `a468bc6ce9bf4f8b5fb804e2eb2241fe65a7cca94626bc9a54c8a730c0ad0e1b`.

Bodies are behaviorally authored for Codex, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Consumer cutover, version 1.3.0

Claude workflow siblings now use exact `-cl` names. The dispatcher distinguishes
them from Codex `-cx` routes without substituting a foreign sibling. The native
parallel route remains unchanged; this naming edit is not runtime qualification.

## Execution route, version 1.6.0

Serial and parallel-ready execution now share one route,
`plan-execute-cx/SKILL.md`. The separate `plan-execute-parallel-cx` route
listed by earlier versions is retired; its parallel guidance folded into
`plan-execute-cx`. This is routing guidance, not runtime qualification.

## Decision asks, version 1.7.0

One route names `decision-clarity-cx` for genuine coordinator asks to the human:
an unresolved choice, material missing fact or required permission. It is a
profile companion read, not a requires edge, approval gate or per-message load.

## Active reads

- `decision-clarity-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `grilling-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `plan-execute-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `plan-review-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `plan-writing-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `requesting-code-review-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `research-team-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `spec-writing-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `spec-review-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `verification-before-completion-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-cx owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
