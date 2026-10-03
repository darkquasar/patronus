# Historical gate fixtures

> Provenance only: the protocol below and RESULTS.md describe the source's
> 2026-08-15 model experiment, not a new Pi test claim or launch instruction.
> The nine numbered plans remain unchanged. Current Pi authority/review rules
> live in SKILL.md. No historical model name supplies model-switch authority.
>
> New local helper tests are separate: `node --test fixtures/helpers.test.mjs`
> from this skill root uses invented plans/Git bytes and launches no agents.

Nine plans with a declared expected mode, used to check that this skill's
proportionality gate routes consistently.

## Why these are run by hand

The gate is prose, evaluated by a model. `go test` tests Go, and Patronus has no
prompt-eval harness yet: `core.yaml`'s own `todo:` still lists "L8 eval: promptfoo CI
gate (P7.5.6)" as unbuilt. So this corpus is committed and the protocol below is run by
a human, once, before the skill enters `core`.

**This is a one-time gate, not a regression guard.** Nothing re-runs it when the gate's
criteria are edited later. If you change a hard trigger, a soft signal, or the precedence
order in SKILL.md, re-run the protocol yourself. The tests in
`cmd/patronus/plan_execute_content_test.go` pin that these files exist and what each
expects; they cannot pin that the gate agrees.

## Protocol

For each fixture:

1. Start a fresh session on the session's default model.
2. Invoke the skill on the fixture, using the fixture's own "Invocation" line verbatim.
3. Record the mode the decision record names, and every section it cites.
4. Repeat five times.

**Pass bar, per fixture:** 5/5 runs agree with the declared expected mode, **and** every
cited section exists in the fixture and actually supports the trigger it is named for. A
right answer reached by a citation that does not hold is a fail: it means the gate landed
on the mode for a reason it could not defend, and the next plan will land differently.

Any fixture below 5/5 means the criteria are not concrete enough. Tighten the wording in
SKILL.md and re-run that fixture. Do not ship the skill into `core` with a fixture below
the bar.

## What each fixture isolates

| Fixture | Isolates |
|---|---|
| 01 | Rule 2 beats the Rule 3 task floor |
| 02 | One soft signal is not two |
| 03 | Two soft signals above the floor |
| 04 | Raw task count is not a criterion |
| 05 | The two-mode decision on a genuinely risky plan |
| 06 | Citations read the whole plan, not one section |
| 07 | Triggers fire on what a plan does, not its vocabulary |
| 08 | Rule 1 overrides a hard trigger |
| 09 | Rule 1 overrides an otherwise-solo assessment |


## Pi provenance boundary (2026-10-01)

The nine numbered fixtures and RESULTS.md are retained source observations at the revision in SOURCE.md. Historical skill names, models and routing statements are quotations, not current invocation or review-budget instructions. They have NOT been rerun as Pi tests; active authority and bounded review/disposition are defined by SKILL.md.
