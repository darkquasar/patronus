#!/usr/bin/env python3
"""Interactive teaching lenses for marimo-teach-topic lessons.

These wrap the marimo/wigglystuff primitives that make a lesson *interactive* —
the things a static page can't do: reveal depth on demand, scrub through a
process one stage at a time, and let the reader tweak an assumption and watch the
consequence move. They exist so a lesson can be deep WITHOUT being a wall of
text: the main line stays readable, and the "complete"-tier detail lives behind a
fold or a slider the reader drives.

All helpers are thin and deterministic (no RNG/clock), so `verify.sh` renders the
same notebook every time.

marimo reactivity, the one rule that bites: a widget must be RETURNED from one
cell (assigned a UNIQUE top-level name), and its `.value`/attribute read in a
*downstream* cell. So the stepper and tangle helpers are used across two cells,
exactly like `quiz.py`'s `mcq`/`check`. Each has a two-cell example in its
docstring.

Depends on: marimo (>=0.13 for mo.mermaid/accordion/stat) and, for `tangle`,
`wigglystuff` (TangleSlider). Mermaid needs no Python package — mo.mermaid is
native and renders from marimo's own bundle.
"""
from __future__ import annotations


# -- depth on demand ---------------------------------------------------------
def deep_dive(mo, title, body, *, lazy=False):
    """A collapsible "go deeper" fold. THE workhorse for depth.

    Put the section's load-bearing explanation in the prose above; put the
    "complete"-tier mechanism detail (the extra hop, the edge case, the internal
    data structure) in here. A skimmer reads the prose; a deep reader opens the
    fold. One section then serves both without bloating the main line.

        deep_dive(mo, "How the scheduler actually scores nodes",
                  "It runs two phases: **filter** (drop nodes that can't fit the "
                  "Pod at all) then **score** (rank the survivors 0-100 on "
                  "spread, affinity, free resources) and binds the top one. ...")

    `title` shows on the closed fold; `body` is markdown. `lazy=True` defers
    rendering the body until opened (use for a heavy diagram/plot inside).
    """
    return mo.accordion({f"▸ {title}": mo.md(body)}, lazy=lazy)


# -- scrub through a process -------------------------------------------------
def stepper(mo, n_steps, label="Step"):
    """A slider that drives a step-through. Return it (unique name) in cell A;
    read `.value` in cell B to pick which stage to render. Pair with
    `mermaid_at` (or index any list of renderables yourself).

        # cell A
        s5_step = stepper(mo, 4, label="Scheduling step")
        s5_step
        # cell B  (re-runs when the slider moves)
        mermaid_at(mo, S5_STAGES, s5_step.value)   # S5_STAGES = [mmd1, mmd2, ...]

    Steps are 1-based (matches what the reader sees). `n_steps` = len(stages).
    """
    return mo.ui.slider(1, n_steps, value=1, show_value=True, label=label)


def play_stepper(mo, n_steps, label="Step", interval_ms=1200, loop=True):
    """Like `stepper`, but AUTO-ADVANCES with a play/pause button — i.e. an
    animation of a step-through (no file, no gif). Uses `wigglystuff.PlaySlider`.
    Return it (unique name) in cell A; read `.value` (1-based) downstream and feed
    `mermaid_at`, exactly like `stepper` — the reader can play, pause, or scrub.

        # cell A
        roll = play_stepper(mo, 5, label="Rolling update", interval_ms=1000)
        roll
        # cell B
        mermaid_at(mo, STAGES, roll.value)

    Requires `wigglystuff`. Keep each frame cheap (it re-renders per tick).
    """
    from wigglystuff import PlaySlider
    return mo.ui.anywidget(PlaySlider(
        min_value=1, max_value=n_steps, step=1,
        interval_ms=interval_ms, loop=loop,
    ))


def mermaid_at(mo, stages, value, *, theme=None):
    """Render stage `value` (1-based) of a list of Mermaid diagram strings.

    Clamps out-of-range values so the cell never errors mid-interaction. Use with
    `stepper` (pass `.value`) or `play_stepper` (pass `.value` — the auto-play
    frame arrives as a dict, which this unwraps) to reveal a diagram one layer at
    a time — the highest-value interactive pattern for a process (a request path,
    a scheduling handoff, a build-up of an architecture). Each stage is a full
    Mermaid source string; typically each adds a node/edge to the previous.

    LAYOUT RULE for a step-through chain: build it **vertically** — a
    `flowchart TD` (top-down) or a `sequenceDiagram`, NOT a horizontal
    `flowchart LR`. `mo.mermaid` scales the SVG to the column WIDTH, so a
    horizontal chain gets smaller and smaller each time you add a step until it's
    unreadable; a vertical chain grows DOWNWARD at a constant, legible width. If a
    step's label is long, vertical is the only thing that stays readable.
    """
    if not stages:
        return mo.md("_(no stages)_")
    if isinstance(value, dict):          # play_stepper (PlaySlider) frame
        value = value.get("value", 1)
    idx = max(1, min(int(value or 1), len(stages))) - 1
    return mo.mermaid(stages[idx], theme=theme)


# -- tweak an assumption, watch it move --------------------------------------
def tangle(mo, amount, min_value, max_value, *, step=1, prefix="", suffix=""):
    """An inline, draggable number embedded in prose (Bret Victor "tangle").

    Reads as normal text until you drag the highlighted number; dependent cells
    recompute live. Great for "what-if" intuition: *"at **80%** cache hit rate,
    latency is **~40ms**"* where both numbers are live.

        # cell A
        hit = tangle(mo, 80, 0, 100, suffix="%")
        mo.md(["If the cache hits ", hit, " of requests..."])   # inline in a hstack
        # cell B  (re-runs as they drag)
        mo.md(f"...effective latency is about **{200 - hit.amount*2:.0f} ms**.")

    Read the value downstream as `hit.amount` (a float) or `hit.value["amount"]`.
    Requires `wigglystuff`.
    """
    from wigglystuff import TangleSlider
    return mo.ui.anywidget(TangleSlider(
        amount=amount, min_value=min_value, max_value=max_value,
        step=step, prefix=prefix, suffix=suffix,
    ))


# -- anchor to external sources ----------------------------------------------
def references(mo, links, *, title="References and further reading"):
    """A collapsible list of external source links that anchors a section in
    authoritative docs (the official docs, a spec, a canonical blog post). Put one
    at the end of each section so the explanation is grounded and the learner can
    go to the source — without cluttering the main line.

        references(mo, [
            ("Kubernetes docs: Pods", "https://kubernetes.io/docs/concepts/workloads/pods/"),
            ("API concepts (watch, resourceVersion)",
             "https://kubernetes.io/docs/reference/using-api/api-concepts/"),
        ])

    `links` is a list of (label, url) pairs, or a {label: url} dict. Rendered as a
    fold so references are available but never in the way.
    """
    if isinstance(links, dict):
        links = list(links.items())
    body = "\n".join(f"- [{label}]({url})" for label, url in links)
    return mo.accordion({f"🔗 {title}": mo.md(body or "_(none)_")})


# -- emphasis ----------------------------------------------------------------
def key_fact(mo, value, label, caption=None):
    """A headline "number to remember" card (wraps `mo.stat`). Use sparingly for
    the one figure a section hinges on (a default, a limit, a ratio).

        key_fact(mo, "110", "Pods per node (default cap)", "raise with --max-pods")
    """
    return mo.stat(value=value, label=label, caption=caption, bordered=True)


# -- render a doc whose YAML fences marimo would otherwise mangle --------------
def render_doc(mo, text):
    """Render a markdown doc/field-manual that may contain ```yaml fences.

    THE PROBLEM: `mo.md` runs a markdown pass that reflows block-sequence lines
    (`  - item`) into loose lists EVEN INSIDE a fenced code block, which mangles
    YAML indentation (injects blank lines, doubles the indent). It only bites YAML
    with block sequences — Kubernetes manifests, configs — not bash or prose.
    (marimo's own docs example dodges it by using inline `[a, b]` flow sequences.)
    None of the usual "fixes" help: raw strings (`r"..."`), `inspect.cleandoc`, or
    `.style({"white-space": "pre"})` all act on the WRONG layer (Python escaping /
    leading indent / CSS) — the corruption happens during markdown PARSING.

    THE FIX: pull ```yaml / ```yml fences OUT of the markdown, syntax-highlight
    them with pygments (which ships with marimo), and emit them via `mo.Html` —
    which is NOT markdown-processed, so nothing reflows. Prose and other fences
    (bash, etc.) stay in `mo.md` and keep marimo's own highlighting.

    Use this in place of `mo.md(read_doc(name))` for any sibling doc that contains
    YAML. A doc with no yaml fence renders identically to `mo.md`.
    """
    import html as _html
    import re as _re

    def _yaml_html(code):
        try:
            from pygments import highlight
            from pygments.formatters import HtmlFormatter
            from pygments.lexers import YamlLexer
            inner = highlight(code, YamlLexer(),
                              HtmlFormatter(noclasses=True, nowrap=True)).rstrip("\n")
        except Exception:                       # pygments missing -> plain, still correct
            inner = _html.escape(code)
        return ("<pre style='background:#f6f8fa;border:1px solid #e1e4e8;"
                "border-radius:6px;padding:12px;overflow-x:auto;font-size:0.82em;"
                "line-height:1.45;margin:0.5em 0'><code style='white-space:pre'>"
                + inner + "</code></pre>")

    _out, _pos = [], 0
    for _m in _re.finditer(r"(?ms)^```(?:yaml|yml)[ \t]*\n(.*?)\n?^```[ \t]*$", text):
        _pre = text[_pos:_m.start()]
        if _pre.strip():
            _out.append(mo.md(_pre))
        _out.append(mo.Html(_yaml_html(_m.group(1).rstrip("\n"))))
        _pos = _m.end()
    _tail = text[_pos:]
    if _tail.strip():
        _out.append(mo.md(_tail))
    return mo.vstack(_out, gap=0.4) if len(_out) != 1 else _out[0]
