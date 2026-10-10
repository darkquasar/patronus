# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`0e8417b` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/plan-writing-cx/LICENSE`: SHA-256 `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986`.
- `artifacts/skills/plan-writing-cx/NOTICE`: SHA-256 `8a988312b1bcf1f029805980a6c7d468097d99ce9fc8986a234f54c618c73c5b`.
- `artifacts/skills/plan-writing-cx/SKILL.md`: SHA-256 `ec17340d085ce482ec4c7b656cbfb77baeafa50072b6ea85a9bf2115121767c8`.
- `artifacts/skills/plan-writing-cx/SOURCE.md`: SHA-256 `d9bf008fc857b8f592e25548092d331cec2e8037f4bd705ab976df19f324c50c`.
- `artifacts/skills/plan-writing-cx/patronus.yaml`: SHA-256 `cbe4a9b3095f13c74e806baeb772e913f54c250042e6a2bd5e9438fff8720b17`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description reworded to start "OpenCode only", attribution note reworded from Codex to
  OpenCode and ending "Static guidance, runtime-unverified."
- Frontmatter renamed; title "Plan writing for Codex" became "... for OpenCode".
- "A read-only Codex worker returns complete draft text ... Native worker availability,
  permissions and authentication must be qualified before authorized use." became a read-only
  OpenCode subagent (the `@patronus-researcher-oc` role through the Task tool) that must be
  listed by `opencode agent list` with its read-only permission rules. The role is named as a
  role, never as a path.
- The plain-prose read "installed tdd-cx SKILL.md" became "installed tdd-oc SKILL.md";
  "Per-task Codex review" became "Per-task OpenCode review"; "Its native Codex qualification"
  became "Its native OpenCode qualification".
- Handoffs to `spec-brainstorming-cx` (SKILL.md, `scripts/README.md` and the validator
  invocation path), `plan-review-cx`, `plan-execute-cx` and `plan-execute-parallel-cx` renamed
  to `-oc`. ADR-0003 folder contract and the testing-strategy paragraph otherwise unchanged.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `plan-execute-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `plan-execute-parallel-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `plan-review-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `spec-brainstorming-oc/SKILL.md` and `spec-brainstorming-oc/scripts/README.md`: exact OpenCode companion reads, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
