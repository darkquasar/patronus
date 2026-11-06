# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`1a9b283` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/requesting-code-review-cx/LICENSE`: SHA-256 `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400`.
- `artifacts/skills/requesting-code-review-cx/NOTICE`: SHA-256 `d93f7b6b2c60f675921c1b8312297fb4373e3657b2001935a199336797bcd18d`.
- `artifacts/skills/requesting-code-review-cx/SKILL.md`: SHA-256 `6998ce46c6dd33978ba306cdaa399f2fd67fa03f34bcb4d7ede0fa0a6072881a`.
- `artifacts/skills/requesting-code-review-cx/SOURCE.md`: SHA-256 `4f874f60466c25d3663e78ed710ac0c72945cee7826413a23b566fee2b637a52`.
- `artifacts/skills/requesting-code-review-cx/code-reviewer.md`: SHA-256 `b1230dd8d592f5e5e7ff9a4cff2ef13802a3ba2c302a1fbffb8223aef8eb2768`.
- `artifacts/skills/requesting-code-review-cx/patronus.yaml`: SHA-256 `81ed4c89432dce678d8771f0008c762b608415906afb4802eda4afbad13af914`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; every `requires`
  entry renamed to its `-oc` twin.
- Frontmatter renamed; title "Request a Codex code review" became "Request an OpenCode code review".
- Step 2: the Codex native worker primitive sentence became the OpenCode Task tool with the
  read-only `@patronus-technical-reviewer-oc` role, admitted only when `opencode agent list`
  shows that role with read-only permissions and budget is approved. The role is named as a
  role (it lands as a separate agent artifact), never as a path.
- Handoffs the `receiving-code-review-cx` route and the `verification-before-completion-cx` route
  renamed to `-oc`.
- `code-reviewer.md`: title "Codex read-only reviewer brief" became "OpenCode read-only
  reviewer brief"; no other change.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `receiving-code-review-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.
- `verification-before-completion-oc/SKILL.md`: exact OpenCode companion read, verify installed/readable before use.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
