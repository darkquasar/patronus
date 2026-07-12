---
id: pat-zr8z
status: closed
deps: []
links: []
created: 2026-07-12T04:46:52Z
type: epic
priority: 0
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
tags: [test-surface]
---
# Test surface — unbind the suite from the catalog's contents

Stream 'test-surface' of docs/specs/01-lifecycle-and-test-surface. Plan: test-surface-plan.md (13 tasks). STRICT ORDER: fixture catalog -> migrate Class-A file-by-file -> delete stubs -> ONLY THEN close the archive-SKIP hole (step 3 is what BREAKS stubBinary). Groups only; tk ready reads deps, never parent.

## Acceptance Criteria

Bumping a pin in recipes/tk.yaml breaks 0 tests (today: ~30); a tampered archive binary at the dest is DETECTED


## Notes

**2026-07-12T04:49:12Z**

NOT ACTIONABLE — this is a GROUPING epic. It appears in 'tk ready' because ready/blocked read ONLY deps, never parent (an epic has no deps, so it always looks ready). That inertness is documented, not a bug: epics group and display; tk dep orders. Work the children.

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/test-surface-plan.md (13 tasks). SPEC: test-surface-spec.md. See also docs/adr/0002-tests-assert-behavior-not-artifacts.md (COMMITTED — readable in any clone). ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T08:56:48Z**

STREAM COMPLETE. All 13 plan tasks done, every commit green (gofmt + go vet + golangci-lint + go test -race). Acceptance VERIFIED against the artifacts: (1) pinning recipes/tk.yaml to a bogus sha256 breaks 0 tests (was ~30) — full suite still green; (2) a tampered archive binary is DETECTED — restoring the old 'return diff.Skip' branch makes TestTamperedArchiveBinaryIsRefetched fail with 'TAMPERED BINARY SURVIVED'; (3) no third-party bytes in the repo (testdata/tk deleted); (4) no test can reach the network (deny-all TestMain — which CAUGHT a real live-network leak in remove_test.go); (5) the coreSkills product guarantee is still LIVE (commenting grilling out of profiles/core.yaml fails 4 assertions). DEVIATION FROM PLAN, approved by the user: the 4 profile tests were Class B, not Class A — they assert the real catalog's CONTENTS. Following the plan literally would have made them green tautologies. They keep their real names and now assert via profile.Resolve / the lock / the plan, which read the catalog's SHAPE and never its PINS. Deploy MECHANICS moved to the fixture, where the FETCH path actually RUNS (stubBinary skipped it entirely) — coverage increased. Follow-ups filed: pat-resg (the install plan under-reports settings.json merges — a consent defect), pat-294n (skills-heartbeat's 1%-rule text needs a home in the Class-C gate).
