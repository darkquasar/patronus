#!/usr/bin/env python3
"""Build a Google Colab / Jupyter .ipynb from a simple cell spec.

Authoring notebook JSON by hand is error prone (one stray comma breaks the
whole file). This builds a valid notebook from a list of cells, so the agent
writes prose and code, not JSON plumbing.

Spec format (JSON file or Python list passed to ``build``):

    [
      {"type": "markdown", "source": "# Title\n\nIntro text."},
      {"type": "code", "source": "import numpy as np", "hide_code": false},
      {"type": "code", "source": "#@title Results\nprint(42)", "hide_code": true}
    ]

`hide_code: true` sets the Colab `cellView: form` metadata so the cell shows
only its title and output, not the source. Pair it with a `#@title` first line
for a clean, results-only report.

CLI:
    build_ipynb.py spec.json out.ipynb
    build_ipynb.py spec.json out.ipynb --hide-all-code   # report mode
"""
from __future__ import annotations

import argparse
import json
import sys


def _source_lines(text: str) -> list[str]:
    """nbformat wants a list of strings, newline-terminated except the last."""
    parts = text.split("\n")
    return [p + "\n" for p in parts[:-1]] + [parts[-1]]


def build(cells: list[dict], hide_all_code: bool = False) -> dict:
    """Return a notebook dict (nbformat 4.5) from a list of cell specs."""
    out = []
    for c in cells:
        kind = c["type"]
        src = _source_lines(c["source"])
        if kind == "markdown":
            out.append({"cell_type": "markdown", "metadata": {}, "source": src})
        elif kind == "code":
            meta = {}
            if c.get("hide_code") or hide_all_code:
                meta["cellView"] = "form"  # Colab: show output/title only
            out.append({"cell_type": "code", "metadata": meta,
                        "execution_count": None, "outputs": [], "source": src})
        else:
            raise ValueError(f"unknown cell type: {kind!r}")
    return {
        "cells": out,
        "metadata": {
            "colab": {"provenance": [], "toc_visible": True},
            "kernelspec": {"name": "python3", "display_name": "Python 3"},
            "language_info": {"name": "python"},
        },
        "nbformat": 4,
        "nbformat_minor": 5,
    }


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("spec", help="JSON file with the list of cell specs")
    ap.add_argument("out", help="output .ipynb path")
    ap.add_argument("--hide-all-code", action="store_true",
                    help="set every code cell to form view (results-only report)")
    args = ap.parse_args(argv)

    with open(args.spec, encoding="utf-8") as fh:
        cells = json.load(fh)
    nb = build(cells, hide_all_code=args.hide_all_code)
    with open(args.out, "w", encoding="utf-8") as fh:
        json.dump(nb, fh, indent=1)
    print(f"wrote {args.out} ({len(cells)} cells)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
