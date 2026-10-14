# Project instructions (AGENTS.md format)

This project follows the open **AGENTS.md** format for context and house rules.
Read the host's effective instructions, accepted brief and latest coordinator
handoff before acting and after resume. Inspect Claude's installed discovery
rules rather than assuming an automatic AGENTS.md fallback when CLAUDE.md is
absent. Nested instructions refine scope; higher-priority constraints still win.

When a section is empty, inspect relevant source/CI and report the gap. Flag
instruction/source drift without rewriting instructions to hide it or treating
source as permission to ignore constraints. Absence grants no editing, Git,
network or installation authority.

## Lifecycle ownership

At plan acceptance the coordinator records one progress authority in the existing
brief/plan. Qualified native state that actually owns delegated runs wins (Pi
today); trackers receive references only. Otherwise use the explicit owner/project
selection, then the existing tracker, or one durable plan-associated record if
neither exists. UI todos alone are not run-owning state. Do not mirror mutable
status among tickets, checkboxes and todos or create a selection ledger.

Only the coordinator admits assignments and transitions. Workers are leaves:
return complete findings, patches and checks, then stop. Read-only workers return
text for the coordinator/runtime to persist. Concurrent writes need qualified
isolated roots and disjoint ownership, never the coordinator checkout. Settle
and capture outputs before serial integration. Existing host/resource limits
apply without mandatory root-wide numerical budgets or new start ledgers.

Existing approval remains valid within scope. Ask for changed targets, material
design choices or missing authority, not a new stage label. Keep inline author
checks, applicable independent spec/plan reviews and one aggregate implementation
review. Task count does not add review chains. At each gate allow at most one
bounded batched correction/disposition by default. Unresolved Critical/Major or
more than two Medium block; one or two Medium residuals require explicit owner
acceptance. Changed interfaces/invariants or an insufficient correction oracle
return to the owner, not an automatic successor review.

On resume recover from the selected authority and retained bytes, read complete
results and inspect actual source and worker settlement. Do not replay settled
work, relaunch live workers or silently switch model/provider/isolation. Preserve
partial evidence when state or settlement is unknown. Completion consumes valid
matching evidence; publication and destructive cleanup remain separate grants.

## Dev environment
- The commands to install dependencies, build, and run the project locally.
- How the workspace is laid out (packages, where to make changes, what not to touch).
- Any environment setup an agent must do before its first edit.

## Testing
- The command that runs the full check suite, and how to scope it to one package/test.
- Where the CI plan lives (e.g. `.github/workflows/`) so local runs match CI.
- Critical behavior uses public-interface test-first slices with observed intended
  RED, minimal GREEN and relevant regressions. Other behavioral work defaults to
  TDD unless an approved exception applies or the owner agrees to a concrete
  reason. Prose/manifests use focused postchecks, not manufactured reds.
- Retain authorized commands, actual exits and readable logs against exact inputs.
  Matching evidence can be reused; relevant drift invalidates reuse. Final
  aggregate checks cover composed delivery. A commit needs its own grant.

## Conventions
- Language, formatting, and naming rules the code already follows — match the surrounding code's idiom rather than importing a personal style.
- Structural rules (where things live, layering, what each module owns).

## PR / commit
- Title and message format.
- The checks (lint, test, typecheck) that must pass before committing.
- Anything required in the description (linked issue, rationale, screenshots).

Nested `AGENTS.md` files deeper in the tree refine these rules for their subtree; the closest file to the code being edited wins.
