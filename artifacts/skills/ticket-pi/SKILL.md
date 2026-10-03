---
name: ticket-pi
description: "Verify delivered tk identity before coordinator mutations"
---

# Ticket: cooperative Pi work graph

This opt-in legacy skill is not selected by Pi core and is not its task authority. Do not mirror upstream-only workflow state into tickets. Generic/non-Pi tk delivery remains unchanged.

The unchanged global tk recipe delivers the pinned script; this skill does not install, intercept or technically prevent ticket writes. The coordinator alone mutates tk, shared status and authorized lessons. Leaves return findings and proposed state changes. Do this preflight initially, after resume/compaction and whenever the executable, recipe, qualification record or root changes.

## Delivered identity preflight (before mutations)

1. Read the current task grant, durable ledger and qualification (Q) record. Obtain the approved canonical executable path from the owned global recipe/receipt, normally `$HOME/.patronus/bin/tk`. PATH discovery is diagnostic only: never run a shadowing PATH command. Resolve the actual file to its canonical approved location, inspect ownership and reject unexpected symlink/redirection, missing or non-executable/nonregular bytes.
2. Read/hash the file WITHOUT executing it. Match the Q record and selected recipe: recipe 1.0.0; upstream v0.3.2, commit `266b9d38d5090f1af4e5184afd21b183b3bbe878`; SHA-256 `408f2c113ecc3bc071507593a78386f1b4cc743be6491c9e9f2627efd4d9902b`. Version identity comes from this pinned source/digest, not an invented `--version` flag. Revalidate current Q evidence and platform (bash plus required awk/sed/date on qualified Linux/Darwin); old candidate evidence is not an operational pass.
3. Record canonical path, recipe/version/commit/digest, Q record hash, owner, checked time and task authority. Missing/wrong identity blocks all ticket mutations. Do not download, chmod, reinstall, use another tk or silently proceed. A matching digest proves integrity, not authority. Invoke only the approved absolute executable after successful preflight, within coordinator scope.

## Work graph loop

One ticket is one verifiable outcome. Coordinator creates one epic per plan to GROUP, one task per independent deliverable, with priority, concern tags, acceptance check, exact plan FILE and verbatim section heading, expected file ownership and a note when that source is ignored/local. Record actual generated IDs in shared metadata only under its write/seeding grant; names are descriptive, never keys. `tk dep <task> <prerequisite>` encodes order: epics/parent links do not schedule. Verify each pointer against actual bytes; never point only at a directory or invent IDs.

After identity preflight, `tk ready` and `tk blocked` orient; `tk show <id>` and `tk ls` inspect. `tk status` is a SETTER, not a report. Coordinator uses create/start/add-note/close/reopen/dep only within scope. Before close read the exact accepted artifact and fresh checks; record durable acceptance and reconcile tickets on resume, never infer completion from ticket status. Commit tracking files only under an explicit Git grant. No leaves claim ready tickets themselves.

## Explicit limited fallback

If tk is absent/wrong, report the exact identity failure. Only an explicit owner-approved Markdown fallback allows the stated non-ticket continuation using [references/markdown-ledger-template.md](references/markdown-ledger-template.md). Record issuer, scope, expiry/revalidation and limitations; coordinator-only writes still apply. A fallback never repairs the executable, omits required global delivery or qualifies this legacy ticket lane as operational. Only this explicitly selected legacy ticket lane remains blocked until its dependency is delivered/verified and operational gates pass. Pi core uses upstream native state without this dependency. Do not use in-context TODOs as the sole recovery record.
