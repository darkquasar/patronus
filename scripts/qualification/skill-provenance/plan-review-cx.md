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

- `artifacts/skills/plan-review/SKILL.md`: SHA-256 `b60c88b441ea15a5a59991975b349f16bfb05b85276fddfbb863bf540e73ae95`.
- `artifacts/skills/plan-review/patronus.yaml`: SHA-256 `6e8931a95698b40e42d5aab81baf5e3b6f4faa30bfd39805754ce317399b49d6`.
- `artifacts/skills/plan-review/plan-reviewer.md`: SHA-256 `8c94dfc363d99cb4f16490e72c8064d2554a759a0155049fa2d749dbbca0fe88`.
- `artifacts/skills/plan-review-pi/SKILL.md`: SHA-256 `364151137c85c9a518d355f31666444d93092f0233f67ca86af1cf5394db3e01`.
- `artifacts/skills/plan-review-pi/SOURCE.md`: SHA-256 `567ab6940bf25724f98ce7690fc01d4e20013697e4d323e7bb36ab3e58eb807e`.
- `artifacts/skills/plan-review-pi/patronus.yaml`: SHA-256 `e68ed7333327a39e4ee165f1357df6e8fb53d70bb078a298366e238f0a7bc0b6`.
- `artifacts/skills/plan-review-pi/plan-reviewer.md`: SHA-256 `a074afe61641492531cadf16d12160ad5a8acf3f7a8ff0bc01771fb572ee0acd`.

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
