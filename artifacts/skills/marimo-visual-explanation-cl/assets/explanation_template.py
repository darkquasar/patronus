#!/usr/bin/env python3
"""TEMPLATE (DECISION) — a decision explained as a marimo notebook.

Use this when the user is CHOOSING between options (it centres on a weighted
matrix and a recommendation). For explaining how a concept/system works — where
nothing is being ranked — use `concept_template.py` instead.

Copy this next to copies of the three helper scripts (`excalidraw_scene.py`,
`markmap_html.py`, `mermaid_tools.py`). This template includes EVERY decision
lens with a `# >>> FILL` block and a `# >>> KEEP IF` note. Your job:

  1. Fill the framing cell with the decision, the options, and your recommendation.
  2. Keep the 3-4 lenses that illuminate THIS decision; DELETE the rest.
     A forced lens is noise. (See the skill's views.md.)

The lenses, each answering a different question:
  • Markmap   — the option space ("what are the alternatives, what does each involve?")
  • Mermaid   — the right shape: quadrant for a tradeoff, flowchart for decision
                logic, timeline for a rollout, sequence for interactions
  • Weighted matrix — "which option wins, and how sensitive is that to priorities?"
  • Table     — options × criteria at a glance
  • Tangle    — "tweak one assumption inline and watch the conclusion move"
  • Excalidraw— the recommendation made concrete

Run:    .venv/bin/marimo edit decision.py
Verify: bash {skillDir}/scripts/verify.sh decision.py

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
    import matplotlib.pyplot as plt
    import pandas as pd
    from excalidraw_scene import Scene
    from markmap_html import markmap_html
    from mermaid_tools import mermaid_panel
    return Scene, markmap_html, mermaid_panel, pd, plt


@app.cell
def _(mo):
    mo.md(
        """
        # 🧭 Decision: <!-- >>> FILL: the decision in a few words -->

        <!-- >>> FILL: 2-4 sentences. State the options (2-3), the criteria that
        matter, and your tentative recommendation UP FRONT — don't bury it. The
        lenses below help the reader pressure-test that recommendation. -->

        **Options:** A · B · C  **Leaning toward:** <!-- your pick, and the one
        thing that would change it -->
        """
    )
    return


@app.cell
def _(mo):
    # A concise, concrete explanation BELONGS before the diagrams. Default to a
    # few tight sentences / bullets — what each option is and the crux of the
    # tradeoff. Go more verbose ONLY if the user asks for a deeper writeup.
    mo.md(
        """
        ## In brief

        <!-- >>> FILL: 3-5 concise, concrete sentences or bullets. What each
        option actually is, and the central tension between them. Keep it tight. -->

        - **Option A** — <!-- what it is, one line -->
        - **Option B** — <!-- what it is, one line -->
        - **The crux** — <!-- the core tradeoff in one sentence -->
        """
    )
    return


@app.cell
def _(mo):
    # >>> KEEP IF: there's a tradeoff or logic to show (usually yes).
    mo.md("## The tradeoff at a glance")
    return


@app.cell
def _(mermaid_panel, mo):
    # Mermaid — PICK THE TYPE(S) that fit (see references/mermaid.md):
    #   quadrantChart (tradeoff 2x2), flowchart (decision logic / process),
    #   timeline (a rollout), sequenceDiagram (interactions over time).
    # Offer MORE THAN ONE in tabs when two angles both help (e.g. the decision
    # logic AND the runtime sequence) and let the reader pick. Collapse to a
    # single mermaid_panel(...) if one diagram says it all.  >>> FILL.
    # mermaid_panel adds a copy-source + download button under each diagram.
    _quadrant = """
    quadrantChart
        title <!-- axis-A --> vs <!-- axis-B -->
        x-axis "low <!-- A -->" --> "high <!-- A -->"
        y-axis "low <!-- B -->" --> "high <!-- B -->"
        quadrant-1 "best"
        quadrant-2 "tradeoff"
        quadrant-3 "avoid"
        quadrant-4 "tradeoff"
        Option A: [0.3, 0.6]
        Option B: [0.7, 0.8]
        Option C: [0.55, 0.35]
    """
    _flow = """
    flowchart TD
        Q{<!-- key question -->} -->|yes| A[Option A]
        Q -->|no| B[Option B]
    """
    mo.ui.tabs({"Tradeoff": mermaid_panel(mo, _quadrant, download_name="tradeoff.mmd"),
                "Decision logic": mermaid_panel(mo, _flow, download_name="decision.mmd")})
    return


@app.cell
def _(mo):
    # >>> KEEP IF: the choice depends on weightable criteria (the usual case).
    mo.md(
        """
        ## Weigh it yourself

        Drag the sliders to match *your* priorities — the ranking updates live.
        The insight is watching the winner change, not the default order.
        """
    )
    return


@app.cell
def _(pd):
    # >>> FILL: score each option 1-5 on each criterion (your judgement).
    scores = pd.DataFrame({
        "Option":     ["Option A", "Option B", "Option C"],
        "Criterion1": [5, 3, 4],
        "Criterion2": [3, 5, 4],
        "Criterion3": [4, 4, 3],
        "Criterion4": [4, 2, 5],
    })
    return (scores,)


@app.cell
def _(mo):
    # One slider per criterion = how much it matters. >>> FILL labels.
    w_c1 = mo.ui.slider(0, 10, value=6, label="Criterion1")
    w_c2 = mo.ui.slider(0, 10, value=5, label="Criterion2")
    w_c3 = mo.ui.slider(0, 10, value=5, label="Criterion3")
    w_c4 = mo.ui.slider(0, 10, value=4, label="Criterion4")
    mo.vstack([mo.md("**Set your priorities:**"), w_c1, w_c2, w_c3, w_c4])
    return w_c1, w_c2, w_c3, w_c4


@app.cell
def _(scores, w_c1, w_c2, w_c3, w_c4):
    _w = {"Criterion1": w_c1.value, "Criterion2": w_c2.value,
          "Criterion3": w_c3.value, "Criterion4": w_c4.value}
    ranked = (scores.assign(Total=lambda d: sum(d[k] * v for k, v in _w.items()))
                    .sort_values("Total", ascending=False).reset_index(drop=True))
    return (ranked,)


@app.cell
def _(plt, ranked):
    fig, ax = plt.subplots(figsize=(6, 3))
    ax.barh(ranked["Option"], ranked["Total"], color="#4C78A8")
    ax.invert_yaxis()
    ax.set_xlabel("Weighted score")
    ax.set_title("Ranking by your weights")
    fig
    return


@app.cell
def _(mo):
    # >>> KEEP IF: a raw options x criteria grid is useful. Otherwise delete.
    mo.md("## Side by side")
    return


@app.cell
def _(mo, scores):
    mo.ui.table(scores, selection=None)
    return


@app.cell
def _(mo):
    # >>> KEEP IF: one assumption dominates the decision. Otherwise DELETE this
    # and the next cell. (Tangle = tweak a number inline, see the sentence move.)
    mo.md("## What if the key number were different?")
    return


@app.cell
def _(mo):
    from wigglystuff import TangleSlider

    # >>> FILL: the dominant assumption and how it flips the call.
    n_units = mo.ui.anywidget(TangleSlider(amount=10000, min_value=0,
                              max_value=1_000_000, step=10000, suffix=" units"))
    unit_cost = mo.ui.anywidget(TangleSlider(amount=0.5, min_value=0,
                                max_value=5, step=0.1, prefix="$", suffix="/unit"))
    return n_units, unit_cost


@app.cell
def _(mo, n_units, unit_cost):
    mo.md(
        f"At {n_units} and {unit_cost}, the cost is "
        f"**${n_units.amount * unit_cost.amount:,.0f}** — "
        f"{'Option A wins' if n_units.amount < 50000 else 'Option B wins'} at this scale."
    )
    return


@app.cell
def _(mo):
    # >>> KEEP (usually): the recommendation made concrete.
    mo.md("## The recommendation")
    return


@app.cell
def _(Scene):
    # Excalidraw = sketch the recommendation in 2D. Use col=/row= for a clean
    # layout: main flow on row 0, supporting nodes on row 1. Mix solid (primary)
    # and dashed (secondary) arrows, group related boxes with a zone, and point
    # one note at the condition that would change the decision. Aim for ~4-6
    # boxes with structure — expressive but not crowded.  >>> FILL.
    _s = Scene(title="Recommendation: <!-- your pick -->")
    _a = _s.box("<!-- box 1 -->", col=0, row=0, color="blue", subtitle="<!-- phrase -->")
    _b = _s.box("<!-- box 2 -->", col=1, row=0, color="green", subtitle="<!-- phrase -->")
    _c = _s.box("<!-- box 3 -->", col=2, row=0, color="green", subtitle="<!-- phrase -->")
    _d = _s.box("<!-- box 4 -->", col=0, row=1, color="yellow", subtitle="<!-- phrase -->")
    _s.arrow(_a, _b, label="<!-- verb -->")
    _s.arrow(_b, _c, label="<!-- verb -->", dashed=True)
    _s.arrow(_a, _d, label="<!-- verb -->", dashed=True)
    _s.zone([_b, _c], label="<!-- what these share -->")
    _s.note("<!-- the condition that would change the decision -->", near=_d)
    dec_scene = _s.to_dict()
    return (dec_scene,)


@app.cell
def _(dec_scene, mo):
    from wigglystuff import Excalidraw

    exca_widget = mo.ui.anywidget(Excalidraw(scene=dec_scene, height=440))
    exca_widget
    return


@app.cell
def _(mo):
    # >>> KEEP IF: a full map of the option space is a useful reference. Markmap
    # goes LAST — it's the "here's everything we considered" recap, not the lead.
    mo.md("## The full option space")
    return


@app.cell
def _(markmap_html, mo):
    # Markmap = the option space as a hierarchy. One `#` root, options as `##`,
    # considerations as `-` leaves. Go 3 levels deep for a thorough recap.  >>> FILL.
    dec_outline = """
    # <!-- the decision -->
    ## Option A
    - consideration
    - consideration
    ## Option B
    - consideration
    ## Option C
    - consideration
    """
    mo.iframe(markmap_html(dec_outline, height=460), height="480px")
    return


@app.cell
def _(mo):
    mo.md(
        """
        ---
        *In brief up top, the diagram and matrix made the tradeoff explicit, the
        sketch commits to a recommendation, and the map recaps the full option
        space. Reweight the sliders if your priorities differ from the defaults.*
        """
    )
    return


if __name__ == "__main__":
    app.run()
