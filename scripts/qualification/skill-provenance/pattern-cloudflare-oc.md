# Source and invocation inventory

OpenCode adaptation of the Codex sibling at source revision
`1a9b283` (branch `feat/opencode-core-lane`, stream base `fbb5b78a4f88f74d472993cd0f9cb25063630e63`).
Disposition: `reuse-cx-body-with-rename`.

- `artifacts/skills/pattern-cloudflare-cx/LICENSE`: SHA-256 `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986`.
- `artifacts/skills/pattern-cloudflare-cx/NOTICE`: SHA-256 `5f078bf899b80172036faed7cc1bd01ddd42c8c3e3a487516845027d19b79a58`.
- `artifacts/skills/pattern-cloudflare-cx/SKILL.md`: SHA-256 `926993983542d17d9bcad87dc7d5f9734955c5bd550d9ba662a6104991f14cdf`.
- `artifacts/skills/pattern-cloudflare-cx/SOURCE.md`: SHA-256 `5b449a4ecfc510a2b2fa55d6212aa570445444babbf3f7e7ccc10c5504f821bb`.
- `artifacts/skills/pattern-cloudflare-cx/patronus.yaml`: SHA-256 `fa1cf465a7485d7d9936a8c11d7530ac84fc8ce94c1e0141696ace27574d9941`.
- `artifacts/skills/pattern-cloudflare-cx/patterns/pattern-001.md`: SHA-256 `9c7d77815e6f2ba623b904c79034a1629e30bd736b607fefb61e47191f3efb3c`.
- `artifacts/skills/pattern-cloudflare-cx/patterns/pattern-002.md`: SHA-256 `21d324ff467bd3e98985a7170f8f3f96ff481208bab449097db258790c9ffb83`.
- `artifacts/skills/pattern-cloudflare-cx/patterns/pattern-003.md`: SHA-256 `6176727f5319434ad6a111dcc05195820704e4d47a518aba48202796ca7537d3`.
- `artifacts/skills/pattern-cloudflare-cx/patterns/pattern-004.md`: SHA-256 `8f922221a4698b2cdd6cca49220f4b191f949df4576aa6b42c809f6cbf04af4a`.
- `artifacts/skills/pattern-cloudflare-cx/patterns/pattern-005.md`: SHA-256 `f2e5b38b2d6478e4e0efa026bb21c5267a9454d10aca4245ed1ec847cdcc9650`.
- `artifacts/skills/pattern-cloudflare-cx/patterns/pattern-006.md`: SHA-256 `72edec8cf8aa269f8132c2ab19ebf3e0cc6830d8012eb82d3c8017391d3d1d2d`.
- `artifacts/skills/pattern-cloudflare-cx/patterns/pattern-007.md`: SHA-256 `208807047bf06cb42b01e6cb5f08bc94c3b06785ec2fe88ab8e70a833ae5bbf2`.

NOTICE and LICENSE are byte copies of the sibling's files.

## Edits against the sibling

- Manifest: `name` suffixed `-oc`, `version: 1.0.0` (new artifact), `targets: [opencode]`,
  description and attribution note reworded from Codex to OpenCode; every `requires`
  entry renamed to its `-oc` twin.
- Frontmatter renamed; heading "Codex local use and handoff" became "OpenCode local use and
  handoff"; "not observed native Codex or Cloudflare behavior" became "... native OpenCode or
  Cloudflare behavior"; the four handoffs (`codebase-design`, `diagnosing-bugs`, `tdd`,
  `pattern-mcp`) renamed to `-oc`. Index body otherwise unchanged.
- `patterns/pattern-001.md` .. `pattern-006.md`: the same two header sentences reworded
  ("Codex local use and handoff", "native Codex or Cloudflare behavior"); pattern bodies
  unchanged.
- `patterns/pattern-007.md`: the same header sentences, its two handoffs (`diagnosing-bugs`,
  `pattern-mcp`) renamed to `-oc`, and "No container API is assumed to exist on Codex" became
  "... on OpenCode".

Bodies are behaviorally authored for OpenCode, not renamed host calls. Source
host names are provenance only. No legacy scripts/templates or runtime APIs
are inherited. Only entry and manifest-declared sidecars are distributed.

## Active reads

- `codebase-design-oc/SKILL.md`: exact installed OpenCode companion, verify readable before use; not a stage grant.
- `diagnosing-bugs-oc/SKILL.md`: exact installed OpenCode companion, verify readable before use; not a stage grant.
- `tdd-oc/SKILL.md`: exact installed OpenCode companion, verify readable before use; not a stage grant.
- `pattern-mcp-oc/SKILL.md`: exact installed OpenCode companion, verify readable before use; not a stage grant.

Mandatory runtime dependencies are named in requires. Offered next-stage and
profile companion reads above are exact routes, not aliases or stage grants.
The core-profile-oc owner must select those companions; absence blocks that
route. No name substitution, placement or content test proves native loading.
