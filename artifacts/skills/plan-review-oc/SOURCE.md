# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`0e8417b` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/plan-review-cx/LICENSE`: SHA-256 `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986`.
- `artifacts/skills/plan-review-cx/NOTICE`: SHA-256 `0725e858af3e1f5321b05eb88d35a9196f21853b0bda489fb1f8ac23c92f5d75`.
- `artifacts/skills/plan-review-cx/SKILL.md`: SHA-256 `369d05b7c99e3aa5097013c65be8150156731b20b5ac7e66d70856dd005a2e70`.
- `artifacts/skills/plan-review-cx/SOURCE.md`: SHA-256 `accb5c8c8ab006a2288f8d01689f92ed1eb1cadc932531a54b42ebb0e5f04a61`.
- `artifacts/skills/plan-review-cx/patronus.yaml`: SHA-256 `b37c7a11f75fff55258c105194b55bc27319ec91eeefddd31bbb56e9fc86f5c9`.
- `artifacts/skills/plan-review-cx/plan-reviewer.md`: SHA-256 `aef880a9466c4d882e44e6e3bfbeab76b26d4ec48bbfbb831760c5774216e5ed`.

NOTICE and LICENSE are byte copies of the sibling's files. The sibling manifest carries no
attribution block (authored Patronus material); this manifest carries none either.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description reworded to start "OpenCode only".
- Frontmatter renamed; title "Plan review for Codex" became "... for OpenCode".
- "The lead may request a fresh-context native Codex reviewer only after capability/authentication
  qualification" became the `@patronus-plan-reviewer-oc` role through the Task tool, only after
  `opencode agent list` shows that role. The role is named as a role, never as a path.
- "native Codex readiness" became "native OpenCode readiness".
- Handoffs to `plan-writing-cx`, `spec-brainstorming-cx` (`scripts/README.md`), `plan-execute-cx`
  and `plan-execute-parallel-cx` renamed to `-oc`. ADR-0003 wording, severity rubric and review
  budget unchanged.
- `plan-reviewer.md`: title "Plan reviewer brief for Codex" became "... for OpenCode"; the
  reviewer sentence names the `@patronus-plan-reviewer-oc` role; the plain-prose read
  "installed tdd-cx SKILL.md" became "installed tdd-oc SKILL.md"; "Codex per-task review"
  became "OpenCode per-task review"; no other change.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `plan-execute-oc/SKILL.md`: exact OpenCode companion read (serial and parallel-ready tasks), verify installed/readable before use.
- `plan-writing-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `spec-writing-oc/scripts/README.md`: exact OpenCode companion read, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
