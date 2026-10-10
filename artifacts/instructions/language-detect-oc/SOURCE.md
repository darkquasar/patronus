# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`fbb5b78a4f88f74d472993cd0f9cb25063630e63`. Disposition: `port-oc`.

- `artifacts/instructions/language-detect-cx/patronus.yaml`: SHA-256 `5f7682a603df58e345f224e77bd155b6933fe2d6fb035a05a47e58c48b98f77f`.
- `artifacts/instructions/language-detect-cx/INSTRUCTIONS.md`: SHA-256 `74caa8d1027d159a18b5c625ef19e12a4621237a9b232a57b5edfe1212db6ab2`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Title and manifest name/description/attribution note reworded from Codex to
  OpenCode; `targets: [opencode]`.
- Closing sentence rewritten as the companion for the omitted session-start
  hook: the instruction is advisory and read by the model at session start
  through the project instructions; OpenCode has no Patronus session-start hook
  and no automatic skill loader stands behind it.
- Detection scope and authority prose kept unchanged.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- No sibling reads.

Mandatory runtime dependencies are named in requires. This instruction declares
none. No name substitution, placement or content test proves native loading.
