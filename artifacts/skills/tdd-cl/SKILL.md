---
name: tdd-cl
description: Use for critical behavioral bugs or safety boundaries, explicit test-first work and public-interface red-green-refactor slices
---

# Test-Driven Development

## Select the strategy

Explicit test-first requests and required project checks win. Critical auth/secrets,
ownership, concurrency/isolation/settlement, migration/removal and reproducible
bugs retain intended public-interface RED, minimal GREEN and negative/legacy
regressions. Characterize uncertain legacy behavior before necessary refactoring.
Routine prose/manifests/mechanical wiring without changed safety invariants use
focused postchecks, not manufactured reds or prose-substring tests. Other behavioral
features require an explicit strategy. Tests do not admit reviewers or attestation
layers. Work only within the accepted assignment; leaves do not delegate.

## Philosophy

**Core principle**: Tests should verify behavior through public interfaces, not implementation details. Code can change entirely; tests shouldn't.

**Good tests** are integration-style: they exercise real code paths through public APIs. They describe _what_ the system does, not _how_ it does it. A good test reads like a specification - "user can checkout with valid cart" tells you exactly what capability exists. These tests survive refactors because they don't care about internal structure.

**Bad tests** are coupled to implementation. They mock internal collaborators, test private methods, or verify through external means (like querying a database directly instead of using the interface). The warning sign: your test breaks when you refactor, but behavior hasn't changed. If you rename an internal function and tests fail, those tests were testing implementation, not behavior.

See [tests.md](tests.md) for examples and [mocking.md](mocking.md) for mocking guidelines.

## Anti-Pattern: Horizontal Slices

**DO NOT write all tests first, then all implementation.** This is "horizontal slicing" - treating RED as "write all tests" and GREEN as "write all code."

This produces **crap tests**:

- Tests written in bulk test _imagined_ behavior, not _actual_ behavior
- You end up testing the _shape_ of things (data structures, function signatures) rather than user-facing behavior
- Tests become insensitive to real changes - they pass when behavior breaks, fail when behavior is fine
- You outrun your headlights, committing to test structure before understanding the implementation

**Correct approach**: Vertical slices via tracer bullets. One test → one implementation → repeat. Each test responds to what you learned from the previous cycle. Because you just wrote the code, you know exactly what behavior matters and how to verify it.

```
WRONG (horizontal):
  RED:   test1, test2, test3, test4, test5
  GREEN: impl1, impl2, impl3, impl4, impl5

RIGHT (vertical):
  RED→GREEN: test1→impl1
  RED→GREEN: test2→impl2
  RED→GREEN: test3→impl3
  ...
```

## Workflow

### 1. Planning

When exploring the codebase, read `CONTEXT.md` (if it exists) so that test names and interface vocabulary match the project's domain language, and respect ADRs in the area you're touching.

Before the behavioral slice, consume valid approval already retained in the brief:

- [ ] Identify the accepted public interface and prioritized behaviors.
- [ ] List public behavior oracles, not implementation steps.
- [ ] Confirm the slice stays within accepted scope and test strategy.
- [ ] Ask only about a real unresolved interface/behavior choice or material change.

Do not reconfirm an already approved interface/test plan every slice. Existing
approval remains valid within scope. Focus on critical paths and complex logic,
with relevant negative/legacy regressions; do not invent exhaustive testing.
A deep-module redesign or another skill workflow is not automatic TDD scope.

### 2. Tracer Bullet

Write ONE test that confirms ONE thing about the system:

```
RED:   Write public behavior test → observe intended failure, not setup error
GREEN: Write minimal code to pass → observe pass; retain relevant regressions
```

This is your tracer bullet - proves the path works end-to-end.

### 3. Incremental Loop

For each remaining behavior:

```
RED:   Write next test → fails
GREEN: Minimal code to pass → passes
```

Rules:

- One test at a time
- Only enough code to pass current test
- Don't anticipate future tests
- Keep tests focused on observable behavior

### 4. Refactor

After the focused tests pass, consider only necessary in-scope [refactor candidates](refactoring.md):

- [ ] Extract duplication
- [ ] Deepen modules (move complexity behind simple interfaces)
- [ ] Preserve accepted behavior and public interface
- [ ] Report out-of-scope structural opportunities instead of expanding work
- [ ] Run tests after each refactor step

**Never refactor while RED.** Get to GREEN first.

## Checklist Per Cycle

```
[ ] Test describes behavior, not implementation
[ ] Test uses public interface only
[ ] Test would survive internal refactor
[ ] Code is minimal for this test
[ ] No speculative features added
```
