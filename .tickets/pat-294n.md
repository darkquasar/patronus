---
id: pat-294n
status: closed
deps: []
links: []
created: 2026-07-12T08:34:26Z
type: task
priority: 2
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
tags: [test-surface]
---
# Assert the vendored hook scripts' load-bearing text (skills-heartbeat's 1% rule)

Dropped from reground_hooks_integration_test.go during the pat-k1h1 migration. The OLD test proved it by deploying --profile core and EXECUTING the placed script — which is no longer possible offline (core wires the gitleaks/tk recipes; see pat-ojb0). The script's BEHAVIOR (enumerate the skills dir, emit JSON) is now proven on the fixture's own hook script. What is NOT covered any more is the real skills-heartbeat.sh's load-bearing TEXT: the '1%' dispatch rule that is the artifact's entire purpose. Natural home: the Class-C catalog validity gate (pat-udo1) — read each hook artifact's script via the catalog (registry.NewLocalRegistry(root).Catalog) and assert its content, rather than a relative ../.. path from the test.

## Acceptance Criteria

A test fails if skills-heartbeat.sh stops emitting the 1% skill-dispatch rule or stops enumerating ~/.claude/skills


## Notes

**2026-07-18T23:49:51Z**

DONE. Added TestSkillsHeartbeatScriptCarriesLoadBearingText to internal/registry/catalog_integrity_test.go (the Class-C catalog validity gate). Reads the real skills-heartbeat.sh THROUGH the catalog (NewLocalRegistry(repoRoot).Catalog() -> find artifact by name -> Source.LocalDir + Hook.Script), not a ../.. relative path. Asserts the two load-bearing pieces: the '(even a 1% chance)' dispatch rule and the '${HOME}/.claude/skills' enumeration, each with a why-message. VERIFIED BY BREAKING IT: sed'd out the 1% wording -> test FAILED with the right message; restored -> PASS. Four gates green (gofmt, vet, golangci-lint 0 issues, go test -race).
