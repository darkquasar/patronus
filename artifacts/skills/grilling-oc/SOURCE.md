# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`0e8417b` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/grilling-cx/LICENSE`: SHA-256 `0e7ac423bf2c6e223b7c5b156f8cf72da49d748e56a1641402c31f22ad07dbb5`.
- `artifacts/skills/grilling-cx/NOTICE`: SHA-256 `e2a7c8c23fb47571a159aa65089dd8665d70eeabca21cbad066c0d7f73ed0203`.
- `artifacts/skills/grilling-cx/SKILL.md`: SHA-256 `0f16387d3b2f17d16bafe28e62d879fd0282489af48e0418da5de4e58a22d1b8`.
- `artifacts/skills/grilling-cx/SOURCE.md`: SHA-256 `634d8658d42a3d42230dbb84987d24f71505ba64f0407ea895d279dc71b4f854`.
- `artifacts/skills/grilling-cx/patronus.yaml`: SHA-256 `aba8c1bb73b92ebbc32ae840c105112b898be553f9c0a34ac962fbb99c013339`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode.
- Frontmatter renamed; title "Stress-test a design in Codex" became "... in OpenCode";
  "available Codex interaction surface" became "available OpenCode interaction surface".
- Handoffs to `spec-brainstorming-cx` and `research-team-cx` renamed to `-oc`.
- ADR-0003 wording (`docs/specs/NN-slug/`, `meta.yaml`, file presence is not approval) and
  every other sentence unchanged.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `research-team-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `spec-brainstorming-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
