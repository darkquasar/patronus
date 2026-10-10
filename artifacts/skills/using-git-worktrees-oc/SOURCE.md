# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`1a9b283` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/using-git-worktrees-cx/LICENSE`: SHA-256 `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400`.
- `artifacts/skills/using-git-worktrees-cx/NOTICE`: SHA-256 `bc44969cd46f0331359bb0ecb68bd6b26e7acfb11437c832dd2e1207ad4e6c7a`.
- `artifacts/skills/using-git-worktrees-cx/SKILL.md`: SHA-256 `cacf640ba4690c2b7c81a1192780d38631d42fa073b72edff223a75124879e8a`.
- `artifacts/skills/using-git-worktrees-cx/SOURCE.md`: SHA-256 `b2c77881767b5425213a2384414e6bc34eeade33e6023fb64379b282708dab80`.
- `artifacts/skills/using-git-worktrees-cx/patronus.yaml`: SHA-256 `54cfa2b2344d56dc133f0b1b30fc27fe264fad9893b271c399f79e01c2375274`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; every `requires`
  entry renamed to its `-oc` twin.
- Frontmatter renamed. Body text unchanged except the two handoffs
  the `plan-execute-cx` route and the `finishing-a-development-branch-cx` route
  renamed to `-oc`.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `plan-execute-oc/SKILL.md`: optional next-stage companion; verify installed/readable before use, not a stage grant or mandatory dependency.
- `finishing-a-development-branch-oc/SKILL.md`: optional completion companion; verify installed/readable before use, not a stage grant or mandatory dependency.

Optional routes do not form a mandatory circular installation dependency.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
