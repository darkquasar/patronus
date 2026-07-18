---
id: pat-d2db
status: closed
deps: [pat-q360, pat-il8m]
links: []
created: 2026-07-12T04:48:50Z
type: task
priority: 0
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-54y7
tags: [lifecycle, security]
---
# Wire the drift guard into patronus scan (two passes)

Plan Task 9 (R7). Files: cmd/patronus/{scan.go,scan_test.go}, internal/render/render.go. NOTE: runScan does not exist — add it (mirror runInstall, install_test.go:20-29). TWO PASSES, and the second IS the point: pass 1 (state -> disk) catches STALE/USER-EDITED/MISSING/ORPHANED; pass 2 (catalog -> disk) catches the UNMANAGED SHADOW — you CANNOT find it by walking state.json, because BY DEFINITION it has no state row. That is exactly how the stale skill hid. Do NOT hand-roll the deploy-path resolution: install already computes it — call it. Two implementations of 'where does this item land' is precisely the divergence this plan is about. Its test uses fixtureRegistry -> depends on the test-surface stream.

## Acceptance Criteria

patronus scan NAMES an unmanaged shadow file on disk — verified by planting one and seeing the verdict fire, not by a unit test


## Notes

**2026-07-12T04:49:02Z**

CROSS-STREAM DEPENDENCY: TestScanReportsDrift uses fixtureRegistry from the test-surface stream (pat-il8m). It is a Class-A test — it asserts Patronus's BEHAVIOR, not the catalog's contents — so it must land on the fixture, not the real catalog. If you must start before the fixture exists, use builtRegistry as a STOPGAP and migrate it when pat-il8m closes; do not leave it bound to the real catalog.

**2026-07-12T04:55:35Z**

PLAN: docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → 'Task 9: Wire the drift guard into patronus scan' — that section carries the exact files, the code, and the step-by-step. ⚠️ docs/specs/ is GITIGNORED: this path exists only in a working tree that has it. If it is absent, the plan was never shared — ask before improvising.

**2026-07-12T04:56:13Z**

PLAN SECTION (verbatim heading): docs/specs/01-lifecycle-and-test-surface/lifecycle-skills-plan.md → "## Task 9: Wire the drift guard into `patronus scan`". It carries the exact files, the code, and the step-by-step. NOTE: docs/specs/ is GITIGNORED — this path exists only in a working tree that has it. If it is absent the plan was never shared; ask, do not improvise.

**2026-07-12T10:18:13Z**

DESIGN (found by reading before writing): pass 2 must NOT hand-roll path resolution — and it does not need to. plan.Compute (internal/plan/compute.go:38, the SAME fn install drives via computePlan) already returns []diff.FileDiff{Path, After, Artifact}: Path = where Patronus WOULD deploy, After = the source bytes it WOULD write. That is BOTH inputs drift.Classify needs for pass 2, from the one existing chokepoint. Better than the plan's sketch, which implied resolving the path and reading the source separately (two chances to diverge). adapter.Engine.resolvePath stays private; no new export needed.

**2026-07-18T11:12:33Z**

DESIGN (③ materialize-on-occupancy, user-approved). Reading the code killed ①: a skill deploys SKILL.md PLUS a copyTree subtree, so deploy paths are DATA-DEPENDENT — no pure path-only fn exists, and a DeployPaths reusing resolvePath would silently miss subtree shadows (exactly where a stale skill hides). Paths must come from Transform against a real source. BUT the deploy DIRECTORY is name-only (ResolveMarker(template,tool,scope) is pure). So pass 2: resolve each artifact's target dir from its NAME, materialize+Transform ONLY the artifacts whose dir is OCCUPIED on disk and not installed. Clean machine fetches nothing (honors materializeSelected's 'don't download to browse'). One source of truth (Transform), no parallel path-deriver. Also reorder command/agent/outputstyle transforms (resolvePath before ReadFile) — behavior-preserving, the path never depends on the body; pure debt-paying.

**2026-07-18T11:23:23Z**

TWO BUGS found by running against the REAL machine (Step 6), not the fixture:
BUG 1 (scope-blind shadow): pass 2 skips an artifact when installed[name] is true, but 'installed' is name-only. brainstorming is installed at LOCAL scope, so a hand-planted shadow at its GLOBAL path (~/.claude/skills/brainstorming) is MISSED — pass 2 skips the whole name. Fix: gate pass 2 on the specific recorded PATHS (the 'recorded' set pass 1 already builds), not on the artifact name. A path pass 1 didn't record is still a shadow even if the same artifact is installed elsewhere.
BUG 2 (recipe rows in the artifact spine): pass 1 feeds ALL state item names into deployDiffs -> plan.Compute, but recipes (beads/tk/gitleaks/ccusage/context7/tdd-guard) are NOT artifacts -> 'unknown artifact' warnings on every scan. Fix: only pass ARTIFACT names to deployDiffs; a recipe FETCH row is already re-verified by classifyFetch and has no source to diff.
Also: the guard correctly named team-research + team-implement as ORPHANED-STATE on this machine — the exact stale skills the whole spec was motivated by.

**2026-07-18T11:28:02Z**

ACCEPTANCE MET: planted a real unmanaged shadow at ~/.claude/skills/brainstorming/SKILL.md (brainstorming installed only at project scope) and 'patronus scan' NAMED it UNMANAGED-SHADOW — the file on disk, not a unit test. Also proved on the real machine: team-research + team-implement show as ORPHANED-STATE (the exact stale skills that motivated the spec). Commits: efc8872 (drift pkg), 0275481 (scan wiring). Two real-machine bugs found and fixed + regression-tested (cross-scope shadow; recipe rows in the artifact spine). KNOWN GAP (filed separately, not this ticket): composed/APPEND instruction files aren't drift-checked yet — byte-compare vs a single source can't match a multi-source fold.
