---
name: tdd-pi
description: "Test-first public-behavior slices for critical-tdd tasks or explicit test-first requests; routine prose/manifests use focused postchecks."
---

## Scope and recovery

Act only within the task's current grant. Research, authorship, review, planning, execution, integration, publication and cleanup are separate actions. Existing explicit approval remains valid within its recorded root, outputs and revalidation conditions; do not ask again merely because the stage changed. Changed scope or destructive actions need revalidation. Before work and after compaction, reread the mandatory brief, native mission state, latest decisions and project instructions, then inspect current artifacts. A leaf never delegates, mutates native mission state or writes lessons. Only the coordinator does so when authorized. Use contact_supervisor need_decision/interview_request for a material unresolved choice and wait for the actual reply; steering delivery is not consent. Do not switch model, provider, protocol or isolation on failure.


# Test-Driven Development

## Select the testing strategy

Record `critical-tdd` or `focused-postcheck` in the existing task brief before
implementation. Select `critical-tdd` for new or changed auth/secrets,
migration/removal, ownership, concurrency/settlement/isolation behavior and
reproducible behavior bugs. Explicit test-first requests and project-required
checks govern. Read this skill when critical TDD is selected.

Routine prose, manifests and mechanical wiring without a changed safety invariant
use `focused-postcheck`: inspect the exact diff, check positive/negative cases and
run the approved focused checks after editing. Preserve existing safety tests and
required fresh independent whole-change implementation review. Do not manufacture
a red cycle per file/task or add prose-substring tests.

Missing or ambiguous strategy, including a legacy brief without one, stops
implementation for a coordinator decision. Unknown behavioral work does not
qualify for focused postchecks. The workflow below applies to `critical-tdd`.

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

Before writing any code (an existing explicit approval satisfies the corresponding step; do not reopen settled choices):

- [ ] Confirm with user what interface changes are needed
- [ ] Confirm with user which behaviors to test (prioritize)
- [ ] Identify opportunities for deep modules (small interface, deep implementation) — read the `codebase-design-pi` SKILL.md for the vocabulary and the testability checks
- [ ] List the behaviors to test (not implementation steps)
- [ ] Get user approval on the plan

Ask: "What should the public interface look like? Which behaviors are most important to test?"

**You can't test everything.** Confirm with the user exactly which behaviors matter most. Focus testing effort on critical paths and complex logic, not every possible edge case.

### 2. Tracer Bullet

Write ONE invented-data test through a public interface for ONE required behavior.
Run it and confirm the intended behavioral failure, not a setup/tool error:

```
RED:   Write test for first behavior → test fails
GREEN: Write minimal code to pass → test passes
```

This tracer bullet proves the path end-to-end. Retain the command, actual exit
and readable red/green logs; run relevant negative-path and legacy regressions.

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

After all tests pass, look for [refactor candidates](refactoring.md):

- [ ] Extract duplication
- [ ] Deepen modules (move complexity behind simple interfaces)
- [ ] Apply SOLID principles where natural
- [ ] Consider what new code reveals about existing code
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
