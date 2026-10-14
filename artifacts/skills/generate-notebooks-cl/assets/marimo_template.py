#!/usr/bin/env python3
"""Minimal marimo notebook showing the patterns that keep marimo happy.

Run it:   .venv/bin/marimo edit marimo_template.py
Verify:   .venv/bin/marimo export html marimo_template.py -o /tmp/out.html

The two rules that bite people are demonstrated below:
1. Every top-level variable name must be unique ACROSS cells (reactive dataflow).
   Use `_`-prefixed names for throwaways; marimo treats them as cell-local.
2. A cell renders its LAST expression. Return a matplotlib figure to show it,
   and return any names other cells need.
"""
import marimo

app = marimo.App(width="medium")


@app.cell
def _():
    import marimo as mo
    return (mo,)


@app.cell
def _(mo):
    mo.md("# Demo\n\nShared helpers are defined once and returned, then reused.")
    return


@app.cell
def _():
    # Define a helper ONCE and return it; other cells receive it as an argument.
    def fmt(x):
        return f"{x:,.1f}"
    data = [3, 1, 4, 1, 5, 9, 2, 6]
    return data, fmt


@app.cell
def _(data, fmt, mo):
    # `_total` is throwaway, so it is `_`-prefixed and stays cell-local.
    _total = sum(data)
    mo.md(f"Sum is **{fmt(_total)}** across {len(data)} points.")
    return


@app.cell
def _(data):
    import matplotlib.pyplot as plt
    fig, ax = plt.subplots(figsize=(5, 3))
    ax.bar(range(len(data)), data, color="#4285f4")
    ax.set_title("Returning the figure renders it")
    fig  # last expression -> displayed
    return


if __name__ == "__main__":
    app.run()
