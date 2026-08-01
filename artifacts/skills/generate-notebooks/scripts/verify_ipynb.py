#!/usr/bin/env python3
"""Execute a .ipynb headless to prove every cell runs, and save its figures.

A notebook that has never been run is a notebook that does not work. This runs
each code cell in order in one shared namespace, with a non-interactive
matplotlib backend, and writes any figures produced to an output directory. It
is the smoke test for a Colab notebook: if this passes, "Runtime, Run all" in
Colab will too.

What it handles:
  * strips Colab/Jupyter magics (lines starting with %, !) which are not valid
    plain Python but are fine in Colab.
  * forces matplotlib's Agg backend so `plt.show()` does not need a display.
  * after each cell, saves any open matplotlib figures as PNGs.

Usage:
    verify_ipynb.py notebook.ipynb [--figdir out_figures]

Exit code is non-zero if any cell raises, so it works as a CI / pre-commit gate.
"""
from __future__ import annotations

import argparse
import json
import os
import sys


def run(path: str, figdir: str | None) -> int:
    import matplotlib
    matplotlib.use("Agg")  # no display needed
    import matplotlib.pyplot as plt

    with open(path, encoding="utf-8") as fh:
        nb = json.load(fh)

    if figdir:
        os.makedirs(figdir, exist_ok=True)

    ns: dict = {}
    fig_count = 0
    for i, cell in enumerate(nb["cells"]):
        if cell["cell_type"] != "code":
            continue
        # Drop magics and shell escapes; keep the rest verbatim.
        src = "".join(
            ln for ln in cell["source"]
            if not ln.lstrip().startswith(("%", "!"))
        )
        try:
            exec(compile(src, f"cell[{i}]", "exec"), ns)
        except Exception as exc:  # noqa: BLE001  (we re-raise after reporting)
            print(f"FAIL cell[{i}]: {type(exc).__name__}: {exc}", file=sys.stderr)
            raise
        # Persist any figures this cell created.
        for num in plt.get_fignums():
            fig = plt.figure(num)
            if figdir:
                dest = os.path.join(figdir, f"cell{i}_fig{num}.png")
                fig.savefig(dest, dpi=110, bbox_inches="tight")
                fig_count += 1
            plt.close(fig)

    print(f"OK: all code cells executed; {fig_count} figure(s) saved"
          + (f" to {figdir}" if figdir else ""))
    return 0


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("notebook", help="path to the .ipynb")
    ap.add_argument("--figdir", default=None, help="directory to save figures into")
    args = ap.parse_args(argv)
    return run(args.notebook, args.figdir)


if __name__ == "__main__":
    sys.exit(main())
