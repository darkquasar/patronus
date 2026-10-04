# Pi content inventory and migration review

## Current inventory boundary

The authoritative selection is `profiles/core-profile-pi.yaml` **2.0.0**, with no
extends edge to Claude core. The profile is catalog data, not an application
membership oracle or proof of runtime support.

The current package and authored-content boundaries are:

- `pi-subagents` and `pi-web-access` are package-manager recipes with the exact refs
  `npm:pi-subagents@0.72.1` and `npm:pi-web-access@0.35.0`. Patronus retains only
  their exact native operation reference/identity plus minimal provenance,
  observation, pending/outcome and profile-ownership state needed for lifecycle
  safety. Pi/npm own package declarations, dependency resolution, scripts and
  installed files. No vendored/static extension payload or npm-internal per-file
  fingerprint ledger is part of this profile.
- `workflow-research-pi`, `workflow-implement-pi` and
  `workflow-peer-review-pi` are separate Patronus-owned installed skill artifacts.
  Their `SKILL.md`, `workflow.js`, request schema and declared fixtures are ordinary
  profile-tracked authored files; they are not files supplied by the native npm
  package manager.
- Normal lifecycle operations preserve and report edited authored files, shared
  profile effects, native declaration/source drift and unresolved manager outcomes.
  Authorized `--force` can remove selected profile-tracked artifacts or a selected
  controlled native identity under the implemented checks. It cannot select an
  unrelated root/tree, delete untracked siblings or npm trees itself, bypass package
  install/trust consent, or suppress Pi/npm/OS failures.

Model-free fixtures validate catalog, placement and lifecycle mechanisms. They do
not qualify actual installed entrypoints, Pi loading, providers, subagent execution,
web behavior, platform coverage or release support. Those claims require separately
bound installed-runtime evidence. Dated observations elsewhere must retain their
stated versions and limits; this page does not invent a completed qualification.

## Superseded reconciliation record

Everything from this heading through the historical matrices and examples below is
preserved as review/provenance evidence for the former static/archive design. It is
superseded as current inventory or operational guidance by the boundary above.
Historical counts, version 1.0.0 membership, directory recipes, payloads, receipt-
derived activation and qualification handoffs must not be read as present behavior.

### Historical reconciliation

- Original core: **42 direct rows**, tk as one transitive row, and installed-only
  go-style-uber/graphify = **45 source rows**. The `I` column marks the **25 installed
  prototype skills**, not proof that those ambient copies match these sources.
- Add the actual pi-subagents runtime package = **46 baseline disposition rows**
  below. E-07's five authority adaptations are **V**, not misleading R reuse.
- E-03 adds required web skill, pointer, role and actual recipe (four additional
  rows below). Current core is **44 unique catalog items**: 31 original selected
  content + 2 web content + 8 roles + 3 recipes. All are explicitly listed.
- The proposed code-intel overlay adds six to make **50** and inherits Go guidance.
  It is not authored/qualified here. No standalone web overlay is created. Earlier
  40/46/43/49 core/code-intel/web/combined counts are superseded, not current claims.
- Native-replaced/omitted rows, Pi host/peer installers, pattern-mcp-pi and graphify-pi
  are absent from core. No optional MCP, guard enforcement, memory daemon or builtin
  agent substitution is implied. Required web remains present even for local-source
  roles; tool availability is checked separately from static placement.

### Historical source disposition matrix

R = portable body reuse in a new identity (metadata/references may change),
V = behavioral adaptation, N = native replacement, O = omit/defer. Source rows
with `@claude` retain the original flavour as provenance, not an active Pi alias.
All selected siblings target Pi only. `tk` is the intentional unchanged agnostic
recipe; pi-subagents/pi-web-access are actual global directory recipes, not skills.

| Original source item | I | Disposition, selected identity and inventory |
| --- | --- | --- |
| agents-spine | no | V agents-spine-pi: context discovery and explicit stage authority. |
| diagram-explain | no | V diagram-explain-pi: ASCII guidance; user format wins. |
| branch-first | no | V branch-first-pi: advisory and subject to Git grant. |
| writing-style-pointer | no | V writing-style-pointer-pi: depends on writing-editorial-pi. |
| skills-dispatch | yes | V skills-dispatch-pi: read installed SKILL.md; no Claude Skill API. |
| skills-dispatch-activate | no | N discovery plus agents-spine-pi reminder; no separate artifact. |
| skills-heartbeat@claude | no | N metadata discovery; no catalog injection artifact. |
| work-state-reground@claude | no | V work-state-reground-pi: reread brief/ledger after resume; no automatic stage advance. |
| language-detect | no | V language-detect-pi: bounded checks; no installs. |
| plan-writing | yes | V plan-writing-pi: ADR-0003, authorized plans and requirement coverage. |
| plan-execute | yes | V plan-execute-pi: retain solo.md, sdd.md, two prompts, three scripts, nine fixtures; adapt authority/budget. |
| grilling | yes | V grilling-pi: optional bounded interview; no stage hop. |
| diagnosing-bugs | yes | V diagnosing-bugs-pi: preserve scripts/hitl-loop.template.sh; no unavailable slash handoff or unapproved stress/setup. |
| tdd | yes | V tdd-pi: preserve tests.md, mocking.md, refactoring.md; prior approval remains valid. |
| spec-brainstorming | yes | V spec-brainstorming-pi: Patronus format and spec-document-reviewer-prompt.md; remove foreign layout/API ceremony. |
| using-git-worktrees | yes | V using-git-worktrees-pi: ownership/prelaunch cleanup authority; no fallback-in-place. |
| finishing-a-development-branch | yes | V finishing-a-development-branch-pi: distinct integrate/publish/discard grants. |
| writing-skills | no | O optional later authoring overlay; retain six sidecar groups if selected. |
| writing-editorial | yes | V writing-editorial-pi: four tiers, fixtures, sectioning, edit-record, attribution; relative paths and inline fallback. |
| research-team@claude | yes | V research-team-pi: keyed native workflows; include RESEARCHER-TEMPLATE.md, DELIVERABLE-TEMPLATES.md, LESSONS-FORMAT.md. |
| plan-execute-parallel@claude | yes | V plan-execute-parallel-pi: native lanes, isolated owners and barriers. |
| codebase-design | yes | V codebase-design-pi: DEEPENING.md, DESIGN-IT-TWICE.md; bounded delegation. |
| domain-modeling | yes | V domain-modeling-pi: ADR-FORMAT.md, CONTEXT-FORMAT.md; project-governed tracking. |
| ddd-distilled | yes | R ddd-distilled-pi: portable rules, preserve inspiration qualification. |
| refactoring-distilled | yes | R refactoring-distilled-pi: behavior-preserving discipline. |
| context7 | no | O optional authenticated integration. |
| pattern-cloudflare | no | O domain overlay, outside initial core. |
| pattern-mcp | yes | V pattern-mcp-pi: code-intel overlay only. |
| github | no | O optional authenticated integration; no issue/PR grant inferred. |
| spec-review | yes | V spec-review-pi: spec-reviewer.md and scoped fresh review. |
| plan-review | yes | V plan-review-pi: plan-reviewer.md; no execution authority. |
| verification-before-completion | yes | V verification-before-completion-pi: observed fresh evidence. |
| requesting-code-review | yes | V requesting-code-review-pi: code-reviewer.md, exact diff/range. |
| receiving-code-review | yes | V receiving-code-review-pi: verify findings before authorized changes. |
| block-secrets | no | O automatic guard; no Pi enforcement claim. |
| gitleaks | no | O from core; optional external explicit scanner candidate 8.30.1 under Q. |
| gitleaks-guard | no | O automatic interception; no dangling requires edge. |
| ccusage | no | N native/session/subagents accounting. |
| ccusage-statusline@claude | no | N session UI and /subagent-cost; no parent-footer child-usage claim. |
| ticket | no | V ticket-pi skill: requires tk, delivered executable preflight, coordinator-only writes and approved Markdown fallback; references/markdown-ledger-template.md required. |
| session-completion | no | V session-completion-pi: stage-bound handoff; no unconditional pull/rebase/push/prune. |
| dispatching-parallel-agents | yes | V dispatching-parallel-agents-pi: independent keyed outputs and memory admission. |
| tk (old transitive dependency) | no | REQUIRED unchanged recipe tk1.0.0 delivering upstreamv0.3.2 pinned script globally; ticket-pi requires tk. |
| pi-subagents (runtime package) | no | REQUIRED exact `npm:pi-subagents@0.72.1`; Pi/npm own install, update, removal and package files; cold-start loading remains separately qualified. |
| go-style-uber (installed-only) | yes | R go-style-uber-pi: REQUIRED core, Go-relevance trigger only; NOTICE, references/style.md and references/LICENSE. |
| graphify (installed-only skill) | yes | V graphify-pi: code-intel overlay, eight named references plus .graphify_version (nine supporting files), SKILL.md separate. |

### Required web additions (owner amendment)

| New source | Disposition / delivery |
| --- | --- |
| web-research-pi | New required capability; SKILL.md, SOURCE.md and declared references: inert web-search/current-model templates, runbook and qualification. SOURCE.md records source evidence; full source receipt/text is separate qualification input, not a shipped sidecar. Requires pi-web-access. |
| web-research-pointer-pi | New inline instruction; requires web-research-pi; no automatic active config target. |
| patronus-web-researcher-pi | New static role; read/bash plus four eagerly registered web tools, research/verification/web skill union, pi-subagents/pi-web-access. |
| pi-web-access | New required exact `npm:pi-web-access@0.35.0`; Pi/npm own install, update, removal and package files; config and cold-start tools remain separately qualified. |

The seven other newly authored roles are listed with complete skills/tool/authority
contracts in [Pi core](pi-core.md#roles-and-launch-contract). They are original
Patronus prompt composition, not copied upstream builtins or renamed legacy agents.
Their two-file roots contain `patronus.yaml` and `agent.md`, both mode0644. No
vendored text requiring an omitted NOTICE or agent sidecar is introduced.

### Historical requires and invocation audit

The generic catalog loader checks all requires edges. Native catalog validation
uses the D-owned byte parser (no JS execution), ensures selected skills are declared
in requires and resolve to Pi-compatible skills. The complete role skill unions
are in native frontmatter and manifests; the local seven each also require
pi-subagents. Technical code-review mode retains spec-review-pi and
verification-before-completion-pi while reading requesting-code-review-pi/code-reviewer.md.
Machine-detectable Markdown links and native skill declarations are linted; plain
prose still needs this classification, not a blind global rename.

### Non-role requires edges (all others have none)

| Source | Declared requires |
| --- | --- |
| agents-spine-pi | ticket-pi |
| language-detect-pi | go-style-uber-pi |
| writing-style-pointer-pi | writing-editorial-pi |
| web-research-pointer-pi | web-research-pi |
| ticket-pi | tk |
| plan-execute-pi | requesting-code-review-pi, pi-subagents |
| research-team-pi, plan-execute-parallel-pi, dispatching-parallel-agents-pi | pi-subagents |
| web-research-pi | pi-web-access |

These are actual manifest edges, distinct from conditional stage suggestions in
prose. Recipe manifests have no further catalog requires; their archive closure
is inventoried separately, not as catalog rows. All role skills and prose-selected
Pi names are inside the explicit core closure; source-name mentions in source
records/licenses/old examples are provenance, not deploy-name aliases.

### Reviewed prose and sidecar classifications

| Active resource | Other selected reads / classification |
| --- | --- |
| agents-spine-pi, work-state-reground-pi, session-completion-pi | ticket-pi; initial/resume preflight, coordinator-only writes. |
| language-detect-pi | go-style-uber-pi only for Go relevance; no install or universal always-load instruction. |
| writing-style-pointer-pi | writing-editorial-pi and its declared tier/sectioning/edit sidecars. |
| skills-dispatch-pi | diagnosing-bugs-pi, grilling-pi, spec-brainstorming-pi, tdd-pi according to task/relevance, not foreign Skill API. |
| plan-writing-pi | plan-execute-pi, plan-execute-parallel-pi, plan-review-pi, ticket-pi as separately granted stages/coordination. |
| plan-execute-pi | requesting-code-review-pi, tdd-pi, ticket-pi, using-git-worktrees-pi, plan-execute-parallel-pi, finishing-a-development-branch-pi; coordinator owns delegation. |
| plan-execute-parallel-pi | requesting-code-review-pi, tdd-pi, ticket-pi, using-git-worktrees-pi; no leaf fanout. |
| research-team-pi | spec-brainstorming-pi, plan-writing-pi, plan-execute-parallel-pi and required web-research-pi; stage suggestions never grant execution. |
| dispatching-parallel-agents-pi | requesting-code-review-pi, ticket-pi, using-git-worktrees-pi; named native workflow APIs are external runtime calls, not guaranteed availability. |
| spec-brainstorming-pi | grilling-pi, research-team-pi, spec-review-pi; spec-document-reviewer-prompt.md is declared. |
| diagnosing-bugs-pi, tdd-pi | codebase-design-pi; shell/test/tool names and code examples are external capabilities, not catalog invocations. |
| finishing-a-development-branch-pi | using-git-worktrees-pi, verification-before-completion-pi; integrate/publish/discard remain distinct grants. |
| grilling-pi | diagram-explain-pi identifies its reproduced charset convention. spec-brainstorming-pi/research-team-pi are availability/grant-qualified suggestions. Plan-writing/review/execution names appear in negative stage-routing guidance, not mandatory dispatch. |
| spec-review-pi, plan-review-pi, requesting-code-review-pi | Their own declared reviewer rubric sidecars; fresh independent parent-launched review, not Claude general-purpose dispatch. |
| writing-editorial-pi, codebase-design-pi, domain-modeling-pi | Own declared sidecars, no additional named skill invocation. Source inspiration/attribution is not a runtime dependency. |
| ddd-distilled-pi, refactoring-distilled-pi, go-style-uber-pi | No active sibling invocation. DDD/refactoring bodies are portable with inspiration/not-the-book caveats. Go body uses declared full guide/license; Go APIs are external code examples. |
| diagram-explain-pi, branch-first-pi | `[claude]` is an illustrative platform label; tools-visual-ascii-arch is attribution. `git checkout -b` is an explicitly authorized external command example, not automatic branch creation. |
| verification-before-completion-pi, receiving-code-review-pi | No mandatory sibling invocation. Verification remains task/resource-grant-bound. `gh api` is an external executable requiring explicit access/reply authority; no installation/auth/publication grant. |
| ticket-pi | tk is the delivered executable, not a skill/tool registry name; canonical pin preflight plus explicitly limited Markdown fallback. |
| web-research-pi, web-research-pointer-pi | research-team-pi / web-research-pi reads. web_search, fetch_content, get_search_content, source_check are activated extension tools, not registered by prose. Current-model OpenAI is separately approved alternative, never DDG fallback. |
| All roles | Their explicit native skill union plus mandatory brief/ledger/instructions and ticket preflight context. Those resolved paths are launcher-bound, not static author-home paths. Role modes/rubrics are described in pi-core.md. |

Historical numbered plan-execute fixtures and RESULTS.md retain upstream evidence,
not current Pi successes. SOURCE.md records original filenames/hashes/modes and
active-read classifications; NOTICE/upstream URLs are provenance. Names in those
records must not be rewritten into fictional upstream Pi sources. Optional
integrations are availability-and-authority qualified. Unknown plain-prose
mappings require review; regex/name replacement is not a semantic oracle.

### Historical sidecars, licenses, modes and original-byte invariance

Generic inventory enumerates declared files including dotfiles, checks regular
files/modes, entry/sidecar existence, NOTICE for attribution, distributable paths,
concrete relative links and author-machine path leaks. Review also checks:

- Go NOTICE plus full pinned references/style.md and references/LICENSE; diagram's
  complete MIT notice is embedded in the emitted inline instruction (its source
  NOTICE alone would not be delivered by the instruction adapter).
- Plan-execute's solo/sdd guides, implementer/task-reviewer prompts, three executable
  scripts, all nine numbered fixture groups and README/RESULTS; inherited dated
  fixture bytes are provenance. New helper tests are invented-data example checks.
- Diagnosing's hitl-loop template; tdd's tests/mocking/refactoring guides;
  spec/plan/code review rubrics; editorial four tiers, tier1 fixtures, sectioning,
  edit record/NOTICE; research's three templates, example/schema/fixture; design's
  DEEPENING/DESIGN-IT-TWICE/NOTICE; domain's ADR/CONTEXT/NOTICE; ticket's Markdown
  ledger; every additional declared SOURCE/NOTICE/license/link, including retained
  parallel-execution source sidecars. Executable modes are not erased on copy.
- Web templates and qualification/runbook remain inert skill files; source
  receipt/text is separate reviewed evidence, not a claimed shipped sidecar.
  There is no implicit write to active web-search.json. Packages retain independent source
  locks, SBOM and license inventories; no third-party code is imported by these
  checks. See [dependency residuals](pi-core-dependencies.md).

At CP-03 base `4c816f3a0916ff779796ddab933e14b3fccefd1c`, the saved recursive
Git tree inventory for artifacts/profiles/recipes/packages/adapters has SHA-256
`c865fc7d3d9ac8770346267564cab2f93c3e12889ccb4692e2c27ffb8b78b488`.
The task adds new content only; existing catalog entries, modes, profiles, recipes,
payloads and any tracked locks must compare identically to that base. The task
report/packet binds the actual before/after comparison and test logs; this hash
is content provenance, not proof of a deployed runtime or newly published asset.
Frozen legacy sidecar debt (pat-yncn) is not silently waived for new Pi content.

### Historical functional-duplicate migration example

D checks exact discovered names/paths/config ownership, **not semantic equivalence**.
Before migration the operator/coordinator uses the old-to-new rows above and records
actual canonical source paths, bytes/hashes, descriptions, scope/precedence,
consumers and an explicit disposition. Different names can still overlap:

| Observed identity | Source recorded in the operator ledger | Description / disposition |
| --- | --- | --- |
| tdd | Existing manually installed prototype at its discovered canonical skill root (record actual path/hash; do not assume ambient source equality) | Test-first development guidance; retain pending review. |
| tdd-pi | Proposed Patronus receipt/state-owned Pi skill at the previewed root, source manifest/version/hash recorded | Task-authority-bound test-first guidance. **Functional overlap flagged** despite distinct names. |

Withhold governed migration/activation until explicit retain/replace decision and
applicable grants. Do not automatically adopt/delete the prototype or treat a
conflict-free static preview as semantic approval. Unknown mappings go to review.
Even a retain-both choice must explain intended dispatch/precedence. Capture old
config/prior bytes and all consumers; shared Claude/tk consumers still count.

After a separately approved migration, a **preview only** example is
`patronus remove tdd-pi --target pi --local`. It removes only that unchanged owned
item on later authorized apply, never the prototype by analogy. There is no
profile-wide inverse or semantic mapping heuristic. Preserve dependencies,
credentials/config backups, sessions, missions, reports and worktrees. Settle
consumers, deregister each exact native extension source before payload removal,
and retain/escalate drift/unknown state; see [the lifecycle runbook](pi-core.md).

### Historical ADR-0003 content example review

The delivered spec/plan/research guidance requires one synthesis per effort and
one spec/plan per stream. Coordinator owns actual meta.yaml. These invented
examples exercise its checklist; they never edit real effort metadata or seed tk.
Parse YAML values, not filenames in comments. Check both directions: each named
research/spec/plan exists; every stream spec/plan file is named. A declined spec
requires an explicit decision. An uncreated/unauthorized plan remains null;
epic stays null and tasks [] before authorized implementation seeding.

**Positive A:** fixture files `demo-research.md`, `alpha-spec.md` exist; plan not
created and no seeding authority. Accept:

```yaml
research: demo-research.md
streams:
  - slug: alpha
    spec: alpha-spec.md
    plan: null
    epic: null
    tasks: []
```

**Positive B:** with separately authorized `alpha-plan.md` now present, change
only plan to `alpha-plan.md`; keep epic null/tasks [] while unseeded. Accept.
**Negative A:** the same named plan when the file is absent: reject, not a future
completeness promise. **Negative B:** an existing `beta-spec.md` omitted from
streams: reject incomplete inventory. **Negative C:** epic `demo-123` or nonempty
tasks without seeding authority: reject invented execution state. **Negative D:**
missing spec with no explicit decline decision: reject coverage claim. These are
content-review examples, not a new metadata schema/parser or per-artifact Go test.

### Historical qualification handoff and evidence boundary

C validation checks current data, not a hardcoded catalog list in application Go.
M tests use invented bytes; exact native role emission/unsafe syntax/lifecycle
mechanisms remain D-owned. QP-03 must separately exercise **all eight exact roles**
from this profile, every mandatory skill/tool/context/provider/output contract,
technical task/whole-branch review and resumed ticket preflight. It also observes
required subagents/DDG functionality and actual native source precedence in the
recorded environment. Missing subagents/web refuses delegated/core claims; no
builtin or optional-web substitution. Unpublished endpoints, runtime/platform and
stronger web bounds remain honest residuals. Counts and this content review do
not qualify a release; commands and support limits are in [Pi core](pi-core.md).
