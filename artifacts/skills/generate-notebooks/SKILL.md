---
name: generate-notebooks
description: >
  Best practices and helper scripts for generating Google Colab (.ipynb) and
  marimo (.py) notebooks, including data-visualisation and dashboard notebooks.
  Use this whenever the user wants to create, build, or scaffold a notebook, a
  Colab, a marimo app, an interactive dashboard, or a data-visualisation /
  charts notebook, even if they do not say the word "notebook" explicitly. Also
  use it when setting up a uv virtualenv for running notebooks locally. It
  covers environment bootstrap (uv + a gitignored .venv), the marimo reactive
  gotchas, the Colab self-contained / code-hiding / sharing patterns, and
  headless verification.
---

# Generating Colab and marimo notebooks

This skill produces two kinds of notebooks and gets the environment right first:

- **marimo** (`.py`, reactive): local interactive apps and dashboards. See
  [references/marimo.md](references/marimo.md).
- **Google Colab** (`.ipynb`): notebooks meant to be shared and run in the
  browser. See [references/colab.md](references/colab.md).

Always do Step 0 first, then pick the format, then build and verify.

## Step 0: Bootstrap the environment (uv + a gitignored .venv)

Notebooks need a Python environment. Standardise on a **uv-managed virtualenv at
the repo root**. uv is the installer: it is fast and resolves deterministically.

Run the bundled script (idempotent), passing only the packages the notebook needs:

```bash
bash {skillDir}/scripts/bootstrap_env.sh \
  marimo matplotlib networkx numpy pandas altair
```

It creates `<repo-root>/.venv` and installs the packages; afterwards use
`.venv/bin/python` and `.venv/bin/marimo`.

**Two side effects are OPT-IN — you must ASK the user before enabling them.** The
script never installs uv or edits `.gitignore` on its own; without the flags it
just prints what it would do. Prefer asking the user about both up front, as a
single question with two options:

- **(a) Install uv?** — needed only if uv is missing (the script exits code **2**
  and says so). Explain exactly what it does: *"uv, the fast Python package
  manager this uses, isn't installed. Install it now? It runs the official
  installer: `curl -LsSf https://astral.sh/uv/install.sh | sh` (or `brew install
  uv`)."* On yes, add `--install-uv`.
- **(b) Gitignore the venv?** — the script detects whether you're in a git repo;
  if so and `.venv/` isn't already ignored it warns (a venv should never be
  committed). Ask: *"Add `.venv/` to this repo's `.gitignore` so the virtualenv
  isn't committed?"* On yes, add `--gitignore-venv`. (Outside a repo, skip this.)

```bash
# after the user says yes to both:
bash {skillDir}/scripts/bootstrap_env.sh \
  --install-uv --gitignore-venv marimo matplotlib
```

## Step 1: Pick the format

| Use | Choose | Why |
|---|---|---|
| Local interactive dashboard, sliders, live re-render | **marimo** | Reactive, runs locally, is also a clean `.py` |
| A notebook to send someone who will run it in a browser | **Colab** | No install for the viewer, runs in Google's cloud |
| Both | Build both | The marimo `.py` for local work, an `.ipynb` for sharing |

If the user said "Colab", "share", or "send", lean Colab. If they said
"dashboard", "interactive", or "marimo", lean marimo. When unsure, ask.

## Step 2: Build

Read the matching reference file and follow it. The non-obvious essentials:

**marimo** ([references/marimo.md](references/marimo.md))
- Every top-level variable name must be unique across cells. Prefix throwaways
  with `_` to make them cell-local. Mismatching this is the #1 failure.
- Define shared helpers once, return them, take them as cell arguments.
- A cell renders its last expression. Return the matplotlib `fig` to show it.
- Start from [assets/marimo_template.py](assets/marimo_template.py).

**Colab** ([references/colab.md](references/colab.md))
- Build the `.ipynb` with `scripts/build_ipynb.py` from a list of cell specs;
  do not hand-write notebook JSON.
- Make it self-contained: first cell does `%pip install -q ...` then imports;
  embed any data inline so it runs without the source repos; link the sources
  in markdown.
- Every figure gets a markdown cell explaining what it shows.
- To present results-only, hide code with `#@title` + form view
  (`build_ipynb.py --hide-all-code`, or `"hide_code": true` per cell).

For both: seed anything random so output is deterministic.

## Step 3: Verify (do not skip)

A notebook that has not been run does not work.

**Colab / .ipynb** — execute every cell headless and save the figures:

```bash
.venv/bin/python {skillDir}/scripts/verify_ipynb.py out.ipynb --figdir /tmp/figs
```

Exit code is non-zero if any cell raises. Open a couple of the saved PNGs and
look at them.

**marimo / .py** — export to HTML, which runs the whole notebook:

```bash
.venv/bin/marimo export html notebook.py -o /tmp/out.html
grep -c "MultipleDefinitionError\|Traceback (most recent" /tmp/out.html   # want 0
grep -c "data:image/png;base64" /tmp/out.html                            # figure count
```

## Step 4: Share a Colab notebook (mind the traps)

- The "Open in Colab" badge works without auth **only for public repos**.
- A **private/org** repo needs Colab's GitHub OAuth app, which the org may
  block. The badge will error for members in that case.
- **Confidential stays confidential.** No-credentials does not mean
  public-safe. Internal data and architecture must not go to a public repo or
  gist. For internal sharing use **Google Drive on the company Workspace
  account** (upload the `.ipynb`, open with Colab, share via Drive). Details in
  [references/colab.md](references/colab.md).

## Bundled files

```
generate-notebooks/
  SKILL.md
  scripts/
    bootstrap_env.sh   - uv + gitignored .venv at repo root (Step 0)
    build_ipynb.py     - build a .ipynb from a cell spec (Colab)
    verify_ipynb.py    - run an .ipynb headless, save figures (Step 3)
  assets/
    marimo_template.py - minimal correct marimo notebook to copy
  references/
    marimo.md          - marimo reactive rules, display, verification
    colab.md           - Colab self-contained, code-hiding, sharing
```

## Gotchas (learned the hard way)

- **marimo `MultipleDefinitionError`**: two cells define the same name. Fix by
  `_`-prefixing throwaways or renaming. This is by far the most common error.
- **Hand-written notebook JSON breaks easily**: use `build_ipynb.py`.
- **`plt.show()` warns or fails headless**: `verify_ipynb.py` forces the Agg
  backend; in Colab `plt.show()` is fine.
- **Empty outputs on a freshly built `.ipynb`**: outputs only persist if the
  notebook was saved after running. Tell the viewer to Run all, or pre-run it.
- **"Open in Colab" OAuth error**: the repo is private/org. Use Google Drive
  for internal sharing instead.
