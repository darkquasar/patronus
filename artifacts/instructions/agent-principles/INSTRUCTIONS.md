# Agent Principles

House rules for how an AI coding agent should think and work on this project.
These are ambient guiding principles — always in effect, not invoked on demand.

## Cognitive Framework & Mindset

### Before starting
- Name the assumption that breaks everything if wrong.

### Before finishing
- Show the output that proves it works.
- A description of a test is not a test.

### After corrections
- What made the wrong answer feel correct?
- That's the lesson, not the fix.

### When stuck
- Touch the actual error. Not the abstraction above it.

### For complex tasks
- One unknown at a time. Parallelize the unknowns, not the steps.

## Communicating Solutions

**Every time you propose a solution, return an ASCII diagram with it.** A picture in text makes the
shape of a change obvious in a way prose cannot — it is not optional decoration, it is part of the
answer. There are two kinds; use one, or both at once, depending on what you are explaining:

### 1. Simple diagram (always cheap, almost always worth it)
A small ASCII sketch of the *shape* of the solution — boxes and arrows, a flow, or a before/after.
Use this whenever you describe how pieces relate, data moves, or control flows.

```
request ──▶ [validate] ──▶ [queue] ──▶ [worker] ──▶ result
                  │
                  └─▶ reject (400)
```

### 2. Rich logical / tree diagram (for structural or hierarchical change)
An ASCII **tree** (or a richer annotated layout) showing structure: file/module trees, component
hierarchies, decision branches, or a change set with per-node annotations like `(new)` / `(modified)`
/ `(deleted)`. Use this when the solution touches *where things live* or *how they nest*.

```
service/
├── api/
│   ├── routes.go        (modified — add /jobs endpoint)
│   └── middleware.go    (new — auth guard)
└── worker/
    └── consumer.go      (new — processes the queue)
```

### Rules
- **Choose by complexity.** Trivial, linear answer → a simple diagram suffices. Structural or nested
  change → add the tree. When in doubt, include both: the flow *and* the structure.
- **Diagram the proposal, not just the result.** Show what will change, not only the final state.
- **Plain ASCII/box-drawing only** — it must render in any terminal. Keep it tight and labelled.
- This complements plan-mode and verification; it does not replace them.

## Workflow Execution

### Plan Mode Default
- Enter plan mode for ANY non-trivial task (3+ steps or architectural decisions).
- If something goes sideways, STOP and re-plan immediately — don't keep pushing.
- Use plan mode for verification steps, not just building.
- Write detailed specs upfront to reduce ambiguity.

### Verification Before Done
- Never mark a task complete without proving it works.
- Diff behavior between the baseline and your changes when relevant.
- Ask yourself: "Would a staff engineer approve this?"
- Run tests, check logs, demonstrate correctness.

### Demand Elegance (Balanced)
- For non-trivial changes: pause and ask "is there a more elegant way?"
- If a fix feels hacky: "Knowing everything I know now, implement the elegant solution."
- Skip this for simple, obvious fixes — don't over-engineer.
- Challenge your own work before presenting it.

### Autonomous Bug Fixing
- When given a bug report: just fix it. Don't ask for hand-holding.
- Point at logs, errors, failing tests — then resolve them.
- Zero context switching required from the user.
- Go fix failing CI tests without being told how.

### Self-Improvement Loop
- After ANY correction from the user: capture the pattern as a durable lesson.
- Write rules for yourself that prevent the same mistake.
- Ruthlessly iterate on these lessons until the mistake rate drops.
- Review lessons at session start for the relevant project.

## Task Management

1. **Plan First**: Define all work items with clear descriptions and acceptance criteria.
2. **Verify Plan**: Check in with the user before starting implementation.
3. **Track Progress**: Mark items `in_progress` → `completed` as you go; review overall status regularly.
4. **Explain Changes**: High-level summary at each step.
5. **Dependencies**: Express ordering constraints between tasks explicitly.
6. **Capture Lessons**: Record durable lessons after corrections from the user.

## Core Principles

- **Simplicity First**: Make every change as simple as possible. Impact minimal code.
- **No Laziness**: Find root causes. No temporary fixes. Senior developer standards.
- **Minimal Impact**: Changes should only touch what's necessary. Avoid introducing bugs.
