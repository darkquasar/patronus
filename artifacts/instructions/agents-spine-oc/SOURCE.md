# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`fbb5b78a4f88f74d472993cd0f9cb25063630e63`. Disposition: `port-oc`.

- `artifacts/instructions/agents-spine-cx/patronus.yaml`: SHA-256 `1d204ea3c313b10b3fcb4e7d5c3fb348b7a0ca29cd8636d076ac36883747f31d`.
- `artifacts/instructions/agents-spine-cx/INSTRUCTIONS.md`: SHA-256 `50ba0f6b566c7ef728ab0c781102d33735fd501c6fd58f80296d18fc07c740e1`.

Upstream provenance (github.com/openai/agents.md, MIT) is inherited through the
sibling; NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Title and manifest name/description/attribution note reworded from Codex to
  OpenCode; `targets: [opencode]`.
- Discovery paragraph rewritten from the OpenCode rules and skills documentation
  and the sandbox probe: root `AGENTS.md` found walking up to the git root, global
  `AGENTS.md` under the OpenCode config directory (`OPENCODE_CONFIG_DIR`, else
  `XDG_CONFIG_HOME/opencode`, else `~/.config/opencode`), the `instructions: []`
  glob array in `opencode.json`, co-visible `.claude/skills` (including nested
  `synced/` trees) and `.agents/skills` roots, and `.opencode/AGENTS.md` not read.
  The Codex-specific user-instruction root and override-file precedence sentences
  were removed.
- "There is no Codex mission API assumed" became "There is no OpenCode mission
  API assumed".
- Added the lessons sentence: "Record and read durable lessons in
  `patronus/LESSONS.md` when that file exists; only the authorized lead edits it".
- "Patronus does not write Codex trust internals" became "Patronus does not write
  OpenCode permission or auth internals".
- Stage, ownership, delegation, parallelism and acceptance prose kept unchanged.

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- No sibling reads.

Mandatory runtime dependencies are named in requires. This instruction declares
none. No name substitution, placement or content test proves native loading.
