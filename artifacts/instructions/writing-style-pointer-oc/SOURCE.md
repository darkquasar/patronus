# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`fbb5b78a4f88f74d472993cd0f9cb25063630e63`. Disposition: `reuse-cx-body-with-rename`.

- `artifacts/instructions/writing-style-pointer-cx/patronus.yaml`: SHA-256 `9d815a05505e69148340788adeaea3a734959f9206830fd0831cc6a343b218b8`.
- `artifacts/instructions/writing-style-pointer-cx/INSTRUCTIONS.md`: SHA-256 `f5dd00a70eab5dad373ad97543028fdace0c5adaf3140bd4ce0ea75b350af444`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Title and manifest name/description/attribution note reworded from Codex to
  OpenCode; `targets: [opencode]`.
- Body audited: apart from the title it names no host, host path or host
  command, so the remaining prose is unchanged.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Decision asks, version 1.1.0

A new paragraph routes genuine human choice, missing-fact and permission asks to
the exact `decision-clarity-oc` skill through the OpenCode `skill` tool or its
resolved file. It is OpenCode-specific prose, not a rename of the Codex
sibling's route. This is guidance, not runtime qualification.

## Active reads

- `decision-clarity-oc/SKILL.md`: exact OpenCode companion read added in version 1.1.0,
  only for a genuine human choice, missing-fact or permission ask; verify
  installed/readable before use.

Mandatory runtime dependencies are named in requires. This instruction declares
none. No name substitution, placement or content test proves native loading.
