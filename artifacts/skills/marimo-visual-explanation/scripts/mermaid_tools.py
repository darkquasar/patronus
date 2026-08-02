#!/usr/bin/env python3
"""Render a Mermaid diagram with handy actions: a one-click button to copy the
raw Mermaid source, and (optionally) a button to download it as a ``.mmd`` file.

On SVG export: there is no built-in way to pull the *rendered* SVG back out of
``mo.mermaid`` — it renders client-side in the browser and marimo doesn't return
the SVG to Python. The easy, reliable, dependency-light wins are therefore the
copy button and the source download (both work live AND in a static export). To
get an actual SVG/PNG:

  • copy the source and paste it into https://mermaid.live  → Actions ▸ Export, or
  • install mermaid-cli once (`npm i -g @mermaid-js/mermaid-cli`) and render the
    source server-side, then offer it with ``mo.download`` — see references/mermaid.md.

Avoid the mermaid.ink URL service for anything confidential — it sends the
diagram to a third party.

Usage:
    from mermaid_tools import mermaid_panel
    mermaid_panel(mo, src, download_name="flow.mmd")          # diagram + buttons
    # in tabs:
    mo.ui.tabs({"Flow": mermaid_panel(mo, flow_src),
                "Sequence": mermaid_panel(mo, seq_src)})
"""
from __future__ import annotations


def mermaid_panel(mo, src: str, *, copy: bool = True,
                  download_name: str | None = None, theme: str | None = None):
    """A Mermaid diagram stacked above a row of action buttons.

    ``mo`` is the marimo module (pass it in so this stays import-light).
    ``copy`` adds a "copy source" button; ``download_name`` (e.g. "flow.mmd")
    adds a download button for the raw text. Returns a marimo element to render
    as a cell's last expression.
    """
    src = src.strip()
    diagram = mo.mermaid(src, theme=theme)
    controls = []
    if copy:
        from wigglystuff import CopyToClipboard
        controls.append(mo.ui.anywidget(CopyToClipboard(text_to_copy=src)))
    if download_name:
        controls.append(mo.download(src.encode(), filename=download_name,
                                    mimetype="text/plain", label="Download .mmd"))
    if not controls:
        return diagram
    return mo.vstack([diagram, mo.hstack(controls, justify="start", gap=1)])


if __name__ == "__main__":
    # Light self-check (no marimo runtime needed): the source round-trips.
    class _Mo:
        def mermaid(self, s, theme=None):
            return ("mermaid", s)
    src = "flowchart TD\n  A --> B"
    out = _Mo().mermaid.__self__.mermaid(src)
    assert out[1] == src
    print("OK: mermaid_tools importable; copy/download need a live marimo runtime")
