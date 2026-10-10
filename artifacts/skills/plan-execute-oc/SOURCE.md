# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`1a9b283` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/plan-execute-cx/LICENSE`: SHA-256 `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400`.
- `artifacts/skills/plan-execute-cx/NOTICE`: SHA-256 `dff8c5e636e85125e71f86c093746600fb852eda78f64d6ebc2701726751e17f`.
- `artifacts/skills/plan-execute-cx/SKILL.md`: SHA-256 `ea4e4d85fc38b7c0d701f7ccfaab72330fe78366324f38209a2a3e47e8105ff0`.
- `artifacts/skills/plan-execute-cx/SOURCE.md`: SHA-256 `11ac5606cca58d3440b672f5fa1115ab8957018284cf34b03dd94aa0f2feaa8c`.
- `artifacts/skills/plan-execute-cx/patronus.yaml`: SHA-256 `d00678b26250de54d18b71a606f7f07ac68372cb1288564fb5aa8c3cce6c8eb4`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; every `requires`
  entry renamed to its `-oc` twin.
- Frontmatter renamed; title "Execute an approved plan in Codex" became "... in OpenCode".
- "Per-task reviewers are read-only Codex workers, available only after current-session
  primitive/role qualification and budget admission" became read-only OpenCode subagents
  (`@patronus-technical-reviewer-oc` through the Task tool), available only after the role
  is listed by `opencode agent list` and budget admission.
- The plain-prose read "installed tdd-cx SKILL.md" became "installed tdd-oc SKILL.md".
- Handoffs to `verification-before-completion-cx`, `requesting-code-review-cx` and
  `receiving-code-review-cx` renamed to `-oc`.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `receiving-code-review-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `requesting-code-review-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `verification-before-completion-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
