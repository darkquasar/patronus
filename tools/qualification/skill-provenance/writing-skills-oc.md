# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`1a9b283` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/writing-skills-cx/LICENSE`: SHA-256 `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400`.
- `artifacts/skills/writing-skills-cx/NOTICE`: SHA-256 `d00f4955f08a5d64bca9ba1ba9c665954de6a0a819e53abecef3e18473df63c0`.
- `artifacts/skills/writing-skills-cx/SKILL.md`: SHA-256 `89037f6cc3f0ab1b60234d3fa9ac12738694c213253c926e54fddb20f0118c67`.
- `artifacts/skills/writing-skills-cx/SOURCE.md`: SHA-256 `1bdd6b367e4b1a375b1e47c2bb17321d22abc20f5e67faaedeae0a1e5c9ed7f9`.
- `artifacts/skills/writing-skills-cx/authoring-best-practices.md`: SHA-256 `72c53f896fd9adb705e4db374447f3afa0a3ead485ebab6d3822a54b2622b9a2`.
- `artifacts/skills/writing-skills-cx/examples/AGENTS_MD_TESTING.md`: SHA-256 `3c064f5a34fb083f6b59cf9ce3c5a47319eb68349992e605bd880ab157af74b2`.
- `artifacts/skills/writing-skills-cx/graphviz-conventions.dot`: SHA-256 `5166a9152c6c3a698e13f427bb12014c7b5ba77757cc508fb0039d11f9de73af`.
- `artifacts/skills/writing-skills-cx/patronus.yaml`: SHA-256 `584eced91b5a4186189ebeb146d6a08fdb498997d4495fa7c7f8bf4bc3de15a3`.
- `artifacts/skills/writing-skills-cx/persuasion-principles.md`: SHA-256 `30116540ee1e9190934ada47edba8ff83b8541ea2b3014394e2804cbe69160a0`.
- `artifacts/skills/writing-skills-cx/testing-skills.md`: SHA-256 `229a00358d5882dfb86baaecbc1960692b94fd1cba11e7f30067ab51cf7baab9`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; every `requires`
  entry renamed to its `-oc` twin.
- Frontmatter renamed; "names ending in `-cx`" became "ending in `-oc`"; the quoted
  description template's host prefix became "OpenCode only: Use when..."; handoffs
  to the `tdd-cx` route (twice) and the `verification-before-completion-cx` route renamed to `-oc`.
- `authoring-best-practices.md`: "Codex-specific behavior" became "OpenCode-specific behavior";
  example route `tdd-cx` renamed to `tdd-oc`.
- `testing-skills.md`: route `tdd-cx` renamed to `tdd-oc`.
- `examples/AGENTS_MD_TESTING.md`: "diagnosing-bugs-cx entry" became "diagnosing-bugs-oc entry";
  the sentence naming the Codex override file (a forbidden host token for `-oc` content) became
  "Do not create a higher-priority shadow file or replace existing project instructions to run
  the audit" (OpenCode has no override file; wording follows `agents-spine-oc`).
- `persuasion-principles.md`: heading "Codex authority boundary" became "OpenCode authority
  boundary"; three occurrences of the adjective meaning "working together" (Unity principle
  bullets and the skill-type table row) became "Cooperative" because the content rules reject
  the case-sensitive six-letter prefix of the original word (it is also the name of a Codex
  feature); the Unity meaning is unchanged.
- `graphviz-conventions.dot` byte-identical.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `tdd-oc/SKILL.md`: exact installed OpenCode companion, verify readable before use; not a stage grant.
- `verification-before-completion-oc/SKILL.md`: exact installed OpenCode companion, verify readable before use; not a stage grant.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
