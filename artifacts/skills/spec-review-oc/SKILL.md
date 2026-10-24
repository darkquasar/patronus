---
name: spec-review-oc
description: "OpenCode only: independently review the exact stream spec against research and source evidence before planning, read-only."
---

# Spec review for OpenCode

Use the owner's selected destination and established project layout throughout.
Any `docs/specs/NN-slug/` path below is a default example, not a location
requirement; preserve the selected effort's metadata and stream relationships.

Use after `{skillsDir}/spec-writing-oc/SKILL.md`, before `{skillsDir}/plan-writing-oc/SKILL.md`. Read `{skillDir}/spec-reviewer.md`. Review checks whether this is the right thing to build; plan review checks whether the implementation promise is buildable.

## Semantic review boundary

Check necessity and sufficiency of outcomes, true constraints, failure guarantees
and structural goals for planning; flag unsupported private design and missing
behavior, DDD consultation/applicability or consequential owner decisions. Preserve
justified public technical detail and selective modeling. Review verification
burden/gates against owner provenance without inventing new thresholds. The rubric
contains the full contract; subjective comparison ratings or unavailable runs
alone create no blocker. Existing objective checks and disposition rules remain.


## Scope and inputs

Require a review grant, exact task brief, project instructions, decisions, source root/revision, spec and research paths/hashes, constraints and test evidence. After context loss, reread inputs and the latest saved report before retrying. Existing approval covers only its recorded root, files, actions and budgets. Ask the lead or owner about material ambiguity and wait for an explicit answer. Review grants no authorship, execution, Git, integration, publication or cleanup authority.

Preserve the agreed effort layout: `docs/specs/NN-slug/`, one research effort, one `meta.yaml`, one folder-level research synthesis and one spec/plan pair per stream. Read parsed metadata and the selected stream's actual spec. Check every named file exists and every `*-spec.md` / `*-plan.md` in the folder is named. Use available project tooling for metadata/file checks; no bundled validator or particular language is required. Report actual checks and any unverified claims. A missing spec or explicitly declined spec blocks spec coverage claims. Do not fabricate it or fill a completeness flag.

## Independent reader

The lead requests a fresh-context read-only OpenCode reviewer (the `@patronus-technical-reviewer-oc` role in spec mode, through the Task tool) only when `opencode agent list` shows that role and authentication is present and authorized. Give exact inputs and constraints, not the author's reasoning transcript. If unavailable, report the independence gate blocked; an explicitly authorized inline self-check must be labelled non-independent and cannot satisfy required independent review. Do not switch model/provider or invent a foreign-host API.

Read-only workers return complete findings to the lead. They do not edit specs, shared checkout files, `meta.yaml`, work records or the index; they do not delegate or install anything. Only the lead writes shared outputs, reads the saved bytes/hashes, verifies consequential claims against current source and records disposition. A path, exit code or completion message alone is not acceptance.

## Disposition and next stage

Findings carry ID, severity, location/quote, evidence, consequence and proposed correction. Normalize by consequence: Critical (catastrophic security/data loss), Major (substantial requirement/correctness/safety failure), Medium (bounded material defect), Low (presentation/low impact). Preserve original reviewer IDs/severities when deduplicating. Never lower severity to pass a threshold.

Use one fresh independent review and at most one bounded batched correction/disposition per exact subject. Unresolved Critical/Major findings block; up to two distinct Medium residuals require explicit owner acceptance, and more than two block. Missing requirements, authority, evidence or tests independently block. Only the lead/owner dispositions findings and admits transitions. The coordinator may resolve a fixed defect only after checking its cited failure against the exact corrected bytes and sufficient relevant positive, negative-path and regression evidence for the existing acceptance criterion. Retain this disposition in the original report as parent-verified, not independently re-reviewed. A changed public interface, safety invariant or acceptance scope, or an insufficient oracle, returns to the owner for a design decision or additional independent-review grant. Blockers remain blockers at the correction cap.

Return source/root/revision and input hashes, criteria checked, actual commands/results, skipped lenses, findings, risks and report bytes. After accepted findings and a valid planning grant, hand off to `{skillsDir}/plan-writing-oc/SKILL.md`. Finishing review never starts planning automatically.
