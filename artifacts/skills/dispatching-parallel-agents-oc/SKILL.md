---
name: dispatching-parallel-agents-oc
description: "OpenCode only: bounded read-only parallel investigation of independent questions using the native Task tool and the Patronus read-only subagent roles."
---

# Bounded OpenCode dispatch

Only the lead dispatches. A worker never nests delegation: OpenCode's native
subagent depth is 1 by default (a primary agent may spawn subagents, a subagent
may not), which matches the spine rule. Use parallel reads when questions are
independent and evidence can be checked separately. Shared state, causal
dependencies or unresolved common design decisions require a lead ordering
decision before launch. Dispatch is subordinate to stage authority.

## Admission and brief

Read the approved brief, actual root/base/head, effective project instructions,
prior outputs and ownership before dispatch. Read-only fanout goes to the
`@patronus-researcher-oc` role through the OpenCode Task tool; when web access
is separately authorized, `@patronus-web-researcher-oc` may take the web
questions. Before launch run `opencode agent list` and confirm each role is
listed as a subagent with its read-only permission rules (`edit: deny`,
`external_directory: deny`, bash limited to inspection commands). If a role is
absent from that list, stop the lane as blocked; do not substitute the built-in
`general` or `explore` agents, invoke a CLI model request, change provider or
model, or fall back to inline investigation without lead authorization.
Authentication and approval are inspected, never assumed; permission `ask`
outcomes are the user's, and `--auto` is not an authorization step.

Bind maximum concurrency, total worker count, timeout, retry budget and resource
reserve before starting. Start with at most two lightweight read-only subagents;
raise concurrency only within measured capacity and the recorded budget.
Serialize expensive tests under the project lock. Retry requires confirmed
settlement, retained failed evidence and explicit remaining budget.

Each Task brief contains: one question, exact root/revision and allowed source
paths, relevant instructions, exclusions, the no-write/no-nesting rule, evidence
criteria, output format and deadline. Name installed `-oc` skills to read
explicitly with the OpenCode `skill` tool rather than relying on description
matching. Workers return complete findings in their final response; the lead
writes shared outputs. Workers must not edit report files, source,
index/branches, graphs, caches or shared service configuration. Read-only intent
is cooperative scope plus the role's permission rules, not an operating-system
sandbox.

## Collect and settle

Each Task call returns the subagent's final response when its child session
ends; read every complete response, not a tool receipt or a child-session title.
Verify consequential claims against current source, bind input and output
hashes, and reconcile contradictions before synthesizing. Empty or missing
results block acceptance; preserve partial evidence on failure.

If a subagent exceeds its deadline, stop it only with the controls the session
exposes and confirm settlement before retry, validation or cleanup. Unknown
liveness retains outputs and allocations for the lead. Do not relaunch a live
worker or extend budgets.

Writing requires separately approved isolated owners, disjoint path claims and
confirmed settlement before serial integration. Native subagents share the lead
checkout and are not proof of write isolation: at most one native writer in a
shared cwd, never concurrent writers in the lead checkout, even on disjoint
files. Isolated parallel writers go through
`{skillsDir}/plan-execute-parallel-oc/SKILL.md`, which is Partial until native
OpenCode workers are qualified.
