---
name: spec-review-cx
description: "Codex only: independently review the exact ADR-0003 stream spec against research and source evidence before planning, read-only."
---

# Spec review for Codex

Use after `{skillsDir}/spec-brainstorming-cx/SKILL.md`, before `{skillsDir}/plan-writing-cx/SKILL.md`. Read `{skillDir}/spec-reviewer.md`. Review checks whether this is the right thing to build; plan review checks whether the implementation promise is buildable.

## Scope and inputs

Require a review grant, exact task brief, project instructions, decisions, source root/revision, spec and research paths/hashes, constraints and test evidence. After context loss, reread inputs and the latest saved report before retrying. Existing approval covers only its recorded root, files, actions and budgets. Ask the lead or owner about material ambiguity and wait for an explicit answer. Review grants no authorship, execution, Git, integration, publication or cleanup authority.

Preserve ADR-0003: `docs/specs/NN-slug/`, one research effort, one `meta.yaml`, one folder-level research synthesis and one spec/plan pair per stream. Read parsed metadata and the selected stream's actual spec. Check every named file exists and every `*-spec.md` / `*-plan.md` in the folder is named. The checker is documented at `{skillsDir}/spec-brainstorming-cx/scripts/README.md`; run it only under authorized host checks and resource limits. A missing spec or explicitly declined spec blocks spec coverage claims. Do not fabricate it or fill a completeness flag.

## Independent reader

The lead requests a fresh-context read-only Codex reviewer only when native worker capability and authentication are present, qualified and authorized. Give exact inputs and constraints, not the author's reasoning transcript. If unavailable, report the independence gate blocked; an explicitly authorized inline self-check must be labelled non-independent and cannot satisfy required independent review. Do not switch model/provider or invent a foreign-host API.

Read-only workers return complete findings to the lead. They do not edit specs, shared checkout files, `meta.yaml`, work records or the index; they do not delegate or install anything. Only the lead writes shared outputs, reads the saved bytes/hashes, verifies consequential claims against current source and records disposition. A path, exit code or completion message alone is not acceptance.

## Disposition and next stage

Findings carry ID, severity, location/quote, evidence, consequence and proposed correction. Normalize by consequence: Critical (catastrophic security/data loss), Major (substantial requirement/correctness/safety failure), Medium (bounded material defect), Low (presentation/low impact). Preserve original reviewer IDs/severities when deduplicating. Never lower severity to pass a threshold.

Use one fresh independent review and at most one bounded correction/disposition under the recorded budget. Unresolved Critical/Major findings block; up to two distinct Medium residuals require explicit owner acceptance, and more than two block. Missing requirements, authority, evidence or tests independently block. Only the lead/owner dispositions findings and admits transitions. Corrected bytes need explicit disposition; no automatic review wave.

Return source/root/revision and input hashes, criteria checked, actual commands/results, skipped lenses, findings, risks and report bytes. After accepted findings and a valid planning grant, hand off to `{skillsDir}/plan-writing-cx/SKILL.md`. Finishing review never starts planning automatically. Static file placement is not a native Codex readiness claim.
