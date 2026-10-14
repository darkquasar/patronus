# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`fbb5b78a4f88f74d472993cd0f9cb25063630e63`. Disposition: `reuse-cx-body-with-rename`.

- `artifacts/instructions/diagram-explain-cx/patronus.yaml`: SHA-256 `54fe22f2846fc3ed6bc3cb7eb11dc1156ab912d074705dcb32230434794bfe35`.
- `artifacts/instructions/diagram-explain-cx/INSTRUCTIONS.md`: SHA-256 `a00a6d3532cbb0554b789edcd185ff4b9e6f5a46cb5705904ba75b123cfdd66b`.

Upstream provenance (github.com/tjboudreaux/cc-visualization-skills, MIT) is
inherited through the sibling; NOTICE and LICENSE are byte copies of the
sibling's files.

## Edits against the sibling

- Title and manifest name/description/attribution note reworded from Codex to
  OpenCode; `targets: [opencode]`.
- Example diagram label "Codex lead" became "OpenCode lead"; the box was widened
  by one column to fit and the connector lines realigned. No other content change.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Decision asks, version 1.1.0

One sentence now limits diagrams in short human decision questions to cases
where a diagram materially clarifies the choice, subject to higher-priority
instructions and the user's format. The same sentence was added to the Codex
sibling, so the reuse-with-rename relation holds; the hashes above remain the
original source. This is guidance, not runtime qualification.

## Active reads

- No sibling reads.

Mandatory runtime dependencies are named in requires. This instruction declares
none. No name substitution, placement or content test proves native loading.
