#!/usr/bin/env python3
"""Turn a markdown outline into a self-contained Markmap mind-map HTML page.

Markmap renders a markdown *outline* as an interactive, zoomable radial mind map.
That makes it the right lens for the **conceptual hierarchy** of an explanation —
"what is this made of, and how do the ideas nest" — which is a genuinely
different question from Mermaid's "how does it flow" or Excalidraw's "what's the
gist". Outline in, mind map out.

There is no Python Markmap renderer; Markmap is a JS library. The robust,
dependency-free way to embed it is the official *autoloader*: a page with the
markdown inside a ``<script type="text/template">`` block and one autoloader
``<script>`` tag from a CDN. We return that HTML string; embed it in marimo with
``mo.iframe(html, height="500px")``.

Caveat worth stating in the notebook: the autoloader fetches Markmap from a CDN
on render, so it needs network the first time, like the Mermaid and Excalidraw
views.

Usage:

    from markmap_html import markmap_html
    md = '''
    # Auth system
    ## Sign-in
    - password
    - OAuth
    ## Tokens
    - access (15 min)
    - refresh (30 d)
    '''
    html = markmap_html(md)          # -> str of a full HTML page
    # in marimo:  mo.iframe(html, height="500px")
"""
from __future__ import annotations

import html as _html
import textwrap

# Pinned majors so the embed is stable; autoloader pulls transitive deps itself.
_AUTOLOADER = "https://cdn.jsdelivr.net/npm/markmap-autoloader@0.18"

# A FIXED pixel height is essential. marimo's `mo.iframe` injects an onload
# auto-resizer that grows the iframe to its content's height; if the markmap SVG
# is sized with `100vh` (viewport height), the two feed back on each other and
# the embed balloons into endless blank space you can scroll forever. Pinning
# html/body/svg to a fixed px height with `overflow:hidden` breaks the loop.
_TEMPLATE = """<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8"/>
<style>
  html, body {{ margin: 0; padding: 0; height: {h}px; overflow: hidden; }}
  #mm {{ width: 100%; height: {h}px; }}
  svg.markmap {{ display: block; width: 100%; height: {h}px; }}
</style>
</head>
<body>
<div class="markmap" id="mm">
<script type="text/template">
{markdown}
</script>
</div>
<script>
window.markmap = {{
  autoLoader: {{ manual: false }},
}};
</script>
<script src="{autoloader}"></script>
</body>
</html>"""


def markmap_html(markdown: str, height: int = 500, autoloader: str = _AUTOLOADER) -> str:
    """Return a full HTML page that renders ``markdown`` as a Markmap.

    ``markdown`` is a normal markdown outline: ``#`` headings nest, and ``-``
    bullets become leaves. Keep it to one ``#`` root for a single centred map.

    ``height`` is the fixed canvas height in pixels — set it to (roughly) the
    height you pass to ``mo.iframe`` so the map fills the frame without the
    runaway blank-space scrolling that a viewport-relative height causes.
    """
    body = textwrap.dedent(markdown).strip()
    # The markdown lives inside a <script> template, so the only thing that can
    # break the page is a literal </script>; defang it.
    body = body.replace("</script>", "<\\/script>")
    return _TEMPLATE.format(markdown=body, autoloader=autoloader, h=int(height))


def escape_label(text: str) -> str:
    """Escape a string for safe use as inline markdown/HTML in an outline node."""
    return _html.escape(text)


if __name__ == "__main__":
    sample = """
    # Auth system
    ## Sign-in
    - password + TOTP
    - OAuth (Google, GitHub)
    ## Tokens
    - access token — 15 min
    - refresh token — 30 days
    ## Storage
    - users table
    - sessions (Redis)
    """
    out = markmap_html(sample, height=520)
    assert "markmap-autoloader" in out
    assert "Auth system" in out
    assert "text/template" in out
    assert "100vh" not in out and "overflow: hidden" in out and "520px" in out
    print(f"OK: generated {len(out)} chars of markmap HTML (fixed-height, no 100vh)")
    print(out[:300] + "\n...")
