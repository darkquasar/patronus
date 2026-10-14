# Marimo notebooks

Marimo notebooks are plain Python files (`.py`), not JSON. Each cell is a
function decorated with `@app.cell`. The runtime is **reactive**: it builds a
dependency graph from the variables each cell defines and reads, and re-runs
downstream cells when an input changes. That model is the source of every
marimo-specific rule below.

Use marimo when the deliverable is an **interactive local app or dashboard**
(sliders, dropdowns, live re-rendering) or you want a notebook that is also a
clean diffable Python file. Use Colab instead when the goal is to **share** a
notebook that others run in the browser (see [colab.md](colab.md)).

## The rules that actually bite

1. **Every top-level variable name must be unique across all cells.** Two cells
   that both define `df`, `i`, or `band` raise `MultipleDefinitionError` at
   load time. This is the single most common failure.
   - For throwaways (loop variables, scratch values), prefix with `_`. Marimo
     treats `_name` as cell-local, so duplicates are fine: `for _i, _row in ...`.
   - For anything another cell needs, give it a unique, descriptive name and
     return it from the cell.

2. **Define shared helpers once, return them, consume them as arguments.** If
   three cells need a `fmt()` helper, define it in one cell, `return fmt`, and
   the cells that use it take `fmt` as a parameter (`def _(fmt, mo): ...`).
   Do not redefine it in each cell.

3. **A cell displays its last expression.** To show a value or a chart, make it
   the last line. For matplotlib, build the figure and put `fig` on the last
   line. For markdown, `mo.md("...")` as the last expression.

4. **No hidden global state.** A cell may not reassign a name another cell
   already defined. If you need to accumulate, build the whole value in one
   cell.

## Display

- Markdown: `mo.md(f"""...""")` as the last expression. f-strings pull in
  computed values.
- Matplotlib: `fig, ax = plt.subplots(...)`; build the plot; last line `fig`.
- Tables/JSON: returning a `dict`, `list`, or dataframe renders it.
- Widgets: `mo.ui.slider(...)`, `mo.ui.dropdown(...)`. Bind to a variable,
  return it, and read its `.value` in a downstream cell to get reactivity.

## Determinism

- Seed anything random: `nx.spring_layout(G, seed=42)`, `np.random.default_rng(0)`.
- Do not call `datetime.now()` or unseeded RNGs in cells whose output you want
  to be stable.

## Run and verify

```bash
# interactive editing (opens a browser, local only)
.venv/bin/marimo edit notebook.py

# read-only app
.venv/bin/marimo run notebook.py

# headless verification: this exercises EVERY cell and is the smoke test.
.venv/bin/marimo export html notebook.py -o /tmp/out.html
```

After export, check for problems:

```bash
grep -c "MultipleDefinitionError\|Traceback (most recent" /tmp/out.html   # want 0
grep -c "data:image/png;base64" /tmp/out.html                            # figure count
```

`marimo export` runs the notebook top to bottom, so a clean export with the
expected number of figures means the notebook genuinely works. A
`MultipleDefinitionError` in the output means rule 1 was violated.

## Layout files

`marimo.App(layout_file="layouts/foo.grid.json")` references a grid layout
relative to the notebook file. If you move the notebook, move the layout with
it (keep it under a `layouts/` folder next to the `.py`), or the reference
breaks.

A ready-to-copy starting point is in `assets/marimo_template.py`.
