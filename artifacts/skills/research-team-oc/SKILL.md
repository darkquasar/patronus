---
name: research-team-oc
description: "OpenCode only: investigate an unknown domain with bounded read-only native subagents and lead-written research synthesis. Use on explicit research request."
---

# Research team for OpenCode

This skill owns research only. Read the approved brief, source root/revision,
project instructions and prior findings before decomposing the unknowns. A folder
is one research effort with many streams; each stream has one spec and one plan.
Use the existing `docs/specs/NN-slug/` folder or obtain authority to create one.
Verify the project's ignored-planning policy; do not edit `.gitignore` without a grant.

## Lead and subagent contract

Read-only subagents return findings to the lead in their final responses. The
lead writes shared outputs, including findings appendices, the single
`<slug>-research.md` synthesis and `meta.yaml`. Researchers never write files in
the shared checkout, including report files. Returned text is the deliverable
for a read-only researcher; a child-session title or Task receipt is insufficient.

1. Survey current source and existing research. Define independent questions,
   source paths, exclusions and evidence requirements. Present the scope and
   outputs for approval if they are not already authorized.
2. Read `{skillsDir}/dispatching-parallel-subagents/SKILL.md`. Dispatch through
   the OpenCode Task tool to the `@patronus-researcher-oc` role; when web access
   is separately authorized, `@patronus-web-researcher-oc` may take the web
   questions. Before launch run `opencode agent list` and confirm each role is
   listed as a subagent with its read-only permission rules. Respect applicable
   existing host/resource or explicitly authorized campaign limits; ordinary
   research needs no numerical root-budget form or cumulative start ledger.
   Admit only useful independent leaf questions. If a role is absent from the
   list or authentication is unavailable, report blocked and stop the lane. Do
   not substitute the built-in `general` or `explore` agents, invoke a CLI
   model, change provider or model, or silently investigate inline.
3. Give each subagent a self-contained question, root/revision, allowed reads,
   no-write/no-nesting rule (native subagent depth is 1), source paths, applicable limits
   and findings format. Name the installed `-oc` skills it must read explicitly
   with the OpenCode `skill` tool. Ask for observations separately from
   inferences, file:line citations or sanitized source URLs, limitations and
   unresolved questions. Web work requires its own approved role and access;
   local evidence grants no network requests.
4. Each Task call returns the subagent's final response when its child session
   ends. Collect every complete final response and validate evidence against
   current source. Empty, missing or unsupported findings block synthesis.
   Preserve partial results on timeout; confirm subagent settlement with the
   controls the session exposes before retry.
5. The lead writes the findings appendices and reconciles contradictions into
   one `<slug>-research.md`. Name that file in `meta.yaml`; keep each new stream's
   `spec: null` and `plan: null`. Preserve existing completed stream entries.
   Every filename named by metadata must exist; every `*-spec.md` and
   `*-plan.md` in the folder must be named in metadata.
6. Report source and output hashes, subagent identities, actual checks,
   limitations and open decisions. Research completion grants no spec or
   implementation stage.

```text
+-------------------+   Task tool, read-only role   +----------------------------+
| OpenCode lead     | ============================> | @patronus-researcher-oc    |
| owns shared files | <============================ | returns findings (final    |
+-------------------+   complete final responses    |  response, no file writes) |
                                                    +----------------------------+
```

Offer `{skillsDir}/spec-writing-oc/SKILL.md` to author a spec from the
validated synthesis, or `{skillsDir}/grilling-oc/SKILL.md` to stress-test unresolved
design choices. Offers require stage authorization; never auto-advance.

This is static OpenCode guidance. Role availability, authentication, native
permission enforcement and runtime handoffs still require qualification for the
selected version/platform.
