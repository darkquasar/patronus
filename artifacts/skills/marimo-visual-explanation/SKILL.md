---
name: marimo-visual-explanation
description: >-
  Build an interactive marimo notebook that explains something visually, instead
  of answering in prose, whenever a few well-chosen visuals would make an idea
  land better than words alone. Two main cases. (1) DECISIONS — comparing options
  and weighing tradeoffs: "X vs Y", "should we use A or B", "help me
  choose/decide", "compare these approaches", "lay out the
  tradeoffs/considerations/risks", "we keep going back and forth on…", "we're
  stuck between…", "monolith vs microservices", "Postgres vs DynamoDB". (2)
  CONCEPTS & SYSTEMS — explaining how something works or is structured: "explain
  how OAuth2 / Raft / TLS / our rate limiter works", "walk me through this
  architecture / algorithm / protocol", "help me understand how X flows", "map
  out how this process fits together". Covers databases, frameworks, languages,
  libraries, architectures, protocols, messaging systems, algorithms, data flows,
  state machines, migration strategies, and the like. Trigger on the
  explaining/deciding intent alone, even with no mention of a diagram, chart,
  visual, or notebook — treat "help me think through / understand / decide this"
  as a build request, not a chat reply. The notebook picks the fitting lenses: an
  option/concept mind map, Mermaid diagrams (quadrant, flowchart, timeline,
  sequence, state), an interactive weighted decision matrix, comparison tables,
  tangle widgets, and an Excalidraw sketch. Do NOT use for: dashboards or charts
  over an existing DATASET (use generate-notebooks); mapping or diagramming an
  existing CODEBASE from its source; code review; or
  implementing an already-made choice.
---

# Visualize an explanation (pick the lenses that fit)

This skill is for the **fuzzy front end** — brainstorming, design, planning,
learning, *before* code exists (or independent of any specific codebase).
Someone is either **weighing alternatives** ("Postgres vs DynamoDB vs Mongo?",
"monolith or microservices?", "REST or gRPC?") or **trying to understand how
something works** ("how does OAuth2 work?", "walk me through Raft", "explain our
rate-limiting approach"), and a few well-chosen visuals make the idea legible.
The deliverable is a **marimo notebook** the user opens, reads, and plays with
(drag a weight slider, expand a mind map, follow a sequence).

These are two shapes of the same job — **make an idea legible with the right few
pictures.** A decision is an explanation that ends in a choice; a concept
explanation is the same lens-picking discipline without the ranking. Decisions
are the most common case, so much of the doctrine below is written in decision
terms — read "the point being made" wherever it says "the decision", and the
same rules apply to a pure concept explanation.

The output is not a fixed set of diagrams. It is a **menu of lenses**, and your
job is to pick the 3-4 that actually illuminate *this* topic and skip the rest.
A forced lens is noise.

## The mindset: lenses serve the point

Every lens must earn its place by answering a question the others don't, in
service of the point being made — the choice, or the thing being understood.
Before adding one, finish: *"This shows ___, which helps the user
understand/decide because ___."* If you can't, drop it.

The cardinal rule: **match the lens — and the Mermaid diagram type — to the
question.** A tradeoff is a quadrant or a weighted matrix, not a sequence
diagram. A process or protocol is a flowchart or a sequence. A rollout is a
timeline. A lifecycle is a state diagram. Never default to one shape; choose.

## The lens menu

| Lens | Best for | Tool | Cost |
|---|---|---|---|
| **Markmap** | the option space — alternatives and their considerations as a hierarchy | `markmap_html.py` → `mo.iframe` | CDN |
| **Mermaid** | the right *shape* for the question — see below | `mo.mermaid` | CDN |
| **Weighted decision matrix** | "which option wins, and how sensitive is that to my priorities?" | `mo.ui.slider` + matplotlib | none (offline-safe PNG) |
| **Comparison table** | options × criteria at a glance | `mo.ui.table` / pandas Styler | none |
| **Excalidraw** | the recommendation / chosen shape at a glance | `excalidraw_scene.py` → `wigglystuff.Excalidraw` | CDN |
| **Tangle widgets** | "tweak one assumption inline and watch the conclusion move" | `wigglystuff.TangleSlider` | none |
| **Plotly radar** *(optional)* | 2-3 options compared across 5+ criteria as overlapping shapes | `plotly` → `mo.ui.plotly` | pip + CDN |

**Mermaid is not one diagram — it's many.** Pick the type that fits, and honour
an explicit request ("show me a sequence diagram"):

- **`quadrantChart`** — a 2×2 tradeoff (effort vs impact, cost vs flexibility). The
  go-to for "where does each option sit".
- **`flowchart`** — decision logic ("if you need X, choose A") or a process/protocol.
- **`timeline`** — sequencing, phases, a migration/rollout.
- **`sequenceDiagram`** — interactions over time between actors (a handshake, an
  auth flow, a request lifecycle). The workhorse for "how does this protocol work".
- **`stateDiagram-v2`** — a lifecycle or state machine (connection states, an
  order's status, a retry/backoff loop).
- **`mindmap`** — concept decomposition.

See [references/mermaid.md](references/mermaid.md) for verified syntax of each.

### Sensible defaults by shape of the topic

**Comparing N options (a decision).** Most decision notebooks want, roughly:
**Markmap** (lay out the options) → **Mermaid `quadrantChart`** (the tradeoff at a
glance) → **weighted decision matrix** (the interactive heart) → **Excalidraw**
(sketch the recommendation), with a **Tangle** line if one assumption dominates.

**Explaining how something works (a concept/system).** There are no options to
rank, so the weighted matrix usually drops out. A good set is: a short prose
brief → **Mermaid** as the spine (a `sequenceDiagram` or `flowchart` for a
process/protocol, a `stateDiagram-v2` for a lifecycle) → an **Excalidraw**
component sketch of how the parts fit → **Markmap** last as a decomposition
recap. Add a **Tangle** if one parameter makes the behaviour click (e.g. window
size, timeout, replica count).

Adjust freely in both cases — the shape of the topic, not a fixed recipe, drives
the lens choice.

### Scale the depth to the topic

The lenses are not capped at the minimal sketch — scale them up when the topic
(or an explicit "explain this thoroughly") warrants real depth:

- **Markmap**: go 3-4 levels deep with many branches — it's zoomable and
  pannable, so a large tree stays navigable. Give it room with a taller height:
  `mo.iframe(markmap_html(md, height=620), height="640px")`.
- **Mermaid**: for a detailed process use a full `flowchart TD/LR` with every
  branch (errors, retries, cache hit/miss, fallbacks); for interactions a full
  `sequenceDiagram` with `alt`/`else`. This is the lens where dense detail
  belongs.
- **Excalidraw**: add more rows and columns via `col=`/`row=`, raise the canvas
  (`Excalidraw(scene=..., height=660)`), and widen `Scene(col_gap=…, row_gap=…)`
  for a larger architecture. The canvas is infinite and zoomable, so a bigger
  scene is fine — keep each box's label short and let the grid handle spacing.

Match depth to the audience: a quick decision or a one-glance concept wants a
tight set of lenses; "walk me through this in detail" wants the deep versions.
Don't pad a simple topic to look deep — depth is for when the subject actually
has it.

### Structure and tone

- **Lead with a concise explanation.** Before the visuals, give a few tight,
  concrete sentences or bullets — the options and the crux of the tradeoff (for a
  decision), or what the thing is and its moving parts (for a concept). Concise
  is the default; write a longer prose section *only if the user asks* to go
  deep. Diagrams support the words; they don't replace them.
- **Order the notebook:** the headline up front (the recommendation for a
  decision, the one-sentence "what this is" for a concept) → the brief
  explanation → the diagrams (Mermaid spine, the weighted matrix if it's a
  decision, the sketch) → **Markmap last**, as a full-space recap. The mind map
  is a closing reference, not the lead.
- **Offer two Mermaid angles in tabs when both genuinely help.** A process is
  often worth showing as both decision-logic and runtime-sequence:

  ```python
  mo.ui.tabs({"Flow": mo.mermaid(flow_src), "Sequence": mo.mermaid(seq_src)})
  ```

  The reader picks the angle. Collapse to a single `mo.mermaid(...)` when one
  diagram says it all — don't add a tab for the sake of it.
- **Give Mermaid diagrams a copy/download action.** Use `mermaid_panel` (from
  `mermaid_tools.py`) instead of a bare `mo.mermaid` so the reader can grab the
  source:

  ```python
  from mermaid_tools import mermaid_panel
  mermaid_panel(mo, src, download_name="flow.mmd")   # diagram + copy + download
  ```

  Note on SVG: there's no built-in way to export the *rendered* SVG from marimo
  (`mo.mermaid` renders client-side). The copy button is the easy path — paste
  into https://mermaid.live to export SVG/PNG — or install mermaid-cli to render
  server-side. See [references/mermaid.md](references/mermaid.md).

## Workflow

### Step 0: Bootstrap the environment

Reuse the repo's uv-managed `.venv` at the repo root. Install only what the
lenses you chose need:

```bash
# core (always):
bash {skillsDir}/generate-notebooks/scripts/bootstrap_env.sh \
  marimo wigglystuff anywidget matplotlib pandas
# only if you use the Plotly radar lens:
uv pip install --python .venv/bin/python plotly
```

Markmap and Mermaid need no Python package (they load from a CDN in-notebook).

**This uses uv, and two actions are OPT-IN — ASK the user before enabling them**
(see generate-notebooks Step 0 for the full flow): **(a) install uv** if it's
missing — the script exits code 2 and prints the installer
(`curl -LsSf https://astral.sh/uv/install.sh | sh`); on yes re-run with
`--install-uv`. **(b) gitignore the venv** — the script detects a git repo and
warns if `.venv/` isn't ignored (a venv should never be committed); on yes add
`--gitignore-venv`. Ask both up front, then re-run with the flags the user approved.

### Step 1: Frame the topic, then choose lenses

Write down, in one or two sentences, what the notebook has to land — *then* pick
the lenses to fit it.

- **A decision:** the decision, the 2-3 options, the criteria that matter, your
  tentative recommendation. For "Postgres vs DynamoDB vs Mongo for a new
  service", a good set: *Markmap* (the three options with sub-considerations —
  consistency, scaling, ops, cost) → *Mermaid `quadrantChart`* (flexibility vs
  ops simplicity) → *weighted decision matrix* (sliders to reweight and watch the
  ranking shift) → *Excalidraw* (sketch the recommended option). Skip the radar
  and timeline — they add nothing here.
- **A concept/system:** the thing, its parts, and the flow or lifecycle that ties
  them together. For "how does OAuth2 authorization-code flow work", a good set:
  a short prose brief → *Mermaid `sequenceDiagram`* (user ↔ app ↔ auth server ↔
  resource server, the token exchange as the spine) → *Excalidraw* (the four
  actors and what each holds) → *Markmap* (the pieces: grant types, tokens,
  scopes, PKCE). No weighted matrix — nothing is being ranked.

If you can't say why a lens helps *this* topic, leave it out.

### Step 2: Build the notebook from a template

Two templates ship with the skill — copy the one that fits, plus **all three
helper scripts**, into the destination dir:

```bash
DEST=notebooks/explanations
mkdir -p "$DEST"
# pick ONE template:
#   decision (options, weighted matrix, recommendation):
cp {skillDir}/assets/explanation_template.py "$DEST/<name>.py"
#   concept/system (no options/matrix; sequence/state + component sketch):
cp {skillDir}/assets/concept_template.py "$DEST/<name>.py"
# always copy the helpers:
cp {skillDir}/scripts/excalidraw_scene.py "$DEST/"
cp {skillDir}/scripts/markmap_html.py "$DEST/"
cp {skillDir}/scripts/mermaid_tools.py "$DEST/"
```

Each template includes its lenses with a `# >>> FILL` block and a `# >>> KEEP IF`
note. **Delete the cells for lenses you didn't choose.** Fill the rest with the
real content. Read the reference for any lens you're unsure about:

- [references/views.md](references/views.md) — the doctrine above in depth, with
  worked examples (a decision and a concept) across the lenses (read this first).
- [references/decision-lenses.md](references/decision-lenses.md) — the weighted
  matrix, comparison table, tangle widgets, and radar, with verified snippets
  (the matrix/radar are decision-specific; the tangle and table suit any topic).
- [references/mermaid.md](references/mermaid.md) — choosing and writing each
  Mermaid diagram type.
- [references/markmap.md](references/markmap.md) — the option-space mind map.
- [references/excalidraw.md](references/excalidraw.md) — the `Scene` builder for
  the recommendation / component sketch (never hand-write Excalidraw JSON).

### Step 3: Verify (do not skip)

A notebook that hasn't been run does not work. Export it headless — this runs
every cell — and confirm no errors:

```bash
bash {skillDir}/scripts/verify.sh notebooks/explanations/<name>.py
```

It checks for `0` tracebacks / `0` MultipleDefinitionError and reports the lens
payloads present. The Mermaid/Markmap/Excalidraw visuals render only in a
browser (CDN-loaded JS); matplotlib charts embed as offline PNGs.

**Mermaid syntax is checked only if `mmdc` is installed.** Because Mermaid parses
in the browser, a syntax error (a stray apostrophe/`;`/`<br/>` in a Note or label
is the classic one — see `references/mermaid.md`) exports "clean" and only breaks
on screen. When `mmdc` (`@mermaid-js/mermaid-cli`) is present, `verify.sh` parses
every diagram and fails loudly; when it's absent it prints `Mermaid lint:
SKIPPED — ... mmdc is missing` and passes anyway. **If you see that SKIPPED line,
relay it to the user** (e.g. "couldn't verify Mermaid syntax — `mmdc` isn't
installed") and eyeball each diagram against the plain-text rule yourself.

### Step 4: Hand it to the user as a running app

Open the notebook as a **read-only marimo app** (background server + URL) — this
is what the user wants: a clean running app to read and interact with (sliders,
tangles), not the editor. Use the bundled launcher and give them the link:

```bash
bash {skillDir}/scripts/serve.sh notebooks/explanations/<name>.py 2718
# -> Serving ... http://localhost:2718   (read-only app)
```

Then tell the user the URL. Do **not** drop them into `marimo edit` by default —
that's the full editing environment, not a finished explanation. (`marimo edit
<name>.py` is available if *you* need to tweak the notebook, but the deliverable
is the served app.) First render fetches the diagram JS from a CDN, so the app
needs network on first open.

## marimo rules that bite

- **Every top-level variable name must be unique across all cells**, or you get
  `MultipleDefinitionError` at load. Prefix throwaways with `_`; the template's
  names are already distinct — keep them so. This is the #1 failure.
- A cell renders its **last expression**. Return the matplotlib `fig`, the
  `mo.md(...)`, the widget, or `mo.iframe(...)` on the last line.
- For reactivity: define a `mo.ui.slider(...)` in one cell and **return it**;
  read `.value` in a downstream cell. The downstream cell re-runs when the slider
  moves. Same for `TangleSlider` (read `.amount`) wrapped in `mo.ui.anywidget`.
- `mo.ui.table(df, selection=None)` to show a dataframe; **never** `df.to_markdown()`
  (needs `tabulate`, which isn't installed — it crashes the export).
- Seed anything random; the `Scene` builder is already deterministic.

## Determinism and the offline caveat

- The `Scene` builder uses a counter for ids/seeds — identical output every run.
- Markmap, Mermaid, Excalidraw, and Plotly fetch JS from a CDN on first render,
  so the notebook needs network the first time it's opened (matplotlib output is
  offline-safe). marimo's own runtime is also CDN-loaded in a static export.
  Note this in the notebook prose if the audience might be offline.

## When NOT to use this skill

- A one-line factual answer that needs no picture.
- The user wants a data dashboard or charts over a *dataset* (real rows, metrics,
  query results) — that's the `generate-notebooks` skill. This skill explains
  ideas and designs, not data.
- The user wants a map/diagram of an *existing codebase* derived from its source
  (architecture of a repo, call graphs, what-imports-what). This skill works from
  concepts and descriptions, not by reading a repo. (Explaining a concept the code
  happens to implement — "how does
  our rate limiter work" at the whiteboard level — is fine here.)
- A single throwaway diagram where a fenced ```mermaid block in your chat reply
  is enough and a whole notebook is overkill. (If they want something to open,
  explore, and keep — build the notebook.)
