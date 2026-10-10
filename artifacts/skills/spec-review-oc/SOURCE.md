# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`0e8417b` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/spec-review-cx/LICENSE`: SHA-256 `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986`.
- `artifacts/skills/spec-review-cx/NOTICE`: SHA-256 `f4915643f9805a2fe39a6b0fb1e7e5efbfb39bc7f5737963ad0f59585e6f0ab4`.
- `artifacts/skills/spec-review-cx/SKILL.md`: SHA-256 `80684045698b0f6dfc04da49252407c654a7c4a3d4f556434cd18c4df2657db3`.
- `artifacts/skills/spec-review-cx/SOURCE.md`: SHA-256 `adc2cf33135f90928b3c827db67f01ac61fca6eb431f75ffb8dc859c525a6daf`.
- `artifacts/skills/spec-review-cx/patronus.yaml`: SHA-256 `ca3c238e29881d1a77e995c480e80898f0b2f7000655be992c5848639ae07523`.
- `artifacts/skills/spec-review-cx/spec-reviewer.md`: SHA-256 `58537e801e3258efecfc068fac40b3d5127b618238a83db125c3f60d35cf3e4e`.

NOTICE and LICENSE are byte copies of the sibling's files. The sibling manifest carries no
attribution block (authored Patronus material); this manifest carries none either.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description reworded to start "OpenCode only".
- Frontmatter renamed; title "Spec review for Codex" became "... for OpenCode".
- "The lead requests a fresh-context read-only Codex reviewer only when native worker capability
  and authentication are present, qualified and authorized." became the
  `@patronus-technical-reviewer-oc` role in spec mode through the Task tool, only when
  `opencode agent list` shows that role and authentication is present. The role is named as a
  role, never as a path.
- "not a native Codex readiness claim" became "not a native OpenCode readiness claim".
- Handoffs to `spec-brainstorming-cx` (SKILL.md and `scripts/README.md`) and `plan-writing-cx`
  renamed to `-oc`. ADR-0003 wording, severity rubric and review budget unchanged.
- `spec-reviewer.md`: title "Spec reviewer brief for Codex" became "... for OpenCode"; the
  reviewer sentence names the `@patronus-technical-reviewer-oc` role in spec mode; no other change.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `plan-writing-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `spec-brainstorming-oc/SKILL.md` and `spec-brainstorming-oc/scripts/README.md`: exact OpenCode companion reads, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
