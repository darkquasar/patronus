# Excalidraw — the recommendation sketch (the `Scene` builder)

Excalidraw renders a hand-drawn whiteboard sketch. Via `wigglystuff.Excalidraw`
the canvas is *editable* in the notebook, preloaded with a scene you build in
Python. It carries the **spatial / structural picture**: how the parts sit
together and what connects to what, drawn as boxes-and-arrows on a whiteboard. In
a decision notebook that's the **recommendation made concrete** (the chosen option
in context, or the finalists side by side, with a grouping zone and a margin note
for the condition that would flip the call). In a lesson it's the anatomy of a
system — the components, the zones they live in, the wires between them.

**Draw as much structure as the picture genuinely needs — freestyle, but aim for
no more than ~20 boxes.** This is not the tight "3–5 boxes only" overview it used
to be. Excalidraw is good at spatial layout (things arranged in 2D, grouped into
zones, with labelled wires), and a lesson's system diagram or a decision's
component map legitimately wants a dozen-plus boxes. The only hard rule is
**legibility**: past ~20 boxes a single canvas turns to spaghetti — split it into
two scenes (e.g. control plane, then data plane) rather than cramming. Use
Excalidraw for the *spatial* story and Mermaid for *sequences, branching flows,
and lifecycles* — pick per what you're showing, not by a box budget.

Never hand-write Excalidraw element JSON (each shape is ~25 fields, arrows need
bindings on both ends, labels are separate bound-text elements). Use the bundled
`scripts/excalidraw_scene.py` `Scene` builder, which emits a valid scene from a
high-level spec.

## The `Scene` API

```python
from excalidraw_scene import Scene

s = Scene(title="...")               # optional heading at top of canvas

box_id = s.box(
    "Label",
    color="blue",                    # blue green yellow red purple orange pink gray
    subtitle="one phrase",           # optional smaller second line
    shape="rectangle",               # or "ellipse" / "diamond"
    x=None, y=None, w=None, h=None,  # omit for auto-layout (placed L→R)
)

s.arrow(src_id, dst_id, label="...", dashed=False)   # bound arrow, stays attached
s.flow([a, b, c], labels=["step1", "step2"])         # connect a→b→c in sequence
s.zone([b, c], label="our infra", color="gray")      # translucent grouping box behind
s.note("the one thing to notice", near=c)            # margin annotation (orange ✎)
s.text("free text", x=600, y=120)                    # free-floating label

scene = s.to_dict()                  # -> Excalidraw(scene=scene)
problems = s.validate()              # [] means structurally sound
```

## Layout — think in rows and columns (don't cram one line)

Placement priority per box: explicit `x`/`y` → grid `col`/`row` → automatic
left-to-right cursor. **For anything past a single straight line, use the grid**
(`box(label, col=, row=)`), which centres the box in cell (col, row). A flat
horizontal layout is the most common cause of clutter: every box on row 0 means
every arrow shares one line, and a secondary arrow ends up running collinear with
the main flow.

The reliable pattern: **main flow on row 0, supporting nodes on row 1.**

```python
s = Scene(title="Recommendation")
client  = s.box("Service",      col=0, row=0, color="blue")
pg      = s.box("Postgres",     col=1, row=0, color="green", subtitle="primary")
replica = s.box("Read replica", col=2, row=0, color="green")
blobs   = s.box("Object store", col=0, row=1, color="yellow")   # below the service
s.arrow(client, pg, label="reads/writes")        # horizontal (row 0)
s.arrow(pg, replica, label="replication", dashed=True)
s.arrow(client, blobs, label="blobs", dashed=True)   # vertical — separate line
s.zone([pg, replica], label="managed DB")
s.note("revisit if writes outgrow one primary", near=blobs)
```

Because the arrow router computes edge-to-edge endpoints, a box on a different
row gets a clean vertical/diagonal arrow that doesn't collide with the
horizontal ones.

**Box spacing is automatic — you don't tune `col_gap` any more.** At render the
scene runs a relayout pass that re-spaces every grid box by its *actual* size:
each column is made as wide as its widest box, each row as tall as its tallest,
with a guaranteed gutter between them. A long `subtitle` that makes one box wide
just pushes its neighbours over — boxes (and the arrows between them) can no
longer overlap no matter how long the labels are. `Scene(col_gap=, row_gap=)`
still set the *provisional* pitch but the relayout supersedes them, so you rarely
touch them; just use `col=`/`row=`.

**Arrows route around boxes in between.** If a third box sits on the straight
path between an arrow's endpoints (e.g. you connect a row-0 box to a row-2 box in
the same column, with a box on row 1 between them), the arrow automatically
elbows out into the gutter beside that box instead of running over it — you don't
have to think about it. That said, the cleanest diagrams still avoid the
situation: prefer connecting adjacent cells, and let a box's *neighbour*, not a
box two rows down, be its arrow target where you can.

**Edge labels sit above/beside the line, not on it — and lift clear when the edge
is short.** `arrow(..., label=...)` places the label as free text offset
perpendicular to the arrow. If the label is *wider than the edge it labels* (a
long label on a short arrow), it is automatically lifted into a clear band **above
both boxes** instead of being squashed onto the line where a box would cover it.
You still get the cleanest result with **short labels** (a word or two), but a
long one no longer collides.

**Arrows that share an endpoint fan apart automatically.** When several arrows
converge on one box (or radiate from one), their tips/tails are spread along the
box edge so you can see each as a distinct connector instead of one merged line —
no manual offsetting needed.

**Expressivity without crowding.** Structure earns its keep: use rows to separate
primary from secondary, **solid arrows for the main path and dashed for
secondary/optional**, `zone`s to group what belongs together, color to carry
meaning (e.g. green = your services, yellow = stores), and `note`s for the key
caveats. A richer sketch is fine — the goal is "more signal", not "boxes for their
own sake". If a box isn't teaching the reader something the neighbouring boxes
don't, drop it.

For the simplest case you can still omit `col`/`row` and just `box()` × N then
`flow([...])` for a single left-to-right line.

## Embedding in the notebook

```python
from wigglystuff import Excalidraw
scene, h = s.fitted(view_w=1060)         # medium app; fit zoom to width + snug height
exca_widget = mo.ui.anywidget(Excalidraw(scene=scene, height=h))
exca_widget          # last expression -> renders
```

**Use `fitted()`.** The widget does *not* zoom-to-fit (a wide scene opens showing
only its top-left corner — looks "zoomed in", and being zoomed makes it fuzzy) and
you can't guess a good pane height (too tall → empty band, drawing looks tiny).
`fitted(view_w=…)` returns `(scene, height)`: it fits zoom to the pane WIDTH and
derives the height from the content — no empty band, diagram fills the pane. Zoom
is capped at 1.0 (never upscale = never blurry).

**`view_w` must match your app's content column** — this is the setting that makes
diagrams fill the pane instead of floating small in it. `marimo.App(width="medium")`
is **1110px** → pass `view_w=1060`; `width="compact"`/`"normal"` is 740px → pass
`view_w=700`; `width="full"` is the viewport → pass a large value. Fitting a medium
app's diagrams into 700 is what made them render at ~60% width. Build wide scenes
compact (`Scene(col_gap=300)`) so the fit zoom stays high. Low-level
`to_dict(fit=True, view_w=, view_h=)` still exists if you need to set the height
yourself; `to_dict(fit=False)` gives the raw 100%.

**A compact toolbar (undo/redo bottom bar, no bottom-left zoom) is reactive, not a
bug.** Excalidraw switches to its compact/mobile layout when the effective canvas
viewport is small — most often because the **browser page is zoomed in**, sometimes
an undersized pane. It's the widget adapting: zoom the page back out (or, if the pane
really is too small, fix `view_w`). Nothing to change in the scene or a widget option.

**The toolbar stays visible** (`fitted(..., zen=False)`, the default). Excalidraw
fixes the toolbar's position — it can't be moved or cleanly tucked away. Zen mode
(`zen=True`) *would* hide it, but its exit control overlays the bottom-left **zoom**
buttons and makes zoom in/out awkward, so it's the worse trade for a lesson people
scrub and zoom. Leave `zen=False`; treat the toolbar as a fact of the widget.

`mo.ui.anywidget(...)` makes it reactive; the user can edit the sketch live. To
read edits back, `exca_widget.scene` holds the current scene dict in a
downstream cell. To save what they drew: the underlying widget has
`.save("name.excalidraw")`.

## Keep it legible (the real constraint — not a box count)

- **Freestyle up to ~20 boxes.** Draw the structure the picture needs. Past ~20
  a single canvas gets unreadable — split into two scenes (by subsystem or by
  phase) instead of shrinking everything.
- **Zones** to say "these belong together" (control plane vs worker node, our
  infra vs theirs); **notes** pointing at what matters (the bottleneck, the risk,
  the surprising part).
- Use **color** to carry meaning (e.g. yellow = data stores, blue = clients,
  green = your services) and keep it consistent across the scene.
- Prefer **rows/columns** (`col=`/`row=`) over one long line so arrows don't pile
  onto a single track — this is what keeps a busy scene readable.
- A small legend (a couple of `note`s or a colour key) is fine on a larger scene;
  don't let the diagram depend on memorising what each colour means.

## How arrows avoid overlapping boxes

Arrows are the easy thing to get wrong: a line drawn from one box's *centre* to
another's runs straight through both rectangles. The builder instead computes
each arrow's endpoints on the box **borders** — the point on the edge facing the
other box — and pushes them out by a small gap, so the line lives entirely in
the whitespace between boxes. It also sets Excalidraw bindings
(`startBinding`/`endBinding` with `focus: 0, gap: 2`, and registers the arrow in
each box's `boundElements`) so the arrow stays snapped to the box edges if you
drag things around in the app.

Two consequences for you:

- **Give boxes room.** Auto-layout leaves ~130px between boxes; if you place
  boxes by hand, keep a comparable gap (the research-backed sweet spot is a
  column pitch of ~300–350px for ~180px-wide boxes). Cramped boxes make even
  edge-routed arrows look busy.
- **Lay boxes out on a grid.** The overlap/routing problem grows with box count,
  but the fix is layout, not fewer boxes: put boxes on distinct `col`/`row` cells
  so arrows have their own lanes. A well-gridded 15-box scene reads cleanly; a
  cramped 6-box row does not.

## How it renders (and the offline caveat)

The widget passes the scene straight into Excalidraw's `initialData`, which runs
it through Excalidraw's `restore()` — that fills in any missing element defaults
and recomputes bound arrow endpoints, which is why the builder only needs the
core geometry plus honest cross-references.

The widget **loads Excalidraw from a CDN (esm.sh) on first render**, so this view
needs network the first time and won't draw fully offline. The Python runs fine
headless (so `verify.sh` confirms the scene is valid), but the *picture* only
appears in a browser with network. Say so in the notebook prose if the audience
might be offline.

## Determinism

The builder assigns ids and seeds from a counter, never from a RNG or clock, so
the same spec always produces byte-identical JSON — reproducible and diffable.

## Validation

`s.validate()` returns a list of structural problems (duplicate ids, bindings or
bound-text pointing at missing elements, missing required fields). Empty list =
sound. The `__main__` block in `excalidraw_scene.py` builds a sample scene and
asserts it validates — run `python excalidraw_scene.py` to sanity-check the
helper itself.
