# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`1a9b283` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/receiving-code-review-cx/LICENSE`: SHA-256 `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400`.
- `artifacts/skills/receiving-code-review-cx/NOTICE`: SHA-256 `fafea4e4aad995ee48c159c0136f63a09f71cd6c55691234aaa44f4171cff7ff`.
- `artifacts/skills/receiving-code-review-cx/SKILL.md`: SHA-256 `4cc165120d8cee652944fdba386baed9cefaa3372a80542eb8e004171c56d072`.
- `artifacts/skills/receiving-code-review-cx/SOURCE.md`: SHA-256 `e7256ac75a590d47e3a227ea40b3ec04ee90329d535199f691612e03bbcc6ebd`.
- `artifacts/skills/receiving-code-review-cx/patronus.yaml`: SHA-256 `3faed935f902550bff002b75eb4809feacbc69cb93b6868a6fda7119c9f89cf3`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; every `requires`
  entry renamed to its `-oc` twin.
- Frontmatter renamed; title "Receive code review in Codex" became "... in OpenCode";
  "Per-task Codex review" became "Per-task OpenCode review"; the plain-prose read
  "installed tdd-cx SKILL.md" became "installed tdd-oc SKILL.md" (a name mention, not a
  `{skillsDir}` route; `tdd-oc` is not a declared requirement, as in the sibling).
- Handoff the `verification-before-completion-cx` route renamed to `-oc`.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `verification-before-completion-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
