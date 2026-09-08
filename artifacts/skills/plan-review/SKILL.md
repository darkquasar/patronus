---
name: plan-review
description: Use at the END of the planning phase, before implementation — dispatches a reviewer applying plan basics (coverage, bite-sized, no placeholders, type consistency) plus engineering / design / devex / strategy lenses, surfacing Critical/Important/Minor findings, with a second model reviewing alongside the first where one is reachable. Advisory; NOT an implementation pre-flight.
---

# Plan Review

Close the planning phase by reviewing the plan before anyone builds from it. The defect this gate
catches most often is **coverage** — a spec requirement that no task actually implements, which
nobody notices until the feature ships without it.

This gate is **advisory**. It surfaces findings — it does not block, and it does not edit the plan.
You decide what to act on.

It runs **once**, at the end of planning. It is not an implementation pre-flight and does not
re-run per task.

**Core principle:** a plan is a promise to build the spec. Check the promise before keeping it.

## When to Use

**Use it when:**
- `plan-writing` has produced a plan and the next step is implementation.
- You are about to execute someone else's plan and want an independent read first.

**Skip it when:**
- The plan is a single task. There is no build order to get wrong and no coverage to miss.

## How to Review

**1. Identify the plan — and the spec it implements.**

Find the plan (`docs/specs/NN-slug/<stream>-plan.md` in the research-effort folder) *and* the spec it was written from (`<stream>-spec.md`, the same `<stream>` prefix, in the same folder). You
cannot review a plan without both: most of the rubric is a comparison between them.

If there is no spec, say so — a plan with no spec to check against can only be reviewed for
internal consistency, not for coverage.

**2. Settle the second model.**

A same-family reviewer shares the author's blind spots. Two models reading the same plan find
different defects, and the overlap between them is partial, so a single reviewer ships real ones.
Before dispatching, settle which second model — if any — reviews alongside the first.

*Read the recorded decision.* Project wins over user:

```sh
CFG=.patronus/config/second-model.yaml
[ -f "$CFG" ] || CFG=~/.patronus/config/second-model.yaml
[ -f "$CFG" ] && cat "$CFG"
```

```yaml
model: codex-mcp        # the discovered id, or `none`
ask_every_time: false   # true re-asks each run and keeps `model` as the default
```

The project file lives under the repo's gitignored `.patronus/`, so a project-level choice is
**personal to one working tree** — not a team policy that travels with the repo. A fresh clone has
no project file and falls back to the user-level one.

*Discover what is reachable*, every run, before honouring any recorded choice. A model recorded
months ago may be gone; discovery is what catches that, which is why nothing here expires or caches.
Probe all three:

- **`codex-mcp`** — an MCP tool in this session (`mcp__codex-mcp__codex`), not a binary. Check the
  session's available tools.
- **`codex`** and **`opencode`** — binaries, on `PATH` *and* at `~/.patronus/bin/`, because Patronus
  installs there and it is not always on `PATH`:

```sh
for m in codex opencode; do
  command -v "$m" || [ -x ~/.patronus/bin/"$m" ] && echo "found: $m"
done
```

Report what discovery found, always — never a bare "none available".

*Then act on the two inputs:*

| Recorded | Still reachable | Do |
|---|---|---|
| a model, `ask_every_time: false` | yes | use it, say so, do not ask |
| a model, `ask_every_time: false` | no | say it vanished, ask again |
| a model, `ask_every_time: true` | yes | ask, with it pre-selected |
| nothing readable, or nothing recorded | any | ask |
| — | nothing discovered | single-model review, caveat in the output |

*Ask via a menu* — `AskUserQuestion` in Claude Code, the platform equivalent elsewhere — never in
prose. A prose offer gets buried in a findings report and read as commentary. Two questions:

1. **Which second model?** One option per discovered model, plus "none — single-model review".
   Sending the plan to another model sends it to another vendor, so this is a consent question and
   the user answers it.
2. **Record this choice?** Use it permanently, or ask again next time. Write the answer to the
   config file, creating `config/` if it is missing. Ask where to record it — project or user — only
   when the choice differs from what is already recorded at user level.

*Degrade quietly.* No config file, an unreadable one, an unknown model id, nothing discovered, or a
second model that errors mid-review are all one path: run the single-model review and **say in the
output that it was single-family**. A second model is an upgrade, never a precondition.

**3. Dispatch the reviewer — both of them, in parallel.**

Dispatch a `general-purpose` subagent, filling the template at [plan-reviewer.md](plan-reviewer.md).

**Placeholders:**
- `{DESCRIPTION}` — one line on what the plan builds
- `{PLAN_PATH}` — path to the plan document
- `{SPEC_PATH}` — path to the spec it implements

Dispatch a subagent rather than reviewing inline: the author of a plan knows what each step *meant*
and will read the gaps closed. A fresh reviewer reads only what is on the page — which is all an
implementer will get.

When a second model was settled on, send it **the same filled template**, in parallel with the
first. Same rubric, same plan, same spec: the value is the different reader, not a different
question.

**4. Merge the findings.**

One list, deduplicated, grouped Critical / Important / Minor. Attribute each finding to the reviewer
that raised it, and **name the disagreements**: a finding one reviewer called Critical and the other
did not raise at all is the most informative thing a two-model review produces. Do not average the
severities — carry the higher one and show the split.

**5. Present the findings.**

Group them Critical / Important / Minor, with task references. Name the lenses the reviewer skipped.
State how the review was run: which models, or that it was single-family and why.

**6. Decide, and say what you decided.**

Offer the choice plainly: fix the findings, accept them and proceed, or revise the plan. Do not
block, and do not proceed past a Critical finding without the user explicitly accepting it.

## The Rubric

The reviewer applies **plan basics** (coverage, bite-sized steps, no placeholders, type
consistency, idiom-aligned, right-sized tasks) plus four lenses — **engineering**, **design**
(user-facing UI only), **DevEx** (developer-facing only), and **strategy** — skipping any lens that
does not apply.

Full rubric: [plan-reviewer.md](plan-reviewer.md)

## Red Flags

**Never:**
- Review the plan without the spec. Coverage is the point, and coverage is a comparison.
- Proceed past a Critical finding without the user explicitly accepting it.
- Treat "the plan looks thorough" as a pass. Thoroughness and coverage are different properties —
  a plan can be exhaustive about the wrong things.

## Where This Sits

`spec-brainstorming` → `spec-review` → `plan-writing` → **plan-review** → the build fork.

Its sibling gate, `spec-review`, closes the spec phase the same way.

## The Fork: how to build it

plan-review is where the pipeline forks into execution. Once the plan is accepted, choose the
build path — this routing is plan-review's, and only plan-review's. The criterion is
**parallelism**, and it is unchanged: can you draw disjoint file-owning boundaries across the
plan's tasks?

- **`plan-execute-parallel` (parallel team)** — a Team Lead spawns teammates in worktree
  isolation, each owning a disjoint concern, then merges. Choose it for a **multi-concern,
  parallelizable** stream: the plan's tasks split cleanly into 2–5 boundaries that touch
  non-overlapping files.
- **`plan-execute` (single stream)** — for everything else: the tasks share files or must land in
  order, so parallelism buys nothing and coordination only adds risk.

A second, orthogonal question lives **inside** the `plan-execute` arm: does independent per-task
review earn its cost for this plan? That is **proportionality**, and `plan-execute` gates on it
itself, choosing a sequential mode or a fresh-subagent-per-task mode. You do not need to answer it
here.

Parallelism decides first, and it decides outright. A plan that is both parallelizable and risky
still goes to `plan-execute-parallel`: its disjoint per-teammate boundaries and merge phase already
impose review structure, and re-partitioning it into per-task subagents would fight that.

Offer the choice; do not gate on it.

## Then offer the ticket mirror

The plan lives in a gitignored markdown file; the `tk` work-graph is committed markdown under
`.tickets/`. Mirroring the plan's tasks into it is what makes them survive a context compaction.
Offer it **after** the findings are presented and the user has decided what to act on — a plan still
being revised should not be frozen into tickets.

**Check tk is there first.** It installs to `~/.patronus/bin/tk`, which is not always on `PATH`, so
a bare `command -v` under-reports it:

```sh
command -v tk || [ -x ~/.patronus/bin/tk ] && echo "tk available"
```

If tk is absent, **say so plainly** and treat the plan's `- [ ]` checkboxes as the source of truth.
Do not offer what cannot be done.

When tk is available, **ask via a menu** (`AskUserQuestion` in Claude Code, the platform equivalent
elsewhere), not in prose — a mirror narrated at the end of a findings report is a step the user never
agreed to, or one they never see. Three options:

- **Mirror now** — one epic to group the plan, one ticket per plan task, `tk dep` for the build
  order. `plan-writing`'s mirror section has the exact commands, the flags that matter, and the
  pointer-verification loop; follow it rather than improvising.
- **Skip** — the plan's checkboxes stay the record.
- **Later** — after the first tasks land.

Say which record is now authoritative, so the next session knows where to look.

**Next:** take the fork above — **`plan-execute`** for a single stream, **`plan-execute-parallel`**
for a parallel one.
