---
name: marimo-teach-topic
description: >-
  Build a LESSON: a larger, multi-section marimo notebook that teaches a whole
  topic end to end with explanatory visuals and interactive lenses (diagrams you
  scrub through, go-deeper folds, inline what-if numbers). Use this when the ask
  is to teach / explain a whole subject as a walkthrough a learner opens and works
  through, especially with a size or DEPTH control. Triggers: "teach me how
  Kubernetes works", "make me a lesson / course / tutorial on <topic>", "walk me
  through the whole of <topic>", "explain <topic> comprehensively / in depth", "a
  study/onboarding notebook on <topic>", plus size/depth asks like "keep it under
  10 pages", "high level overview only", "go deep / a proper course". Depth
  defaults to MEDIUM (unpack each mechanism); "overview" = simple, "thorough /
  deep" = complete. Explanation is the default; check-your-understanding QUIZZES
  are OFF unless the user asks ("quiz me", "with checks"). It composes the
  marimo-visual-explanation skill for the static diagrams (Mermaid, Markmap,
  Excalidraw) and adds what a lesson needs: section structure, a depth+breadth
  model, interactive lenses, and optional exercises. Do NOT use for: a single
  concept or decision that wants a few focused visuals (that is
  marimo-visual-explanation); a dashboard/charts over a DATASET (generate-notebooks);
  or mapping an existing codebase from its source.
---

# Teach a topic (a multi-section lesson notebook)

This skill builds a **lesson**: a learner opens it, reads section by section, and
answers check questions as they go. It is the *big* sibling of
`marimo-visual-explanation`. That skill explains **one** thing with a few focused
pictures and is deliberately tight. This one teaches a **whole subject** across
many sections, each with its own explanation and (usually) a check.

## This skill owns the pedagogy — it only borrows the drawing tools

Read this carefully, because it is the whole point of the split:

- **Borrow the lens MECHANICS from `marimo-visual-explanation`** — the helper scripts
  (`excalidraw_scene.py`, `markmap_html.py`, `mermaid_tools.py`), the
  verify/serve scripts, and the *syntax* references
  (`references/mermaid.md`, `markmap.md`, `excalidraw.md`). A diagram is drawn the
  same way whether it lives in a decision note or a lesson.
- **Do NOT borrow its STRUCTURE doctrine.** Ignore its `views.md`,
  `decision-lenses.md`, and its templates. Those say "pick 3-4 lenses, be
  concise, one framing, Markmap last" — correct for a single-topic note, *wrong*
  for a lesson. A lesson is intentionally longer, repeats a section rhythm many
  times, leads with a roadmap, and quizzes as it goes. **The structure below is
  this skill's own; marimo-visual-explanation does not get a vote on it.**

If you ever feel pulled toward "just make a tight 4-lens notebook", you are
reaching for the wrong skill — that is marimo-visual-explanation. Here, the shape is
a course, not a diagram.

## Two dials: DEPTH and BREADTH

A lesson is sized on two independent axes. Set both, deliberately.

**Depth = how far each section unpacks its mechanism (default: medium).** This is
the dial that decides whether a lesson feels shallow. A shallow section names a
part and states the outcome ("the scheduler picks a Node and the Pod runs");
a deep one unpacks the mechanism.

| Depth | The user says | Each load-bearing concept gets |
|---|---|---|
| **simple** | "overview / high level / TL;DR" | what it does + **what it talks to and what it does NOT** + how (a step or two) + the common misconception. Even the floor is a real mechanism, never a bare one-liner. |
| **medium** *(default)* | *(nothing) / "explain it properly"* | **broadens AND deepens:** subtopics a simple lesson lists become their own sections; the internals (data structures, phases, the exact object written) + edge cases + worked examples go in the **MAIN LINE**, not hidden. ~3× a simple. |
| **complete** | "thorough / deep / a real course / the works / go all in" | everything in medium **plus the full-course apparatus**: a per-tool/command **field manual** (what/where+creds/install/use/read-output), one or more **end-to-end scenario walkthroughs** (five-block phases as tabs + a defender diagram), a **default runbook**, nested sub-headings, **inline body links**, and **environment realism** (where each command runs + creds; assume the real managed/prod env). A multi-hour course. ~9× a simple. |

For every load-bearing component, every tier must answer: **(1) what it does,
(2) what it communicates with — and what it deliberately does not, (3) how, in
actual steps, (4) the misconception.** #2 and #3 are the ones that get skipped and
are exactly what "too high level" means.

**Complete is a different artifact, not just folded.** It wraps the topic in a
field manual + scenario walkthroughs + a runbook, carries the mechanism in the
**main prose** (folds only for genuine asides), states **where every command runs
and what creds it needs**, links sources **inline**, and for large content loads
sibling `.md` files via a `read_doc()` helper. Full spec + the
simple→medium→complete worked example (the scheduler) in
[references/lesson-design.md](references/lesson-design.md).

**Breadth = how many sections (a "page" = one `##` section, ~1-3 screens):**

| The user says | Sections |
|---|---|
| "high level / overview" | 3-5 |
| *(no size given — default)* | 6-9 |
| "under N pages" | ≈ N |
| "very granular / a real course" | 12-20+ |

Depth and breadth are orthogonal — "deep" ≠ "granular". State your section plan
*and* target depth before building (Step 1). If a length cap collides with depth,
prefer **fewer sections at full depth** and **say what you dropped** rather than
silently truncating.

**Quizzes are OFF by default.** A lesson is pure explanation unless the user asks
for checks ("quiz me", "with checks", "test me"). See the optional exercise
primitive below.

## Lesson anatomy (the fixed rhythm)

1. **Title + "what you'll learn"** — 2-4 bullets of learning objectives and the
   rough length + depth ("~8 sections, 15 min, explained in depth").
2. **Roadmap** — a Markmap (or short ordered list) of the whole journey UP FRONT.
   Unlike marimo-visual-explanation, the map *leads* here: it orients the learner.
3. **Sections** — the repeating unit, one per subtopic:
   - `## <section title>`
   - an explanation at the target **depth**, carried by the lens that fits — a
     Mermaid `sequenceDiagram`/`flowchart`/`stateDiagram`, an **Excalidraw scene**
     (a first-class recurring lens here — see below), a Markmap, a **table** (for a
     set of items compared across the same fields — a reference, a matrix, an IOC
     inventory, a timeline; Markdown in `mo.md` or `mo.ui.table` when large), prose
     — plus the **interactive lenses** that make it live: a `stepper` (or
     `play_stepper` to auto-play) step-through for a process, a `deep_dive` fold for
     optional detail, a `tangle` for a what-if, `mo.ui.tabs` for multiple framings,
     an **animation**
     (animated SVG via `mo.iframe`, or a matplotlib GIF via `mo.image`) for a
     concept in motion. Pick per the *question the section answers*. Full menu:
     [references/interactive-lenses.md](references/interactive-lenses.md).
   - a **`references` fold** anchoring the section in authoritative sources
     (official docs / spec) — a default part of the rhythm, not an extra. Use real,
     current URLs; 2-4 per section.
   - **no check** — unless the user asked for quizzes (then space them out; see
     the optional primitive below).
4. **Recap + next steps** — what was covered, and where to go deeper.

**Excalidraw is a recurring lens in a lesson.** The borrowed `excalidraw.md` is
now freestyle (aim for ≤20 boxes, split beyond that). Use Excalidraw for the
spatial/anatomy story (components + how they're wired and grouped) and expect
*several scenes across one lesson* — cluster anatomy early, networking later, etc.
Mermaid still owns sequences, branching flows, and lifecycles.

**Embed Excalidraw with `Scene.fitted()`, not a guessed height.** The widget does
NOT zoom-to-fit (a wide scene opens showing only its top-left — "zoomed in") and
has no idea how tall the drawing is (guess too big → huge empty band; the diagram
looks tiny/lost). `fitted(view_w)` returns `(scene, height)`: it fits the zoom to
the pane WIDTH and derives a snug pane height from the content.

**`view_w` MUST match the app's content column — this is the #1 thing to get
right.** The lesson template is `App(width="medium")`, whose column is **1110px**,
so pass **`view_w=1060`**:

```python
_scene, _h = _sc.fitted(view_w=1060)   # medium app (1110px column)
mo.ui.anywidget(Excalidraw(scene=_scene, height=_h))
```

Pass `view_w=700` ONLY for a `width="compact"`/`"normal"` app (740px column), or a
large value for `width="full"`. **Passing 700 in a `medium` app is the classic bug**
(and a recurring regression): the scene fits into ~60% of the column, so it renders
wide-but-short and the drawing looks tiny/lost. Use `view_w=1060` for the `medium`
template. Leave `zen=False` (the default) — zen mode's exit control overlays the
bottom-left zoom buttons.

Separately, and *not* a bug to fix in the notebook: Excalidraw **reactively** shows
a compact toolbar (a bottom bar with only undo/redo, and the bottom-left zoom
control hidden) when the effective canvas viewport is small — most often because the
**browser page is zoomed in**, occasionally an undersized pane. If a reader reports
that, it's the widget adapting; zoom the page back out (or size the pane correctly).

For a **wide** scene (4+ columns), also build it compact — `Scene(col_gap=300,
row_gap=170)` — so the fit zoom stays high (bigger text) instead of shrinking to
squeeze all columns into the column width.

## The exercise primitive (bundled `quiz.py`) — OPTIONAL, off by default

**Only include checks when the user explicitly asks** ("quiz me", "with checks",
"test my understanding"). A quiz after every block makes a topic harder to read,
not easier. When asked, space them out (one every few sections on load-bearing
ideas), never one per block.

Checks are multiple-choice with instant, teaching feedback. marimo reactivity
needs **two cells** (define the widget and return it; read `.value` downstream),
and every widget needs a **unique variable name** across the whole notebook
(`q1_pick`, `q2_pick`, …). Use the bundled helpers:

```python
# cell A — ask (return the widget; unique name per question)
q1_pick = mcq(mo, "What binds a Pod to a Node?",
              ["kubelet", "kube-scheduler", "etcd", "kube-proxy"])
q1_pick

# cell B — react (reads q1_pick.value; re-runs when they choose)
check(mo, q1_pick.value, "kube-scheduler",
      "The scheduler watches for unscheduled Pods and binds each to a suitable "
      "Node; the kubelet on that Node then actually runs it.")
```

`check` shows a green "correct" / red "not quite" callout with the explanation —
so a wrong answer still *teaches*. Write distractors that are plausible (common
misconceptions), and make the explanation earn its place. Open-ended prompts can
use `mo.accordion({"Show answer": mo.md(...)})` instead. See
[references/lesson-design.md](references/lesson-design.md) for good-question rules.

## Workflow

### Step 0: Bootstrap the environment
Reuse the repo's uv-managed `.venv`. A lesson needs only marimo + the lens deps:

```bash
bash {skillsDir}/generate-notebooks/scripts/bootstrap_env.sh \
  marimo wigglystuff anywidget matplotlib pandas
```
Mermaid and Markmap load from a CDN (no Python package).

**This uses uv, and two actions are OPT-IN — ASK the user before enabling them**
(full flow in generate-notebooks Step 0): **(a) install uv** if missing — the
script exits code 2 and prints the installer
(`curl -LsSf https://astral.sh/uv/install.sh | sh`); on yes re-run with
`--install-uv`. **(b) gitignore the venv** — the script detects a git repo and
warns if `.venv/` isn't ignored (never commit a venv); on yes add
`--gitignore-venv`. Ask both up front, then re-run with the approved flags.

### Step 1: Outline the lesson — set depth AND breadth, then choose lenses
Write the section list first, scaled to the two dials (tables above): the learning
objectives, the ordered sections, the **target depth** (default medium — for each
load-bearing concept, plan to answer what/talks-to/how/misconception), the lens
each section will use, and which interactive lenses (`stepper`, `deep_dive`,
`tangle`, tabs, Excalidraw scene) carry it. Only plan checks if the user asked.
*Then* build. If you can't say what a section teaches, cut it.

### Step 2: Build from the template + copy the borrowed + own helpers
This skill depends on `marimo-visual-explanation` being installed alongside (it reuses
its lens helpers). If that directory is missing, stop and say so.

```bash
DEST=notebooks/lessons
mkdir -p "$DEST"
cp {skillDir}/assets/lesson_template.py "$DEST/<name>.py"
cp {skillDir}/scripts/interactive.py "$DEST/"                # own: interactive lenses
cp {skillDir}/scripts/quiz.py "$DEST/"                       # own: exercises (only if quizzing)
VE={skillsDir}/marimo-visual-explanation/scripts                             # borrowed: static lenses
cp "$VE/excalidraw_scene.py" "$VE/markmap_html.py" "$VE/mermaid_tools.py" "$DEST/"
```

The template has the roadmap, worked sections (explanation + interactive lens),
and a recap, with `# >>> FILL` / `# >>> DUPLICATE` markers.
**Duplicate the section block once per subtopic.** Prefer native `mo.mermaid`
(no CDN) so diagrams can be driven by a `stepper`; read
[references/interactive-lenses.md](references/interactive-lenses.md) for the
interactive lenses and the borrowed
`../marimo-visual-explanation/references/{mermaid,markmap,excalidraw}.md` for static
lens syntax. If (and only if) the user asked for quizzes, add `mcq`/`check`
blocks, incrementing the `q<N>_pick` names.

### Step 3: Verify (reuse marimo-visual-explanation's verifier)
```bash
bash {skillsDir}/marimo-visual-explanation/scripts/verify.sh notebooks/lessons/<name>.py "$PWD/.venv"
```
Same checks (0 tracebacks / 0 MultipleDefinitionError, payloads present) plus the
optional Mermaid lint when `mmdc` is installed. If it prints
`Mermaid lint: SKIPPED — ... mmdc is missing`, relay that to the user and eyeball
each diagram for plain-ASCII Note/label text (see the borrowed `mermaid.md`).

### Step 4: Serve it as a read-only app (with live reload)
```bash
bash {skillsDir}/marimo-visual-explanation/scripts/serve.sh notebooks/lessons/<name>.py 2718 "$PWD/.venv"
```
`serve.sh` runs with `--watch`, so when you edit the `.py` on the next iteration,
the open app reloads live — no restart needed. Give the user the URL. The lesson
is interactive (step-throughs, folds, tangles), so the served app is the real
deliverable, not the editor.

**Iterating on a lesson:** the reader steers by telling you in chat what to extend
("go deeper on scheduling; add an RBAC section"). Regenerate/extend the `.py` and
the watched app reloads. (A served `marimo run` app is read-only and cannot accept
persisted input, so there is no in-notebook feedback widget — chat is the channel.)

## marimo rules that bite (same as the lens skill)
- **Unique top-level variable names across ALL cells** — the #1 failure. Each
  question needs its own `q<N>_pick`; each interactive widget its own name
  (`s5_step`, `hit`, …); prefix throwaways with `_`.
- A cell renders its **last expression**; return the widget / `mo.md` / `fig`.
- **Import shared names ONCE.** `from wigglystuff import Excalidraw` in two section
  cells is a `MultipleDefinitionError` (the name is a top-level def in each). Import
  `Excalidraw` (and anything reused across sections) in the imports cell and pass it
  in — a lesson has several Excalidraw scenes.
- Reactivity = define the widget and return it in one cell, read `.value` (or
  `.amount` for a tangle) in a *downstream* cell. `check(...)` / `mermaid_at(...)`
  go in the downstream cell.
- **Keep Mermaid Note/label text plain ASCII** (no apostrophes / `;` / `<br/>`) —
  it renders in the browser and a stray char aborts the whole diagram. This
  applies to every stage string in a `stepper` step-through too.

## When NOT to use this skill
- One concept or a decision that wants a few focused visuals → `marimo-visual-explanation`.
- A dashboard or charts over a real dataset → `generate-notebooks`.
- A map of an existing codebase derived from its source.
- A quick factual answer that needs no lesson.
