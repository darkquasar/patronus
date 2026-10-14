---
name: plan-execute-parallel-cl
description: Use when explicitly executing an accepted plan with native Claude teammates owning disjoint isolated file boundaries
---

# Parallel Plan Execution

The coordinator selects the execution mode from actual ownership/dependencies,
not a plan-reviewer instruction. Read accepted spec/plan, source anchors and tests.
Existing valid approval remains valid. Do not seed tk or mirror todos unless that
existing tracker is the selected authority and seeding is explicitly in scope.
Respect repository ignore/tracking policy. Qualified native run-owning state wins;
UI todos alone do not replace the project authority.

## Native isolated assignments

Verify actual Agent support for isolation: "worktree", auth/permissions and
wait/stop controls. Fill TEAMMATE-TEMPLATE.md with exact assignments. The coordinator
launches native Agent writers with isolation: "worktree" and records actual returned
agentId, worktree path and worktreeBranch, not guessed names or task owner fields.
Verify root/base/head and disjoint path claims before writes. Overlap requires
serial work or one owner. Preparation never starts dependent work early.
No default five-teammate cap or mandatory root budget form. A CLI supervisor is
used only if separately explicitly selected and authorized, never as native default.

Leaves do not pull a concern queue, select successor tickets or merge peers.
Commit only if separately granted: a checked patch and full report are delivery
without a commit. Only the coordinator integrates settled patches serially under
its grant. Capture exact delivered bytes and read all reports before integration.
Do not auto-merge or discard conflicts. Use PROVENANCE-GUIDE.md for authorized
traceability, not source headers or competing mutable status.

## Lifecycle control

Only the coordinator admits assignments, reviewers, corrections and successors.
Implementers, fixers, researchers and reviewers are leaves: complete the assigned
subject, return complete evidence and stop. No nested agents, scope expansion or
shared progress mutations. An exact coordinator-authorized deterministic worktree
allocation is setup only, not permission to choose roots or launch agents.

Use the sole progress authority already selected in the brief, not a new ledger,
ticket mirror or todo system. Read effective instructions, brief, current source
and latest handoff initially and after resume. Do not replay settled work.
No mandatory numerical root start/concurrency/deadline budget is required.
Existing host/resource limits and explicitly authorized campaign limits apply.

Verify actual native tools, permissions/authentication and wait/stop controls.
Missing tools block that lane, without provider/model switching or silent mode
fallback. Read complete reports and checked patches after settlement; status or
branch existence is not acceptance. Complete returned text is valid delivery when
write tools are unavailable; the coordinator persists it, without missing-file
redispatch. Unknown liveness/capture retains allocations and blocks dependents.

Critical auth/secrets, ownership, concurrency/isolation/settlement, migration/
removal and reproducible bugs retain intended public-interface RED, minimal
GREEN and negative/legacy regressions through tdd-cl. Characterize uncertain
legacy before refactoring. Routine prose/manifests use focused postchecks;
other behavior needs an explicit strategy. No manufactured prose-substring reds.
Separate expensive check admission from writer concurrency; account for nested
parallelism, isolate mutable outputs and reuse compatible caches.

## Review and correction

No automatic per-task reviewer, repeat-until-approved or per-finding fixer chain.
An earlier independent review requires a named changed safety/compatibility
invariant or irreversible boundary before dependent consumption. Task count,
framework use or weak verification alone creates no gate. This distinct boundary
review never replaces the one independent aggregate implementation review.

After settled work, run fresh composed final checks, including integrated consumers
and requirement coverage. Green leaf checks cannot waive aggregate failure.
The coordinator admits one aggregate review through requesting-code-review-cl
and ../requesting-code-review-cl/code-reviewer.md, bound to exact root/base/head,
dirty/untracked inputs, requirements, diff/bytes and actual retained evidence.
Consume complete findings and source IDs. Admit at most one batched correction/
disposition, not one fixer per finding. Fixers cannot launch reviewers.
Unresolved Critical/Major or more than two Medium block; at most two Medium
residuals need explicit owner acceptance. Low findings receive disposition.

Verify corrected exact bytes against the cited defect and affected positive/
negative/regression evidence. Label corrections parent-verified, not independently
re-reviewed. Changed public interface, safety invariant or acceptance scope, or
an insufficient oracle, returns to the owner for a decision/additional grant.
Do not automatically launch another review wave.

## Completion

Settle live workers on stop and preserve partial evidence. Report changed files,
actual checks/options/exits/logs, input identities, disposition, limitations and
next authorized action. Matching evidence can satisfy unchanged-input finishing;
final aggregate checks stay fresh. Commit, integration, publication and destructive
cleanup remain separate grants. No automatic merge, push, branch deletion or
worktree removal.

```text
+-------------+   native isolated assignment   +-------------+
| Coordinator | =============================> | Leaf writer |
+-------------+ <============================= +-------------+
       |              evidence and settlement
       v
+-------------+   composed exact bytes         +-------------+
| Integration | =============================> | One review  |
+-------------+                                +-------------+
```

$ARGUMENTS
