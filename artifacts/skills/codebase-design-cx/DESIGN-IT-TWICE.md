# Design it twice

Use the vocabulary in `{skillDir}/SKILL.md` and dependency categories in
`{skillDir}/DEEPENING.md`. Compare alternatives for an approved interface problem,
not a new implementation stage.

1. Frame constraints, current callers, source paths and dependency categories.
   Read relevant CONTEXT.md and ADRs. Explain the seam and the behavior callers
   need. State what is out of scope.
2. Produce two or three genuinely different designs locally: a minimal interface
   with one to three entry points, a flexible design, and a common-caller design.
   Each includes types/parameters, invariants, ordering, error modes, caller
   example, hidden implementation, adapters and trade-offs.
3. Compare depth (leverage), locality and seam placement, then recommend a design
   and explain the consequence for callers and tests. Do not automatically edit
   source or advance to implementation.

Optional independent comparison must be separately authorized by the lead, with a total
worker count, deadline, output binding and read-only ownership. Read the existing
`{skillsDir}/dispatching-parallel-agents-cx/SKILL.md`; do not invent another worker
framework. A leaf never delegates. Workers return complete findings to the lead,
who alone writes shared outputs. Missing runtime capability blocks that optional
route; it does not authorize a substitute provider or in-place writer fallback.
