# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`1a9b283` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `port-oc`.

- `artifacts/skills/dispatching-parallel-agents-cx/LICENSE`: SHA-256 `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400`.
- `artifacts/skills/dispatching-parallel-agents-cx/NOTICE`: SHA-256 `e6bed6e6e8819dc08b6c7322d4bafe1a603f91072ccb4f23d7ed388a8f7e065f`.
- `artifacts/skills/dispatching-parallel-agents-cx/SKILL.md`: SHA-256 `3af96f6d013af98edfab25a5ee045fafea0cca4a9f816106071dc6ca42d459e5`.
- `artifacts/skills/dispatching-parallel-agents-cx/SOURCE.md`: SHA-256 `267a72523403876f125e9fba3e3fed0eaf32b3bda684e08051ed89f44fe2655e`.
- `artifacts/skills/dispatching-parallel-agents-cx/patronus.yaml`: SHA-256 `30eca2e76f42e0ded7d428b9087d36b87a6169931a15d92086903d1f3a0787f5`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; every `requires`
  entry renamed to its `-oc` twin.
- Body ported (not renamed): the Codex native worker/wait primitive prose became the
  OpenCode Task tool with the `@patronus-researcher-oc` role (and `@patronus-web-researcher-oc`
  when web is authorized), admission by `opencode agent list` showing the role as a read-only
  subagent, stop when the role is absent (no built-in `general`/`explore` substitution),
  bounded fanout (start at most two), no nesting (native `subagent_depth` default 1), explicit
  installed `-oc` skill reads through the OpenCode `skill` tool, Task results read as the
  subagent's final response. Roles are named as roles, never as paths.
- Writers: "This milestone has no writer supervisor" became the one-native-writer-per-shared-cwd
  rule and the route to the `plan-execute-parallel-oc` entry (Partial).
- Budget, settlement, retry and read-only-scope paragraphs kept in substance.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `plan-execute-parallel-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
