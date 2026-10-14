# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`0e8417b` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `port-oc`.

- `artifacts/skills/research-team-cx/LICENSE`: SHA-256 `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986`.
- `artifacts/skills/research-team-cx/NOTICE`: SHA-256 `4aaa75257aa3ea95276a24dbf82686f7922a18a6a036bd201a3b739c2c254a88`.
- `artifacts/skills/research-team-cx/SKILL.md`: SHA-256 `401b6d2162091a03d2a2ed26d4418714f751869a944bf7f89eef86f46dda6a08`.
- `artifacts/skills/research-team-cx/SOURCE.md`: SHA-256 `fd49a256c8ee680e0571c84bb1f1eb813eb5b9ba7ec131b8039889499423c11d`.
- `artifacts/skills/research-team-cx/patronus.yaml`: SHA-256 `2c1c0b8ec3280ba1c6b2f51d56d9a996ae3f6bbdb6fa8a7f9ad3b054cab42aac`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; `requires` renamed to
  `dispatching-parallel-agents-oc`.
- Body ported (not renamed): "Codex native worker primitive actually exposed in the current
  session, with an observed read-only role" became the OpenCode Task tool with the
  `@patronus-researcher-oc` role (and `@patronus-web-researcher-oc` when web is separately
  authorized), admission by `opencode agent list` showing the role as a read-only subagent,
  stop when the role is absent (no built-in `general`/`explore` substitution). Roles are named
  as roles, never as paths.
- Step 3 names the OpenCode `skill` tool for explicit installed `-oc` reads and states native
  subagent depth 1 for the no-nesting rule.
- Step 4: "the available Codex wait mechanism" became the Task call returning the subagent's
  final response when its child session ends; settlement uses the controls the session exposes.
- "Workers" wording became "subagents"/"researchers"; the diagram names the role.
- ADR-0003 wording (`docs/specs/NN-slug/`, one research file, `meta.yaml` invariants,
  `spec: null`/`plan: null`), fanout bounds (start at most two, never more than four), budget,
  settlement and read-only paragraphs kept in substance. Handoffs to `spec-brainstorming-cx`
  and `grilling-cx` renamed to `-oc`.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `dispatching-parallel-agents-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `grilling-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `spec-writing-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
