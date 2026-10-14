# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`0e8417b` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/spec-brainstorming-cx/LICENSE`: SHA-256 `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986`.
- `artifacts/skills/spec-brainstorming-cx/NOTICE`: SHA-256 `97c2c63c99dde5250f1c17d6fc9f4bbfecc17e4237ec3bfa44007d4fe6a930db`.
- `artifacts/skills/spec-brainstorming-cx/SKILL.md`: SHA-256 `485587fd4f3be9fde050caf1e87647fb5eac0ad7355df7ca9d01aba8c3e69b6f`.
- `artifacts/skills/spec-brainstorming-cx/SOURCE.md`: SHA-256 `e9bd41d62edc8b728a463cf5ed9e11e18ee5783c0623909af3f3733db46f9e04`.
- `artifacts/skills/spec-brainstorming-cx/patronus.yaml`: SHA-256 `05da46e6aa8dcf99d8e6d41a502edb26ad96553473b8108d24be932a1aa0e43c`.
- `artifacts/skills/spec-brainstorming-cx/scripts/README.md`: SHA-256 `ae4fe72f76d8457a6c119bd8b75e74a5531ecf52203bbf55c4d79108d9b1a31e`.
- `artifacts/skills/spec-brainstorming-cx/scripts/go.mod`: SHA-256 `90fe08aa25fa674322bd86994a84373c473dd7f21851524c42b1f21a560ff5ea`.
- `artifacts/skills/spec-brainstorming-cx/scripts/go.sum`: SHA-256 `3f8f6a8ac1048ed9b6cb66a4e2795ff956d52fd447239c67eed83943fa91ac35`.
- `artifacts/skills/spec-brainstorming-cx/scripts/main.go`: SHA-256 `4ec7c6a23a4abc03dfe12271dd068c19ff6415e149f6b5bf64d23225a97b26d0`.
- `artifacts/skills/spec-brainstorming-cx/spec-document-reviewer-prompt.md`: SHA-256 `c3bdddba71c8b7ff345077d35f3abd813e93dc3e1d9c348999b629d9f23d7f04`.

NOTICE and LICENSE are byte copies of the sibling's files. The four files under
`scripts/` (`README.md`, `go.mod`, `go.sum`, `main.go`) are byte copies as well.

## Validator audit

`scripts/main.go` was read in full: it imports only `fmt`, `os`, `path/filepath`, `strings`
and `gopkg.in/yaml.v3`, reads `meta.yaml` and the folder listing, and prints a diagnostic.
It makes no process execution and no call to any host binary or model; nothing was changed.
`scripts/README.md` names the repository Go test that exercises the command interface on
invented fixtures; that name is provenance, not a host invocation, and the file is unchanged.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description reworded to start "OpenCode only", attribution note reworded from Codex to
  OpenCode and ending "Static guidance, runtime-unverified."; `files` unchanged.
- Frontmatter renamed; title "Spec brainstorming for Codex" became "... for OpenCode";
  "available Codex interaction surface" became "available OpenCode interaction surface".
- "Use native Codex workers only if present, qualified and separately authorized. Missing
  capability or authentication blocks that lane." became native OpenCode subagents (the
  read-only `@patronus-researcher-oc` role through the Task tool) only if
  `opencode agent list` shows the role; missing role or authentication blocks that lane.
  The role is named as a role, never as a path.
- "not native Codex qualification" became "not native OpenCode qualification".
- Handoffs to `research-team-cx`, `grilling-cx`, `spec-review-cx` and `plan-writing-cx`
  renamed to `-oc`. ADR-0003 folder contract, the `meta.yaml` example and the validator
  invocation are unchanged.
- `spec-document-reviewer-prompt.md`: the `spec-review-cx` route renamed to `-oc`; "fresh-context
  read-only Codex worker" became a fresh-context read-only OpenCode subagent (the
  `@patronus-technical-reviewer-oc` role in spec mode); no other change.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `grilling-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `plan-writing-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `research-team-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `spec-review-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
