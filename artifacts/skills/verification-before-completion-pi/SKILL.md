---
name: verification-before-completion-pi
description: Use when about to claim work is complete, fixed, or passing, before committing or creating PRs - requires running verification commands and confirming output before making any success claims; evidence before assertions always
---

# Verification Before Completion

## Task authority

This skill grants no command execution, mutation, commit, publication or delegation
authority. Run verification only within the task's approved scope and resource
budget; do not install tools or broaden access to make a check run. If a required
command is unavailable or unapproved, report the work as unverified and the
completion gate as blocked, with the missing evidence and required decision.
Never replace fresh evidence with an assumption of success.

## Overview

Claiming work is complete without verification is dishonesty, not efficiency.

**Core principle:** Evidence before claims, always.

**Violating the letter of this rule is violating the spirit of this rule.**

## The Iron Law

```
NO COMPLETION CLAIMS WITHOUT FRESH VERIFICATION EVIDENCE
```

Freshness follows the checked inputs, command and acceptance scope, not the message
boundary. A later status report may cite an observed run only when all relevant
identity still matches:

- exact checked file bytes, including relevant dirty and untracked inputs;
- the full command and its options;
- relevant dependency, tool and environment identity;
- the claim's acceptance scope; and
- readable saved output and the actual exit status.

HEAD equality alone is insufficient. Consequential drift, changed acceptance,
missing logs or unknown identity invalidates reuse: run the authorized check
again or report it blocked. Cite the run, hashes, scope, logs and exit when reusing
it. Final aggregate verification at the delivery boundary stays fresh, even when
narrower task evidence was reused. This rule preserves mandatory initial and
post-compaction brief, native-state and instruction reads.

## The Gate Function

```
BEFORE claiming any status or expressing satisfaction:

1. IDENTIFY: What command proves this claim?
2. BIND: Compare the complete evidence identity above; if it does not match,
   RUN the FULL authorized command. Run final aggregate checks freshly.
3. READ: Full saved output and actual exit status, including reported failures
4. VERIFY: Does output confirm the claim?
   - If NO: State actual status with evidence
   - If YES: State claim WITH evidence
5. ONLY THEN: Make the claim

Skip any step = lying, not verifying
```

## Common Failures

| Claim | Requires | Not Sufficient |
|-------|----------|----------------|
| Tests pass | Complete test output and exit status bound to matching inputs/scope | Unbound previous run, "should pass" |
| Linter clean | Linter output: 0 errors | Partial check, extrapolation |
| Build succeeds | Build command: exit 0 | Linter passing, logs look good |
| Bug fixed | Test original symptom: passes | Code changed, assumed fixed |
| Critical-TDD regression works | Intended behavioral red, minimal change, green and relevant regressions | Test passes once |
| Agent completed | VCS diff shows changes | Agent reports "success" |
| Requirements met | Line-by-line checklist | Tests passing |

## Red Flags - STOP

- Using "should", "probably", "seems to"
- Expressing satisfaction before verification ("Great!", "Perfect!", "Done!", etc.)
- About to commit/push/PR without verification
- Trusting agent success reports
- Relying on partial verification
- Thinking "just this once"
- Tired and wanting work over
- **ANY wording implying success without identity-bound observed verification**

## Rationalization Prevention

| Excuse | Reality |
|--------|---------|
| "Should work now" | Verify against identity-bound observations |
| "I'm confident" | Confidence ≠ evidence |
| "Just this once" | No exceptions |
| "Linter passed" | Linter ≠ compiler |
| "Agent said success" | Verify independently |
| "I'm tired" | Exhaustion ≠ excuse |
| "Partial check is enough" | A narrow check proves only its checked scope |
| "Different words so rule doesn't apply" | Spirit over letter |

## Key Patterns

**Tests:**
```
✅ [Run authorized tests or read matching saved run] [See: 34/34 pass, exit 0]
   "The checked tests pass" [cite inputs, command, scope and log]
❌ "Should pass now" / "Looks correct"
```

**Regression tests (TDD Red-Green):**
```
✅ Public-behavior test → Run (intended failure) → Minimal fix → Run (pass)
   → Relevant regressions (pass)
❌ "I've written a regression test" (without red-green verification)
```

**Build:**
```
✅ [Run build] [See: exit 0] "Build passes"
❌ "Linter passed" (linter doesn't check compilation)
```

**Requirements:**
```
✅ Re-read plan → Create checklist → Verify each → Report gaps or completion
❌ "Tests pass, phase complete"
```

**Agent delegation:**
```
✅ Agent reports success → Check VCS diff → Verify changes → Report actual state
❌ Trust agent report
```

## Testing strategy

Use the task brief's selected `critical-tdd` or `focused-postcheck` strategy and
project-required checks. Critical TDD needs observed intended behavioral red/green
slices and relevant regressions. Routine prose, manifests or mechanical wiring
without a changed safety invariant use focused postchecks; do not invent a red
cycle for them. Explicit test-first requests govern. Missing or ambiguous strategy
stops for a coordinator decision before implementation. Evidence reuse waives no
required tests, acceptance criteria or fresh independent whole-change review.

## Why This Matters

From 24 failure memories:
- your human partner said "I don't believe you" - trust broken
- Undefined functions shipped - would crash
- Missing requirements shipped - incomplete features
- Time wasted on false completion → redirect → rework
- Violates: "Honesty is a core value. If you lie, you'll be replaced."

## When To Apply

**ALWAYS before:**
- ANY variation of success/completion claims
- ANY expression of satisfaction
- ANY positive statement about work state
- Committing, PR creation, task completion
- Moving to next task
- Delegating to agents

**Rule applies to:**
- Exact phrases
- Paraphrases and synonyms
- Implications of success
- ANY communication suggesting completion/correctness

## The Bottom Line

**No shortcuts for verification.**

Bind the evidence identity. Run when it does not match, and always run fresh final
aggregate checks. Read the output and exit status. THEN state the bounded result.

This is non-negotiable.
