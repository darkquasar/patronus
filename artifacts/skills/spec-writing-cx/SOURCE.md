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

- `artifacts/skills/spec-brainstorming/NOTICE`: SHA-256 `62f56f819c36b9316809693d2b3a3e480c4b9a2ad6c3bdbfb6cb09a306823804`.
- `artifacts/skills/spec-brainstorming/SKILL.md`: SHA-256 `b06c2b310a669c15f47d3b9b15b9224c150980a0581d7f52dcc152e1f00d2011`.
- `artifacts/skills/spec-brainstorming/patronus.yaml`: SHA-256 `e85cfc00774e4dcf940cff3431a0013ec8eb2e5dc88f98d43aab1396aa16591d`.
- `artifacts/skills/spec-brainstorming/spec-document-reviewer-prompt.md`: SHA-256 `fa8e0e3ee02b76dab8e3aa8bf455c975b2cc136abbc94d2353fae0f6dc1c2c99`.
- `artifacts/skills/spec-brainstorming-pi/NOTICE`: SHA-256 `62f56f819c36b9316809693d2b3a3e480c4b9a2ad6c3bdbfb6cb09a306823804`.
- `artifacts/skills/spec-brainstorming-pi/SKILL.md`: SHA-256 `4eec8e179a99e123e669cfc7b72226520813898c2480759c0aaf2cc290d3fef8`.
- `artifacts/skills/spec-brainstorming-pi/SOURCE.md`: SHA-256 `cfd1aa7af4535c838d939339dd8a5fb159cf8736ea138c410f80e40d884a0d1e`.
- `artifacts/skills/spec-brainstorming-pi/patronus.yaml`: SHA-256 `4676c234d7c5fa6d22705075b1b583ea5449714be0c7bffdb02f29cb87895d36`.
- `artifacts/skills/spec-brainstorming-pi/spec-document-reviewer-prompt.md`: SHA-256 `c9cb172d8999ab46498d9ceafaed1ec58e57398c151c7ad18c7901d259f38463`.

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
