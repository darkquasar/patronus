# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`1a9b283` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `port-oc`.

- `artifacts/skills/pattern-mcp-cx/LICENSE`: SHA-256 `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986`.
- `artifacts/skills/pattern-mcp-cx/NOTICE`: SHA-256 `086b1d102b26715f0cb18e43a072c78f56f4593edf683c5f825d33188f3c91da`.
- `artifacts/skills/pattern-mcp-cx/SKILL.md`: SHA-256 `b21fb936df1aef8720fc352cb063b58b4aa44893e44e139c24b026e4e24bb3a6`.
- `artifacts/skills/pattern-mcp-cx/SOURCE.md`: SHA-256 `688217e73fc29d93e52cbd0ba4248bdc4709479a25875f00e19eaad2b501da8b`.
- `artifacts/skills/pattern-mcp-cx/patronus.yaml`: SHA-256 `bdaf39f3efa390d9c3d872085ce8eb712637d7ce44447fe0e38fa1dab099300a`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; every `requires`
  entry renamed to its `-oc` twin.
- Body ported from the Codex MCP facts to the OpenCode ones (research surfaces section 7,
  CLI help 1.18.35): `opencode.json` `mcp.<name>` with `type: local` (`command: []`, `cwd`,
  `environment`, `enabled`, `timeout`) and `type: remote` (`url`, `headers`, `oauth`, `enabled`,
  `timeout`); global and project file locations and merge precedence including
  `OPENCODE_CONFIG_CONTENT`; `opencode mcp add|list|auth|logout|debug`; MCP tools exposed as
  `<server>_<tool>` and gated by `permission`/legacy `tools` rules; OAuth tokens in
  `~/.local/share/opencode/mcp-auth.json` never read or copied; Patronus writes structured
  transport leaves only (`type` plus `url` or `command`) and refuses commented files; one inert
  invented remote example with a placeholder header value (no credential, no token shape).
- Trust, query ownership and shared-service paragraphs kept in substance with OpenCode wording
  ("Patronus does not write OpenCode permission or auth internals").

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- No sibling reads.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
