# OpenCode core static reuse and authored-body audit

Source identity: `cf83372db7ca3d796785cbfab51f44f37d1273b7`.
This is a static content/provenance record, not a native acceptance receipt.
`reuse-audit.yaml` hashes every distributed file of the three reused skills and
the whole three recipe files. The shape follows the Codex audit; actual bytes
were read rather than inferred from multi-target metadata. None of the six items
is changed. Drift needs a fresh body/sidecar/dependency/handoff audit, not a hash
refresh alone.

## Six unchanged items

| Identity | Entire surface | Rationale and limits |
|---|---|---|
| domain-modeling 1.0.1 | patronus.yaml, SKILL.md, ADR-FORMAT.md, CONTEXT-FORMAT.md, NOTICE | Host-neutral glossary/ADR files, local links only; no requires or lane handoff. mattpocock/skills commit 6eeb81b5fcfeeb5bd531dd47ab2f9f2bbea27461, MIT. Source Git grants still govern writes/commits. |
| ddd-distilled 1.0.0 | patronus.yaml, SKILL.md, NOTICE | Host-neutral decision rules; no sidecars/requires/worker APIs. ciembor/agent-rules-books commit 9c8763613514e4047d75c089533e09bc4b493c28, MIT. LLM-generated/inspired-by, not the book; no legal-review claim. |
| refactoring-distilled 1.0.0 | patronus.yaml, SKILL.md, NOTICE | Host-neutral change rules; no workflow handoff. Same ciembor commit and caveat above, not an authoritative reproduction of the book. |
| context7 1.0.0 | recipes/context7.yaml | Wire-only hosted HTTP, opencode declared, no package execution or secret. upstash/context7 provenance. Native initialization, optional owner-managed key and rate limits pending. |
| github 1.0.0 | recipes/github.yaml | Wire-only hosted HTTP, opencode declared; github/github-mcp-server provenance. No credential copying; owner-managed auth/native MCP pending. |
| gitleaks 1.0.0 | recipes/gitleaks.yaml | Target-agnostic fetch-only CLI, six OS/arch assets pinned to v8.30.1; gitleaks/gitleaks provenance. Pins read, not downloaded/revalidated. Binary presence is not native commit-guard protection. |

All six remain Partial/runtime-pending for native loading or effects in the
ledger. Source neutrality is not behavioral equivalence. The neutral skills may
also be visible in `.claude/skills` or `.agents/skills`. Identical same-name
bytes are warned with both paths, but OpenCode requires unique native names
across locations; duplicate visibility remains a caveat. Different bytes refuse
admission. No unsuffixed workflow sibling is selected.

## Authored-body dispositions, per file

This binds the existing delivered OC-T8..T12/T5/T6 files to their recorded
per-file authoring methods, not a new port or independent behavioral audit.
Each skill/instruction SOURCE.md records sibling revisions/hashes, exact edits,
unchanged sidecars and active reads. Historical contributor source paths are
provenance, not live installed roots. Agents were ported from Pi roles with
OpenCode permissions and skill reads; OC-CD1 gives spec/plan authors edit: allow.
Settings are original spec/plan data. Guard scripts intentionally copy their
Codex sibling bytes, hash-bound in SOURCE.md. Retained NOTICE wording is upstream
provenance, not a host call; four B3 NOTICEs still describe their Codex origin.

The table hashes manifest, entry, expanded declared sidecars and provenance.
Recorded method describes the adaptation; each SOURCE.md supplies per-file
exceptions and exact edits, including byte-identical sidecars. The ledger uses
only port-oc/reuse-unsuffixed/omit, not reuse-cx-body-with-rename as a disposition.
Every native behavior remains pending. A hash or source method is not runtime
proof.

| Source file | Recorded authored-body method (see SOURCE.md for exact edits) | SHA256 |
|---|---|---|
| `artifacts/skills/codebase-design-oc/DEEPENING.md` | reuse-cx-body-with-rename | `be2f0defa739610128e7404ec72b0d8855c19fb6a5842dc26d099da76bf62144` |
| `artifacts/skills/codebase-design-oc/DESIGN-IT-TWICE.md` | reuse-cx-body-with-rename | `fb7151c183c4159219c20b2b2198e9ed73e230de77fa1760e52e133305a89c38` |
| `artifacts/skills/codebase-design-oc/LICENSE` | reuse-cx-body-with-rename | `0e7ac423bf2c6e223b7c5b156f8cf72da49d748e56a1641402c31f22ad07dbb5` |
| `artifacts/skills/codebase-design-oc/NOTICE` | reuse-cx-body-with-rename | `385ce09311133284122d5d131f4d7a9c7f4634df4f6c3712e06c212c260222ef` |
| `artifacts/skills/codebase-design-oc/SKILL.md` | reuse-cx-body-with-rename | `ecf5db98e1cba1b831dc4dde62eb9cba4cbe7e55b41cb6f0704e9e136ca0478a` |
| `artifacts/skills/codebase-design-oc/SOURCE.md` | reuse-cx-body-with-rename | `fc778c82b4df3a76541438bfd9ad98451de316127befa52621b79f1bce1bba5a` |
| `artifacts/skills/codebase-design-oc/patronus.yaml` | reuse-cx-body-with-rename | `4462de3bf1727df169f58e513f6ec571228bdc7596299fd276d96c418ae7f9e6` |
| `artifacts/skills/diagnosing-bugs-oc/LICENSE` | reuse-cx-body-with-rename | `0e7ac423bf2c6e223b7c5b156f8cf72da49d748e56a1641402c31f22ad07dbb5` |
| `artifacts/skills/diagnosing-bugs-oc/NOTICE` | reuse-cx-body-with-rename | `b7e139dc12c93977085519aeeb0d79a2a34d265d7de9bbfd29819b17e233e15d` |
| `artifacts/skills/diagnosing-bugs-oc/SKILL.md` | reuse-cx-body-with-rename | `33c1710428aea9166d0a6eee5049f863dc1c16585c0914f9c5ddf722f0601a9b` |
| `artifacts/skills/diagnosing-bugs-oc/SOURCE.md` | reuse-cx-body-with-rename | `24df9b7ba4d3a2827523f1a90a3952e742779b4f50f3946b83d04527b53d24ce` |
| `artifacts/skills/diagnosing-bugs-oc/patronus.yaml` | reuse-cx-body-with-rename | `6cdbf4a25fd92fa3f83d5bb765a1f6f364704ba5f7124677874c82453145a343` |
| `artifacts/skills/diagnosing-bugs-oc/scripts/hitl-loop.template.sh` | reuse-cx-body-with-rename | `e91d667942d40d63e306187189f000f327c22a00aa8f00c7d69dfd40fdc141b7` |
| `artifacts/skills/dispatching-parallel-agents-oc/LICENSE` | port-oc | `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400` |
| `artifacts/skills/dispatching-parallel-agents-oc/NOTICE` | port-oc | `e6bed6e6e8819dc08b6c7322d4bafe1a603f91072ccb4f23d7ed388a8f7e065f` |
| `artifacts/skills/dispatching-parallel-agents-oc/SKILL.md` | port-oc | `7f73365c9cbd3bc50dcfb919284e9fd16e49b378ab5b7a736ac9c74e8500b224` |
| `artifacts/skills/dispatching-parallel-agents-oc/SOURCE.md` | port-oc | `4740802a334465a01c8556868f8faa4778a7374008705d167fe5e3c48f617aeb` |
| `artifacts/skills/dispatching-parallel-agents-oc/patronus.yaml` | port-oc | `e8ee6657fd6a546f82bcaadfdcfe19a786a9985c677fb1607fc9f1adb2c049ba` |
| `artifacts/skills/finishing-a-development-branch-oc/LICENSE` | reuse-cx-body-with-rename | `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400` |
| `artifacts/skills/finishing-a-development-branch-oc/NOTICE` | reuse-cx-body-with-rename | `82bd63d7dfb67857a31281a0ccffd511a6e346c6e12ea2848945239f282b2761` |
| `artifacts/skills/finishing-a-development-branch-oc/SKILL.md` | reuse-cx-body-with-rename | `a074bbd8dbb63f5d67c08a368a108b1abbf3066cee75e007783b61c537b62d32` |
| `artifacts/skills/finishing-a-development-branch-oc/SOURCE.md` | reuse-cx-body-with-rename | `27d0edbf914a2a9db171b6f093881cb87aaa9791e3cee358bf35f8be82107510` |
| `artifacts/skills/finishing-a-development-branch-oc/patronus.yaml` | reuse-cx-body-with-rename | `7564e50437f86e8e87276744201b9bf782c43487d7ea9e34655db303e38fc880` |
| `artifacts/skills/grilling-oc/LICENSE` | reuse-cx-body-with-rename | `0e7ac423bf2c6e223b7c5b156f8cf72da49d748e56a1641402c31f22ad07dbb5` |
| `artifacts/skills/grilling-oc/NOTICE` | reuse-cx-body-with-rename | `e2a7c8c23fb47571a159aa65089dd8665d70eeabca21cbad066c0d7f73ed0203` |
| `artifacts/skills/grilling-oc/SKILL.md` | reuse-cx-body-with-rename | `d17dfad8485214e28d78b68950a9a0613f732b066324cd382eb599e2ff4bbc81` |
| `artifacts/skills/grilling-oc/SOURCE.md` | reuse-cx-body-with-rename | `d34cc2c69596e8de8b2e52f63bc74c57fb33579d5fcffc5d172cf7a86c21bd78` |
| `artifacts/skills/grilling-oc/patronus.yaml` | reuse-cx-body-with-rename | `44f977c5fce828238389ec2207298bb5907943c7c7a454fd8ea108923a96017b` |
| `artifacts/skills/pattern-cloudflare-oc/LICENSE` | reuse-cx-body-with-rename | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |
| `artifacts/skills/pattern-cloudflare-oc/NOTICE` | reuse-cx-body-with-rename | `5f078bf899b80172036faed7cc1bd01ddd42c8c3e3a487516845027d19b79a58` |
| `artifacts/skills/pattern-cloudflare-oc/SKILL.md` | reuse-cx-body-with-rename | `7dbffb0679eb1113d7936a170209dd04101954049252140cf0d7073b1e3e5c78` |
| `artifacts/skills/pattern-cloudflare-oc/SOURCE.md` | reuse-cx-body-with-rename | `8789463fd25b81a2c3290f712b657df9e9b5fc1635ec789afc1592b5b78ec517` |
| `artifacts/skills/pattern-cloudflare-oc/patronus.yaml` | reuse-cx-body-with-rename | `4346bba3f95dec6566f77fcbcbc138fdd2a5d8ba3acfc39a054e7581a5dc7fb1` |
| `artifacts/skills/pattern-cloudflare-oc/patterns/pattern-001.md` | reuse-cx-body-with-rename | `d9ce04376657f61b1511112882a4125b8671e156062b69d7c9f8c0f7db5d6b9c` |
| `artifacts/skills/pattern-cloudflare-oc/patterns/pattern-002.md` | reuse-cx-body-with-rename | `c83770a56f6efbc3f304dae15838e87393e9f6e1d53318ca7d9db821b44cea82` |
| `artifacts/skills/pattern-cloudflare-oc/patterns/pattern-003.md` | reuse-cx-body-with-rename | `6ab188fb7bebcec17f3ac934241f257383bcc70d2d2590d48e2562362691ee3e` |
| `artifacts/skills/pattern-cloudflare-oc/patterns/pattern-004.md` | reuse-cx-body-with-rename | `7bdfea454921b934f23acf4e9729bdd76d1b89d58d3973304a56997b09f59830` |
| `artifacts/skills/pattern-cloudflare-oc/patterns/pattern-005.md` | reuse-cx-body-with-rename | `8eb1f16a7c536c9e405438f349e10bd81a57d89f83c8e574507d45bc71482377` |
| `artifacts/skills/pattern-cloudflare-oc/patterns/pattern-006.md` | reuse-cx-body-with-rename | `d554702f650d250cf1a9799dc88f4e2a1f76f75544b6360184d7f9bafdc1ddc1` |
| `artifacts/skills/pattern-cloudflare-oc/patterns/pattern-007.md` | reuse-cx-body-with-rename | `3d0d8263b2ca07f2d72a5a02fcfbfd6911526a28423dac5a92be179bf96913fe` |
| `artifacts/skills/pattern-mcp-oc/LICENSE` | port-oc | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |
| `artifacts/skills/pattern-mcp-oc/NOTICE` | port-oc | `086b1d102b26715f0cb18e43a072c78f56f4593edf683c5f825d33188f3c91da` |
| `artifacts/skills/pattern-mcp-oc/SKILL.md` | port-oc | `b6b89878a65fbdf16aaad789ca41a4fb8bd9ee8563f01590a25252b12da00eda` |
| `artifacts/skills/pattern-mcp-oc/SOURCE.md` | port-oc | `d1383e791c4dbb751b93331bac87a4515e35b7e6cf18e9f7698b531b158f1d56` |
| `artifacts/skills/pattern-mcp-oc/patronus.yaml` | port-oc | `a910a872d3a21ef0e53fec2ea23f65312d4f36670c7d1c17eb955b9a97f9d31f` |
| `artifacts/skills/plan-execute-oc/LICENSE` | reuse-cx-body-with-rename | `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400` |
| `artifacts/skills/plan-execute-oc/NOTICE` | reuse-cx-body-with-rename | `dff8c5e636e85125e71f86c093746600fb852eda78f64d6ebc2701726751e17f` |
| `artifacts/skills/plan-execute-oc/SKILL.md` | reuse-cx-body-with-rename | `209eff8ff1b0c2cd0428361f55957e308b29a3be1bb58cdb7f89dbd112536cd4` |
| `artifacts/skills/plan-execute-oc/SOURCE.md` | reuse-cx-body-with-rename | `7525ebed3186128b2b5bb236dc7185a9a9cb75a87326a31233c04c9d81e21b2f` |
| `artifacts/skills/plan-execute-oc/patronus.yaml` | reuse-cx-body-with-rename | `74d0a65c2db66b79c7d783c9ffb2e2e76faca4a4a61b45a5130ccee32481f441` |
| `artifacts/skills/plan-execute-parallel-oc/LICENSE` | port-oc | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |
| `artifacts/skills/plan-execute-parallel-oc/NOTICE` | port-oc | `f06979caadf3af300ba27c87dee139bedad4f0a86caf2cd5b37884ef1b955714` |
| `artifacts/skills/plan-execute-parallel-oc/SKILL.md` | port-oc | `85c53efad49f2c4ca834310d4e8228c885710d4d82cdbd5b333cf942690793c7` |
| `artifacts/skills/plan-execute-parallel-oc/SOURCE.md` | port-oc | `a41c8ccc1e78d68723894d83220b5f0973a9624bdc10f9cdf96b1df6b014b894` |
| `artifacts/skills/plan-execute-parallel-oc/fixtures/test_worker_precheck.py` | port-oc | `0e88e8f30a852edaaeac0d86272adf0e097b6b3b14d9177f90963ad3e9bb214a` |
| `artifacts/skills/plan-execute-parallel-oc/patronus.yaml` | port-oc | `66a2db7af7915e3121701d0ff0414ff693045e809bc6cd1be3efb75ea487698e` |
| `artifacts/skills/plan-execute-parallel-oc/scripts/worker-precheck.sh` | port-oc | `7ef744d57101aa64a8f681107f7283db3f645df0c2dd487faba5de93be08f664` |
| `artifacts/skills/plan-review-oc/LICENSE` | reuse-cx-body-with-rename | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |
| `artifacts/skills/plan-review-oc/NOTICE` | reuse-cx-body-with-rename | `0725e858af3e1f5321b05eb88d35a9196f21853b0bda489fb1f8ac23c92f5d75` |
| `artifacts/skills/plan-review-oc/SKILL.md` | reuse-cx-body-with-rename | `d4e7f55fcfb26cd6a7687328e70320462f908ece6ea98b35462626a6a83de63b` |
| `artifacts/skills/plan-review-oc/SOURCE.md` | reuse-cx-body-with-rename | `976bfae88f09e108655834f75d920f5d87ae532dccd09a1f80a37b8c0ea3feed` |
| `artifacts/skills/plan-review-oc/patronus.yaml` | reuse-cx-body-with-rename | `0505ae211a20d0d17fcbddbe720b20ca48636a5e357e5ffa20d817f120075060` |
| `artifacts/skills/plan-review-oc/plan-reviewer.md` | reuse-cx-body-with-rename | `acb0e35a8ded6faa9561cd36aacaf0ea5e8e0bda176872fee88c4935a9a3227d` |
| `artifacts/skills/plan-writing-oc/LICENSE` | reuse-cx-body-with-rename | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |
| `artifacts/skills/plan-writing-oc/NOTICE` | reuse-cx-body-with-rename | `8a988312b1bcf1f029805980a6c7d468097d99ce9fc8986a234f54c618c73c5b` |
| `artifacts/skills/plan-writing-oc/SKILL.md` | reuse-cx-body-with-rename | `aa81bad7d8b1613ef041c28ec7d626c72f9c3994a15e6f6e4ae130c6c4363531` |
| `artifacts/skills/plan-writing-oc/SOURCE.md` | reuse-cx-body-with-rename | `2855d3689c0597dd16f3f5ae11668cb9a1cbd9f49fa33f0e4aa86f0d4b31d9f8` |
| `artifacts/skills/plan-writing-oc/patronus.yaml` | reuse-cx-body-with-rename | `6986a068a09f09968a65dc9ac67b8c758aae59c1a969a238fb1e0ccdec120634` |
| `artifacts/skills/receiving-code-review-oc/LICENSE` | reuse-cx-body-with-rename | `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400` |
| `artifacts/skills/receiving-code-review-oc/NOTICE` | reuse-cx-body-with-rename | `fafea4e4aad995ee48c159c0136f63a09f71cd6c55691234aaa44f4171cff7ff` |
| `artifacts/skills/receiving-code-review-oc/SKILL.md` | reuse-cx-body-with-rename | `6d251f156c2c8899facede608de8920f458ae863d476fe0be8a58bae0e6cd63c` |
| `artifacts/skills/receiving-code-review-oc/SOURCE.md` | reuse-cx-body-with-rename | `bb4cc0fe260e832c1933e062ae342729d5f65ace38592150e02cfd4d28b42d86` |
| `artifacts/skills/receiving-code-review-oc/patronus.yaml` | reuse-cx-body-with-rename | `14a03fa82b706b08cdfd0b66ebcef3877a4bab1383ef5f6c4e096e3db38095e4` |
| `artifacts/skills/requesting-code-review-oc/LICENSE` | reuse-cx-body-with-rename | `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400` |
| `artifacts/skills/requesting-code-review-oc/NOTICE` | reuse-cx-body-with-rename | `d93f7b6b2c60f675921c1b8312297fb4373e3657b2001935a199336797bcd18d` |
| `artifacts/skills/requesting-code-review-oc/SKILL.md` | reuse-cx-body-with-rename | `010715274eaf4f492ff6de2fda54096152c6c6061f0d0e8e6b473490090dd4e1` |
| `artifacts/skills/requesting-code-review-oc/SOURCE.md` | reuse-cx-body-with-rename | `47a19886d24cf5cdd467959a34a2792253b66a35dafe8adaba02c5468471152c` |
| `artifacts/skills/requesting-code-review-oc/code-reviewer.md` | reuse-cx-body-with-rename | `0637d561f27ddf68a5732e3f0ff167a33d2839eee53a84c15afc59c585cdaf02` |
| `artifacts/skills/requesting-code-review-oc/patronus.yaml` | reuse-cx-body-with-rename | `213c9a92547152724fec202cd683c6f325c815855bd8b855c4f425a25d99f7be` |
| `artifacts/skills/research-team-oc/LICENSE` | port-oc | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |
| `artifacts/skills/research-team-oc/NOTICE` | port-oc | `4aaa75257aa3ea95276a24dbf82686f7922a18a6a036bd201a3b739c2c254a88` |
| `artifacts/skills/research-team-oc/SKILL.md` | port-oc | `e58cea3e86ed9603ffc1896efd2f6ce315571c9f0fbf01705444a1703d227783` |
| `artifacts/skills/research-team-oc/SOURCE.md` | port-oc | `269f0d0bffafabdddc47f0058e7af8242746e019c15b20eaec1631680b810148` |
| `artifacts/skills/research-team-oc/patronus.yaml` | port-oc | `c120b851b65437d2f77bd63b6b4900ab6cb5796bcf3360d6f99829ec0f34b2ae` |
| `artifacts/skills/skills-dispatch-oc/LICENSE` | port-oc | `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400` |
| `artifacts/skills/skills-dispatch-oc/NOTICE` | port-oc | `5c0420787dde2acff3ac4c0ae097582e4d689c6960d12daaaec046edd8bc1658` |
| `artifacts/skills/skills-dispatch-oc/SKILL.md` | port-oc | `173bb7428479982a2510945826860bc9635e0ec161bf8d64c6eee9cbcf5b5352` |
| `artifacts/skills/skills-dispatch-oc/SOURCE.md` | port-oc | `31176b32a663d07a1be1c4be1f90e113914d8fe6741becf14b17b643cf21dda6` |
| `artifacts/skills/skills-dispatch-oc/patronus.yaml` | port-oc | `d9611da49057cafd7829b036b00be44c51ae29ed0e5dad3d30633d3e412727fc` |
| `artifacts/skills/spec-brainstorming-oc/LICENSE` | reuse-cx-body-with-rename | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |
| `artifacts/skills/spec-brainstorming-oc/NOTICE` | reuse-cx-body-with-rename | `97c2c63c99dde5250f1c17d6fc9f4bbfecc17e4237ec3bfa44007d4fe6a930db` |
| `artifacts/skills/spec-brainstorming-oc/SKILL.md` | reuse-cx-body-with-rename | `d05dee739887e7ddf45533720ff419aa3245a792958446e8167f7e0a98ca7cd5` |
| `artifacts/skills/spec-brainstorming-oc/SOURCE.md` | reuse-cx-body-with-rename | `d9e0526545c0d8a6e1fe56f9259a5f33ea752c3370fe32bdf681e1891a651c83` |
| `artifacts/skills/spec-brainstorming-oc/patronus.yaml` | reuse-cx-body-with-rename | `6eb35e219238069c1209c27b8eeb343f4e10481fa5421e85037f956da50a26ba` |
| `artifacts/skills/spec-brainstorming-oc/scripts/README.md` | reuse-cx-body-with-rename | `ae4fe72f76d8457a6c119bd8b75e74a5531ecf52203bbf55c4d79108d9b1a31e` |
| `artifacts/skills/spec-brainstorming-oc/scripts/go.mod` | reuse-cx-body-with-rename | `90fe08aa25fa674322bd86994a84373c473dd7f21851524c42b1f21a560ff5ea` |
| `artifacts/skills/spec-brainstorming-oc/scripts/go.sum` | reuse-cx-body-with-rename | `3f8f6a8ac1048ed9b6cb66a4e2795ff956d52fd447239c67eed83943fa91ac35` |
| `artifacts/skills/spec-brainstorming-oc/scripts/main.go` | reuse-cx-body-with-rename | `4ec7c6a23a4abc03dfe12271dd068c19ff6415e149f6b5bf64d23225a97b26d0` |
| `artifacts/skills/spec-brainstorming-oc/spec-document-reviewer-prompt.md` | reuse-cx-body-with-rename | `0d90c0e55baaa10f6233eab0c97cf8d3e30e6b84e560c72c03e33bfe19a72c8d` |
| `artifacts/skills/spec-review-oc/LICENSE` | reuse-cx-body-with-rename | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |
| `artifacts/skills/spec-review-oc/NOTICE` | reuse-cx-body-with-rename | `f4915643f9805a2fe39a6b0fb1e7e5efbfb39bc7f5737963ad0f59585e6f0ab4` |
| `artifacts/skills/spec-review-oc/SKILL.md` | reuse-cx-body-with-rename | `1076a8f8e022017fa4753ce456946aa76e96809a582d345e802ee8c7e8090083` |
| `artifacts/skills/spec-review-oc/SOURCE.md` | reuse-cx-body-with-rename | `0bdf4f8a6b95d091a5947e17b9f0b534ed9738d084e2fe3e752451f2c7a0dd65` |
| `artifacts/skills/spec-review-oc/patronus.yaml` | reuse-cx-body-with-rename | `3aa5fa53b526d70756106dcdf8bfde801c911ea1c08007584340f7d49a96579c` |
| `artifacts/skills/spec-review-oc/spec-reviewer.md` | reuse-cx-body-with-rename | `5886de5492f5fe9632f2714f70ecb154bf897ae1b8e02f494e7e21f5c891cf48` |
| `artifacts/skills/tdd-oc/LICENSE` | reuse-cx-body-with-rename | `0e7ac423bf2c6e223b7c5b156f8cf72da49d748e56a1641402c31f22ad07dbb5` |
| `artifacts/skills/tdd-oc/NOTICE` | reuse-cx-body-with-rename | `6539080e889b0a3e625f6b1c7f6715b11ef2482eba19b50637594f9ca9a10748` |
| `artifacts/skills/tdd-oc/SKILL.md` | reuse-cx-body-with-rename | `39e0066b4300a335027f013a73f91da53dc07e00d31e0b6c0c97b25903606ac8` |
| `artifacts/skills/tdd-oc/SOURCE.md` | reuse-cx-body-with-rename | `e1fcf7ebb0de809c3fdd87104bc97f351b9f379576340efdcffaf8ac34ff5c5d` |
| `artifacts/skills/tdd-oc/mocking.md` | reuse-cx-body-with-rename | `3ceb807fdf4a47d6a93d4d9a891e5ba6d362a6247bd08adc451feebfc17361ef` |
| `artifacts/skills/tdd-oc/patronus.yaml` | reuse-cx-body-with-rename | `e2fa0b9030eca331fdc099b765185f9537093aebf82c73a8ec0e57a4f4bf19cf` |
| `artifacts/skills/tdd-oc/refactoring.md` | reuse-cx-body-with-rename | `54fced22dd1911b7094c3fe7979b7c1a40d40be307482c4adf7dc0588f27d6cc` |
| `artifacts/skills/tdd-oc/tests.md` | reuse-cx-body-with-rename | `e12182f5c4c6a844b7c6ab4b8d2623b642aaaae5460fe58ed087efd4f3d5fb8a` |
| `artifacts/skills/using-git-worktrees-oc/LICENSE` | reuse-cx-body-with-rename | `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400` |
| `artifacts/skills/using-git-worktrees-oc/NOTICE` | reuse-cx-body-with-rename | `bc44969cd46f0331359bb0ecb68bd6b26e7acfb11437c832dd2e1207ad4e6c7a` |
| `artifacts/skills/using-git-worktrees-oc/SKILL.md` | reuse-cx-body-with-rename | `d7f394559ce3d0e4c81274bc8a471ae6a4e9631daaa5982d97d861def2bb73e3` |
| `artifacts/skills/using-git-worktrees-oc/SOURCE.md` | reuse-cx-body-with-rename | `1226a38ef9a744e6d7bc227200b8f93bf702d4163cf6a6d5f542c428f7adb500` |
| `artifacts/skills/using-git-worktrees-oc/patronus.yaml` | reuse-cx-body-with-rename | `3f241361d1be856d1b77b118d386623fc4b29e1ce224bcc1bf0a7abf4882f9b9` |
| `artifacts/skills/verification-before-completion-oc/LICENSE` | reuse-cx-body-with-rename | `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400` |
| `artifacts/skills/verification-before-completion-oc/NOTICE` | reuse-cx-body-with-rename | `1fff198ada52e5c5178203e01e5b1c18069407b43ed8628d23915e07348d12f9` |
| `artifacts/skills/verification-before-completion-oc/SKILL.md` | reuse-cx-body-with-rename | `0f377da605c5fd4ed37508e61a7088b53c060044669c1413d7079e25596a5c91` |
| `artifacts/skills/verification-before-completion-oc/SOURCE.md` | reuse-cx-body-with-rename | `d4e8976b6766134170a7aa11ff2d0991c61d89aca97ddd0801728e35c2805b37` |
| `artifacts/skills/verification-before-completion-oc/patronus.yaml` | reuse-cx-body-with-rename | `f6987ffd731e6f784345f154cf786edf74f615ba19db6289c60c3bbcc3b2ee25` |
| `artifacts/skills/writing-skills-oc/LICENSE` | reuse-cx-body-with-rename | `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400` |
| `artifacts/skills/writing-skills-oc/NOTICE` | reuse-cx-body-with-rename | `d00f4955f08a5d64bca9ba1ba9c665954de6a0a819e53abecef3e18473df63c0` |
| `artifacts/skills/writing-skills-oc/SKILL.md` | reuse-cx-body-with-rename | `da2b02ed041bdcd08b18ca116143b5339512a5b64350a8067cc44966aa8f7053` |
| `artifacts/skills/writing-skills-oc/SOURCE.md` | reuse-cx-body-with-rename | `82e47ea507e2a69f625036af727e4418915bc4cbc812111eedb5a5ea0f9d4a7e` |
| `artifacts/skills/writing-skills-oc/authoring-best-practices.md` | reuse-cx-body-with-rename | `dd0b7e0f91ea0cdf4072f328d9e8af0e9c4e27581c4868fc0ffbb168c853e049` |
| `artifacts/skills/writing-skills-oc/examples/AGENTS_MD_TESTING.md` | reuse-cx-body-with-rename | `c5027c238c2429066d0da4dea2caf41003f8174ff8f8ddef61fa1487f465076c` |
| `artifacts/skills/writing-skills-oc/graphviz-conventions.dot` | reuse-cx-body-with-rename | `5166a9152c6c3a698e13f427bb12014c7b5ba77757cc508fb0039d11f9de73af` |
| `artifacts/skills/writing-skills-oc/patronus.yaml` | reuse-cx-body-with-rename | `a6c65cfd97ffbf593b87241864251df38dcd403562bcb88e8991fde18f0d8020` |
| `artifacts/skills/writing-skills-oc/persuasion-principles.md` | reuse-cx-body-with-rename | `c9217b6e7f43a99297caebc8c847738d5ad4cb1768e4ca0880ab7a7f6c10edb0` |
| `artifacts/skills/writing-skills-oc/testing-skills.md` | reuse-cx-body-with-rename | `e75c506ef3fe8f3cf635f05c35dfd88af65a9115ec5fdba613079dfa2dba3ee3` |
| `artifacts/instructions/agents-spine-oc/INSTRUCTIONS.md` | port-oc | `a72f0f1751f3e64a246e9884fcca2ebfaf4b44b3ff5d117e6fb921e793ac87d8` |
| `artifacts/instructions/agents-spine-oc/LICENSE` | port-oc | `13df7812ca53ecaae1cb4a868844bb598373047ae1d580e4debfbef1dd5b6915` |
| `artifacts/instructions/agents-spine-oc/NOTICE` | port-oc | `ea4316c11b03bbf26c7fe178ac18c6e75115dfb6b876bbe5f9609285d6b342ce` |
| `artifacts/instructions/agents-spine-oc/SOURCE.md` | port-oc | `66e00187b78b4822fe88d63032b499925ccb7969cef4fd2ea947879aa80d0dfb` |
| `artifacts/instructions/agents-spine-oc/patronus.yaml` | port-oc | `89c6ba9cdb9697fa8ff249d1dea2e8bb9e62c8133484bf18eb8fed24e21da7b5` |
| `artifacts/instructions/branch-first-oc/INSTRUCTIONS.md` | reuse-cx-body-with-rename | `838e22d6fb3a7f1b2c486108072460f4621a950e9ecadfc5e26187d3892ee8e9` |
| `artifacts/instructions/branch-first-oc/LICENSE` | reuse-cx-body-with-rename | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |
| `artifacts/instructions/branch-first-oc/NOTICE` | reuse-cx-body-with-rename | `9966e7e7ef14b401717545af341f21e64d5279c535b561c3d58ed3e11ed9aeaa` |
| `artifacts/instructions/branch-first-oc/SOURCE.md` | reuse-cx-body-with-rename | `91b4272387bd7efb76f841ce019856c9e63e5ec4dcabd480d8961bdb96c9a890` |
| `artifacts/instructions/branch-first-oc/patronus.yaml` | reuse-cx-body-with-rename | `31c4d3eb0331f6a599f6a0dbe22caff71dce68d4e713293cf90e16d75d28fab5` |
| `artifacts/instructions/diagram-explain-oc/INSTRUCTIONS.md` | reuse-cx-body-with-rename | `568a56626c7da60012ae67c7e7334eb0a3575c47f3101fc014efcaded6a38988` |
| `artifacts/instructions/diagram-explain-oc/LICENSE` | reuse-cx-body-with-rename | `e9482f6e42088344ff06248c10e5c86416e4a0b7f85d39f75a45ddecb387325f` |
| `artifacts/instructions/diagram-explain-oc/NOTICE` | reuse-cx-body-with-rename | `69b77811cb7d87bd5f1d313261df64230ba2f41cf4ed1744801d91f97bea9aef` |
| `artifacts/instructions/diagram-explain-oc/SOURCE.md` | reuse-cx-body-with-rename | `ae3122cacf5c58c7fd22970824b0d75c146db5a6904bf411cff153092d1c4f6b` |
| `artifacts/instructions/diagram-explain-oc/patronus.yaml` | reuse-cx-body-with-rename | `1d70542ce5c9978aa7ad92914d6a0aeeed2bac37cc2ff3332f6e2a2f3fda369a` |
| `artifacts/instructions/language-detect-oc/INSTRUCTIONS.md` | port-oc | `0e188e0b3d671e88a9f47278b85c0c820b3d18b209cbf382c9a00d8d15f511a0` |
| `artifacts/instructions/language-detect-oc/LICENSE` | port-oc | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |
| `artifacts/instructions/language-detect-oc/NOTICE` | port-oc | `cdd2285c54bc795323d591701122540b64f325331e32d152a3d553979cf5be25` |
| `artifacts/instructions/language-detect-oc/SOURCE.md` | port-oc | `9b22a9c0b0f5edff6c95b371704bfc0285da8b45b631b1b417de81bcfb910ac9` |
| `artifacts/instructions/language-detect-oc/patronus.yaml` | port-oc | `a631cda416d23ca5a57550b2107861637245192b0f0556659e2fe387340e084f` |
| `artifacts/instructions/session-completion-oc/INSTRUCTIONS.md` | reuse-cx-body-with-rename | `06a3d84098b8894c2be23230dd5778fabaea6d774986a4fbd206c412d613d6f0` |
| `artifacts/instructions/session-completion-oc/LICENSE` | reuse-cx-body-with-rename | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |
| `artifacts/instructions/session-completion-oc/NOTICE` | reuse-cx-body-with-rename | `7e94086eddd080c4548126e00eb3b805fc1e67353595c098e471129672257ce7` |
| `artifacts/instructions/session-completion-oc/SOURCE.md` | reuse-cx-body-with-rename | `778dd7096b7c3d45198011d5296d16223fc2da8b176619e7c866e50bf7b16c46` |
| `artifacts/instructions/session-completion-oc/patronus.yaml` | reuse-cx-body-with-rename | `43ca4dca148ef52555b8c8af689565e9c39c6bef53163f1bde84d38762d91b37` |
| `artifacts/instructions/work-state-reground-oc/INSTRUCTIONS.md` | port-oc | `c8df74d53d0b0131850c4fe36d729b34cb1d3447637100aa34943f95f41697d9` |
| `artifacts/instructions/work-state-reground-oc/LICENSE` | port-oc | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |
| `artifacts/instructions/work-state-reground-oc/NOTICE` | port-oc | `87f3695a3d7724ca41c3550d2b63ab52b79116849727dfa410b92459b092a533` |
| `artifacts/instructions/work-state-reground-oc/SOURCE.md` | port-oc | `780eb402043f5e42fdd505c23376bb0e0bfaf49518c5de91d2a6645faf0defc8` |
| `artifacts/instructions/work-state-reground-oc/patronus.yaml` | port-oc | `b0b589b9748877fb48477d96b838de661577c30d99e30d86835091269944bf07` |
| `artifacts/instructions/writing-style-pointer-oc/INSTRUCTIONS.md` | reuse-cx-body-with-rename | `bc0a3e375b9156f84e0ece023aeddb7b9e5fc3237ec8ed8e34d2a33230a336e5` |
| `artifacts/instructions/writing-style-pointer-oc/LICENSE` | reuse-cx-body-with-rename | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` |
| `artifacts/instructions/writing-style-pointer-oc/NOTICE` | reuse-cx-body-with-rename | `22a3ed9d16291e8a4c738d4ea4d571f72220940205f9a9a104216ac3f466cb09` |
| `artifacts/instructions/writing-style-pointer-oc/SOURCE.md` | reuse-cx-body-with-rename | `6432cd612a0171c70ed7a1f40be4209a8008ffbaedc37c8406d11f1e725528b3` |
| `artifacts/instructions/writing-style-pointer-oc/patronus.yaml` | reuse-cx-body-with-rename | `2f40c2ec5d21e30b077a63c90bf7069dce636890db041ef7fbd301711849e9b6` |
| `artifacts/agents/patronus-plan-author-oc/agent.md` | Pi role port | `338a583619af769dc27bb09bf555dce23aed57aa69b87db1b0ed70e3f3fa071f` |
| `artifacts/agents/patronus-plan-author-oc/patronus.yaml` | Pi role port | `705ba587d64703a7f5136857e4391834533480e93f6b0ad1322c76054750c974` |
| `artifacts/agents/patronus-plan-reviewer-oc/agent.md` | Pi role port | `8e373a808f87304c403c8fa64d87778033c47913f654bb70c4f5f31277d15e0d` |
| `artifacts/agents/patronus-plan-reviewer-oc/patronus.yaml` | Pi role port | `22fe3f30490f165461c301eca47cb3d1ea6aa63fcd88a666fb61040dea3b9b8a` |
| `artifacts/agents/patronus-researcher-oc/agent.md` | Pi role port | `7922e248ead0180be1be059d6a73d7979ee4574e4aed65b5f2dbdff45e284b9f` |
| `artifacts/agents/patronus-researcher-oc/patronus.yaml` | Pi role port | `929d21be0f5733a2d629e29cfb38f7c13fb2275cf880e755cd62f053462d7c96` |
| `artifacts/agents/patronus-spec-author-oc/agent.md` | Pi role port | `b1c1294ce44397ee5ed35fdabc7aa6ee3be46325a638883ce8b9ab2ec8118d38` |
| `artifacts/agents/patronus-spec-author-oc/patronus.yaml` | Pi role port | `0b15e35574156fb36d7e696c45365ae6b465c72d0cf229b21f025640a1ecf91f` |
| `artifacts/agents/patronus-technical-reviewer-oc/agent.md` | Pi role port | `fb9ddef686a5976351a466166958842baf654bd6179fa2337e2075cdc5369098` |
| `artifacts/agents/patronus-technical-reviewer-oc/patronus.yaml` | Pi role port | `ebbb67be8bf83c3bad80aff9a6e7dae1c694b0dc924bd7cab1ea2a9fa6623f15` |
| `artifacts/agents/patronus-web-researcher-oc/agent.md` | Pi role port | `02a91b295f98bc11d1af7a378d116f31147ea2a31b97e37e0744c4b9fcc0147b` |
| `artifacts/agents/patronus-web-researcher-oc/patronus.yaml` | Pi role port | `78b083237ad1344c1444641cf9dfeb0bda6fcf65ecab1f21626f8daeb652f579` |
| `artifacts/agents/patronus-workflow-security-reviewer-oc/agent.md` | Pi role port | `1f5428c1bde862de5159d40e06d4ac02bf6cf6375bae8f1f054340beb48384db` |
| `artifacts/agents/patronus-workflow-security-reviewer-oc/patronus.yaml` | Pi role port | `5e44a911a1311486674b5fa10a66c9a04be8dba47050fe8f87369074f415dad2` |
| `artifacts/agents/patronus-writer-oc/agent.md` | Pi role port | `61064c4807e8a563420a268a2b0196d464991ad23e1e1d0c638942971743f39b` |
| `artifacts/agents/patronus-writer-oc/patronus.yaml` | Pi role port | `29aec7e2a705cfa49140a97d782f55c31f09a897f4155a08f217c7360c693a7b` |
| `artifacts/hooks/block-secrets-oc/SOURCE.md` | byte-copy script; OpenCode delivery manifest | `b0571dd9f35a974634bc12ac8d3660ea54ca500bf9ecf6b317ce2af9a3dc67fe` |
| `artifacts/hooks/block-secrets-oc/block-secrets.py` | byte-copy script; OpenCode delivery manifest | `cfd78d4f424b39a7577f52f8a41d1c515ac28a67681b98ca786369dae5a46fa5` |
| `artifacts/hooks/block-secrets-oc/patronus.yaml` | byte-copy script; OpenCode delivery manifest | `cb8e129fba4821cbf1e8815a9255d9dbae61b2f40000619876fe8303de2db41a` |
| `artifacts/hooks/gitleaks-guard-oc/SOURCE.md` | byte-copy script; OpenCode delivery manifest | `d8967db7daab94f9a00b2701fa2f3b4b8c50edaab447f47681ee43ccdf9c1432` |
| `artifacts/hooks/gitleaks-guard-oc/gitleaks-guard.py` | byte-copy script; OpenCode delivery manifest | `172606b6a0ca3cae4f5ca16a5c32c9fe1c08baa1057a6fb581f4d64d312473e5` |
| `artifacts/hooks/gitleaks-guard-oc/patronus.yaml` | byte-copy script; OpenCode delivery manifest | `a2d8a9eddb41e10c83dca1abc6485d00049ac5b64e769051f950100ced76fd3a` |
| `artifacts/settings/permission-defaults-oc/SOURCE.md` | original setting | `8f3cb4844ac6655c542151d5f0205afe114769e2a3cff211a71fc335314b7216` |
| `artifacts/settings/permission-defaults-oc/patronus.yaml` | original setting | `3c6d02f3f8e23a530ee807a352a80740b504f309c06b2a3cd85a4a4df051f164` |
| `artifacts/settings/skill-visibility-oc/SOURCE.md` | original setting | `cf3bd72062919a60d4d35e5b3dbb0479d32e8378b5380f0fdf96dc569ea5b7a2` |
| `artifacts/settings/skill-visibility-oc/patronus.yaml` | original setting | `213472f294f940ccaf40ddecc3d71caaf6cbbb85717ac07c16a295b1a6191274` |

## Omissions and remaining limits

Legacy dispatch/startup/heartbeat/reground/language callbacks remain omitted;
advisory instructions do not replace them. Editorial/voice-model and tk
work-graph integration are excluded; Claude ccusage/statusline are N/A.
Every baseline disposition and endpoint is in core-opencode.yaml.

The shared isolated-writer-supervisor is not a seventh baseline reuse item:
it is an additional shared helper at 1.0.0. Retained fake-worker and OpenCode
precheck fixtures are historical and not run by current gates. Native result,
permission merge and settlement remain pending. runDir permits the entire
evidence root outside the worktree, not just the result; denied outside-write
needs native observation. Cooperative permission rules are not an OS sandbox.

Guard copies keep best-effort and --pure disablement limits. Historical Node
observations in SOURCE.md are not Bun proof. Roles, MCP, mask effects, both-scope
double execution and fresh-session lifecycle remain OC-N01..N17 runtime-pending.
No parity, credentials or native permission enforcement is established here.
