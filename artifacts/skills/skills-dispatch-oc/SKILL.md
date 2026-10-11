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
(`-cx`, `-pi` or unsuffixed `.claude` skills) may therefore be co-visible in the
same `<available_skills>` list. Select exact `-oc` names for this OpenCode lane.
Do not fuzzy-match an unsuffixed or other-host sibling when the named `-oc`
skill is absent. Missing/unreadable required skills or tools stop the lane as
blocked. Placeholders resolve file paths only; Markdown name replacement is not
skill activation or a substitute for an OpenCode body.

## Exact routes

- Unknown domain: `{skillsDir}/research-team-oc/SKILL.md`.
- Settle or author a spec: `{skillsDir}/spec-brainstorming-oc/SKILL.md`.
- Stress-test choices when requested: `{skillsDir}/grilling-oc/SKILL.md`.
- Review a spec: `{skillsDir}/spec-review-oc/SKILL.md`.
- Author an approved plan: `{skillsDir}/plan-writing-oc/SKILL.md`.
- Review and route a plan: `{skillsDir}/plan-review-oc/SKILL.md`.
- Approved serial execution: `{skillsDir}/plan-execute-oc/SKILL.md`.
- Approved parallel execution: `{skillsDir}/plan-execute-parallel-oc/SKILL.md`
  (shared bounded helper at
  `{skillsDir}/isolated-writer-supervisor/scripts/supervisor.py`; native OpenCode
  workers remain Partial/runtime-pending).
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
actual reads. Missing or ambiguous strategy stops before implementation for a
lead decision, including a legacy brief with no strategy. Completion verification
is common to reviews; declared spec mode selects spec-review-oc, while
implementation mode selects requesting-code-review-oc and its implementation
rubric. Missing or ambiguous mode stops for a lead decision. Keep the capabilities
available without eagerly loading both rubrics.

Before work and after compaction/resume, reread the brief, current task state and
latest decisions, project/global instructions and every selected skill. Inspect
the latest run/output before retry. Revalidate settled approval or qualification
on relevant input drift, not for each unchanged new message. A route description
never qualifies the supervisor or native subagents; bounded launch, settlement,
capture and integration still follow the shipped parallel skill and their exact
grants. No new controller or host API is introduced here.

An available name proves neither tool activation nor authentication. Delegation
goes through the OpenCode Task tool to the installed `@patronus-<role>-oc`
subagent roles only after `opencode agent list` shows the role; if a role or
native primitive is unavailable or auth-blocked, report blocked and stop. Do not
invoke a CLI model, install a replacement or change
model/provider/protocol/isolation. This is advisory dispatch guidance; no
startup hook is registered or claimed.
