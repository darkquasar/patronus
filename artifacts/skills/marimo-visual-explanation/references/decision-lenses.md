# Interactive lenses — the reactive views

These are the interactive lenses. The **weighted matrix** and **radar** are
decision-specific (they rank/compare options); the **comparison table** and
**tangle** widget suit any topic — a tangle is just as useful for "watch the
behaviour move as this parameter changes" in a concept explanation as for "watch
the winner change" in a decision. Each snippet below was verified to render live
and export clean in marimo. They lean on `mo.ui` reactivity, so they shine in
`marimo run`/`edit`; in a static HTML export the chart/table renders but the
interactivity is frozen at its default — fine, and worth a note in the prose.

Remember the marimo rule throughout: **every top-level variable name must be
unique across cells**; prefix throwaways with `_`, and `return` anything a later
cell reads.

## 1. Weighted decision matrix — the centerpiece

The single highest-value lens for a tradeoff. Sliders set how much each
criterion matters; each option is pre-scored 1-5 per criterion; the weighted
total re-ranks **live** as the user drags. It answers the real question:
"*which option wins, and how sensitive is that to my priorities?*" Deps
(matplotlib, pandas) are already in the venv, and the chart embeds as an offline
PNG.

```python
@app.cell
def _():
    import matplotlib.pyplot as plt
    import pandas as pd
    return pd, plt

@app.cell
def _(pd):
    # Pre-score each option 1-5 on each criterion (your judgement).
    scores = pd.DataFrame({
        "Option":      ["Postgres", "DynamoDB", "MongoDB"],
        "Consistency": [5, 3, 3],
        "Scale":       [3, 5, 4],
        "Low ops":     [4, 5, 4],
        "Flexibility": [5, 2, 4],
        "Low cost":    [4, 3, 3],
    })
    return (scores,)

@app.cell
def _(mo):
    # One slider per criterion = how much it matters (0-10).
    w_consistency = mo.ui.slider(0, 10, value=7, label="Consistency")
    w_scale       = mo.ui.slider(0, 10, value=5, label="Scale")
    w_ops         = mo.ui.slider(0, 10, value=6, label="Low ops")
    w_flex        = mo.ui.slider(0, 10, value=4, label="Flexibility")
    w_cost        = mo.ui.slider(0, 10, value=5, label="Low cost")
    mo.vstack([mo.md("**Set your priorities:**"),
               w_consistency, w_scale, w_ops, w_flex, w_cost])
    return w_consistency, w_cost, w_flex, w_ops, w_scale

@app.cell
def _(scores, w_consistency, w_cost, w_flex, w_ops, w_scale):
    _w = {"Consistency": w_consistency.value, "Scale": w_scale.value,
          "Low ops": w_ops.value, "Flexibility": w_flex.value, "Low cost": w_cost.value}
    ranked = (scores.assign(Total=lambda d: sum(d[k] * v for k, v in _w.items()))
                    .sort_values("Total", ascending=False).reset_index(drop=True))
    return (ranked,)

@app.cell
def _(plt, ranked):
    fig, ax = plt.subplots(figsize=(6, 3))
    ax.barh(ranked["Option"], ranked["Total"], color="#4C78A8")
    ax.invert_yaxis()                      # winner on top
    ax.set_xlabel("Weighted score")
    ax.set_title("Drag the sliders — the ranking updates")
    fig                                    # last expression renders it
    return
```

Notes: the matplotlib `fig` must be the cell's last expression. Don't put emoji
in matplotlib labels (font gaps). Tell the user in prose that the *insight* is
watching the ranking flip as they reweight — not the default order.

## 2. Comparison table — options × criteria

The natural baseline. Use `mo.ui.table` (sortable) for the plain grid, or a
pandas Styler heatmap when you want the winning cell per row to pop.

```python
@app.cell
def _(mo, scores):
    mo.ui.table(scores, selection=None)        # sortable grid
    return

# Optional heat-shaded version (winner per criterion stands out):
@app.cell
def _(mo, scores):
    _num = scores.set_index("Option")
    _styled = _num.style.background_gradient(cmap="Greens", axis=0)
    mo.Html(_styled.to_html())
    return
```

**Never use `df.to_markdown()`** — it needs the `tabulate` package (not
installed) and crashes the export. Build markdown tables by hand or use the
above.

## 3. Tangle widgets — tweak one assumption inline

`wigglystuff.TangleSlider` puts an editable number inside a sentence: the reader
drags it and the conclusion updates. Perfect for the "what if this one number
were different" moment in brainstorming.

```python
@app.cell
def _(mo):
    from wigglystuff import TangleSlider
    return (TangleSlider,)

@app.cell
def _(TangleSlider, mo):
    users = mo.ui.anywidget(TangleSlider(amount=10000, min_value=0,
                            max_value=1_000_000, step=10000, suffix=" users"))
    cost  = mo.ui.anywidget(TangleSlider(amount=0.5, min_value=0, max_value=5,
                            step=0.1, prefix="$", suffix="/user/mo"))
    return cost, users

@app.cell
def _(cost, mo, users):
    # Interpolate the widget OBJECTS into the markdown to get inline editable
    # numbers; read their values via `.amount`.
    mo.md(
        f"At {users} and {cost}, monthly infra is "
        f"**${users.amount * cost.amount:,.0f}** — "
        f"{'DynamoDB on-demand wins' if users.amount < 50000 else 'a provisioned cluster is cheaper'}."
    )
    return
```

`TangleChoice` is the categorical sibling (read `.choice`). Both must be wrapped
in `mo.ui.anywidget(...)`.

## 4. Plotly radar — overall profile comparison (optional)

Best when comparing 2-3 options across **5+ criteria as overlapping shapes** —
the eye reads "this one is spiky on scale but weak on consistency" faster than a
table. Costs a `plotly` install plus a CDN load, so use it only when the shape
comparison genuinely helps (and you didn't already cover it with the matrix).

```python
# requires: uv pip install --python .venv/bin/python plotly
@app.cell
def _(mo):
    import plotly.graph_objects as go
    return (go,)

@app.cell
def _(go, mo):
    _criteria = ["Consistency", "Scale", "Low ops", "Flexibility", "Low cost"]
    _data = {"Postgres": [5, 3, 4, 5, 4], "DynamoDB": [3, 5, 5, 2, 3]}
    _fig = go.Figure()
    for _name, _vals in _data.items():
        _fig.add_trace(go.Scatterpolar(
            r=_vals + [_vals[0]], theta=_criteria + [_criteria[0]],
            fill="toself", name=_name))            # repeat first point to close
    _fig.update_layout(polar=dict(radialaxis=dict(range=[0, 5])), showlegend=True)
    mo.ui.plotly(_fig)
    return
```

## Picking among these

- A tradeoff with weightable criteria → **weighted matrix** (almost always) +
  maybe a **table** for the raw scores.
- A close call across many dimensions → add a **radar**.
- A single dominant assumption ("it's all about cost per user") → a **tangle**
  line makes that explorable.
- Don't include all four. Two decision lenses plus a Markmap and an Excalidraw is
  usually plenty.
