# Google Colab notebooks

Colab runs **Jupyter `.ipynb`** files (JSON), in Google's cloud, in the
browser. Use Colab when the point is to **share** a runnable notebook with
people who should not have to clone a repo or install anything. Use marimo
instead for local interactive dashboards (see [marimo.md](marimo.md)).

## Build the .ipynb, do not hand-write it

Notebook JSON is fragile. Use `scripts/build_ipynb.py`: write a list of cell
specs (markdown and code), and it emits valid notebook JSON. This avoids the
single-stray-comma class of bugs entirely.

```bash
.venv/bin/python {skillDir}/scripts/build_ipynb.py spec.json out.ipynb
```

## Make it self-contained

A shared notebook must run for someone who has none of your local context.

- **First code cell installs deps and imports**: start with
  `%pip install -q matplotlib networkx numpy pandas` then the imports. Colab
  pre-installs the common scientific stack, but pinning what you need makes it
  robust.
- **Embed the data the notebook needs.** Do not read local files or reach into
  private repos. If the notebook visualises a dataset, embed it inline (a JSON
  string the first cell parses). A few KB is fine and makes the notebook
  reproduce anywhere. If the embedded data was extracted from a source, link
  the source in a markdown cell.
- **Recompute from the embedded data** so the figures are genuinely derived,
  not pasted.

## Deterministic and explained

- Seed any randomness (`spring_layout(..., seed=42)`), so re-runs match.
- Every figure gets a markdown cell that says what it shows and what to read
  from it. A graph with no explanation is noise to a new reader.

## Hiding code for a clean report

To present a notebook as results-only, hide the code cells:

- Per cell: add `#@title <Title>` as the first line and set the cell to form
  view. `build_ipynb.py` does this when a cell spec has `"hide_code": true`.
- Whole notebook: `build_ipynb.py ... --hide-all-code`.
- In the Colab UI a viewer can still expand the code via the "Show code" bar.

Outputs only appear without running if the notebook is saved WITH outputs.
A freshly generated notebook has empty outputs, so either tell the viewer to
"Runtime, Run all" (a few seconds), or pre-run it before sharing.

## Verify before sharing

```bash
.venv/bin/python {skillDir}/scripts/verify_ipynb.py out.ipynb --figdir /tmp/figs
```

This executes every code cell headless (stripping `%`/`!` magics, forcing the
Agg backend) and saves the figures. If it passes, "Runtime, Run all" in Colab
will too. Open a couple of the saved PNGs and actually look at them.

## Sharing: the OAuth and confidentiality traps

The "Open in Colab" badge URL is:

```
https://colab.research.google.com/github/<owner>/<repo>/blob/<branch>/path/to/nb.ipynb
```

- This works with **no authentication only for PUBLIC repos**.
- For a **private or org** repo, Colab must authenticate to GitHub via OAuth,
  and the org can block the Colab OAuth app, producing a permission error even
  for members. Do not expect the badge to "just work" on a private repo.
- **Confidential data stays confidential.** Absence of credentials in a
  notebook does not make it public-safe. Internal architecture, identifiers,
  and embedded internal data are still confidential. Do not publish such a
  notebook to a public repo or gist.

For internal sharing, prefer **Google Drive on the company Workspace account**:
upload the `.ipynb`, open with Colab, and share via normal Drive permissions
(restrict to the domain or named people). This bypasses GitHub OAuth and keeps
the file inside the company tenancy. If the user's Drive is synced locally
(macOS: `~/Library/CloudStorage/GoogleDrive-<email>/My Drive/...`), copying the
file into the synced folder uploads it.

Badge markdown:

```markdown
[![Open In Colab](https://colab.research.google.com/assets/colab-badge.svg)](https://colab.research.google.com/github/<owner>/<repo>/blob/<branch>/path/nb.ipynb)
```
