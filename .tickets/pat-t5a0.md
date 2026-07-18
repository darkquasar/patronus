---
id: pat-t5a0
status: open
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
