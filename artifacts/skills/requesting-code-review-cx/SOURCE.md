# Source and invocation inventory

Codex adaptation at source revision `46247c13c3de38008c59995c5b1b697f78d3f9e5`.

- `artifacts/skills/requesting-code-review/patronus.yaml`: SHA-256 `de2fa3abce884238ec60b13fa9a1feed095d6287ee02458e27d0ae28961c80d0`.
- `artifacts/skills/requesting-code-review/SKILL.md`: SHA-256 `1bd1e79d051fd4c34705c5fb617550cd9e7b818b3b5a291e7eefd0d4a0746124`.
- `artifacts/skills/requesting-code-review/code-reviewer.md`: SHA-256 `b2f2ec7596925fe52dac158fdfbca19b3a7d779d619c481e6706a6c0001662d3`.

Bodies are behaviorally authored for Codex, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `receiving-code-review-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.
- `verification-before-completion-cx/SKILL.md`: exact Codex companion read, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-cx owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
