# Source and invocation inventory

Codex adaptation authored for Patronus (2026-10-06), source revision `46247c13c3de38008c59995c5b1b697f78d3f9e5`.
The legacy and Pi siblings below are read-only design sources, not active Codex invocations.
Bodies and review briefs were rewritten for Codex stage grants, lead-owned shared outputs,
read-only worker returns and explicit capability/authentication gates. No host API was
renamed into a fictional Codex equivalent. No native runtime qualification is claimed.

ADR-0003 preserves one research effort under `docs/specs/NN-slug/`, one `meta.yaml`,
one research synthesis and one spec/plan pair per stream. Lead-owned filename metadata
is checked through parsed YAML in both directions. The spec skill adds a model-free
Go validator; other planning skills read its declared scripts/README.md through the
actual installed sibling placeholder. The pinned external YAML dependency is not vendored.

## Source bytes

- `artifacts/skills/spec-review/SKILL.md`: SHA-256 `8d386713e984878b7f7a29c923856a9e7578dc28b40a7fbff664c0077f1a4da5`.
- `artifacts/skills/spec-review/patronus.yaml`: SHA-256 `25e90427918270ea0383e8390dc4d86f2719df85e65c1493dd28e16f20844ef8`.
- `artifacts/skills/spec-review/spec-reviewer.md`: SHA-256 `3a8512773670dc76b8dc743b4afbc607324e9add3e86fac05a49aab79826f5a0`.
- `artifacts/skills/spec-review-pi/SKILL.md`: SHA-256 `dccd6dbff4785b4071fb57a5d5e26f017ffe5cfe1dba5164370a459581ec19c1`.
- `artifacts/skills/spec-review-pi/SOURCE.md`: SHA-256 `bbe414961a63938e740acc718dbecd32a9708240d3c40ec70d4fd565c4bc5b52`.
- `artifacts/skills/spec-review-pi/patronus.yaml`: SHA-256 `80a783de12431d4452f9f9f25549e9522f8aa15d8439e57cca05fde7428a90b3`.
- `artifacts/skills/spec-review-pi/spec-reviewer.md`: SHA-256 `3fbb4f1d35457e08ae84c4dd30ccfa391da31b657041bde8ae80dd743bb60267`.

## Licensing and sidecars

The authored Codex adaptation follows Patronus repository GPL-3.0 licensing; the full
license is bundled as `LICENSE`. Where upstream brainstorming/writing-plans material
is adapted, `NOTICE` retains the complete MIT license, copyright and upstream revision.
Review skills and the validator are authored Patronus material, not upstream copies.
`patronus.yaml` declares every delivered supporting file/directory. Source hashes are
provenance evidence, not dependency pins or claims about installed runtime behavior.

## Active route semantics

Read `SKILL.md` at the exact installed `{skillsDir}` sibling path when a stage is
separately granted. Read sidecars through `{skillDir}` or `{skillsDir}`; these are the
installer's real path substitutions, not a mechanism that executes Markdown text.
Codex worker capability is discovered in the current session under lead authority;
missing tools/authentication block that lane. Historical source filenames and APIs in
this provenance inventory do not register tools or authorize their invocation.
