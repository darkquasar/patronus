# Pi core: delivery, roles and qualification

`core-profile-pi` 2.0.0 is an explicit Pi catalog profile, not an alias for or an
extension of Claude `core`. It includes workflow/design/review guidance, the
language-triggered Go guide, eight native roles and two required Pi-managed package
recipes: **pi-subagents and pi-web-access**. Required web is not an optional overlay.
[The source inventory](pi-content-inventory.md) records membership and dispositions.
Original Claude artifacts and locks are unchanged.

**Status: Pi-native delivery implemented; activation and runtime qualification are
separate.** Patronus validates and places its authored static resources. For
`npm:pi-subagents@0.72.1` and `npm:pi-web-access@0.35.0`, Pi/npm own installation,
update, removal and package files; Patronus records exact refs, bounded pending
intent and observed selected metadata. Catalog validation alone is not native
activation or core-exact-pin qualification. Do not infer permission to install,
trust, authenticate, launch, publish or migrate from this document. Pi 0.87.1,
Node 22.22.1 and required host peers remain externally provisioned. Read
[dependency provenance and activation](pi-core-dependencies.md) and
[delivery/ownership](pi-delivery.md) before an authorized operation.

See [Pi-native deployment validation](pi-native-validation.md) for the current
install/update/remove and actual resource-loading observations.

## Explicit selection and preview

These are **preview-only** examples for the selected catalog and roots. They do
not acquire publication authority, apply changes or launch Pi. Package operations
must retain the exact selected refs `npm:pi-subagents@0.72.1` and
`npm:pi-web-access@0.35.0`; do not substitute ambient or unpinned packages.

```sh
patronus install --profile core-profile-pi --target pi --global
patronus install --profile core-profile-pi --target pi --local
patronus update core-profile-pi --target pi --local
patronus update core-profile-pi --target pi --global
patronus remove patronus-researcher-pi --target pi --local
```

Save/hash the preview, exact refs, selected metadata, canonical roots and grant.
Changed inputs need a fresh preview. Apply requires separately authorized
`--deploy`; updates require exactly one explicit scope. A local operation needs
both compatible selected global packages first, otherwise it refuses before local
writes. Establish those globals through Pi in a separate operation. Local removal
leaves globals intact; profiles do not own independent dependency lifetimes.

Static admission inspects discoverable files, structured config/manifests and
static resource declarations. Known-name/path/config collisions, malformed or
unreadable required static inputs, unsafe paths, ownership/drift and mixed-context
consent still block the whole selection. Configured executable extensions alone
are not blanket blockers; dynamic registrations/resolver-only discovery are
visibly runtime-unverified. Static success means no conflict in that inventory,
not complete runtime collision detection. Patronus never executes Pi, extensions,
helpers, npm/git resolvers, `npm root -g`, provider discovery or MCP connections
for this check. No mandatory extension deactivation or `PI_OFFLINE` follows from
runtime uncertainty. Qualification records its actual environment, including
`PI_OFFLINE` if deliberately used; it is not OS/network containment.

## Roles and launch contract

Native files deploy as `<agent-dir>/agents/<identity>.md` globally or
`.pi/agents/<identity>.md` locally, with matching runtime name. They use the
[finite native subset](agent-artifacts.mdx#selected-pi-native-agent-subset), not a
YAML-to-agent translation. Each is a separately versioned static agent with
sole Pi target, explicit `agent.md`, no sidecars/Overrides and declared skills
plus pi-subagents. The definitions do not shadow upstream builtins.

| Identity | Responsibility and subject authority | Selected skills (all mandatory) |
| --- | --- | --- |
| patronus-researcher-pi | Local-source research; source read-only, full returned findings | research-team-pi, verification-before-completion-pi |
| patronus-spec-author-pi | All assigned evidence, then only assigned spec/bundle/output | spec-brainstorming-pi, writing-editorial-pi, verification-before-completion-pi |
| patronus-technical-reviewer-pi | Spec/architecture **or** task/whole-branch implementation review; no subject writes | spec-review-pi, requesting-code-review-pi, verification-before-completion-pi |
| patronus-workflow-security-reviewer-pi | Authority, lifecycle, trust and failure review; no subject writes | spec-review-pi, verification-before-completion-pi |
| patronus-plan-author-pi | Requirement-covered plans; assigned documents/output only | plan-writing-pi, writing-editorial-pi, verification-before-completion-pi |
| patronus-plan-reviewer-pi | Independent coverage/sequence/ownership review; no subject writes | plan-review-pi, verification-before-completion-pi |
| patronus-writer-pi | Separately authorized implementation in assigned worktree/files | plan-execute-pi, tdd-pi, verification-before-completion-pi |
| patronus-web-researcher-pi | Required qualified DDG research, no subject writes | research-team-pi, verification-before-completion-pi, web-research-pi; additionally requires pi-web-access |

The original seven explicitly enable project/global context and skill inheritance,
use `defaultContext: fresh`, `allowNestedSubagents: false`, native unquoted lists
and `outputMode: file-only`. Minimum tools are read/bash; only spec/plan authors
and implementation writer add write/edit. Reviewers/researcher return complete
content for runtime persistence. Bash is powerful: these restrictions are
**cooperative source-read-only scope**, not filesystem isolation. Authors omit
acceptanceRole; researcher/reviewers use read-only, and only implementation uses
writer with separately configured host verification. No model/provider pin or
static output/extensions/skillPath/defaultReads is installed. The web role's
four web tools require separate eager activation; they are not added to the
seven local-source roles. Optional MCP requires separately qualified operator
configuration, never a hidden core dependency or invented per-call tools field.

Before each background launch, the **coordinator** must:

1. Verify exact Pi/subagents/provider/host-peer environment and requested tools
   in the effective child registry. Inspect operator overrides and extension/tool
   loading plan. An absent tool, unavailable skill, inactive/wrong subagents or
   unsupported native field **refuses that launch** with a diagnostic; no builtin,
   prompt-only emulation or model/protocol fallback. This is a cooperative launch
   preflight, not a new Patronus launcher/permission engine.
2. Bind an exclusive absolute output and resolved mandatory reads: brief, ledger,
   decisions, effective project/global instructions, all selected SKILL.md files
   and mode rubric. Require actual read acknowledgement and input hashes.
   Discoverability is not evidence of consumption. Inheritance must not suppress
   approved provider loading. Foreground use requires separate exact-role smoke.
3. Choose the technical reviewer's mode explicitly. Spec mode reads spec-review-pi's
   `spec-reviewer.md`. Task/whole-branch implementation mode reads
   requesting-code-review-pi's `code-reviewer.md`, exact base/head/diff and requirements.
   Call-level skills replace defaults where supported: retain **spec-review-pi,
   requesting-code-review-pi, verification-before-completion-pi**, then add the
   chosen rubric to resolved reads. Never replace the union with just code review.
4. Supply fresh reviewers exact subject bytes/hashes, revision, tests and rubric,
   not the author's reasoning transcript. Set resource/run/deadline budgets;
   default to one measured child. Admit expensive work only with known headroom
   (this deployment: >=750 MiB free and available, >=500 MB reserve). Use one
   writer per disjoint worktree; prelaunch native cleanup/capture authority or
   coordinator-retained cwd is chosen before dispatch, never fallback-in-place.
5. Record logical task key, run/session/mission IDs, attempt lineage, ownership,
   stage/action/root/output grant, hashes, budget and asks in the durable ledger.
   Only coordinator mutates tickets/shared metadata/lessons. Read ticket-pi and
   check canonical delivered tk path/version/commit/digest against qualification
   initially and after resume before mutations. Explicit limited Markdown
   fallback never replaces required delivery or proves qualified core.
6. Require the complete output plus structured acceptance (criteria/evidence,
   hashes, changed files, checks/actual results/logs, residual risks). Read/hash the
   saved bytes; file-only pointers, dispatch receipts and successful exit are not
   completion. Missing output/tools/verification holds. Preserve partial evidence.

Leaves never dispatch. After compaction reread brief, ledger and decisions and
inspect latest run/output before retry. Grants distinguish research, authorship,
review, planning, implementation, integration, publication and cleanup; a valid
existing grant continues within scope. Changed scope/destructive actions need
revalidation. Supervisor requests wait for actual replies; steering is not consent.

Coordinator transitions require all dependent work settled, outputs readable and
hash-bound, fresh uncached/digest-bound host verification, independent review of
those bytes, dispositions and next-action authority. Maximum TWO review cycles,
optional second. Normalize by consequence and deduplicate source IDs: unresolved
Critical/Major or >2 distinct Medium block; <=2 Medium require explicit owner
acceptance; Low corrections optional. Requirement/test/authority/evidence failures
independently block. No spare capacity is needed to close a clean final wave;
no third wave is inferred. Changed bytes need fresh checks and disclosed review
coverage, not a hash placed beside cached success.

## Clean sandbox bootstrap and later lifecycle

This checklist is **not established by install success** and needs separately
authorized deployment qualification. It does not change the live host.

- Use disposable, explicitly selected HOME/PI_CODING_AGENT_DIR/workspace and record
  exact environment/configuration, external Pi/Node/provider/peer identities and
  actual local archive transport. Keep credentials in narrowly injected ephemeral
  secret files, never reports, images or committed config.
- Deliver/verify `tk`, then ask Pi to install the exact `pi-subagents@0.72.1` and
  `pi-web-access@0.35.0` refs in the approved scope. Inspect selected settings and
  package manifests. Inventory duplicate npm/git/local/one-shot declarations and all
  consumers. Pi/npm alone mutate package files; Patronus records bounded intent and
  refuses missing imports, ambiguous aliases or unconfirmed post-state.
- Apply the inert reviewed web template explicitly to the selected agent root's
  `web-search.json` in the clean bootstrap; existing config instead needs per-key
  preview/prior/ownership/drift review. Follow the [web runbook](../artifacts/skills/web-research-pi/references/runbook.md).
  Defaults are DDG-only/eager, workflow none, raw fetch, no automatic summaries or
  background fetch/paid fallback, and disabled browser/cookie/clone/media routes
  where supported. Sanitize cookie-enabling opt-ins. Observe disabled behavior.
  Use <=5 results, includeContent false, source_check fetchContent false, selective
  fetch and get_search_content <=2000-character pages. `maxInlineContentChars:8000`
  is ordinary presentation, not network/cumulative token/cost or OS-egress bounds.
- Keep cache/log/session/mission/output/worktree data outside Pi/npm-owned package
  directories; observe writes. Run cold/reload and **every exact role's** background
  context/skill/output/provider/tool smoke, including technical task/whole-branch
  review mode and initial/resumed ticket preflight. Inspect effective discovery
  and overrides. Exercise all four web tool response/pagination/error paths and
  disabled routes. Missing mandatory web is not optional local-only core success.
- Before global update or removal, settle all sessions/descendants/background/shared
  consumers and record reload acknowledgement. Preview the exact operation, delegate
  it to Pi, and verify the resulting settings/package observation before clearing
  pending intent. Never separately edit or delete npm-owned files. Unknown
  consumers/ownership/drift mean retain/escalate.
  Local removal preserves globals, credentials, external tools, settings backups,
  sessions, missions, outputs and worktrees. No profile inverse/refcount, blanket
  delete, auto-prune or automatic rollback. Restored code is not restored runtime.

## Validation subjects and repeatable commands

**M: application capabilities** use invented manifests/bytes and isolated roots.
DP-07 lifecycle/native-byte/refusal tests exercise mechanisms, not named core
installation. **C: catalog checks/review** enumerate declarations, references,
licenses/modes, profile resolution and native syntax; membership lives in YAML
and the reviewed inventory, never a second executable name/count list.
**I: deployment qualification** observes selected real runtime bytes under a
separate grant. Neither M nor C establishes I. Ordinary tests never import the
acquired extensions or use provider/network credentials.

With provisioned local Go cache and admitted RAM, run from the source root:

```sh
# Fixture-only root BEFORE affected CLI tests; do not point it at live Pi config.
PI_CODING_AGENT_DIR=$(mktemp -d /tmp/patronus-pi-check.XXXXXX)
export PI_CODING_AGENT_DIR
# Generic C (including discovered adapters and native role references)
flock -w 300 /home/agent/.local/state/patronus-code-intel/pi-implementation-test.lock \
  env GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOMAXPROCS=2 \
  go test -p 1 ./internal/registry -count=1 -v
# M plus generic C and retained legacy regressions, not runtime qualification
flock -w 300 /home/agent/.local/state/patronus-code-intel/pi-implementation-test.lock \
  env GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOMAXPROCS=2 \
  go test -p 1 ./... -count=1 -v
rmdir "$PI_CODING_AGENT_DIR" # only this owned empty fixture root
unset PI_CODING_AGENT_DIR
# C-class workflow example doubles, not a Pi launch or scheduling proof
node --test artifacts/skills/research-team-pi/fixtures/workflow-example.test.mjs
```

Record nonzero tests, commands/exits/logs, exact revision/input/output/config
hashes and limitations. For I, use QP-03's authorized operational procedure and
[qualification records](pi-qualification/record-format.md), not a `go test` success
or an ambient installed session. No core-exact-pin/public distribution/bounded-
unattended claim is made here; stronger web bounds remain unresolved (pat-n6za).
