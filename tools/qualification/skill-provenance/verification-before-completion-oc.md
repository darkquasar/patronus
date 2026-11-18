# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`1a9b283` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/verification-before-completion-cx/LICENSE`: SHA-256 `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400`.
- `artifacts/skills/verification-before-completion-cx/NOTICE`: SHA-256 `1fff198ada52e5c5178203e01e5b1c18069407b43ed8628d23915e07348d12f9`.
- `artifacts/skills/verification-before-completion-cx/SKILL.md`: SHA-256 `ff95cc969f940edd6eab68a622219fd1327c9248450395102aff45a3d880ea1e`.
- `artifacts/skills/verification-before-completion-cx/SOURCE.md`: SHA-256 `da558af7bd43d04967b10fd6df1c61a8950963f5cd5e8d322d6fa866b67a2732`.
- `artifacts/skills/verification-before-completion-cx/patronus.yaml`: SHA-256 `eef32f34e987e9bbd62a71e115b12acb9e54999f66a0331f6667fdb6b07e1403`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; every `requires`
  entry renamed to its `-oc` twin.
- Frontmatter `name`/`description` renamed; title "Evidence before completion in Codex"
  became "... in OpenCode"; "native Codex readiness" became "native OpenCode readiness".
  No other content change.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- No sibling reads.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
