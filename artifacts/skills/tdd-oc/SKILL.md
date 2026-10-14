---
name: tdd-oc
description: "OpenCode only: Use when implementing features, behavior changes or bugfixes, refactoring tested code, or selecting proportionate checks for nonbehavioral edits."
---

# Test-Driven Development

Test one public behavior at a time: observe the intended failure, implement the
minimum change, then refactor while green. A test that only passes after coding
does not demonstrate that it would have caught the missing behavior.

## Tiered testing discipline

Explicit test-first requests and required project checks govern. Classify by the
changed contract, not the file extension or perceived size of the edit.

1. **Critical behavior: mandatory TDD.** New or changed auth/secrets, ownership,
   concurrency/isolation/settlement, migration/removal and reproducible behavioral
   bugs require an observed intended RED, minimal GREEN and relevant negative-path
   and legacy regressions. Missing test capability blocks the affected change;
   report the exact limitation rather than substituting unverified implementation.
2. **Other behavioral development: default TDD.** New features and behavior
   changes use the same red-green-refactor loop, with coverage proportionate to
   their public contract. A different strategy needs a concrete reason and owner
   agreement, unless already approved by the task or project policy. For a purely
   behavior-preserving refactor, use existing coverage, characterize uncertain
   legacy behavior and work from green; do not invent a failing assertion for
   behavior that is not changing.
3. **Nonbehavioral work: focused postchecks.** Routine prose, manifests and
   mechanical wiring without a changed safety invariant use exact-diff inspection,
   relevant positive/negative cases and approved focused checks after editing.
   Do not manufacture red cycles or prose-substring tests. Generated outputs are
   checked through their generator and resulting contract; changed generator or
   build/test behavior belongs in the appropriate behavioral tier above.

Exploration within the existing task grant can use small, invented-data probes.
Passing a prototype or manual probe is not TDD evidence for the final change.
Bring retained exploratory code under the appropriate testing tier before delivery;
do not automatically delete existing work to recreate a test-first history.

In workflows with a strategy field, use `critical-tdd` for test-first work
(including the default tier) and `focused-postcheck` for eligible postchecks.
These are strategy labels, not separate skills or required directories. Use the
existing brief or handoff if present; no new ledger or approval is needed merely
to select the default. Ask only about a real uncertainty, conflicting requirement
or proposed exception.

## Tests that survive refactoring

Verify caller-visible outcomes through public interfaces. Prefer real code paths;
mock external boundaries when needed, not owned collaborators or private methods.
A useful test name describes a capability, such as "user can retrieve a created
account". Internal call-order assertions usually describe an implementation.

A passing test written afterward can still provide regression coverage, but it
may only confirm assumptions shared with the implementation. Observing the right
RED establishes that this test detects the missing behavior. Manual checks are
useful exploration, not a repeatable regression oracle.

Read [tests.md]({skillDir}/tests.md) for good/bad examples and
[mocking.md]({skillDir}/mocking.md) when choosing a boundary double.

## Prepare the slice

Read relevant source, callers, tests, local ADRs and `CONTEXT.md` where present.
Identify the accepted public interface and prioritize required behaviors, including
relevant failure paths. Reuse valid interface and test-plan approval; do not
reconfirm settled decisions every cycle or require a separate plan for a small fix.

When an interface is difficult to test, consult
`{skillsDir}/codebase-design-oc/SKILL.md` if installed,
or use the project's interface/testability guidance within scope.
A structural redesign is not automatic TDD work.

## Red-green-refactor, vertically

1. **RED:** write one deterministic, invented-data test for one required public
   behavior. Run it before changing that behavior. Verify the failure comes from
   the missing behavior, not an import, syntax, fixture or tool error.
2. **GREEN:** implement only what this behavior needs. Rerun the focused test and
   relevant negative-path and legacy regressions. Resolve failures before adding
   more behavior; do not weaken a valid assertion to accommodate the code.
3. **REFACTOR:** once the relevant tests are green, make only useful, in-scope,
   behavior-preserving improvements. Preserve the accepted interface and rerun
   tests after each small step. No refactor is required just to complete the cycle.
4. **REPEAT:** choose the next behavior using what the previous slice taught you.
   Retain actual commands, exits and readable red/green output in the existing
   evidence location.

The first slice is a tracer bullet: it proves that the public path and test runner
work together. Do not write all tests first and all implementation afterward.
That commits to imagined interfaces before the earlier slices can inform them.

```text
test one behavior => observe RED => minimal GREEN => optional REFACTOR
       ^                                                    |
       +---------------- next behavior ---------------------+
```

Read [refactoring.md]({skillDir}/refactoring.md) for candidates when a
necessary cleanup emerges. Characterize uncertain legacy behavior before changing
its structure; a characterization test may start green and is not a bug's RED.

## Worked bugfix example

Invented Node example using the built-in runner, no dependency installation.
Buggy `quantity.mjs` accepts zero:

```javascript
export function isValidQuantity(value) {
  return Number.isInteger(value);
}
```

Keep existing positive/fractional coverage and add the zero regression in
`quantity.test.mjs`:

```javascript
import test from 'node:test';
import assert from 'node:assert/strict';
import { isValidQuantity } from './quantity.mjs';

test('accepts a positive integer quantity', () => {
  assert.equal(isValidQuantity(2), true);
});
test('rejects a fractional quantity', () => {
  assert.equal(isValidQuantity(1.5), false);
});
test('rejects a zero quantity', () => {
  assert.equal(isValidQuantity(0), false);
});
```

Run `node --test quantity.test.mjs`. RED must be the zero test's assertion:
actual `true`, expected `false`; the existing two tests still pass.
Then make the minimal GREEN change:

```javascript
export function isValidQuantity(value) {
  return Number.isInteger(value) && value > 0;
}
```

Rerun the same command: all three tests must pass. No structural refactor is needed
here. If later cleanup is justified, keep these tests unchanged and green.

## When the cycle gets stuck

| Observation | Next action |
|---|---|
| New test passes immediately | Check whether the behavior already exists or the test misses it. Strengthen the oracle only when the requirement warrants it; never force an artificial failure. |
| Test errors before its assertion | Repair imports, fixtures or runner setup, then rerun to establish the intended RED. |
| Setup requires many internal mocks | Move the test to a useful public seam; simplify setup or inject an external dependency within scope. |
| GREEN breaks an existing test | Determine whether the old contract remains required. Fix the regression, or obtain approval for a real contract change. |
| Implementation was written first | Preserve the work, add meaningful coverage and report the missing test-first evidence. For critical work, resolve the evidence gap with the owner; do not claim a retroactive RED or automatically delete code. |

## Completion evidence

Confirm the intended behavioral RED where required, focused GREEN, relevant
negative/legacy coverage and green behavior-preserving refactors. Inspect the exact
diff for speculative features and assertions coupled to internals. Run required
aggregate checks on the composed final inputs; focused slice success is not a
whole-change pass.

Return actual commands, exits, readable logs and any limitations through the
existing handoff. Test cycles do not launch reviewers, fixers or successor workers;
review remains at the already authorized boundaries. Stay within owned paths and
authorized commands. Testing grants no installation, live-service probing, commit,
publication or cleanup authority.
