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

- `artifacts/skills/plan-writing/NOTICE`: SHA-256 `5e9088bbbd0b94716da0b68c344d836d3b5a37953ac5eaf45deba5289a50b73e`.
- `artifacts/skills/plan-writing/SKILL.md`: SHA-256 `94902aeb91ff77ff740aca76a5d1d6125e656f7e0886c1d50fa9579cd4c0a965`.
- `artifacts/skills/plan-writing/patronus.yaml`: SHA-256 `556902203488f8cfc405ce92c3e348d2d2930b6ec69aa6f784911e4efbb3e2cd`.
- `artifacts/skills/plan-writing-pi/NOTICE`: SHA-256 `5e9088bbbd0b94716da0b68c344d836d3b5a37953ac5eaf45deba5289a50b73e`.
- `artifacts/skills/plan-writing-pi/SKILL.md`: SHA-256 `136aa1db560bcdc3c1eb6b08d1b1648c53b867af9b6f95e46244879d8c3a55aa`.
- `artifacts/skills/plan-writing-pi/SOURCE.md`: SHA-256 `6e2d997d3fbcddb315b73b6723d931fad128e7980fd3e0ea882468f953576fa1`.
- `artifacts/skills/plan-writing-pi/patronus.yaml`: SHA-256 `44a550985ed3f9379fcec69729b5596eb31780866858128b2ec996ce6f256b08`.

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
