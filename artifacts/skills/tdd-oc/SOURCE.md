# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`1a9b283` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/tdd-cx/LICENSE`: SHA-256 `0e7ac423bf2c6e223b7c5b156f8cf72da49d748e56a1641402c31f22ad07dbb5`.
- `artifacts/skills/tdd-cx/NOTICE`: SHA-256 `6539080e889b0a3e625f6b1c7f6715b11ef2482eba19b50637594f9ca9a10748`.
- `artifacts/skills/tdd-cx/SKILL.md`: SHA-256 `50b1cc022a5cb56b17a26e8ebb60b7bfd2c68a152eacb85996793bd0b8424d49`.
- `artifacts/skills/tdd-cx/SOURCE.md`: SHA-256 `6d29fd2514b8394bc0f3c663571c95fbf3ab63f0cae08dd6df52c209511a805d`.
- `artifacts/skills/tdd-cx/mocking.md`: SHA-256 `3ceb807fdf4a47d6a93d4d9a891e5ba6d362a6247bd08adc451feebfc17361ef`.
- `artifacts/skills/tdd-cx/patronus.yaml`: SHA-256 `32d9ccbaf64d0317ecca8439e33f7a0aef3eab59fe9d31bf50dbbe426b14c1ad`.
- `artifacts/skills/tdd-cx/refactoring.md`: SHA-256 `54fced22dd1911b7094c3fe7979b7c1a40d40be307482c4adf7dc0588f27d6cc`.
- `artifacts/skills/tdd-cx/tests.md`: SHA-256 `e12182f5c4c6a844b7c6ab4b8d2623b642aaaae5460fe58ed087efd4f3d5fb8a`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; every `requires`
  entry renamed to its `-oc` twin.
- Frontmatter renamed; heading "Codex scope and evidence" became "OpenCode scope and evidence";
  "not observed native Codex behavior" became "... native OpenCode behavior"; handoff to the
  `codebase-design-cx` route renamed to `-oc`. Upstream body text otherwise unchanged.
- `mocking.md`, `refactoring.md`, `tests.md` byte-identical.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `codebase-design-oc/SKILL.md`: exact installed OpenCode companion, verify readable before use; not a stage grant.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
