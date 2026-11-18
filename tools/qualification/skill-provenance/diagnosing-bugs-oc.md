# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`1a9b283` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/diagnosing-bugs-cx/LICENSE`: SHA-256 `0e7ac423bf2c6e223b7c5b156f8cf72da49d748e56a1641402c31f22ad07dbb5`.
- `artifacts/skills/diagnosing-bugs-cx/NOTICE`: SHA-256 `b7e139dc12c93977085519aeeb0d79a2a34d265d7de9bbfd29819b17e233e15d`.
- `artifacts/skills/diagnosing-bugs-cx/SKILL.md`: SHA-256 `194cbb4270ee24fdf2acef74ceef5fafec4ea665a26d583bd3f6a207baf3f8a6`.
- `artifacts/skills/diagnosing-bugs-cx/SOURCE.md`: SHA-256 `93065724349b4a4e19482bdc1cb9bfa517e686f5eba03bc2d1ba4788c1702596`.
- `artifacts/skills/diagnosing-bugs-cx/patronus.yaml`: SHA-256 `858970068466e08509cd142e75e7970b0687ef36c126fdb7fcf9184a196a5b4e`.
- `artifacts/skills/diagnosing-bugs-cx/scripts/hitl-loop.template.sh`: SHA-256 `e91d667942d40d63e306187189f000f327c22a00aa8f00c7d69dfd40fdc141b7`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; every `requires`
  entry renamed to its `-oc` twin.
- Frontmatter renamed; heading "Codex scope and evidence" became "OpenCode scope and evidence";
  "not observed native Codex behavior" became "... native OpenCode behavior"; handoff to the
  `codebase-design-cx` route renamed to `-oc`. Upstream body text otherwise unchanged.
- `scripts/hitl-loop.template.sh` byte-identical (mode 100644 as in the sibling).

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `codebase-design-oc/SKILL.md`: exact installed OpenCode companion, verify readable before use; not a stage grant.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
