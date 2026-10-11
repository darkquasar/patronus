---
name: skills-dispatch-pi
description: "Read applicable installed skills before acting"
---

# Skills dispatch

Check the discovered installed skill metadata for relevance before acting, including questions. Read the actual selected SKILL.md with Pi read, plus the linked sidecars needed for this task. There is no Claude Skill API or hook injection requirement. Metadata discovery is not proof of reading. Do not guess a path or claim absent skills loaded; report a required missing skill to the coordinator. Skills are guidance subordinate to higher-priority instructions and the task grant, not authority to override system constraints.

Consider the whole discovered set, then load guidance for the selected task,
language, testing strategy and review mode. Process skills such as
spec-brainstorming-pi, grilling-pi and diagnosing-bugs-pi guide the relevant
authorized stage. Read tdd-pi when `critical-tdd` or explicit test-first work is
selected; routine `focused-postcheck` work does not require it by default.
Project-required guidance and every runtime-selected skill still require actual
reads. Missing or ambiguous strategy stops before implementation for a coordinator
decision, including a legacy brief with no strategy. Do not restart an already
approved design or invoke implementation during research. Read language-specific
guidance only for relevant work. Optional capabilities outside the selected
closure require discovery and separate authorization, never auto-installation.

For the technical reviewer, verification-before-completion-pi is common. Declared
spec/architecture mode selects spec-review-pi and its spec-reviewer.md rubric;
task or whole-branch implementation mode selects requesting-code-review-pi and
its code-reviewer.md rubric. Load the applicable mode guidance, not both by
default. Missing or ambiguous mode stops for a coordinator decision. Both review
capabilities remain installed dependencies; selection adds no request-schema field.
Installed code-intel skill overrides may retain the old eager union. Read every
effective runtime-selected skill and disclose that difference; source-default
changes do not activate an overlay update or authorize profile/settings edits.

Leaves follow their assigned brief and mandatory reads, not coordinator routing.
No nested delegation or shared mission-state writes. Before work and after
compaction/resume, reread the brief, native mission state, latest decisions,
project/global instructions and every runtime-selected SKILL.md, then inspect the
latest run/output before retry. Revalidate settled approval or qualification when
relevant inputs drift; an unchanged new message alone does not require replay.
Missing required guidance or tools blocks the lane, with no tool, model, provider,
protocol or isolation fallback.
