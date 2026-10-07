---
name: grilling-cx
description: "Codex only: bounded one-question-at-a-time interview to stress-test a plan or design on request, without stage advance."
---

# Stress-test a design in Codex

Use only when requested. Read the brief, current design and project instructions.
Answer source-resolvable questions by bounded source reads before interviewing.
Set a question/time budget with the lead/owner; do not extend it automatically.
Ask one question at a time, state the recommended answer and evidence, and wait
for the actual reply. Steering is not consent to a changed architecture or scope.

Walk dependent decisions in order. For a structural ambiguity, include a compact
ASCII diagram with labeled boxes, directional edges, `=>` for sync and `~>` for
async, no tabs and at most 100 columns, unless the requested format forbids it.

The result is clarified choices, assumptions, open questions and evidence returned
to the lead. Read-only workers return findings; only an authorized lead writes
shared outputs. This interview grants no edits, dispatch, planning, implementation,
Git mutation or publication. No native interview tool is assumed: use the actual
available Codex interaction surface and stop if required input cannot be obtained.

## Upstream offers only

Inspect the existing `docs/specs/NN-slug/` folder and `meta.yaml` without treating
file presence as approval. If a spec exists, ask whether to fold the clarified
choices into it through `{skillsDir}/spec-brainstorming-cx/SKILL.md`. With research
but no spec, offer that same skill; do not loop back to research. With neither,
offer `{skillsDir}/research-team-cx/SKILL.md` for domain unknowns or
`{skillsDir}/spec-brainstorming-cx/SKILL.md` for design settling.

Check offered companion files are installed and readable. These are suggestions,
not gates. There is no forward planning or execution hop from this interview.
Do not launch another stage automatically. Unavailable or auth-blocked required
primitives stop the lane, with no CLI model or engine/provider/protocol fallback.
