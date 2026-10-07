---
name: research-team-cx
description: "Codex only: investigate an unknown domain with bounded read-only workers and lead-written research synthesis. Use on explicit research request."
---

# Research team for Codex

This skill owns research only. Read the approved brief, source root/revision,
project instructions and prior findings before decomposing the unknowns. A folder
is one research effort with many streams; each stream has one spec and one plan.
Use the existing `docs/specs/NN-slug/` folder or obtain authority to create one.
Verify the project's ignored-planning policy; do not edit `.gitignore` without a grant.

## Lead and worker contract

Read-only workers return findings to the lead in their final responses. The
lead writes shared outputs, including findings appendices, the single
`<slug>-research.md` synthesis and `meta.yaml`. Workers never write files in the
shared checkout, including report files. Returned text is the deliverable for a
read-only worker; a launch receipt or completion notification is insufficient.

1. Survey current source and existing research. Define independent questions,
   source paths, exclusions and evidence requirements. Present the scope and
   outputs for approval if they are not already authorized.
2. Read `{skillsDir}/dispatching-parallel-agents-cx/SKILL.md`. Use only the
   Codex native worker primitive actually exposed in the current session, with
   an observed read-only role. Bind maximum concurrency (start with at most two,
   never more than four), total worker count, deadline and retry budget before launch.
   If the primitive or authentication is unavailable, report blocked and stop the lane.
   Do not invoke a CLI model, replace the engine or silently investigate inline.
3. Give each worker a self-contained question, root/revision, allowed reads,
   no-write/no-nesting rule, source paths, budget and findings format. Ask for
   observations separately from inferences, file:line citations or sanitized
   source URLs, limitations and unresolved questions. Web work requires its own
   approved tools and access; local evidence grants no network requests.
4. Wait for every assigned worker to finish through the available Codex wait
   mechanism. Collect the complete final responses and validate evidence against
   current source. Empty, missing or unsupported findings block synthesis.
   Preserve partial results on timeout; confirm worker settlement before retry.
5. The lead writes the findings appendices and reconciles contradictions into
   one `<slug>-research.md`. Name that file in `meta.yaml`; keep each new stream's
   `spec: null` and `plan: null`. Preserve existing completed stream entries.
   Every filename named by metadata must exist; every `*-spec.md` and
   `*-plan.md` in the folder must be named in metadata.
6. Report source and output hashes, worker identities, actual checks, limitations
   and open decisions. Research completion grants no spec or implementation stage.

```text
+-------------------+   native read-only launch   +-------------------+
| Codex lead        | =========================> | Codex workers     |
| owns shared files | <========================= | return findings   |
+-------------------+   complete final responses +-------------------+
```

Offer `{skillsDir}/spec-brainstorming-cx/SKILL.md` to author a spec from the
validated synthesis, or `{skillsDir}/grilling-cx/SKILL.md` to stress-test unresolved
design choices. Offers require stage authorization; never auto-advance.

This is static Codex guidance. Native worker availability, authentication and
runtime handoffs still require qualification for the selected version/platform.
