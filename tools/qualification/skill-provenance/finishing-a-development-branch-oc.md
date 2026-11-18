# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`1a9b283` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/finishing-a-development-branch-cx/LICENSE`: SHA-256 `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400`.
- `artifacts/skills/finishing-a-development-branch-cx/NOTICE`: SHA-256 `82bd63d7dfb67857a31281a0ccffd511a6e346c6e12ea2848945239f282b2761`.
- `artifacts/skills/finishing-a-development-branch-cx/SKILL.md`: SHA-256 `963056d2defcd6f669ff1dc5a41698aef2e78f0119b22458fc05dc81c941ab69`.
- `artifacts/skills/finishing-a-development-branch-cx/SOURCE.md`: SHA-256 `cf87f0f5b8d94a9feaf655c8efdbf91d00ddc4838c5feccf0e0676e9ce38faf0`.
- `artifacts/skills/finishing-a-development-branch-cx/patronus.yaml`: SHA-256 `4c8aa23a069cb7ad20f44053f8691a6b4a2130b845f57f70963402a883fb63b3`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; every `requires`
  entry renamed to its `-oc` twin.
- Frontmatter renamed. Body text unchanged except the three handoffs
  the `verification-before-completion-cx` route, the `requesting-code-review-cx` route
  and the `using-git-worktrees-cx` route renamed to `-oc`. The separate integration,
  publication and cleanup grants are kept as written.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Shared worktree route, version 1.1.1

The cleanup route and `requires` entry now name the shared
`using-git-worktrees` skill, which replaced the retired
`using-git-worktrees-oc` bundle. Pointer rename only; integration,
publication and cleanup grants are unchanged.

## Active reads

- `using-git-worktrees/SKILL.md`: exact installed shared companion, verify readable before use; not a stage grant.
- `verification-before-completion-oc/SKILL.md`: exact installed OpenCode companion, verify readable before use; not a stage grant.
- `requesting-code-review-oc/SKILL.md`: exact installed OpenCode companion, verify readable before use; not a stage grant.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
