---
name: tdd-oc
description: "OpenCode only: Test-first public-behavior slices for critical-tdd tasks or explicit test-first requests; routine prose/manifests use focused postchecks."
---

# Behavior-sized testing

Read the recorded strategy, owned paths, authorized commands and applicable
resource limits. Reuse valid interface and behavior approval within scope.
Ask only about missing approval, changed targets or a material unresolved choice,
not to reconfirm each slice. No automatic installs, live-service probes, commits,
publication or cleanup.

## Select the strategy

Use `critical-tdd` for new or changed auth/secrets, ownership,
concurrency/isolation/settlement, migration/removal and reproducible behavioral
bugs. Explicit test-first requests and project-required checks govern.
Routine prose, manifests and mechanical wiring without a changed safety
invariant use `focused-postcheck`: inspect exact diff and positive/negative
cases, then run approved focused checks. Preserve existing safety tests.
Other behavior requires an explicit recorded strategy; missing or ambiguous
strategy stops implementation for the coordinator's decision. Do not invent
red cycles or prose-substring tests.

## Short public-behavior slices

Read relevant source, callers, tests, ADRs and CONTEXT.md where present. List
required behaviors and their public interfaces, not implementation steps.
For interface/testability design, read
`{skillsDir}/codebase-design-oc/SKILL.md` within the existing grant.

1. Write one invented-data test through a public interface for one required
   behavior. Run it and observe the intended behavioral failure, not a setup
   or tool error.
2. Implement the minimum change. Run the test to green and retain actual
   commands, exits and readable red/green output.
3. Run relevant negative-path and legacy regressions, then repeat for the next
   behavior. Do not write all tests first and all implementation later.
4. Characterize uncertain legacy behavior before necessary refactoring.
   Refactor from green in small behavior-preserving steps, checking after each.

Tests should survive internal refactors: verify caller-visible outcomes rather
than private methods, internal call order or mocks of owned collaborators.
Use [tests.md]({skillDir}/tests.md) for examples,
[mocking.md]({skillDir}/mocking.md) for boundary mocks and
[refactoring.md]({skillDir}/refactoring.md) for refactor candidates.

Test cycles do not admit reviewers, fixers or successor agents. The coordinator
owns named boundary reviews and the required aggregate implementation review.
Return complete evidence and stop within the assigned scope.
