---
id: pat-t5a0
status: closed
deps: []
links: []
created: 2026-07-18T11:27:03Z
type: task
priority: 2
assignee: darkquasar
tags: [lifecycle]
---
# Re-compose agent-rules into built CLAUDE.md after the antipattern-heuristic edit


## Notes

**2026-07-18T11:28:02Z**

CLAUDE.md is a COMPOSED file (has <!-- patronus:start agents-spine --> markers) — deployed by patronus install, not hand-edited. So the agent-rules heuristic reaches it only via re-deploy: patronus install agent-rules --tool claude (whichever scope). NOTE: the drift guard does NOT currently flag this as STALE because instructions are APPEND-composed (a MERGE fold of many sources into one file); pass 1 byte-compares the deployed file against ONE source's would-be bytes, which won't match a multi-source fold. Composed-file drift is a known gap, separate from pat-d2db's CREATE-file scope.

**2026-07-19T00:04:14Z**

DONE. Re-composed agent-rules into the deployed CLAUDE.md via 'patronus install agent-rules --tool claude --local --deploy'. CLAUDE.md is UNTRACKED (a local deploy artifact, gitignored), so this is a safe local re-fold with no repo diff. VERIFIED: the 'simplest solution that does not become an antipattern' tie-breaker (commit e6ba7fa) now appears in the deployed agent-rules section, and the composed-file drift guard (pat-wkp3) confirms agent-rules is no longer STALE. Also re-folded the 'ticket' section (likewise stale since pat-06p3). CLAUDE.md now has zero STALE sections. NOTE (out of scope, harmless, left as-is): 'MISSING beads' persists — a dead state row for the beads section that 793c06b replaced with ticket; 'patronus remove beads --local' would clear the orphaned row (it SKIPs, the section is already gone). That's state hygiene, not this ticket.
