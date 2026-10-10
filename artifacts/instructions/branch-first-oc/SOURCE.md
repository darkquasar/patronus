# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`fbb5b78a4f88f74d472993cd0f9cb25063630e63`. Disposition: `reuse-cx-body-with-rename`.

- `artifacts/instructions/branch-first-cx/patronus.yaml`: SHA-256 `fc9c377ef37613052bc1848fe4f1bdd14c263ba62e6f38c6ba764e7a9df99223`.
- `artifacts/instructions/branch-first-cx/INSTRUCTIONS.md`: SHA-256 `838e22d6fb3a7f1b2c486108072460f4621a950e9ecadfc5e26187d3892ee8e9`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest name/description/attribution note reworded from Codex to OpenCode;
  `targets: [opencode]`.
- Body audited: it names no host, no host path and no host command, so the
  INSTRUCTIONS.md bytes are identical to the sibling's.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- No sibling reads.

Mandatory runtime dependencies are named in requires. This instruction declares
none. No name substitution, placement or content test proves native loading.
