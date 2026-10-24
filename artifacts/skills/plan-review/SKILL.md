---
name: plan-review
description: Use when independently reviewing an exact written implementation plan and its approved spec before execution
---

# Review an implementation plan

Use the owner's selected destination and established project layout throughout.
Any `docs/specs/NN-slug/` path below is a default example, not a location
requirement; preserve the selected effort's metadata and stream relationships.

This optional Codex/OpenCode identity remains available. Review closes planning,
not execution preflight per task. Every written plan, including a single task,
needs one independent plan review unless the owner explicitly declines that
subject and retains the exception in the existing brief. Author self-check is
inline, not this independent review. A required spec review remains a distinct
subject and is not repeated here.

## Semantic review boundary

Check actual spec-to-plan fidelity, inherited constraints versus planner choices,
requirement/task/check links, vocabulary/boundary/invariant preservation and
reasoned start/join deliverables. Runtime order alone is not build order; inspect
accepted inputs, ownership, isolation and settlement before accepting parallelism.
Review unsupported verification burdens/gates against recorded owner policy;
subjective comparison ratings alone do not block. Report feasibility without
choosing execution mode; retain the existing independent-review disposition.


## Bind the subject and reviewer

Read effective instructions, approved brief/decisions, current source root/base/head
and exact plan/spec bytes initially and after resume. Resolve the stream's
`docs/specs/NN-slug/<stream>-plan.md` and approved spec through parsed `meta.yaml`,
not guessed names. A missing referenced spec blocks coverage claims. An explicit
owner-declined spec is the only plan-only exception: retain `spec: null` and
`spec_declined:`, review internal consistency and disclose unavailable spec coverage.

Only the coordinator admits one read-only reviewer and supplies exact plan AND
spec, requirement/source anchors, relevant input hashes, accepted constraints,
verification evidence and the [plan-reviewer.md]({skillDir}/plan-reviewer.md) rubric.
The reviewer returns complete native text; the coordinator persists the report.
Use actual available native controls/authentication and role restrictions, without
blanket qualification ceremony or model/provider/protocol/CLI fallback.

One reviewer is the default. A second model requires explicit coordinator admission
and any necessary owner/provider consent, not automatic discovery, probing or a
permanent configuration ritual. This skill neither reads nor writes
`.patronus/config/second-model.yaml` (project or user scope) to select reviewers.
Existing files are left untouched; presence is not a launch grant. Retain valid
reviewer choices in the existing brief, without another ledger or repeated consent.
Missing tools/authentication block the selected lane, not an automatic substitute.

## Review and disposition

Apply plan basics (coverage, bite-sized actions, placeholders, type/path consistency,
project idioms and independently testable deliverables), engineering/build order,
user-facing design, developer-facing DevEx and strategy/scope lenses. Skip only
inapplicable lenses and state why; skipped does not mean clean. Check actual source
anchors and interface/predecessor availability, not merely the author's claims.
The ownership bullet also checks the recorded or defaulted `Slicing:` value and
the execution graph: agreement with task text, consistent IDs, no missing edges or
cycles, correct `start`/`join` labels, shared-file hazards resolved by one owner,
serialization or an authorized seam split, no unaccepted owned collaborators
hidden behind mocks, and J/C/R barriers present.

Consume complete findings, preserve source IDs and original severities, and
deduplicate by defect without dropping evidence. Use Critical/Major/Medium/Low by
consequence; never downgrade to fit the threshold. Unresolved Critical/Major or
more than two distinct Medium block. Up to two Medium residuals require explicit
owner acceptance; Low receives recorded disposition. Missing authority,
requirements or tests independently block.

Admit at most one bounded batched correction/disposition per gate by default.
The fixer is a leaf and cannot launch another reviewer. The coordinator verifies
fixed defects against their cited criterion, exact corrected plan bytes and
sufficient requirement/source/verification evidence. Record parent-verified
correction, not independent re-review. Changed public interface, safety invariant,
acceptance scope or insufficient oracle returns to the owner for a decision or
additional independent-review grant. Critical/Major residuals cannot be waived.
No automatic task-review or successor fix/review waves.

## Authority and next action

Only the coordinator selects execution mode after plan acceptance and an execution
grant. Disjoint deliverables need checked isolated owners and settlement before
serial integration; shared-file or ordered tasks run serially. This reviewer
identifies ownership/order defects, but cannot choose mode, launch implementers,
change shared progress or grant execution/commit/publication/cleanup.

Record the sole progress authority once in the accepted brief: qualified native
state that actually owns delegated runs first; without it, explicit project/owner
selection, otherwise the existing tracker or one durable plan-associated record.
UI todos alone do not outrank a tracker. Do not mirror status into tickets, plan
checkboxes or a new ledger. Seed selected tickets once under one owner only when
that tracker is authoritative, respecting actual ignore/tracking policy.

For native host-specific guidance, Codex uses installed
`{skillsDir}/plan-review-cx/SKILL.md` and OpenCode uses
`{skillsDir}/plan-review-oc/SKILL.md`. These are optional own-host routes, not
cross-host requirements or substitutions when an explicitly selected route is
missing. Return review subject/hashes, complete findings/disposition, actual
checks/exits/logs, limitations and next authorized action; leaves then stop.
Existing valid grants survive stage labels. No new root-budget form is required;
applicable existing host/resource or explicitly granted campaign limits still apply.
