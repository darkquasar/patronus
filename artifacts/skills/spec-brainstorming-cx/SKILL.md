---
name: spec-brainstorming-cx
description: "Codex only: explore approved design and evidence, then author an ADR-0003 stream spec before planning."
---

# Spec brainstorming for Codex

Read the task brief, project instructions, recorded decisions, current source and supplied research before designing. After context loss, reread those inputs and the latest saved outputs, checking their hashes before retrying. Prior research is an input, not permission to rerun it.

## Authority and workers

Research, spec authorship, review, planning, implementation, integration, publication and cleanup are separate stage grants. Existing approval remains valid within its recorded root, files and budgets. A changed target, destructive action or unresolved design choice requires an explicit decision from the lead or owner. Stop and ask through the available Codex interaction surface; wait for the reply. Never infer approval from steering or skill completion.

Only the lead writes shared outputs and `meta.yaml`. A read-only worker returns complete findings or draft text to the lead, with source paths and hashes; it does not edit the shared checkout, delegate, commit or change workflow state. The lead checks actual returned bytes before saving them. Use native Codex workers only if present, qualified and separately authorized. Missing capability or authentication blocks that lane. Do not invent another host's API, change model/provider, or substitute concurrent writers.

## Explore and design

- Read the folder's synthesis from `meta.yaml` and all supplied findings first. Clarify intent, constraints, success criteria and non-goals. When evidence already settles a choice, confirm it rather than reopen research.
- For an unknown domain, propose a separately authorized research stage using `{skillsDir}/research-team-cx/SKILL.md`. Consume its one folder-level research synthesis here. Large known work needs decomposition into streams, not a second investigation.
- Compare two or three plausible approaches with tradeoffs, recommend one, and present components, data flow, failure handling and verification at proportional depth. A small ASCII diagram can clarify a consequential seam.
- Obtain decisions only for unresolved choices. Optional `{skillsDir}/grilling-cx/SKILL.md` can stress-test an approved design under its own grant. Spec authorship does not authorize coding.

## Folder contract (ADR-0003)

Use `docs/specs/NN-slug/`: one research effort, one `meta.yaml`, one folder-level research synthesis and one spec/plan pair per stream. A second unrelated research effort gets another folder. A split plan requires a split stream/spec. Preserve project tracking and ignore policy; do not edit `.gitignore` or invent a task ledger.

For a newly authorized effort, inspect existing folder numbers, choose the next `NN` and let the lead create the folder and its sole `meta.yaml`. Starting without research is allowed: leave `research: null` until the one synthesis is written, rather than naming an absent file. Existing efforts reuse their metadata and research; append an independent stream only under its authorship grant.

The lead writes `docs/specs/NN-slug/<stream>-spec.md`. Update only that stream's `spec:` and `updated:` after the authorized file exists. Leave `plan: null` until planning writes it. Name files, never boolean completeness flags. Spec review and plan review answer different questions. If the owner explicitly declines a spec, record `spec: null` and `spec_declined:` with the decision/date, never fabricate a spec.

```yaml
slug: 09-example-effort
intent: "Investigate one invented system"
created: 2026-10-06
updated: 2026-10-06
research: example-effort-research.md
streams:
  - slug: example-stream
    intent: "One independently buildable change"
    spec: example-stream-spec.md
    plan: null
    epic: null
```

`research`, `spec` and `plan` may be null or absent while unfinished. Presence of a filename is the completeness record. Preserve unrelated fields, streams and any project-managed work references. Metadata records artifacts; it grants no stage authority.

## Author and validate

Write requirement IDs, testable acceptance checks, scope/non-goals, source-verified interfaces, failure/rollback behavior, authority boundaries and evidence limits. Self-check missing sections, placeholders and contradictions. A self-check cannot replace an independent review.

Run the bundled model-free validator, described in `{skillDir}/scripts/README.md`, against the absolute folder path:

```sh
(cd "{skillDir}/scripts" && GOPROXY=off GOTOOLCHAIN=local go run -mod=readonly . "$SPEC_FOLDER")
```

Set `SPEC_FOLDER` to the actual absolute `docs/specs/NN-slug/` path before running. Respect the project's resource lock. Go 1.25+ and the pinned YAML module already available locally are prerequisites; do not install or fetch dependencies under this skill. Missing prerequisites block validation, never count as a pass. The validator parses YAML and enforces both invariants: every file named by `research` or `streams[].{spec,plan}` exists, and every `*-spec.md` / `*-plan.md` in the folder is named. Comments are not references.

## Review and handoff

Under a review grant, request fresh independent `{skillsDir}/spec-review-cx/SKILL.md` with the exact spec, brief, research/source inputs and hashes. Include `{skillDir}/spec-document-reviewer-prompt.md` as a supplement, not another review cycle. Read-only reviewers return findings to the lead. The lead writes the report and dispositions material findings within the approved review budget.

After acceptance, `{skillsDir}/plan-writing-cx/SKILL.md` is the planning destination. It must read the written spec and metadata, not infer requirements from a filename. Completion does not automatically start that stage: require a valid planning grant and settled review gate. Return actual file paths/hashes, checks with results, unverified claims, decisions and the next authorized action. Static delivery of these instructions is not native Codex qualification.
