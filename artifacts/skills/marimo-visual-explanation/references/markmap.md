# Markmap — the option-space lens

Markmap turns a markdown *outline* into an interactive, zoomable radial mind map.
In a decision notebook this is the lens that **lays out the option space**: the
2-3 alternatives, each with its sub-considerations (consistency, scaling, ops,
cost…), as a hierarchy you can scan before judging. More broadly it answers "what
is this **made of**, and how do the parts nest" — a taxonomy, not a flow, so it's
orthogonal to the arrow-based lenses.

There's no Python Markmap renderer, so the bundled `scripts/markmap_html.py`
emits a self-contained HTML page using Markmap's official CDN *autoloader*, which
you embed with `mo.iframe`.

## Usage

```python
from markmap_html import markmap_html

mm_outline = """
# Topic (one root)
## Major part A
- detail
- detail
## Major part B
- detail
  - sub-detail
## Major part C
- detail
"""
# Pass a fixed `height` and make the iframe ~20px taller.
mo.iframe(markmap_html(mm_outline, height=460), height="480px")
```

## Gotcha: the runaway blank-space scroll (always pass `height`)

`mo.iframe` injects an onload auto-resizer that grows the iframe to fit its
content. If the embedded Markmap sizes its SVG with a viewport-relative height
(`100vh`), the two feed back on each other and the embed balloons into endless
blank space you can scroll forever — extremely annoying. `markmap_html` avoids
this by pinning html/body/svg to a **fixed pixel height** with `overflow:hidden`,
which is why you should pass `height=<px>` and set the `mo.iframe` height to
roughly the same value. Never reintroduce `100vh` into the template.

## Authoring the outline

- **One `#` root.** A single root gives one centred map. Multiple `#` headings
  make several disconnected maps — usually not what you want.
- **`##`, `###` nest** as branches; **`-` bullets** are leaves. Indent bullets
  with two spaces to nest them.
- Markmap renders inline markdown in nodes: `**bold**`, `` `code` ``, and links
  work. Keep node text short — a mind map is scannable phrases, not sentences.
- This is the place for *decomposition*: "Auth = sign-in + tokens + storage;
  tokens = access + refresh". Don't sneak sequence or data-flow in here — that's
  Mermaid's job.

## Good vs bad outline

Good (a taxonomy):

```
# Payment system
## Methods
- card
- bank transfer
- wallet
## Risk
- fraud scoring
- 3-D Secure
## Settlement
- batching
- reconciliation
```

Bad (a flow forced into headings — use Mermaid for this instead):

```
# Payment
## Step 1: user enters card
## Step 2: we charge it
## Step 3: we settle
```

## Caveat

The autoloader fetches Markmap from a CDN on render, so this view needs network
the first time, like the Mermaid and Excalidraw views. The HTML is generated
deterministically, so the notebook itself is reproducible.

## Safety

`markmap_html.py` defangs any literal `</script>` in your outline so it can't
break out of the embedded template. You don't need to escape anything yourself
for normal prose; if you're injecting untrusted text, `escape_label()` is
available.
