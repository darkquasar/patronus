# OpenCode project instructions

Read effective user/project instructions, the approved task brief, latest lead
handoff and current source before acting and after resume. OpenCode reads the
project `AGENTS.md` found by walking up from the working directory to the git
root, the global `AGENTS.md` under its config directory (`OPENCODE_CONFIG_DIR`,
else `XDG_CONFIG_HOME/opencode`, else `~/.config/opencode`), and every path or
glob listed in the `instructions: []` array of `opencode.json`. Skills are
co-visible from `.claude/skills` (including nested `synced/` trees) and
`.agents/skills` roots next to OpenCode's own skill roots. `.opencode/AGENTS.md`
is not read as instructions. Verify the installed OpenCode configuration before
assuming custom discovery. Nested instructions refine their scope. Do not create
a higher-priority shadow file or rewrite instructions to hide source drift.

Useful sections are dev environment (approved commands/layout), testing (focused
and full checks), conventions (language/ownership) and commit/review rules. When
absent, inspect source/CI and report the gap. Absence grants no installation,
network, editing or Git authority. System/developer instructions remain higher
priority than project prose and skills.

The lead binds root/base/head, stage/action, owned files, output paths,
predecessors, acceptance and open decisions in the approved brief and retained
handoff, plus existing host/resource or explicitly granted campaign limits where
applicable. Ordinary work needs no numerical root-budget form or start ledger.
Research, authoring, review, planning, execution, integration, publication and
cleanup are separate grants. Existing approval remains valid within scope;
changed targets, destructive actions or material design choices need revalidation.
Changed stage names alone do not require repeated grants.
There is no OpenCode mission API assumed by these instructions. Record and read
durable lessons in `patronus/LESSONS.md` when that file exists; only the
authorized lead edits it.

Select the sole mutable progress authority once in the accepted brief. Qualified
native mission/workflow state that actually owns delegated runs stays authoritative;
project records reference it without mirroring status. Without such native state,
use explicit project/owner selection, otherwise the existing tracker, or one
durable plan-associated record if none exists. Native UI todos alone do not outrank
a project tracker. Do not create a selection ledger or competing status mirror.

Only the coordinator admits workers, execution modes and transitions. Researchers,
implementers, fixers and reviewers are leaves: return complete evidence and stop,
without successors, added gates, nesting or shared progress mutations. Read-only
workers return complete text through available native delivery; the coordinator
persists shared outputs. Inspect current native capabilities, authentication,
role restrictions and wait/stop controls for the assigned operation. Report actual
missing or auth-blocked primitives; do not require a blanket qualification ceremony
before usable native tools. Never switch model/provider/protocol or invoke a CLI
model to bypass a blocked lane.

Parallel reads need independent questions and capability-matched read-only roles.
Writes require approved isolated owners, disjoint path claims and
settlement before serial integration. Native workers prove no write isolation;
concurrent writers in the lead checkout are forbidden. Parallel implementation
uses native Task subagents only with checked allocated roots and permitted tool
targeting, not a mandatory CLI supervisor. A leaf may prepare only an exact
allocation already authorized by the coordinator, never select successors.
CLI workers require explicit user mode selection; optional worktree-only setup
launches no agents. Existing mode approval remains valid within scope; a materially
different alternative needs coordinator authorization, never failed-engine fallback.
Settle workers and capture complete results before serial integration. Admit heavy
builds/tests separately from writer concurrency, account for nested parallelism
and isolate mutable outputs. Leaf tests do not prove the integrated composition.

Read complete worker outputs and checked patches, bind their hashes and report
actual tests before acceptance. Keep one independent review per required spec,
plan and aggregate implementation subject; an explicit owner-declined spec/plan
review stays in the brief. Task count alone creates neither a skip nor an extra
gate. Earlier reviews need a named safety/compatibility invariant or irreversible
boundary with its own acceptance point. No automatic task-level review chains.
One matching aggregate review covers feature/branch/pre-merge triggers for the
same exact inputs and coverage. Use at most one bounded batched correction per
gate; unresolved Critical/Major or more than two distinct Medium block, and up
to two Medium residuals require explicit owner acceptance. Missing requirements,
authority or tests independently block. Correction is parent-verified against the
cited defect, exact corrected bytes and positive/negative/regression evidence,
not retrospectively independently reviewed. Changed interface, safety invariant
or acceptance scope, or an insufficient oracle, needs an owner decision or an
additional independent-review grant. No automatic successor review/fix waves.
Static placement is runtime-unverified. Trust, auth and native loading are
separate gates; Patronus does not write OpenCode permission or auth internals.
No automatic push, worktree removal, stash deletion or cleanup follows from a
completed stage.
