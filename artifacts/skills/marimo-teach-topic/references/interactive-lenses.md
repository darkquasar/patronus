# Interactive lenses — making a lesson do things a page can't

This is marimo-teach-topic's **own** reference (alongside `lesson-design.md`). The
borrowed `mermaid.md` / `markmap.md` / `excalidraw.md` cover the *static* lenses.
This file covers the interactive ones — the reason a lesson lives in marimo and
not in a Markdown file. Use them to make a lesson **deep without being a wall of
text**: the main line stays readable, and detail/exploration is something the
reader *pulls* (opens a fold, drags a slider) rather than something dumped on them.

Most of these are wrapped in the bundled `interactive.py` (`deep_dive`, `stepper`
+ `mermaid_at`, `tangle`, `key_fact`) so you don't re-derive them. The raw APIs
are documented here so you know what's available and can reach past the wrappers.

All verified against marimo 0.23 + wigglystuff 0.5. Every widget that has a value
follows the **two-cell rule**: return the widget (unique top-level name) in one
cell; read its value in a *downstream* cell — same as `quiz.py`.

## Native Mermaid: `mo.mermaid` (use this, not a CDN)

marimo renders Mermaid natively — no `mermaid_panel`/CDN needed for a lesson.

```python
mo.mermaid("flowchart LR; A[client] --> B[API] --> C[(etcd)]", theme=None)
```

The big win: the diagram is a **Python string**, so you can build it dynamically
and re-render it reactively → step-throughs (below). Keep Note/label text plain
ASCII (same rule as static Mermaid). Docs: https://docs.marimo.io/api/diagrams/

## Step-through a process (the highest-value pattern)

A slider in one cell + a dependent cell that rebuilds the diagram = the reader
scrubs through a process one stage at a time, at their own pace. Ideal for a
request path, a scheduling handoff, a protocol, or building an architecture up
layer by layer. Wrapped as `stepper` + `mermaid_at`:

```python
# cell A — the control (unique name!)
sched_step = stepper(mo, 5, label="Scheduling step")
sched_step

# cell B — the view (re-runs on drag). Each stage adds one hop.
SCHED_STAGES = [
    "sequenceDiagram\n  participant U as kubectl\n  participant A as API",
    "sequenceDiagram\n  participant U as kubectl\n  participant A as API\n  U->>A: apply Pod",
    # ...one string per stage, usually additive
]
mermaid_at(mo, SCHED_STAGES, sched_step.value)
```

**Build the chain VERTICALLY** — a `sequenceDiagram` or a `flowchart TD`
(top-down), never a horizontal `flowchart LR`. `mo.mermaid` scales the SVG to the
column *width*, so a horizontal chain shrinks a little more with every step you
add until the boxes are unreadable; a vertical chain grows *downward* at a
constant, legible width. Rule of thumb: if a step has more than ~3 nodes, or the
node labels are more than a couple of words, it must be vertical. For an additive
`flowchart TD` stepper, emit only the nodes/edges up to the current step:

```python
STEPS = ['N["1 Cordon node"]', 'O["2 Orphan the pod"]', 'P["3 Deny-all NetworkPolicy"]']
CHAIN = ["N", "O", "P"]
def _td(n):                       # first n steps, wired top-down
    body = "\n".join(STEPS[:n])
    edges = "".join(f"{CHAIN[i]} --> {CHAIN[i+1]}\n" for i in range(n - 1))
    return "flowchart TD\n" + body + "\n" + edges
mermaid_at(mo, [_td(i) for i in range(1, len(STEPS) + 1)], step.value)
```

For **hands-free playback** — an *animation* of the step-through — use `play_stepper`
(wraps `wigglystuff.PlaySlider`, a slider with a play/pause button):

```python
# cell A
roll = play_stepper(mo, 5, label="Rolling update", interval_ms=1400)
roll
# cell B — mermaid_at unwraps PlaySlider's frame automatically
mermaid_at(mo, ROLL_STAGES, roll.value)
```

`play_stepper`'s frame arrives as a dict (`.value["value"]`); `mermaid_at` handles
both that and a plain `stepper` value, so usage is identical.

## Depth on demand: `mo.accordion` (`deep_dive`)

The workhorse for the depth model. Put the section's main idea in prose; tuck the
"complete"-tier mechanism (the extra hop, the data structure, the edge case)
behind a fold. Skimmer reads the prose; deep reader opens it.

```python
deep_dive(mo, "How the scheduler scores nodes",
          "Two phases: **filter** removes nodes that cannot fit the Pod, then "
          "**score** ranks survivors 0-100 (spread, affinity, free resources) and "
          "binds the top one. ...")            # -> mo.accordion({"▸ ...": mo.md(...)})
```

Raw: `mo.accordion({"Title": mo.md("...")}, multiple=False, lazy=False)`. Pass
`lazy=True` when the body holds a heavy diagram/plot.
Docs: https://docs.marimo.io/api/layouts/accordion/

## Tweak an assumption: `wigglystuff.TangleSlider` (`tangle`)

An inline draggable number in a sentence — "what-if" intuition without a control
panel. Two cells: define inline, read `.amount` downstream.

```python
# cell A
hit = tangle(mo, 80, 0, 100, suffix="%")
mo.hstack([mo.md("If the cache hits "), hit, mo.md(" of requests...")], gap=0)
# cell B
mo.md(f"...effective latency is about **{200 - hit.amount*2:.0f} ms**.")
```

Read as `hit.amount` (float) or `hit.value["amount"]`. Siblings: `TangleChoice`
/ `TangleSelect` (inline cycling options, read `.choice`).
Docs: https://koaning.github.io/wigglystuff/reference/tangle/

## Multi-view one concept: `mo.ui.tabs`

Show the same idea framed several ways without scrolling — e.g. **Diagram | Code
| Analogy | Common mistakes**. Lets the reader pick the framing that clicks.

```python
mo.ui.tabs({
    "Diagram":  mo.mermaid("flowchart LR; A-->B"),
    "Code":     mo.md("```yaml\nkind: Deployment\n```"),
    "Gotchas":  mo.md("- forgets the selector\n- wrong namespace"),
})
```

Docs: https://docs.marimo.io/api/inputs/tabs/

## Tables: the right lens for structured, comparable facts

A **table is a lens**, not a fallback — reach for it whenever the content is a set
of items each carrying the *same fields*, and the teaching point is the
**comparison down a column** or the row-by-row lookup. Prose that lists "A does X
and talks to Y; B does P and talks to Q; C does …" is a table wearing a disguise —
the reader has to hold every row in their head to compare them. Put it in a table
and the comparison is immediate. Classic fits: a **comparison** (option A vs B vs
C across criteria), a **field/flag reference** (each command flag + what it does),
a **capability/permission matrix** (role × resource → allowed?), an **IOC / entity
inventory** (indicator, type, first-seen, verdict), a **timeline** (time · actor ·
action), a **mapping** (attacker step → detection signal → artifact).

Most tables are just **Markdown in `mo.md`** — marimo renders GFM tables natively,
so no widget is needed:

```python
mo.md("""
| Log source | Records the... | Join key | Gotcha |
|---|---|---|---|
| AuraRequest | controller called | `requestId` | no object names |
| cosis (DB)  | SOQL executed    | `rootRequestId` | not customer-visible |
""")
```

Keep cells terse (a phrase, not a paragraph); if a cell needs a paragraph, it
wants prose or a `deep_dive`, not a table row. Bold the column that carries the
lesson. Keep pipes `|` out of cell text (escape as `\|`). A **wide** table (5+
columns) will scroll on marimo's ~740px column — drop non-load-bearing columns or
split into two tables rather than shrinking every cell.

**When the data is large or the reader should explore it, upgrade to a real
widget** (these auto-display; no `mo.md`):

- `mo.ui.table(rows, selection="single"|"multi", page_size=…)` — a **sortable,
  paginated, selectable** table. Read `.value` downstream (the selected rows) to
  drive an explanation off the row the reader clicks. Use when there are more than
  ~15 rows, or when sorting/filtering is itself part of the lesson.
- A **pandas / polars DataFrame** returned as a cell's last expression renders as
  an interactive table (sort, search, column stats) for free — ideal for showing
  real tabular data you computed.
- `mo.ui.dataframe(df)` / `mo.ui.data_explorer(df)` — a GUI to filter/transform or
  chart a frame, when "how you'd slice this data" is the thing being taught.

Rule of thumb: **static, curated, ≤~15 rows → Markdown table** (it reads as part
of the prose); **live, large, or explorable → `mo.ui.table` / a DataFrame**. Don't
force a table on two or three items that a sentence handles, and don't paste a
giant frame the reader can't scan — sample or aggregate it first.
Docs: https://docs.marimo.io/api/inputs/table/ · GFM tables render in any `mo.md`.

## Emphasis: `mo.callout`, `mo.stat` (`key_fact`)

- `mo.callout(mo.md("..."), kind="info"|"success"|"warn"|"danger"|"neutral")` —
  a boxed tip/warning. (The quiz `check` already uses callouts.)
- `key_fact(mo, "110", "Pods per node", "default cap")` → `mo.stat` card for the
  one number a section hinges on. Use sparingly.

Docs: https://docs.marimo.io/api/layouts/stat/

## Before / after: `mo.image_compare`

Native slider between two images (a "before refactor / after", a topology change).
`mo.image_compare(before=<src>, after=<src>)`. For two blocks of text/diagrams
instead of images, use `mo.ui.tabs` or a `stepper` of length 2.
Docs: https://docs.marimo.io/api/media/image_compare/

## Layout glue you'll reach for

`mo.hstack` / `mo.vstack` (side-by-side diagram + prose; `gap`, `align`, `justify`),
`mo.carousel([...])` (a slide deck flow), `mo.sidebar([...])` (a persistent
glossary/nav), `mo.nav_menu({...})` (a table of contents). LaTeX works in `mo.md`
via `$...$` / `$$...$$` for any math. Docs: https://docs.marimo.io/api/layouts/

## Also in the toolbox (reach for when the topic fits)

From wigglystuff (`mo.ui.anywidget(Widget(...))`): `SortableList` ("put these
steps in order" exercise), `Slider2D` (pick an x,y — gradients, vectors),
`EdgeDraw`/`GraphWidget` (draw/explore graphs — graph algorithms, networks),
`Matrix` / core `mo.ui.matrix` (fill-in matrices for math/CS), `drawdata`
(sketch a dataset — ML/stats intuition). `mo.ui.altair_chart` / `mo.ui.plotly`
give **selection** charts (the reader clicks points and you explain the
selection). Don't force these — use one only when it teaches the specific topic
better than prose + a static picture would.

## Anchor in sources: `references`

Ground each section in authoritative external docs (the official docs, a spec, a
canonical post) with a `references` fold — so the explanation isn't free-floating
and the learner can go to the source. Put one at the **end of most sections**; it
folds away so it never clutters the main line.

```python
references(mo, [
    ("Kubernetes docs: Pods", "https://kubernetes.io/docs/concepts/workloads/pods/"),
    ("API concepts (watch, resourceVersion)",
     "https://kubernetes.io/docs/reference/using-api/api-concepts/"),
])
```

`links` is a list of `(label, url)` pairs (or a `{label: url}` dict). Use **real,
current** URLs for the topic — prefer the canonical/official docs; 2-4 per section
is plenty. This is a default part of the section rhythm, not an extra.

## Animations (only when motion itself carries information)

**Animate only when the *motion* teaches something a static picture or a driven
step-through cannot.** A dot sliding along a line, a box that pulses, any
decorative motion — is noise; it looks like an animation without being one, and it
actively cheapens the lesson. Before animating, ask: *what does moving show that a
labelled diagram doesn't?* If you can't answer, don't animate.

Motion earns its place for: **a quantity evolving over time/parameter** (a
distribution shifting, a curve converging, load vs replicas), **an algorithm's
state changing step by step**, or **a physical/among-parts process** where timing
or overlap is the point. For a **process flow** (a request path, a handoff, a
rollout) a **driven `stepper`/`play_stepper` is almost always better** than a true
animation — the learner controls the pace and each step is labelled. Reach for a
real animation mainly for *plotted, evolving quantities*.

When it does earn its place (JS-driven animations need `mo.iframe`, which runs
scripts; `mo.Html` does not — native GIF/MP4 animate via `mo.image`/`mo.video`):

- **matplotlib → GIF via `mo.image`** — the workhorse, for a plotted quantity
  evolving. Pillow writer (no ffmpeg); the browser `<img>` loops the bytes.
  ```python
  import io, matplotlib.pyplot as plt, numpy as np
  from matplotlib.animation import FuncAnimation
  fig, ax = plt.subplots(); (ln,) = ax.plot([], []); ax.set_xlim(0, 6.3); ax.set_ylim(-1, 1)
  x = np.linspace(0, 6.3, 200)
  anim = FuncAnimation(fig, lambda f: (ln.set_data(x, np.sin(x + f/8)), (ln,))[1],
                       frames=40, blit=True)
  buf = io.BytesIO(); anim.save(buf, writer="pillow", fps=15); plt.close(fig)
  mo.image(buf.getvalue(), width=480)
  ```
  Also: `anim.to_jshtml()` in `mo.iframe` for a JS player with a scrubber;
  `mo.as_html(px_fig)` for a Plotly `animation_frame=` play button. Deps:
  matplotlib + pillow; keep frames modest (GIF size grows fast).
- **`play_stepper`** (above) — an auto-playing step-through. This is the honest
  "animation" for a process; prefer it over faking motion with SVG.

Animated SVG/CSS via `mo.iframe` *can* be emitted by an LLM, but it almost always
lands as the decorative-motion anti-pattern above — avoid it unless the schematic
motion genuinely encodes timing/order the reader must see.

## Native integrations & the extension surface (what marimo gives you)

Beyond the lenses above, marimo renders a lot out of the box and has real
escape hatches — reach for these when a topic needs more than the built-ins:

- **Auto-displayed libraries:** matplotlib, Plotly, Altair, seaborn, HoloViews;
  pandas/polars frames as interactive tables. Reactive: `mo.ui.altair_chart(chart)`
  (brush/select points → filtered data back into Python — great "select and see"),
  `mo.ui.plotly`, `mo.mpl.interactive(fig)` (pan/zoom).
- **`mo.ui.code_editor(language=...)`** — a live editor (python/sql/js/...); let a
  learner change a line and re-run reactively ("edit this, watch the output").
- **Structure/navigation:** `mo.carousel([...])` (slide deck), `mo.sidebar([...])`
  (persistent glossary/TOC), `mo.nav_menu({...})`, `mo.routes({...})` (hash
  multipage), `mo.lazy(fn)` (defer a heavy widget until opened), `mo.tree([...])`.
- **Run arbitrary JS/CSS libraries:** `mo.iframe(html)` **executes `<script>`** —
  the sanctioned way to drop in a CDN lib (D3, vis-network / cytoscape for graphs,
  three.js) for one visual. `mo.Html`/`mo.as_html` do **not** run scripts.
- **Custom widgets:** author or embed an **anywidget** (JS/TS + CSS) via
  `mo.ui.anywidget(...)` — this is how wigglystuff (Excalidraw, tangles) plugs in.
- **Whole-notebook CSS / JS** (confirmed mechanisms):
  - Global CSS/theming: `marimo.App(css_file="custom.css")` (or `pyproject.toml`
    `[tool.marimo.display] custom_css=[...]`) — applies in edit **and** run mode.
  - CDN JS / `<head>` (fonts, D3, KaTeX): `marimo.App(html_head_file="head.html")`
    — **run mode only** (not edit mode); for edit-time JS, load via `mo.iframe`.

  Reach for whole-notebook CSS/JS rarely — a lesson almost always wants a per-cell
  `mo.iframe` or an anywidget, not global injection.

## When NOT to be interactive

Interactivity is for *mechanism the reader should feel* — a process that unfolds,
an assumption with a consequence, a layer worth hiding. A fact that's just a fact
is better as one clear sentence. A step-through with two near-identical stages, a
tangle whose number changes nothing downstream, an accordion hiding one line —
all noise. Interactive when it earns it; prose when it doesn't.
