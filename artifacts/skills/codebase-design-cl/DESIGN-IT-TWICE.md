# Design It Twice

When the user wants to explore alternative interfaces for a chosen deepening candidate, use this coordinator-admitted sub-agent pattern. Based on "Design It Twice" (Ousterhout) — your first idea is unlikely to be the best.

Uses the vocabulary in [SKILL.md](SKILL.md) — **module**, **interface**, **seam**, **adapter**, **leverage**.

## Process

### 1. Frame the problem space

Before launching any sub-agents, write a user-facing explanation of the problem space for the chosen candidate:

- The constraints any new interface would need to satisfy
- The dependencies it would rely on, and which category they fall into (see [DEEPENING.md](DEEPENING.md))
- A rough illustrative code sketch to ground the constraints — not a proposal, just a way to make the constraints concrete

Show this to the user. The user reads and thinks while any admitted sub-agents work.

### 2. Launch admitted design sub-agents

The coordinator admits only the independent designs that are useful for this candidate, each under a genuinely different constraint, within the current grant and actual host capacity. There is no minimum count and no automatic launch. Read `{skillsDir}/dispatching-parallel-subagents/SKILL.md` and launch the admitted designs through its read-only mode; it owns admission preflight, result collection, settlement and failure boundaries. Each sub-agent must produce a **radically different** interface for the deepened module and return it in its final response without writing files.

Prompt each sub-agent with a separate technical brief (file paths, coupling details, dependency category from [DEEPENING.md](DEEPENING.md), what sits behind the seam). The brief is independent of the user-facing problem-space explanation in Step 1. Give each admitted agent a different design constraint, for example:

- Minimal: "Minimize the interface — aim for 1–3 entry points max. Maximise leverage per entry point."
- Flexible: "Maximise flexibility — support many use cases and extension."
- Common caller: "Optimise for the most common caller — make the default case trivial."
- Ports & adapters (if applicable): "Design around ports & adapters for cross-seam dependencies."

Include both [SKILL.md](SKILL.md) vocabulary and CONTEXT.md vocabulary in the brief so each sub-agent names things consistently with the architecture language and the project's domain language.

Each sub-agent outputs:

1. Interface (types, methods, params — plus invariants, ordering, error modes)
2. Usage example showing how callers use it
3. What the implementation hides behind the seam
4. Dependency strategy and adapters (see [DEEPENING.md](DEEPENING.md))
5. Trade-offs — where leverage is high, where it's thin

### 3. Present and compare

Present designs sequentially so the user can absorb each one, then compare them in prose. Contrast by **depth** (leverage at the interface), **locality** (where change concentrates), and **seam placement**.

After comparing, give your own recommendation: which design you think is strongest and why. If elements from different designs would combine well, propose a hybrid. Be opinionated — the user wants a strong read, not a menu.
