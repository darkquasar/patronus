---
name: plan-writing-oc
description: "OpenCode only: write a requirement-covered ADR-0003 implementation plan from an approved stream spec, before execution."
---

# Plan writing for OpenCode

Read the approved spec, task brief, decisions, project instructions and actual source seams first. After context loss, reread inputs and the latest saved outputs, verify their hashes and reconcile completed work before retrying. Plan writing authors plans, not replacement specs or production code.

## Stage grant and spec gate

Research, authorship, review, planning, execution, integration, publication and cleanup have separate grants. Continue within existing approval's root, files and applicable resource limits; ask the lead or owner for material unresolved choices or changed targets and wait for the reply. Skill completion is not permission to execute, commit or publish.

Find this stream's spec through parsed `meta.yaml` and read the actual file. No spec named or a missing referenced file blocks coverage claims: route to `{skillsDir}/spec-writing-oc/SKILL.md`. Do not derive a spec from research or reverse-engineer it from an orphan plan. An explicit owner-declined spec is the only plan-only exception; preserve `spec: null` and `spec_declined:` and disclose that coverage against a spec cannot be checked. If a plan already exists, obtain revision/supersession authority before editing reviewed or user-modified bytes.

## Folder contract (ADR-0003)

Use `docs/specs/NN-slug/`, one research effort with one `meta.yaml`, one folder-level research synthesis and one spec/plan pair per stream. Multiple independent changes share the research but get separate stream/spec/plan pairs. A stream's plan is `docs/specs/NN-slug/<stream>-plan.md`, co-located with `<stream>-spec.md`. Follow project tracking policy without editing `.gitignore` or mirroring work into another ledger.

Only the lead writes shared outputs and metadata. A read-only OpenCode subagent (the `@patronus-researcher-oc` role through the Task tool) returns complete draft text or findings, source anchors and hashes to the lead; it never writes shared files, changes workflow state or delegates. The role must be listed by `opencode agent list` with its read-only permission rules, and available authentication, before coordinator-admitted use. Missing capability blocks the lane without provider/model/host substitution.

After saving the plan, the lead updates only this stream's `plan:` and `updated:`. `plan: null` stays until the file exists. Preserve `research:`, other streams, `spec:` and recorded work references. Name the file, never `plan: true`. Validate both directions: every named research/spec/plan file exists and every `*-spec.md` / `*-plan.md` in the folder is named. Run the parsed-YAML checker documented at `{skillsDir}/spec-writing-oc/scripts/README.md` under authorized project checks:

```sh
(cd "{skillsDir}/spec-writing-oc/scripts" && GOPROXY=off GOTOOLCHAIN=local go run -mod=readonly . "$SPEC_FOLDER")
```

`SPEC_FOLDER` is the actual absolute effort folder. Go 1.25+ and cached pinned YAML dependency are required; missing prerequisites block validation, and this skill grants no installation.

## Write a buildable plan

Start with the goal, source spec/hash, architecture, tech stack and exact global constraints. Map every requirement ID to a task and acceptance check. Each task names owned files/source anchors, consumed and produced interfaces, predecessor dependencies and an independently testable deliverable. Fold scaffolding and docs into the deliverable that needs them. Split only where ownership and evidence can be independent.

Use the installed tdd-oc defaults and record strategy/rationale in the existing
brief/handoff. Explicit test-first requests and required project checks govern.
Critical auth/secrets, ownership, concurrency/isolation/settlement, migration/
removal and reproducible behavioral bugs require public-behavior TDD. Other
features/behavior changes default to TDD; a different strategy needs a concrete
reason and owner agreement unless already approved. Use `critical-tdd` for
short test-first slices: invented-data intended RED, minimum GREEN and relevant
negative/legacy regressions. Pure behavior-preserving refactors work from green
coverage and characterize uncertain legacy behavior first. Nonbehavioral prose/
manifests/mechanical wiring without a changed safety invariant use
`focused-postcheck`, not manufactured reds or prose-substring tests. Omission
alone applies these defaults and records selection, without another approval or
ledger; genuine classification, conflict or exception ambiguity needs owner
direction. Keep existing safety tests. Tests do not admit successor reviewers.

Include real code/signatures and known commands/results, never vague "handle edge cases" steps or references to undefined interfaces. Record unknown decisions as blockers instead of inventing approved code. Commits are optional steps only under a separate Git grant.

Include negative paths, migration/rollback, ownership, applicable host/resource limits and a durable handoff. Retain required fresh independent whole-change implementation review even for small deliveries; only the owner may explicitly waive it. Earlier boundary OpenCode review requires a named changed safety/compatibility invariant or irreversible boundary with its own acceptance point and consequence. Using an existing shared framework alone is insufficient. A saved local plan is not evidence that another checkout can read it. The lead binds source/base, output paths, authority and dependencies in the existing project work record; there is no mandatory foreign-host mission or ticket API here.

## Progress authority

At plan acceptance the coordinator records the sole mutable progress authority
once in the existing plan/brief. Native state that actually owns delegated runs
remains authoritative, even when a project tracker is selected. Otherwise use
the explicit project/owner selection, then the existing tracker, or one existing
durable plan-associated record. A UI todo list alone does not outrank the tracker.
Specs/plans hold intent, metadata indexes artifacts and logs retain observations;
do not mirror progress into checklists, native todos, tk or another ledger.
Respect actual ignore/tracking policy; seed selected tickets once under one owner,
not automatically. Recovery reads this authority and retained results without
replaying settled work.

## Review and handoff

Check requirement coverage, placeholders and name/type consistency inline;
do not dispatch an author self-check. Every written plan, including a single-task
plan, needs one independent plan review unless the owner explicitly declines
that subject and the decision is retained in the existing brief. Task count
neither skips the review nor adds another reviewer. Under the review grant,
request one fresh independent `{skillsDir}/plan-review-oc/SKILL.md` with the exact plan AND spec, source evidence, checks and hashes. Inline author checks cannot replace required independence. No automatic second
model, discovery loop or repeated consent for already valid grants. Read-only reviewers return complete findings to the lead; the lead writes the report and dispositions them.

Review acceptance does not authorize implementation. Once the gate is settled and execution is granted, the coordinator selects the
execution mode, not the reviewer. Use `{skillsDir}/plan-execute-oc/SKILL.md` for ordered/shared-file tasks. Disjoint writers use `{skillsDir}/plan-execute-parallel-oc/SKILL.md` with native subagents and checked allocated worktrees. Separate authoring dependencies from integration barriers, and let read-only scouts prepare later waves while admitted writers work. If the plan lacks a usable split, ask whether to decompose the first wave, reusing any existing decomposition grant. No CLI supervisor or extra tracker is required; native delegation alone does not prove isolation. Return saved file hashes, actual validation, decisions, residuals and the next authorized action.
