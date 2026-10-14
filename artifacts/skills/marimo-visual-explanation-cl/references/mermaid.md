# Mermaid — pick the diagram type that fits the question

Mermaid is built into marimo: `mo.mermaid(diagram_str)` (renders client-side from
a CDN). The skill's whole stance is that **Mermaid is many diagram types, not
one** — your job is to choose the shape that illuminates *this* topic or
decision, and to honour an explicit request ("show me a timeline").

All six types below were verified to render and export cleanly via `mo.mermaid`.

## Choosing the type

| The thing you're showing | Type | Why |
|---|---|---|
| A tradeoff / where options sit | `quadrantChart` | a 2×2 places each option visually |
| Decision logic ("if X, choose A") or a process | `flowchart` | branches and steps |
| Sequencing, phases, a rollout/migration | `timeline` | events along time |
| Interactions over time between actors | `sequenceDiagram` | order + who-calls-whom |
| Concept decomposition | `mindmap` | nested ideas radiating from a center |
| Something that moves between states | `stateDiagram-v2` | states + transitions |

The anti-pattern is defaulting to a flowchart (or a sequence diagram) for
everything. For a decision skill, **`quadrantChart` is often the most useful
single diagram** — reach for it first when the topic is a tradeoff.

## Verified snippets

### quadrantChart — the tradeoff 2×2 (reach for this first)

```
quadrantChart
    title Cost vs Flexibility
    x-axis "Less flexible" --> "More flexible"
    y-axis "Lower cost" --> "Higher cost"
    quadrant-1 "Flexible but pricey"
    quadrant-2 "Pricey, less flexible"
    quadrant-3 "Cheap, less flexible"
    quadrant-4 "Sweet spot"
    Option A: [0.3, 0.4]
    Option B: [0.75, 0.8]
    Option C: [0.6, 0.35]
```

Points are `Label: [x, y]` with **0-1 floats**. Name the axes by their two ends
(`"low" --> "high"`). Keep titles short.

### flowchart — decision logic or process

```
flowchart TD
    Start{Need multi-region writes?} -->|yes| Dynamo[DynamoDB]
    Start -->|no| Q2{Rich queries / joins?}
    Q2 -->|yes| PG[Postgres]
    Q2 -->|no| Mongo[MongoDB]
```

`TD` top-down (decision trees) or `LR` left-right (pipelines). Quote node text
with special characters: `A["GET /users (paginated)"]`.

### timeline — phases / rollout

```
timeline
    title Migration plan
    section Phase 1
        Stand up new DB : dual-write
    section Phase 2
        Backfill : verify parity
    section Phase 3
        Cut over reads : retire old DB
```

### sequenceDiagram — interactions over time

```
sequenceDiagram
    participant C as Client
    participant S as Service
    participant D as DB
    C->>S: request
    S->>D: query
    alt found
        D-->>S: row
        S-->>C: 200
    else missing
        S-->>C: 404
    end
```

### mindmap — concept decomposition

```
mindmap
  root((Caching))
    Where
      Client
      CDN
      App
    Risks
      Staleness
      Invalidation
```

### stateDiagram-v2 — state machine

```
stateDiagram-v2
    [*] --> Idle
    Idle --> Running : start
    Running --> Idle : stop
    Running --> Failed : error
    Failed --> Idle : reset
```

## Two diagrams in tabs (let the reader pick the angle)

A process often deserves two views — the **decision logic / branches**
(`flowchart`) and the **runtime sequence** (`sequenceDiagram`). Rather than
choosing for the reader or stacking both, offer them in tabs:

```python
mo.ui.tabs({
    "Flow (every branch)":      mo.mermaid(flow_src),
    "Sequence (who calls whom)": mo.mermaid(seq_src),
})
```

Verified to render and export cleanly. Use it when both angles genuinely help;
collapse to a single `mo.mermaid(...)` when one diagram says it all. The same
trick works for any pair (e.g. a `quadrantChart` tradeoff alongside a `flowchart`
decision tree).

## Copy the source / export an SVG

Use `mermaid_panel` (bundled `mermaid_tools.py`) instead of a bare `mo.mermaid`
to add a one-click **copy-source** button and an optional **.mmd download** —
both work live and in a static export:

```python
from mermaid_tools import mermaid_panel
mermaid_panel(mo, src, download_name="flow.mmd")
# in tabs:
mo.ui.tabs({"Flow": mermaid_panel(mo, flow_src),
            "Sequence": mermaid_panel(mo, seq_src)})
```

**Getting an actual SVG/PNG.** There is *no* built-in way to pull the rendered
SVG back out of `mo.mermaid` — it renders client-side and marimo never returns
the SVG to Python. So the options, easiest first:

1. **Copy the source → https://mermaid.live → Actions ▸ Export** (SVG/PNG). The
   copy button makes this two clicks. Best default.
2. **Render server-side with mermaid-cli** and offer it via `mo.download`:
   ```python
   # one-time: npm i -g @mermaid-js/mermaid-cli   (pulls headless Chromium)
   import subprocess, pathlib
   pathlib.Path("d.mmd").write_text(src)
   subprocess.run(["mmdc", "-i", "d.mmd", "-o", "d.svg"], check=True)
   mo.download(pathlib.Path("d.svg").read_bytes(), filename="diagram.svg",
               mimetype="image/svg+xml", label="Download SVG")
   ```
   Real SVG, but a heavy dependency — only add it if SVG export is a hard
   requirement.
3. Avoid the `mermaid.ink` URL service for anything confidential — it sends the
   diagram to a third party.

## Syntax that bites (all types)

- **Indentation matters** for `mindmap`, `timeline`, `quadrantChart`. Keep it
  consistent; the leading whitespace from a triple-quoted Python string is fine
  (Mermaid ignores a uniform indent), but ragged indentation breaks these three.
- **Quote node labels** containing `()`, `:`, `,`: `A["text (x)"]`.
- **Keep `Note`, edge-label, and arrow-label text plain ASCII prose.** In
  `sequenceDiagram`/`flowchart`, the text after `Note over A,B:` or on a `-->>`
  arrow is *unquoted* and read to end-of-line, so a stray punctuation char there
  aborts the whole diagram with a cryptic `Parse error ... got 'INVALID'`. The
  ones that actually bit in practice: an **apostrophe** (`won't`), a **semicolon**
  (`a; b`), and inline **`<br/>`**. Write "will not", split clauses with a full
  stop, and keep notes to one line. (Node text you can rescue by quoting —
  `A["a; b"]` — but note/label text has no such escape hatch, so just keep it
  clean.) This is the single most common render failure and it slips past
  `verify.sh`, because Mermaid parses in the browser, not during the headless
  export — see below.
- **Mermaid parses in the browser, not during the headless export.** The HTML
  export confirms the notebook's Python runs and the diagram *payload* is present,
  but Mermaid (like Markmap and Excalidraw) only parses when a browser renders it
  from the CDN — so a Mermaid syntax error exports "clean" and only surfaces on
  screen. `verify.sh` closes this gap **when `mmdc` is installed**: it extracts
  every Mermaid block and parses it with `mmdc`, failing loudly on bad syntax.
  When `mmdc` is absent it prints `Mermaid lint: SKIPPED — ... mmdc is missing`
  and passes anyway (graceful degradation), so **without `mmdc` you still must
  eyeball every diagram for the plain-text rule above.** `mmdc`
  (`npm i -g @mermaid-js/mermaid-cli`) is a heavy Node+Chromium dependency, so the
  skill treats it as optional, never assumed.
- Use **`stateDiagram-v2`**, not `stateDiagram`.
- `mo.mermaid(src, theme="dark")` to force a theme; omit to follow the app.
- Comments: `%% like this`.
