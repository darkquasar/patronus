#!/usr/bin/env python3
"""TEMPLATE (CONCEPT/SYSTEM) — how something works, explained as a marimo notebook.

Use this when the user wants to UNDERSTAND something (a protocol, algorithm,
architecture, data flow, lifecycle) rather than choose between options. There is
nothing to rank here, so there is no weighted matrix and no recommendation — the
spine is a Mermaid diagram of the flow/lifecycle, backed by a component sketch and
a decomposition recap. For a DECISION between options, use
`explanation_template.py` instead.

Copy this next to copies of the three helper scripts (`excalidraw_scene.py`,
`markmap_html.py`, `mermaid_tools.py`). This template includes the concept lenses
with a `# >>> FILL` block and a `# >>> KEEP IF` note. Your job:

  1. Fill the framing cell with the one-sentence "what this is" and its parts.
  2. Keep the 2-4 lenses that illuminate THIS topic; DELETE the rest.
     A forced lens is noise. (See the skill's views.md concept example.)

The lenses, each answering a different question:
  • Mermaid    — the spine: sequenceDiagram (a protocol / conversation over time),
                 flowchart (a process with branches), stateDiagram-v2 (a lifecycle)
  • Excalidraw — the components and how they fit ("who holds/does what")
  • Tangle     — "change this one parameter, watch the behaviour move" (optional)
  • Markmap    — the pieces as a decomposition, a closing reference

Run:    .venv/bin/marimo edit concept.py
Verify: bash {skillDir}/scripts/verify.sh concept.py

marimo gotcha: every top-level variable name must be unique ACROSS cells. The
names below are already distinct — keep them; prefix throwaways with `_`.
"""
import marimo

app = marimo.App(width="medium")


@app.cell
def _():
    import marimo as mo
    return (mo,)


@app.cell
def _():
    import os
    import sys
    _here = os.path.dirname(os.path.abspath(__file__))
    if _here not in sys.path:
        sys.path.insert(0, _here)
    from excalidraw_scene import Scene
    from markmap_html import markmap_html
    from mermaid_tools import mermaid_panel
    return Scene, markmap_html, mermaid_panel


@app.cell
def _(mo):
    mo.md(
        """
        # 🔍 How <!-- >>> FILL: the thing --> works

        <!-- >>> FILL: 1-2 sentences. What this is, in plain terms, and why it
        exists / what problem it solves. Put the one-line mental model UP FRONT. -->
        """
    )
    return


@app.cell
def _(mo):
    # A concise, concrete explanation BELONGS before the diagrams. Default to a
    # few tight sentences / bullets — the moving parts and how they relate.
    # Go more verbose ONLY if the user asks for a deeper writeup.
    mo.md(
        """
        ## In brief

        <!-- >>> FILL: 3-5 concise, concrete sentences or bullets. The main parts
        and the single idea that ties them together. Keep it tight. -->

        - **<!-- part 1 -->** — <!-- what it does, one line -->
        - **<!-- part 2 -->** — <!-- what it does, one line -->
        - **The key idea** — <!-- the one insight that makes it click -->
        """
    )
    return


@app.cell
def _(mo):
    # >>> KEEP: the flow/lifecycle is the spine of a concept explanation.
    mo.md("## The flow, step by step")
    return


@app.cell
def _(mermaid_panel, mo):
    # Mermaid — PICK THE TYPE that fits (see references/mermaid.md):
    #   sequenceDiagram (a protocol / conversation over time between actors),
    #   flowchart (a process with branches / error paths),
    #   stateDiagram-v2 (a lifecycle: states and transitions).
    # Offer TWO angles in tabs only when both genuinely help (e.g. the runtime
    # sequence AND the state lifecycle); collapse to one if it says it all. >>> FILL.
    # mermaid_panel adds a copy-source + download button under each diagram.
    _sequence = """
    sequenceDiagram
        participant A as <!-- actor 1 -->
        participant B as <!-- actor 2 -->
        participant C as <!-- actor 3 -->
        A->>B: <!-- request / message -->
        B->>C: <!-- next step -->
        C->>B: <!-- response -->
        B->>A: <!-- result -->
    """
    _states = """
    stateDiagram-v2
        [*] --> Idle
        Idle --> Working: <!-- trigger -->
        Working --> Done: <!-- success -->
        Working --> Idle: <!-- retry / reset -->
        Done --> [*]
    """
    mo.ui.tabs({"Sequence": mermaid_panel(mo, _sequence, download_name="flow.mmd"),
                "Lifecycle": mermaid_panel(mo, _states, download_name="states.mmd")})
    return


@app.cell
def _(mo):
    # >>> KEEP IF: one parameter makes the behaviour click (window size, timeout,
    # replica count, batch size). Otherwise DELETE this and the next cell.
    # (Tangle = tweak a number inline, see the consequence move.)
    mo.md("## What changes if you turn this knob?")
    return


@app.cell
def _(mo):
    from wigglystuff import TangleSlider

    # >>> FILL: the parameter that governs the behaviour, and its consequence.
    param = mo.ui.anywidget(TangleSlider(amount=3, min_value=1, max_value=10,
                            step=1, suffix=" replicas"))
    return (param,)


@app.cell
def _(mo, param):
    mo.md(
        f"With **{param.amount}**, the system "
        f"{'tolerates more failures but costs more to coordinate' if param.amount > 3 else 'is cheaper but less fault-tolerant'} "
        f"— <!-- >>> FILL: the real consequence at this setting -->."
    )
    return


@app.cell
def _(mo):
    # >>> KEEP (usually): the components made concrete.
    mo.md("## The pieces and how they fit")
    return


@app.cell
def _(Scene):
    # Excalidraw = sketch the components in 2D. Use col=/row= for a clean layout:
    # main path on row 0, supporting parts on row 1. Mix solid (primary path) and
    # dashed (secondary) arrows, group related boxes with a zone, and point a note
    # at the one subtlety worth flagging. Draw as much structure as it needs —
    # freestyle up to ~20 boxes; past that, split into two scenes.  >>> FILL.
    _s = Scene(title="<!-- the system -->: the parts")
    _a = _s.box("<!-- part 1 -->", col=0, row=0, color="blue", subtitle="<!-- role -->")
    _b = _s.box("<!-- part 2 -->", col=1, row=0, color="green", subtitle="<!-- role -->")
    _c = _s.box("<!-- part 3 -->", col=2, row=0, color="yellow", subtitle="<!-- role -->")
    _d = _s.box("<!-- part 4 -->", col=1, row=1, color="green", subtitle="<!-- role -->")
    _s.arrow(_a, _b, label="<!-- verb -->")
    _s.arrow(_b, _c, label="<!-- verb -->")
    _s.arrow(_b, _d, label="<!-- verb -->", dashed=True)
    _s.zone([_b, _c], label="<!-- what these share -->")
    _s.note("<!-- the one subtlety worth flagging -->", near=_d)
    concept_scene = _s.to_dict()
    return (concept_scene,)


@app.cell
def _(concept_scene, mo):
    from wigglystuff import Excalidraw

    exca_widget = mo.ui.anywidget(Excalidraw(scene=concept_scene, height=440))
    exca_widget
    return


@app.cell
def _(mo):
    # >>> KEEP IF: a full decomposition of the pieces is a useful reference.
    # Markmap goes LAST — the "here's every part / term" recap, not the lead.
    mo.md("## The full picture")
    return


@app.cell
def _(markmap_html, mo):
    # Markmap = the concept decomposed as a hierarchy. One `#` root, sub-systems
    # as `##`, details as `-` leaves. Go 3 levels deep for a thorough recap.  >>> FILL.
    concept_outline = """
    # <!-- the concept -->
    ## <!-- sub-system 1 -->
    - detail
    - detail
    ## <!-- sub-system 2 -->
    - detail
    ## <!-- sub-system 3 -->
    - detail
    """
    mo.iframe(markmap_html(concept_outline, height=460), height="480px")
    return


@app.cell
def _(mo):
    mo.md(
        """
        ---
        *The mental model up top, the diagram traces the flow, the sketch shows how
        the parts fit, and the map recaps every piece. Follow the sequence tab for
        the runtime path; the lifecycle tab for the states it moves through.*
        """
    )
    return


if __name__ == "__main__":
    app.run()
