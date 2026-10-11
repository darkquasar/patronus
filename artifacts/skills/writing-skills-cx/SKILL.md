---
name: writing-skills-cx
description: "Codex only: Use when creating, editing or auditing skill instructions and their local references before an authorized deployment."
---

# Write skills that can be checked

A skill is a reusable technique, pattern or reference, not a transcript of one
project. Author only the approved files. Authoring, audit, runtime testing,
deployment and publication are separate grants. No automatic nested agents,
paid calls, installations, commits or pushes. A leaf never delegates.

## Structure and discovery

Use exact harness-visible names ending in `-cx`, matching directory and
frontmatter. A description starts with "Codex only: Use when..." and gives
observable triggers, not a summary that invites skipping the body. Keep the
entry concise: purpose, applicability, steps, evidence and common failures.
Put heavy references in declared sidecars and use `{skillDir}` for local reads.
Cross-skill reads use exact installed companion routes, for example
`{skillsDir}/tdd-cx/SKILL.md`. Name required
companions in the manifest's `requires`; verify them installed before use.
Missing routes block, never fuzzy-match to a foreign sibling.

Read `{skillDir}/authoring-best-practices.md` for progressive disclosure and
reference organization. `{skillDir}/persuasion-principles.md` explains wording
trade-offs, not permission to override the user's constraints. Use positive
output recipes for shape failures, required template slots for omissions and
observable conditions for conditional behavior. Reserve prohibitions for actual
safety or discipline violations. Do not manufacture authority or urgency.

## Proportionate verification

Classify the skill: discipline, technique, pattern or reference. Record the
failure it should prevent and the acceptance oracle. Static source checks cover
identity, targets, dependency closure, references and placement, not native
compliance. For behavior changes use an approved baseline/scenario comparison
as described in `{skillDir}/testing-skills.md`. Code test discipline is described
in `{skillsDir}/tdd-cx/SKILL.md`, not an unconditional costly ritual for every
content edit. Existing approval governs scope and proportionate checks.

A test campaign needs its own explicit lead grant with case count, runtime,
resource/spend limits, deadline and output bindings before any worker launch.
No automatic raw API sampling, repeated model calls or test harness creation.
Use local invented cases where sufficient. Report unavailable runtime checks as
blocked. Never delete untested source or other owners' work to satisfy a ritual.

## Audit and handoff

Check every active sidecar and route, not just SKILL.md. Bundle referenced files,
NOTICE, license and source inventory. Historical source names belong in
provenance, not executable handoffs. No Markdown include substitution or assumed
host tool names. See `{skillDir}/examples/AGENTS_MD_TESTING.md` for bounded cases.
`{skillDir}/graphviz-conventions.dot` is optional diagram reference only; no
renderer or package installation is automatically invoked.

Before claiming a check passed read
`{skillsDir}/verification-before-completion-cx/SKILL.md`. Return exact files,
input/output hashes, observed cases/exits/logs, skipped checks and residual risks.
Deployment/publication requires separate approval and qualified native evidence;
static placement alone is not readiness.
