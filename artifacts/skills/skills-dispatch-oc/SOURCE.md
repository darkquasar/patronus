# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`0e8417b` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `port-oc`.

- `artifacts/skills/skills-dispatch-cx/LICENSE`: SHA-256 `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400`.
- `artifacts/skills/skills-dispatch-cx/NOTICE`: SHA-256 `5c0420787dde2acff3ac4c0ae097582e4d689c6960d12daaaec046edd8bc1658`.
- `artifacts/skills/skills-dispatch-cx/SKILL.md`: SHA-256 `7cad047c7445632b15758ed8474f535f0b56bcb1b787fd8a3c1d6a7548c025be`.
- `artifacts/skills/skills-dispatch-cx/SOURCE.md`: SHA-256 `e7c52d1c2f4d1ee3d3bb421a4ebc2fc53d02932589f0f262a516f823870ae743`.
- `artifacts/skills/skills-dispatch-cx/patronus.yaml`: SHA-256 `9eeb7a93c3be64318c1894d77d9a70133c4bcbb82e19ae73fbb28549cf794917`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode.
- Body ported (not renamed): the inventory read names the OpenCode `skill` tool for
  explicit installed reads; a new paragraph states that OpenCode reads four project skill
  roots (`.opencode/skills`, `.opencode/skill`, `.claude/skills`, `.agents/skills`) plus the
  global roots, so other-host siblings may be co-visible and the exact `-oc` name is the one
  to load (surfaces section 3, probe 1).
- Routing table: the ten destinations renamed to their `-oc` twins (`research-team-oc`,
  `spec-brainstorming-oc`, `grilling-oc`, `spec-review-oc`, `plan-writing-oc`,
  `plan-review-oc`, `plan-execute-oc`, `plan-execute-parallel-oc`,
  `requesting-code-review-oc`, `verification-before-completion-oc`); the parallel helper note
  says native OpenCode workers remain Partial/runtime-pending.
- Plain-prose reads `tdd-cx`, `spec-review-cx` and `requesting-code-review-cx` became the
  `-oc` names; "Codex worker" wording became OpenCode subagent wording, with native depth 1
  for the no-delegation rule.
- Final paragraph: delegation goes through the OpenCode Task tool to the installed
  `@patronus-<role>-oc` roles (named as roles, never as paths) only after
  `opencode agent list` shows the role; absent roles stop the lane. "no SessionStart hook"
  became "no startup hook".

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Local correction, version 1.0.1

Corrected the dispatcher helper reference to the existing distributed
`isolated-writer-supervisor/scripts/supervisor.py`, matching the parallel skill's
route. No helper is added or moved; native workers remain Partial/runtime-pending.
A focused content regression checks the concrete Python route's file existence
and manifest distribution. This does not qualify native loading or execution.

## Active reads

- `grilling-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `plan-execute-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `plan-execute-parallel-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `plan-review-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `plan-writing-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `requesting-code-review-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `research-team-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `spec-brainstorming-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `spec-review-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `verification-before-completion-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
