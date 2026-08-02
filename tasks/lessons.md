# Lessons

Durable, project-relevant lessons. Routine findings belong in the research doc; this file is for
patterns that should change how future work is done.

---

## L1 — A claim is not evidence. Verify against the artifact, never the report.

**2026-07-12 · from `01-lifecycle-and-test-surface`**

This session found the same defect in six places, and the Team Lead committed it three times.
**The thing that REPORTS the state had diverged from the thing that IS the state, and nothing
reconciled them:**

| The claim | The reality |
|---|---|
| Installed skill says `TeamCreate` | Source says `Agent` (fixed in f521de5) |
| `.agents/skills/plan-review:61` says "plan → beads" | Source says "plan → ticket" |
| `ticket/INSTRUCTIONS.md:166` says "no epics, flat graph" | tk accepts `-t epic` + `--parent` |
| `p77_acceptance_test.go:47-51` says "a real FETCH" | `:57` stubs it — the FETCH never runs |
| `brainstorming:136` says "`docs/specs/` is gitignored" | It was not (fixed: 9ab1dd3) |
| **Patronus reports `SKIP` (= "verified")** | **It never hashed the file** ← security hole |

**Why:** Patronus *records* the truth and then never reads it back. `state.go:52-58` stores a
`Checksum` for every deployed file; `remove/compute.go:103` has a function literally named
`driftsFromChecksum` — wired into `remove` and **nothing else**. Same shape as the binary
`SKIP` hole. The data is there; nobody consults it.

**How to apply:** before asserting anything about state — a test's coverage, an installed
artifact, a binary's integrity, a subagent's progress — **open the artifact.** A status field, a
code comment, a doc sentence, and a skill's prose are all *claims*. Treat them as hypotheses to
check, never as findings. When you write a check, ask: *does this compare against the thing, or
against something that merely describes the thing?*

---

## L2 — A subagent's self-report is unverifiable — including its excuse.

**2026-07-12 · from `01-lifecycle-and-test-surface`**

Running 4 researchers, the Team Lead:
1. Polled the filesystem once, saw no files, and **declared 3 of 4 agents failed.** They were
   mid-flight; one delivered 14k of excellent findings minutes later.
2. Accepted an agent's claim that *"the harness blocks subagents from writing report files"* and
   **wrote it into the findings as a protocol constraint.** It was **false** — another subagent had
   already written a 14k file to that same directory.
3. Saw `TaskList` report a stream `completed` with **no deliverable produced**.

**How to apply:**
- **Absence of an artifact is NOT evidence of failure while the agent is still running.** Wait for
  the completion notification (background agents auto-notify — do not poll).
- **A member's excuse is as unverifiable as its status.** Check it before propagating it.
- **Use two channels:** the member returns findings as its **final message** (primary) *and* writes
  the artifact (secondary). The lead cannot distinguish "write denied" from "chose not to write"
  from "still working" by looking at the filesystem alone.
- **Agent names drift.** Spawned `teamproto`; `TaskList` showed owner `researcher-c`;
  `SendMessage{to:"researcher-c"}` → *"No agent named 'researcher-c' is reachable."* Address by the
  spawn-assigned name; keep the returned `agentId`. Never trust a task's `owner` field.
- **A reviewer mutated production code.** One left `internal/recipe/recipe.go` flipped from `Skip`
  to `Fetch` in the working tree. **Spawn prompts for reviewers must forbid repo modification
  explicitly**, and the lead must `git status` after they finish.

---

## L2b — Subagents reliably fail to REPORT. Design the protocol to PULL, not to be PUSHED.

**2026-07-12 · from `01-lifecycle-and-test-surface` · n=7, reproducible**

Across this session, **seven** subagents (4 researchers, 3 reviewers) were spawned. Every one of them
went **idle without delivering its findings to the lead**, including two given an explicit,
hardened contract that said *"your findings must be the TEXT of your final message; do not end
without reporting."*

**What they actually did:**
- 2 did **excellent work** but never reported it: one wrote a 14k findings file (found only because
  the lead listed the directory); one authored a fixture probe (found only because the lead spotted
  a `.bak` in the scratchpad — and it had also left `internal/recipe/recipe.go` **mutated**).
- The rest produced nothing retrievable.
- One marked its task `completed` with **no deliverable**.
- One **renamed itself** and became unreachable by the name the lead assigned.

**What the contract DID fix:** after adding *"NEVER modify any file in the repo"*, no further
production-code mutation occurred. **Prompt contracts can enforce PROHIBITIONS.**

**What the contract did NOT fix:** *"report your findings as reply text"* — ignored by every agent,
every time. **Prompt contracts cannot enforce DELIVERY.**

**How to apply — this is a protocol requirement, not a prompt-wording problem:**
- **The lead must PULL, never wait to be PUSHED.** Assume the return channel will not fire.
- **Give every member a deliverable the lead can go and FETCH** — a file at a lead-chosen path — and
  then **the lead reads it**. Do not build a protocol whose only channel is the agent's cooperation.
- **Poll the ARTIFACT, not the status.** `TaskList` said `completed` for a stream that produced
  nothing (see [[L1]] — a claim is not evidence).
- **Budget for it.** If a stream's findings must reach the lead, the lead must plan to go get them —
  or do the work inline. On this session the lead ultimately did 2 of 4 streams itself, and caught
  5 defects in its own specs by verification the reviewers never delivered.
- **Corollary:** for pure *reading* work (no parallel writes, no isolation needed), inline is often
  strictly better than fan-out. The coordination and retrieval cost exceeded the parallelism benefit.
  See [[L3]].

---

## L3 — Research streams don't need worktrees. Implementation streams do.

**2026-07-12 · from `01-lifecycle-and-test-surface`**

Empirically probed `Agent` + `isolation: "worktree"` (Claude Code):

| | |
|---|---|
| Worktree path | `.claude/worktrees/agent-<id>/` |
| Branch | `worktree-agent-<id>` — **the caller CANNOT choose the name** |
| **Commit reachable from the main repo?** | **YES** — `git cat-file`/`show`/`branch -a` all see it |
| **Can the lead merge it?** | **YES** — `git merge <branch> --no-ff` works |
| Auto-cleaned? | **Only if unchanged.** A worktree with a commit **persists**, and `.claude/` is gitignored → **leaks are invisible to `git status`** |
| Subagents spawning subagents | **Works** |

**This disproves `team-implement/SKILL.md:79-81`**, which forbids native isolation *"so you retain
ownership of the branches to merge"* — the lead **can** merge them. It bans the right tool for a
wrong reason.

**How to apply:** if members only READ and return findings (research), **use no worktree** —
isolation buys nothing and adds cleanup debt. Reserve worktrees for members that **write in
parallel** (implementation). And whoever creates a worktree must clean it up: nothing is reclaimed
once a commit exists.

---

## L4 — Tests must assert behavior, not the presence of artifacts.

**2026-07-12 · from `01-lifecycle-and-test-surface`**

~30 integration tests build the **real catalog** to test Patronus, so real upstream pins leak into
tests that do not care about them. It stayed invisible only because
`internal/recipe/recipe.go:207-209` **SKIPs an archive on mere file presence, without hashing** — so
`stubBinary`'s 17 dummy bytes satisfied it. The tests *looked* like they tested installs; **the sha
— the entire trust anchor — was never exercised.** `tk`, the first raw delivery, made the latent
coupling bite.

**How to apply:**
- A test must bind to **Patronus's behavior**, never to the **catalog's contents**. If bumping a pin
  in `recipes/*.yaml` breaks a test that isn't about that recipe, the test is wrong.
- **A fixture's sha is computed from bytes the test invents** — never copied from upstream. Then
  nothing can drift. (Proven: `patronus build` builds a full registry from a `t.TempDir()` tree.)
- **Never fetch attacker-controllable remote bytes in CI** — that executes third-party code in a
  credentialed pipeline, on every PR, including forks, before human review.
- **Never weaken a sha check to serve a test.** The check IS the trust anchor; a test-only bypass
  makes the tested and shipped paths diverge exactly there.
- **Never vendor third-party bytes as test inputs** to satisfy a pin.

---

## L3 — A ticket's premise drifts too. Verify it against the artifacts before planning the fix.

**2026-08-02 · from `06-nine-tickets-triage`**

Two of nine tickets had a stale premise that would have sent the fix the wrong way:

- **pat-qo3r** said "writing-plans has NO spec.md step." Opening the artifacts showed
  team-research and writing-plans had ALREADY moved to per-stream spec/plan output
  (ADR-0003). The real work was hand-off *rewiring* (a behavior-preserving move of
  spec-synthesis prose), not the parity *build* the ticket described. Planning from the
  ticket text would have re-built something that exists.
- **pat-670z** assumed we should author a serena-first pointer mirroring
  graphify-query-pointer. Investigating serena's UPSTREAM showed serena self-delivers its
  whole "prefer me over grep" contract server-side (`--context claude-code` tool-exclusion
  + `mcp__serena__initial_instructions`). Authoring our own would duplicate it and create a
  second source of truth that drifts.

**How to apply:**
- A ticket's OBSERVED/PLAN text is a *claim written at capture time* — treat it like any
  other claim (L1) and open the artifacts it references before designing. The graph moves
  between ticket-write and ticket-do.
- **Check the upstream before wiring anything a tool might already do itself**
  (`no-duplicate-native-capability`). The value-add is the thin part the tool *cannot*
  carry (here: the serena-vs-graphify boundary + firing the un-fired trigger), not a
  re-statement of what it ships.
- When a premise is stale, say so explicitly in the deliverable and re-scope — don't
  silently implement the ticket's literal words.

## L4 — Distinguish a deliberate design decision from an undocumented accident.

**2026-08-02 · from `06-nine-tickets-triage`**

pat-78oj asked "why aren't recipes artifacts?" Two axes were tangled in that one question,
and they had OPPOSITE answers:
- Recipes not being an `ArtifactType` is a **defended decision** (`manifest.go:40-43`:
  shape is computed via `Recipe.Shape` so no `type:` can drift).
- Recipes lacking a `version:` is an **accident** — the only statement was a status-quo
  description in a plan doc with no rationale; it fell out of the version-guard's
  directory-based design, not a principle.

The fix followed the distinction: recipes GAIN versioning on the shared `Meta` header
(like they already carry `Requires`/`Role`) WITHOUT becoming an ArtifactType.

**How to apply:** when something looks like an omission, search for the rationale
(ADR/CHANGELOG/code comment) before either defending OR "fixing" it. If no rationale
exists, it is an accident you may correct — and you should WRITE the missing ADR as part of
the fix (we added ADR-0004), so the next person inherits a decision, not another silent
gap. A description of the status quo is not a justification for it.

## L5 — Test Patronus's mechanism, not the catalog's contents. Enumerating item names in Go is an anti-pattern that scales into absurdity.

**2026-08-02 · from `06-nine-tickets-triage`**

We wrote (and the repo already had) tests of the form "profile X resolves to the names
[a, b, c]" — e.g. `TestCodeIntelHasSerenaAndGraphify`, `coreSkills`. ADR-0002 point 6
had blessed these as "product guarantees." They are the wrong instrument, and the
package-manager analogy settles it: **npm does not unit-test that `left-pad` is in its
registry.** A package manager tests its MECHANISM (resolve → verify → place) on invented
fixtures, and validates registry CONTENTS with a publish-time schema/lint — never with a
per-package assertion in the installer's suite. Enumerating a profile's items in Go is that
anti-pattern in miniature: every catalog edit forces a matching test edit, so the test
becomes a speed-bump everyone re-syncs by reflex — and a test you routinely edit to match
the change it guards has lost the power to guard. At thousands of items it is obviously
ridiculous; at 58 it is just small enough to tolerate, which is how the anti-pattern
survives.

**How to apply:**
- Three buckets: (1) MECHANISM (does `Resolve` apply an `@tool` flavour? does a hook fold?
  does remove round-trip?) → test on INVENTED fixtures, once. (2) VALIDITY as a GENERAL
  PROPERTY (every profile resolves; no declared layer is hollow; the type/role ontology is
  consistent) → one catalog-wide test, zero per-item upkeep
  (`profile_structure_test.go`, `catalog_integrity_test.go`). (3) CONTENTS correctness
  (should `core` ship `grilling`?) → NOT a test; it is decided in the profile YAML and
  reviewed in the PR that changes it.
- The regression the name lists feared — a profile silently losing the tooling it promises —
  is bucket (2): assert "every declared layer resolves to ≥1 item for some tool," not the
  specific names.
- Deleting a name-mirroring test is correct even if it was committed earlier. Do NOT
  "migrate" it by renaming to fixture names — that yields a green tautology (the other
  failure mode ADR-0002 names). Delete it; recover the invariant structurally.
- This overrode a standing ADR. When you reverse a documented decision, AMEND the ADR
  in place with a dated supersedes-note (we amended ADR-0002 point 6), so the next reader
  inherits the corrected decision, not two contradictory ones.
