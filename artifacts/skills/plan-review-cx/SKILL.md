---
name: plan-review-cx
description: "Codex only: independently review exact ADR-0003 plan/spec coverage, build order and interfaces before execution, read-only."
---

# Plan review for Codex

Every written plan, including one task, requires one independent review unless
the owner explicitly declines that subject in the existing brief. A second
reviewer/model is a separately admitted start, never automatic discovery or
repeated consent. Inline author checks are not dispatched reviews.

Use after `{skillsDir}/plan-writing-cx/SKILL.md` closes planning. Read `{skillDir}/plan-reviewer.md`. Review the written plan AND its source spec, not the author's intent or only the last changed task.

## Scope and folder contract

Require a review grant, task brief, project instructions, recorded decisions, exact plan/spec/research input paths and hashes, source root/revision and authorized check evidence. After context loss, reread these inputs and the latest saved outputs before retrying. Existing approval remains valid within its recorded root, files and applicable resource limits. A material ambiguity or changed target requires an explicit lead/owner decision; wait for the reply. Review grants no implementation, commit, integration, publication or cleanup authority.

Preserve ADR-0003: `docs/specs/NN-slug/`, one research effort, one `meta.yaml`, one folder-level research synthesis and one spec/plan pair per stream. Resolve this stream's filenames through parsed metadata and read both files. Check every named file exists and every `*-spec.md` / `*-plan.md` in the folder is named. The checker is documented at `{skillsDir}/spec-writing-cx/scripts/README.md`; execution requires authorized host checks and resource limits. Metadata names artifacts, not boolean completeness flags.

An owner-declined spec must have `spec: null` and a recorded `spec_declined:` decision. Such a plan can be reviewed for internal consistency only, never spec coverage. Missing referenced inputs otherwise block. Do not reverse-engineer a spec, overwrite a plan, or silently extend scope.

## Fresh read-only review

The lead may request a fresh-context native Codex reviewer only with exposed native tools, read-only scope, authentication and complete
delivery under the review grant. Supply exact inputs and constraints, not the author transcript. Missing capability blocks required independence; an authorized inline self-check remains labelled non-independent. Do not switch provider/model or launch another host as fallback.

Read-only workers return complete findings to the lead, without subject/shared checkout/metadata/index/work-record writes or delegation. Only the lead writes shared outputs and verifies returned bytes/hashes. A completion message or successful exit cannot replace reading the full report and verifying source-backed claims.

## Findings and handoff

Each finding has ID, severity, task/section/quote, evidence, consequence and proposed correction. Normalize Critical (catastrophic security/data loss), Major (substantial requirement/correctness/safety failure), Medium (bounded material defect), Low (presentation/low impact). Preserve original severities and source IDs when deduplicating, with disagreement visible. No threshold-driven downgrade.

Use one fresh independent review and at most one bounded batched correction/disposition per exact subject. Unresolved Critical/Major findings block; up to two distinct Medium residuals need explicit owner acceptance, and more than two block. Missing tests, requirements, evidence or authority independently block. Only the lead/owner dispositions findings and grants transition. Corrected bytes need explicit disposition; never launch another review wave automatically.

The coordinator chooses execution mode after acceptance under an execution
grant, using `{skillsDir}/plan-execute-cx/SKILL.md` for serial and
parallel-ready tasks. The reviewer checks dependency, ownership, isolation and
settlement feasibility, plus the recorded or defaulted slicing and the execution
graph (task agreement, consistent IDs, no missing edges or cycles, correct
`start`/`join`, resolved shared-file hazards, no unaccepted collaborators hidden
behind mocks, J/C/R barriers), without choosing or starting execution.
Parallel work uses disjoint allocated roots, never concurrent writers in the
lead checkout; mode changes need coordinator admission, not a failed-engine
fallback.

Return exact source/root/revision and input hashes, coverage map, actual commands/results, skipped lenses, canonical findings, residuals and full report bytes. Acceptance plus a valid execution grant may permit a later execution stage; review completion alone cannot start it.
