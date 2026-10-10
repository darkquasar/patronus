# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`1a9b283` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/codebase-design-cx/DEEPENING.md`: SHA-256 `be2f0defa739610128e7404ec72b0d8855c19fb6a5842dc26d099da76bf62144`.
- `artifacts/skills/codebase-design-cx/DESIGN-IT-TWICE.md`: SHA-256 `45334c1e8cb514f956e01fe2601e6f77e3871f798b4d50f3a0b1f3f94559dcb8`.
- `artifacts/skills/codebase-design-cx/LICENSE`: SHA-256 `0e7ac423bf2c6e223b7c5b156f8cf72da49d748e56a1641402c31f22ad07dbb5`.
- `artifacts/skills/codebase-design-cx/NOTICE`: SHA-256 `385ce09311133284122d5d131f4d7a9c7f4634df4f6c3712e06c212c260222ef`.
- `artifacts/skills/codebase-design-cx/SKILL.md`: SHA-256 `15b8f6f1912934dd7d60e44cc248c4ca9f1a7af3fb4785499ab8dcb8a2cb1be3`.
- `artifacts/skills/codebase-design-cx/SOURCE.md`: SHA-256 `b0fa1b40c21af3363d2a509d7a21f56e0ddbe85e04295aee26e738fd6f6d1c42`.
- `artifacts/skills/codebase-design-cx/patronus.yaml`: SHA-256 `78526d87331a818e6d67c99e12230053a56dac57a57e75cd19fc515fca3671a3`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; every `requires`
  entry renamed to its `-oc` twin.
- Frontmatter renamed; heading "Codex scope and evidence" became "OpenCode scope and evidence";
  "not observed native Codex behavior" became "... native OpenCode behavior". Upstream body
  text (including its original em-dashes and box-drawing diagrams) unchanged.
- `DESIGN-IT-TWICE.md`: handoff to the `dispatching-parallel-agents-cx` route renamed to `-oc`;
  no other change. `DEEPENING.md` byte-identical.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `dispatching-parallel-agents-oc/SKILL.md`: exact installed OpenCode companion, verify readable before use; not a stage grant.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
