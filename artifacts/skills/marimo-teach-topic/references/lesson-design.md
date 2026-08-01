# Lesson design — depth, structure, interactivity, and (optional) checks

This is marimo-teach-topic's **own** doctrine. It borrows *how to draw* from
marimo-visual-explanation's lens references, but the shape of a lesson is decided
here. Where the two disagree (e.g. "Markmap last" vs "roadmap first", or
"3-5 boxes" vs "freestyle"), this file wins for lessons.

There are **two independent dials**: **breadth** (how many sections — how much of
the topic you cover) and **depth** (how far each section unpacks its mechanism).
The old model only had breadth; that is why lessons came out shallow — lots of
sections, each stating a fact and stopping. Set both, deliberately.

---

## Dial 1 — DEPTH (default: medium)

Depth is *how far you unpack the mechanism*, not how many sections you have. A
shallow lesson names the parts; a deep one shows how each part actually works.
**Default to medium.** Go simple when the user asks for an overview/TL;DR; go
complete when they ask for "thorough / a real course / go deep / complete / the
works".

| Depth | The user says | What each load-bearing concept gets |
|---|---|---|
| **simple** (~1×) | "overview / high level / TL;DR / just the gist" | what it does **+ what it talks to (and what it deliberately does not) + how, in a step or two + the common misconception**. Even the *floor* is a real mechanism, never a bare one-liner. |
| **medium** (~3×) *(default)* | *(nothing) / "explain it properly"* | **broadens and deepens.** *Broaden:* subtopics a simple lesson would only list become their own sections. *Deepen:* the internal mechanism (data structures, phases, the exact object written) + edge cases + worked examples live in the **main line**; `deep_dive` folds only for genuinely optional tangents. |
| **complete** (~9×, "the full course") | "thorough / deep / a real course / complete / the works / go all in" | everything in medium, **plus the full-course apparatus below** — a per-tool field manual, end-to-end scenario walkthroughs, a default runbook, nested sub-headings, inline body links, and environment realism. This is a multi-hour course, not a note. |

### What "complete" adds (the full-course apparatus)

`complete` is a different *kind* of artifact, not just longer prose. On top of the
medium mechanism, it carries:

- **A tool/command field manual.** Every tool or command the topic uses gets its
  own `#### <name>` block with **What & when · Where & creds · Install · Use ·
  Read the output (what to look for)**. For a big topic these get long — put them
  in **sibling `.md` files loaded via a `read_doc()` helper** so the `.py` stays
  navigable (see "Loading big content from .md" below).
- **End-to-end scenario walkthroughs.** One or more *realistic, cited* scenarios
  walked step by step — plausible, not contrived (ground them in real reported
  cases). For an adversarial / ops topic, give each phase the five-block format
  **Actor (what happens) → Signal (how you detect it) → Collect (what you grab) →
  Artifact (what it proves) → Respond (what you do to contain it here)**, and
  render each phase's blocks as **`mo.ui.tabs`** to keep the page lean. Pair each
  scenario with BOTH a "what happened" diagram and a "what the responder does"
  diagram.
- **A default end-to-end runbook section** — the "in most cases, do this" ordered,
  copy-pasteable sequence, so the reader has a spine to follow before the scenarios
  show it applied.
- **Nested sub-headings.** Sections use `###`/`####` structure, not a flat
  subheading list; subsections *expand* rather than staying terse.
- **Inline links in the body.** Link the authoritative source at first mention,
  *inline* — not only in a `references` fold.
- **Environment realism.** State **where each command runs and what
  credentials/permissions it needs**, and assume the realistic production
  environment (e.g. a managed cloud cluster), not a toy local setup. Ambiguous
  "run this" with no location is a defect at this tier.

Everything must still pass `verify.sh` (0 Python errors, Mermaid parses, **YAML
blocks parse**). Paste real YAML/commands, not hand-skewed approximations.

### The "unpack the mechanism" checklist (what depth actually means)

For every **load-bearing** component or step (not every noun — the ones the topic
turns on), a medium/complete section should answer:

1. **What does it do?** — its job in one line.
2. **What does it communicate with — and what does it NOT?** This is the most
   commonly skipped one and the most clarifying. Naming the boundary ("it only
   ever talks to X; it never touches Y") kills whole classes of misconception.
3. **How does it do it?** — the actual steps/phases, not just the outcome. If the
   verb is "creates" / "handles" / "manages", replace it with the mechanism.
4. **What do people get wrong about it?** — the misconception, stated and corrected.

If a section can't answer 1-3 for its subject, it's below "simple" — that's the
gap to close, and usually the difference between the grades the reader feels. (A
bare one-sentence "names + outcome" summary — "the scheduler picks a Node and the
Pod runs" — is *below the floor*: no mechanism, no boundary, no misconception.
Don't ship it even at simple.)

### Worked example — the same section at each depth (the scheduler)

**simple** — what/talks-to/how/misconception:
> The **kube-scheduler** places Pods on Nodes. It talks to **only one thing: the
> API server** — it *watches* for Pods with no `nodeName` and, when it picks a
> Node, it *writes a Binding object* back through the API server. It never talks
> to a Node, a kubelet, or etcd directly, and **it does not create the Pod or the
> container** — it only records the decision. Choosing a Node is a two-step pass:
> **filter** (drop Nodes that can't fit the Pod — not enough CPU/memory, failed
> taints/affinity/node-selector) then **score** (rank the survivors and take the
> best). *Common mix-up:* people think the kubelet chooses the Node, or that the
> scheduler launches the container. Neither — the scheduler only decides; the
> kubelet on the bound Node is what actually runs it.

**medium** — simple, plus the internals in the MAIN line (default depth):
> *(the simple prose above, then, not folded away:)* the
> scheduler keeps a local cache of Node/Pod state fed by **informers/watches**;
> unscheduled Pods sit in a **scheduling queue**; the **filter** phase runs
> predicate plugins (`NodeResourcesFit`, `NodeAffinity`, `TaintToleration`, …)
> and **score** runs priority plugins (`NodeResourcesBalancedAllocation`,
> `ImageLocality`, spread) each 0-100, weighted and summed; if nothing fits it
> may **preempt** lower-priority Pods; the winning choice is an **assume** in
> cache then a **Bind** API call writing `pod.spec.nodeName`. The bound Node's
> kubelet — which is *also* watching the API server — sees a Pod now assigned to
> it, and only then pulls the image and starts containers. Worked example: a Pod
> requesting 2 CPU with a GPU toleration on a 3-node cluster → filter drops the
> 2 nodes without the GPU taint tolerated / without 2 free CPU → score ranks the
> one survivor → Bind → that node's kubelet runs it.

**complete** — the whole topic as a course. The medium prose above is now just
*one section*; around it, complete adds a **field manual** (a `#### kube-scheduler`
block: how to read its logs/events, `kubectl get events`, `--v=10` scheduler
tracing, where it runs and what RBAC you need to inspect it), an **end-to-end
walkthrough** ("a Pod stuck Pending — trace it from `kubectl apply` to running,
what fires, what you'd check"), a **default runbook** ("to debug any scheduling
problem, do this"), nested sub-headings, and inline links to the scheduler docs.
Same mechanism, wrapped in the full-course apparatus.

Notice: medium *deepens* a section; complete *builds a course around* it. That's
the dial.

### Loading big content from .md (complete tier)

A complete lesson's field manuals and scenario walkthroughs get long. Embedding
them as giant Python triple-quoted strings makes the `.py` unreadable and invites
escaping bugs (a `\n` inside a shell command, a stray `"""`). Instead, keep each
big block in a **sibling `.md` file** and load it at render:

```python
# in the imports cell
def read_doc(name):
    return (Path(_here) / name).read_text(encoding="utf-8")
# in a section cell — use render_doc, NOT mo.md, if the doc contains YAML (below)
render_doc(mo, read_doc("field_manual_detection.md"))
```

`verify.sh` scans those sibling `.md` files for Mermaid and YAML fences too, so
they're checked. To render a five-block scenario as tabs, parse the `.md` phases
(`#### Phase N`, then `**Actor** / **Signal** / **Collect** / **Artifact** /
**Respond**`) and feed each phase's fields to `mo.ui.tabs`.

**YAML gotcha — use `render_doc`, not `mo.md`, for any doc with `yaml` fences.**
`mo.md` runs a markdown pass that reflows block-sequence lines (`  - item`) into
loose lists **even inside a ```yaml fence**, mangling the indentation (injected
blank lines, doubled indent) of Kubernetes manifests / configs. It only bites YAML
*block sequences* — bash, prose, and inline-flow YAML (`[a, b]`) are fine, which is
why marimo's own docs example (all flow sequences) doesn't show it. The usual
"fixes" **don't work** (tested): raw strings `r"..."`, `inspect.cleandoc`, and
`.style({"white-space": "pre"})` all act on the wrong layer — the corruption is in
markdown *parsing*. `interactive.py`'s **`render_doc(mo, text)`** fixes it: it pulls
`yaml`/`yml` fences out, pygments-highlights them, and emits them via `mo.Html`
(not markdown-processed), while prose and other fences stay in `mo.md`. `render_doc`
is drop-in for `mo.md(read_doc(...))` and identical when there's no YAML.

### Depth ≠ verbosity

Deeper means *more mechanism*, not more words around the same claim. Every added
sentence should carry a fact the reader didn't have: a boundary, a step, an
object name, an edge case. If you can cut a sentence and lose no mechanism, cut it.

**Where the depth lives depends on the tier.** At **medium**, a `deep_dive` fold
keeps the main line readable while making complete-tier detail *available* to a
deep reader — depth on demand. At **complete**, the reader *asked* for the depth,
so the mechanism belongs in the **main prose**; folds are reserved for genuine
tangents (a historical note, an adjacent edge case) a reader can skip without
missing the point. A complete lesson whose depth is all folded reads as a medium —
that's the trap to avoid.

---

## Dial 2 — BREADTH (how many sections)

A **"page" = one section**: an `##` heading, its explanation (one or more lenses),
optional interactive bits. Roughly one to three screens of scroll. Map the ask:

| The user says | Sections |
|---|---|
| "overview / high level / TL;DR" | 3-5 |
| *(nothing specified — default)* | 6-9 |
| "under N pages" | ≈ N (a length cap is real — see below) |
| "granular / thorough / a proper course" | 12-20+ |

- **State the section plan before building** (SKILL.md Step 1) — with the depth
  you're targeting per section.
- **A length cap is a real constraint.** If "under 8 pages" collides with the
  depth you want, prefer **fewer sections at full depth** over many shallow ones,
  and *tell the user what subtopics you dropped and why*. Silent truncation reads
  as "this is the whole topic" when it isn't.
- Breadth and depth are orthogonal: "granular" (more, smaller sections) is not
  the same request as "deep" (each section unpacked further). Read which one the
  user asked for; when unsure, default breadth 6-9 at medium depth.

---

## Lesson anatomy (the rhythm)

1. **Title + objectives** — what the learner will be able to do after, plus rough
   size ("~8 sections, 15 min") and the depth ("explained in depth").
2. **Roadmap first** — a Markmap (or ordered list) of the journey. It *leads*: a
   learner needs the map before the territory. (Deliberate inversion of
   marimo-visual-explanation, where the mind map closes a single-topic note.)
3. **Sections** — the repeating unit. Each: a heading, a concise explanation at
   the target depth, carried by the fitting lens, with interactive bits where they
   earn their place, and a **`references` fold anchoring it in authoritative
   sources** (official docs / spec) so the explanation isn't free-floating. A
   **check only if the user asked for checks** (see below).
4. **Recap + go-deeper** — the through-line and where to continue.

---

## Choosing the lens per section

Match the shape to what the section teaches (same discipline as the borrowed
`mermaid.md`, applied per section), and reach for an **interactive** lens when the
reader should *feel* the mechanism (full menu in `interactive-lenses.md`):

- a protocol / conversation over time → Mermaid `sequenceDiagram`
- a process with branches → Mermaid `flowchart`
- a lifecycle / states → Mermaid `stateDiagram-v2`
- **how components sit together in space** → **Excalidraw `Scene`** (see below)
- a decomposition / taxonomy → Markmap
- **a set of items compared across the same fields** (options vs criteria, a
  flag/field reference, a permission matrix, an IOC inventory, a timeline, a
  step→signal→artifact mapping) → a **table** (Markdown in `mo.md`; upgrade to
  `mo.ui.table` / a DataFrame when it's large or explorable). See
  `interactive-lenses.md` → "Tables". Prose that lists parallel facts row by row
  is a table in disguise — a table makes the down-column comparison immediate.
- a quantitative relationship → a matplotlib chart, or a **`tangle`** what-if
- **a process worth watching unfold** → a **`stepper` step-through** of a diagram
  (or **`play_stepper`** to auto-play it as an animation; or an animated SVG via
  `mo.iframe` / a matplotlib GIF via `mo.image` for a concept in motion)
- **an optional aside a reader can skip** → a **`deep_dive`** fold (at *medium*,
  also for depth-on-demand; at *complete*, keep core mechanism in the main line)
- multiple framings of one idea → **`mo.ui.tabs`** (Diagram | Code | Gotchas)

Don't force a lens on a section that's fine as two sentences, and don't add
interactivity that changes nothing (see interactive-lenses.md's "when NOT to").
A lesson is prose *and* pictures *and* the occasional thing the reader drives.

### A section is not capped at one block or one visual

The unit is the **idea**, not the paragraph. A short paragraph + one diagram is
right when the idea is small — and *wrong* when the section is chunky. A chunky
section (several linked mechanisms, or a misconception to clear before the
mechanics land) earns **multiple prose blocks and more than one visual, of any
type, under the same heading**. Signs a section wants splitting into blocks:

- **A reader would ask "wait, what even *is* X?" before the mechanics make sense.**
  Then lead with a framing block (+ its own visual) that defines the *object* the
  mechanism operates on, and only then the mechanics block (+ its visual). Skipping
  this is the classic "I came out more confused" failure: explaining what acts on a
  thing before establishing what the thing *is*.
- **Two visuals show different lenses of the same thing** — e.g. a `stateDiagram`
  of an object's phases *and* a `sequenceDiagram` of the actors handing it off over
  time. Keep both; different lenses reinforce, they don't duplicate.
- **Don't pad.** Each block must teach something its neighbours don't (the same
  rule as boxes in a scene). Length and visual count follow the idea — never a
  quota, in either direction.

*Worked example — section 9 (scheduling) in the k8s lesson:* it read as confusing
while it was one paragraph + one sequence diagram, because "the scheduler assigns a
node to a Pod that has none" is nonsense until the reader knows a Pod is first a
*record of intent*, not a running process. Fixed as **one heading → two prose
blocks → two visuals**: a framing block ("a Pod is a record, not a process") + a
`stateDiagram` of the object's phases (answering "how can it exist with no node"),
then the scheduler-mechanics block + the `sequenceDiagram` scrubber of the handoff.

### Excalidraw in a lesson (overrides the borrowed "overview only" bias)

The borrowed `excalidraw.md` is now freestyle (**aim for ≤20 boxes, split beyond
that**) — good. In a lesson, treat Excalidraw as a **first-class, recurring lens**,
not a one-off closing sketch:

- Use it for the **spatial/anatomy story** — the components of a system and how
  they're wired and grouped (control plane vs worker node, client vs our infra).
  Mermaid still owns sequences, branching flows, and lifecycles.
- **Multiple Excalidraw scenes across one lesson is expected** — a cluster-anatomy
  scene early, a networking scene later, an ownership scene later still. Don't
  ration them.
- Lay boxes on the **grid** (`col=`/`row=`), zones to group, color consistently,
  dashed for secondary paths, a `note` for the caveat. Legibility is the only cap.

---

## Checks / quizzes — OFF by default

**Default: no questions.** A wall of "check your understanding" after every block
interrupts reading and makes a topic *harder* to follow, not easier. Only include
checks when the user **explicitly asks** — "quiz me", "with checks", "add
exercises", "test my understanding". If they didn't ask, the lesson is pure
explanation.

When checks *are* requested (using the bundled `quiz.py` — `mcq`/`check`/`reveal`):

- **Space them out.** Roughly one every few sections, on load-bearing ideas —
  never one per block, even when asked. Denser only if the user says "lots of
  quizzes".
- **One clearly-correct answer.** If two options are defensible, fix the question.
- **Plausible distractors** = real misconceptions (e.g. "the kubelet schedules
  Pods"). A silly distractor teaches nothing.
- **Explanations teach.** `check`'s feedback shows on right *and* wrong answers —
  write it to stand alone: why the answer is right and why the tempting wrong one
  isn't.
- **Check the idea, not the wording.** Ask what the concept *does*, not which term
  the prose used.
- **Open-ended when options are artificial** — `reveal(mo, prompt, answer)` for
  "in your own words" / design questions instead of forcing four choices.

---

## What this skill borrows vs owns

- **Borrows (mechanics):** `excalidraw_scene.py`, `markmap_html.py`,
  `mermaid_tools.py`, `verify.sh`, `serve.sh`, and the syntax references
  `mermaid.md` / `markmap.md` / `excalidraw.md` — all from marimo-visual-explanation.
- **Owns (pedagogy):** this file, `interactive-lenses.md`, the section rhythm, the
  depth + breadth model, `quiz.py` (optional exercises), `interactive.py`
  (interactive lenses), and `lesson_template.py`.
- **Ignores:** marimo-visual-explanation's `views.md`, `decision-lenses.md`, and its
  templates — their "few lenses, concise, one framing" doctrine is for a single
  note and does not govern a lesson.
