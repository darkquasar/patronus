---
name: skills-dispatch-oc
description: "OpenCode only: read applicable installed skills and resolve exact -oc workflow routes before acting. Advisory guidance, not an activation hook."
---

# OpenCode skill dispatch

Before acting, inspect the current OpenCode skill inventory and read each
applicable installed skill explicitly with the OpenCode `skill` tool (or the
file-read tool on the resolved path). Follow system/developer instructions, task
authority and scoped project instructions first. Skills refine behavior within
that authority; they never override higher-priority instructions or grant new
tools, installs, stages or Git mutations. A dispatched subagent reads its
assigned skills, executes only its task and never delegates (native subagent
depth is 1).

OpenCode reads four project skill roots (`.opencode/skills`, `.opencode/skill`,
`.claude/skills`, `.agents/skills`) plus the global roots under the OpenCode
config directory and `~/.claude/skills`. Siblings written for other hosts
(`-cx`, `-pi` or Claude `-cl` skills) may therefore be co-visible in the
same `<available_skills>` list. Select exact `-oc` names for this OpenCode lane.
Do not fuzzy-match an unsuffixed or other-host sibling when the named `-oc`
skill is absent. Missing/unreadable required skills or tools stop the lane as
blocked. Placeholders resolve file paths only; Markdown name replacement is not
skill activation or a substitute for an OpenCode body.

## Exact routes

- Unknown domain: `{skillsDir}/research-team-oc/SKILL.md`.
- Settle or author a spec: `{skillsDir}/spec-writing-oc/SKILL.md`.
- Stress-test choices when requested: `{skillsDir}/grilling-oc/SKILL.md`.
- Review a spec: `{skillsDir}/spec-review-oc/SKILL.md`.
- Author an approved plan: `{skillsDir}/plan-writing-oc/SKILL.md`.
- Review a written plan: `{skillsDir}/plan-review-oc/SKILL.md`.
- Approved execution, serial or parallel-ready: `{skillsDir}/plan-execute-oc/SKILL.md`
  (admitted parallel leaves use native Task subagents and checked allocated roots;
  CLI workers require explicit user choice).
- Independent review: `{skillsDir}/requesting-code-review-oc/SKILL.md`.
- Completion evidence: `{skillsDir}/verification-before-completion-oc/SKILL.md`.

These are installed profile companion reads, not permission to advance stages.
Check the selected file exists and is readable before using its route. An offered
next step needs authorization; existing approved stage authority remains valid
within scope. Never auto-start planning, execution or publication.

Select other installed guidance by the task, language, testing strategy and
review mode, with host compatibility checks. Read tdd-oc for `critical-tdd` or an
explicit test-first request; routine `focused-postcheck` work does not require it
by default. Project-required guidance and every selected skill still require
actual reads. Other features/behavior default to TDD unless an approved concrete
exception applies; read tdd-oc for that behavior too. Behavior-preserving
refactors use green coverage and characterize uncertainty. Omission applies and
records these defaults in the existing brief, without another approval stop.
Genuine classification/conflict/exception ambiguity needs owner direction.
Completion verification
is common to reviews; declared spec mode selects spec-review-oc, while
implementation mode selects requesting-code-review-oc and its implementation
rubric. Missing or ambiguous mode stops for a lead decision. Keep the capabilities
available without eagerly loading both rubrics.

The coordinator chooses execution mode, admits leaves and records the one
progress authority at plan acceptance through
`{skillsDir}/plan-writing-oc/SKILL.md`. Consume that selection for recovery;
do not expand plan checklists into todo/tk mirrors. Run-owning native state wins
over a project tracker; UI todos alone do not. Ordinary development requires no
root numerical budget form; existing host/resource and explicit campaign limits
still apply. Existing valid grants remain valid until material scope/input drift.

Before work and after compaction/resume, reread the brief, current task state and
latest decisions, project/global instructions and every selected skill. Inspect
the latest run/output before retry. Revalidate scope and actual capabilities
on relevant input drift, not for each unchanged new message. A route description
never proves native isolation; checked launch, settlement,
capture and integration still follow the shipped execution skill and their exact
grants. No new controller or host API is introduced here.

An available name proves neither tool activation nor authentication. Delegation
goes through the OpenCode Task tool to the installed `@patronus-<role>-oc`
subagent roles only after `opencode agent list` shows the role; if a role or
native primitive is unavailable or auth-blocked, report blocked and stop. Do not
invoke a CLI model, install a replacement or change
model/provider/protocol/isolation. This is advisory dispatch guidance; no
startup hook is registered or claimed.
