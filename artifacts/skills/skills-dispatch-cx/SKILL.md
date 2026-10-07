---
name: skills-dispatch-cx
description: "Codex only: read applicable installed skills and resolve exact -cx workflow routes before acting. Advisory guidance, not an activation hook."
---

# Codex skill dispatch

Before acting, inspect the current Codex skill inventory and read each applicable
installed `SKILL.md` with the available file-read tool. Follow system/developer
instructions, task authority and scoped project instructions first. Skills refine
behavior within that authority; they never override higher-priority instructions
or grant new tools, installs, stages or Git mutations. A dispatched worker reads
its assigned skills, executes only its task and never delegates.

Select exact names for this Codex lane. Do not fuzzy-match an unsuffixed or other
host sibling when the named skill is absent. Missing/unreadable required skills
or tools stop the lane as blocked. Placeholders resolve file paths only; Markdown
name replacement is not skill activation or a substitute for a Codex body.

## Exact routes

- Unknown domain: `{skillsDir}/research-team-cx/SKILL.md`.
- Settle or author a spec: `{skillsDir}/spec-brainstorming-cx/SKILL.md`.
- Stress-test choices when requested: `{skillsDir}/grilling-cx/SKILL.md`.
- Review a spec: `{skillsDir}/spec-review-cx/SKILL.md`.
- Author an approved plan: `{skillsDir}/plan-writing-cx/SKILL.md`.
- Review and route a plan: `{skillsDir}/plan-review-cx/SKILL.md`.
- Approved serial execution: `{skillsDir}/plan-execute-cx/SKILL.md`.
- Approved parallel execution: `{skillsDir}/plan-execute-parallel-cx/SKILL.md`
  (bundled bounded helper at
  `{skillsDir}/plan-execute-parallel-cx/scripts/supervisor.py`; native Codex workers
  remain Partial/runtime-pending).
- Independent review: `{skillsDir}/requesting-code-review-cx/SKILL.md`.
- Completion evidence: `{skillsDir}/verification-before-completion-cx/SKILL.md`.

These are installed profile companion reads, not permission to advance stages.
Check the selected file exists and is readable before using its route. An offered
next step needs authorization; existing approved stage authority remains valid
within scope. Never auto-start planning, execution or publication.

Select other installed guidance by the task, language, testing strategy and
review mode, with host compatibility checks. Read tdd-cx for `critical-tdd` or an
explicit test-first request; routine `focused-postcheck` work does not require it
by default. Project-required guidance and every selected skill still require
actual reads. Missing or ambiguous strategy stops before implementation for a
lead decision, including a legacy brief with no strategy. Completion verification
is common to reviews; declared spec mode selects spec-review-cx, while
implementation mode selects requesting-code-review-cx and its implementation
rubric. Missing or ambiguous mode stops for a lead decision. Keep the capabilities
available without eagerly loading both rubrics.

Before work and after compaction/resume, reread the brief, current task state and
latest decisions, project/global instructions and every selected SKILL.md. Inspect
the latest run/output before retry. Revalidate settled approval or qualification
on relevant input drift, not for each unchanged new message. A route description
never qualifies the supervisor or native workers; bounded launch, settlement,
capture and integration still follow the shipped parallel skill and their exact
grants. No new controller or host API is introduced here.

An available name proves neither tool activation nor authentication. If a native
primitive is unavailable or auth-blocked, report blocked and stop. Do not invoke
a CLI model, install a replacement or change model/provider/protocol/isolation.
This is advisory dispatch guidance; no SessionStart hook is registered or claimed.
