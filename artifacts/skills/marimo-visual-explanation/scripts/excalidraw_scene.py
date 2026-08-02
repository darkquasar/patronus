#!/usr/bin/env python3
"""Build a valid Excalidraw *scene* from a high-level node/edge/zone spec.

Hand-writing Excalidraw element JSON is miserable: every shape needs ~25 fields,
arrows need bindings on both ends, and labels are separate "bound text" elements
that have to be cross-referenced. This module hides all of that so an agent can
declare the *overview* it wants and get back a scene dict that
``wigglystuff.Excalidraw(scene=...)`` renders.

The design bias is the OVERVIEW lens (see the skill's views.md): a handful of big
boxes, the main arrows, a couple of grouping zones and margin notes — the gist a
reader gets in five seconds. It is deliberately not a precise blueprint; that is
the Mermaid view's job.

Why this produces something Excalidraw accepts: the widget passes the scene
straight into Excalidraw's ``initialData``, which runs it through Excalidraw's
``restore()``. ``restore`` fills in any missing defaults and recomputes bound
arrow endpoints from the bindings, so we only need the core geometry and honest
cross-references. We still emit a full field set for stability across versions.

Determinism: ids and seeds come from a counter, never from a RNG or clock, so the
same spec always yields byte-identical JSON (important for reproducible
notebooks and diffable output).

Quick start:

    from excalidraw_scene import Scene

    s = Scene(title="Request lifecycle")
    client = s.box("Client", color="blue",  subtitle="browser / app")
    api    = s.box("Core API", color="green", subtitle="auth + routing")
    data   = s.box("Data", color="yellow", subtitle="Postgres + cache")
    s.flow([client, api, data])              # auto-place L→R and connect
    s.zone([api, data], label="our infra")   # translucent grouping rectangle
    s.note("cache miss = the slow path", near=data)
    scene = s.to_dict()                       # -> pass to Excalidraw(scene=scene)

Run this file directly to build a sample scene and self-validate it.
"""
from __future__ import annotations

import json
import math
from typing import Iterable, Optional

# Excalidraw's named background palette (light, friendly, distinct).
COLORS = {
    "blue":   "#a5d8ff",
    "green":  "#b2f2bb",
    "yellow": "#ffec99",
    "red":    "#ffc9c9",
    "purple": "#d0bfff",
    "orange": "#ffd8a8",
    "pink":   "#fcc2d7",
    "gray":   "#e9ecef",
    "transparent": "transparent",
}
_STROKE = "#1e1e1e"
_FONT_HAND = 1   # Excalidraw font families: 1=hand-drawn, 2=normal, 3=code
_LINE_HEIGHT = 1.25


def _text_width(text: str, font_size: int) -> float:
    """Rough width of a single line of hand-drawn text, in px.

    Good enough to size boxes so labels fit; Excalidraw re-measures bound text
    on load, so this only needs to be in the right ballpark.
    """
    longest = max((len(line) for line in text.split("\n")), default=0)
    return longest * font_size * 0.6


def _wrap_text(text: str, font_size: int, max_px: float) -> str:
    """Word-wrap ``text`` so no line exceeds ``max_px`` px, keeping any existing
    newlines as hard breaks.

    A ``note`` is prose and would otherwise render as ONE very long line — which
    blows up the scene's content width and forces ``fitted()`` to zoom far out,
    yielding a short, cramped pane (Excalidraw then collapses its bottom-left zoom
    UI and leaves a big empty band). Wrapping keeps a note's width bounded so the
    fit zoom stays high and the pane stays a sensible height.
    """
    out = []
    for para in text.split("\n"):
        line = ""
        for word in para.split(" "):
            trial = word if not line else line + " " + word
            if line and _text_width(trial, font_size) > max_px:
                out.append(line)
                line = word
            else:
                line = trial
        out.append(line)
    return "\n".join(out)


def _edge_point(box: dict, toward_x: float, toward_y: float, gap: float):
    """Point on ``box``'s border in the direction of (toward), pushed out by gap.

    This is the anti-overlap workhorse: an arrow drawn from one box's edge-point
    to the other's never crosses into either rectangle, unlike a centre-to-centre
    line (which is what produced arrows running through the boxes).
    """
    cx = box["x"] + box["width"] / 2
    cy = box["y"] + box["height"] / 2
    dx = toward_x - cx
    dy = toward_y - cy
    if dx == 0 and dy == 0:
        return cx, cy
    # scale the direction until it hits the nearest box side
    tx = (box["width"] / 2) / abs(dx) if dx else math.inf
    ty = (box["height"] / 2) / abs(dy) if dy else math.inf
    t = min(tx, ty)
    ex = cx + dx * t
    ey = cy + dy * t
    length = math.hypot(dx, dy)
    return ex + dx / length * gap, ey + dy / length * gap


def _seg_hits_rect(sx, sy, ex, ey, box, margin: float = 10.0) -> bool:
    """Does the segment (sx,sy)->(ex,ey) pass through ``box`` (expanded by margin)?

    Sampled along the segment — simple and robust enough for the handful of
    boxes in an overview, and it's what lets us detect an arrow that would run
    over a box sitting between its endpoints.
    """
    rx0, ry0 = box["x"] - margin, box["y"] - margin
    rx1 = box["x"] + box["width"] + margin
    ry1 = box["y"] + box["height"] + margin
    for _i in range(25):
        _t = _i / 24.0
        px, py = sx + (ex - sx) * _t, sy + (ey - sy) * _t
        if rx0 <= px <= rx1 and ry0 <= py <= ry1:
            return True
    return False


def _longest_segment(points):
    """Return the (p1, p2) consecutive pair spanning the greatest distance."""
    best, bp = -1.0, (points[0], points[-1])
    for _p1, _p2 in zip(points, points[1:]):
        _d = math.hypot(_p2[0] - _p1[0], _p2[1] - _p1[1])
        if _d > best:
            best, bp = _d, (_p1, _p2)
    return bp


def _fan_endpoints(points, start_shift: float, end_shift: float):
    """Nudge the first/last point perpendicular to its adjacent segment.

    When several arrows converge on one box (or fan out of one), routing lands
    them all on the same side-midpoint and they collapse into what looks like a
    single edge. Shifting each arrow's tip (and tail) tangentially along the box
    edge by a per-arrow offset separates them so it's clear which is which.
    """
    pts = [(float(x), float(y)) for x, y in points]
    if end_shift and len(pts) >= 2:
        (x1, y1), (x2, y2) = pts[-2], pts[-1]
        dx, dy = x2 - x1, y2 - y1
        L = math.hypot(dx, dy) or 1.0
        pts[-1] = (x2 + (-dy / L) * end_shift, y2 + (dx / L) * end_shift)
    if start_shift and len(pts) >= 2:
        (x1, y1), (x2, y2) = pts[0], pts[1]
        dx, dy = x2 - x1, y2 - y1
        L = math.hypot(dx, dy) or 1.0
        pts[0] = (x1 + (-dy / L) * start_shift, y1 + (dx / L) * start_shift)
    return pts


class Scene:
    """Accumulates Excalidraw elements and serialises them to a scene dict."""

    def __init__(self, title: Optional[str] = None, theme: str = "light",
                 col_gap: float = 360.0, row_gap: float = 190.0):
        self._elements: list[dict] = []
        # Arrows / zones / notes are recorded as specs and only built at render
        # (in _materialize), AFTER a relayout pass has spaced the grid boxes by
        # their actual sizes. Building them eagerly would bake in coordinates from
        # the provisional (often overlapping) box positions.
        self._arrow_specs: list[dict] = []
        self._zone_specs: list[dict] = []
        self._note_specs: list[dict] = []
        self._materialized = False
        self._counter = 0
        self.theme = theme
        self._cursor_x = 140.0          # where the next auto-placed box goes
        self._cursor_y = 200.0
        # Grid placement: box(col=, row=) centres a box in cell (col, row).
        # Rows are the missing piece that lets you lay things out in 2D instead
        # of one cramped horizontal line — put the main flow on row 0 and
        # supporting nodes on row 1, and arrows route cleanly between them.
        # A wide column pitch keeps arrows comfortably longer than their labels.
        self._origin_x = 200.0          # centre of cell (0, 0)
        self._origin_y = 200.0
        self._col_gap = col_gap         # centre-to-centre column pitch
        self._row_gap = row_gap         # centre-to-centre row pitch
        if title:
            self.title(title)
            # Leave clear vertical space below the title so the first row of
            # boxes — and the zone that wraps them, which extends above its
            # boxes for its own label — never collide with the title.
            self._origin_y = 300.0
            self._cursor_y = 300.0

    # -- id / seed generation (deterministic) -------------------------------
    def _next_id(self, kind: str) -> str:
        self._counter += 1
        return f"{kind}-{self._counter}"

    def _seed(self) -> int:
        # A fixed-but-varied seed per element keeps the hand-drawn look stable.
        return 1000 + self._counter * 7

    def _base(self, eid: str, etype: str, x, y, w, h, **over) -> dict:
        el = {
            "id": eid,
            "type": etype,
            "x": float(x),
            "y": float(y),
            "width": float(w),
            "height": float(h),
            "angle": 0,
            "strokeColor": _STROKE,
            "backgroundColor": "transparent",
            "fillStyle": "solid",
            "strokeWidth": 2,
            "strokeStyle": "solid",
            "roughness": 1,
            "opacity": 100,
            "groupIds": [],
            "frameId": None,
            "roundness": {"type": 3},
            "seed": self._seed(),
            "version": 1,
            "versionNonce": self._seed() * 3,
            "isDeleted": False,
            "boundElements": [],
            "updated": 1,
            "link": None,
            "locked": False,
        }
        el.update(over)
        return el

    # -- primitives ---------------------------------------------------------
    def title(self, text: str, x: float = 120, y: float = 90, font_size: int = 28) -> str:
        """A free heading at the top of the canvas."""
        return self.text(text, x=x, y=y, font_size=font_size)

    def text(self, content: str, x: float, y: float, font_size: int = 20,
             color: str = _STROKE, align: str = "left") -> str:
        """A free-floating text element (not bound to any container)."""
        eid = self._next_id("text")
        w = _text_width(content, font_size)
        h = font_size * _LINE_HEIGHT * (content.count("\n") + 1)
        el = self._base(eid, "text", x, y, w, h,
                        strokeColor=color, roundness=None)
        el.update({
            "text": content,
            "fontSize": font_size,
            "fontFamily": _FONT_HAND,
            "textAlign": align,
            "verticalAlign": "top",
            "containerId": None,
            "originalText": content,
            "lineHeight": _LINE_HEIGHT,
            "autoResize": True,
            "baseline": int(font_size * 0.9),
        })
        self._elements.append(el)
        return eid

    def box(self, label: str, x: Optional[float] = None, y: Optional[float] = None,
            w: Optional[float] = None, h: Optional[float] = None,
            color: str = "blue", subtitle: Optional[str] = None,
            shape: str = "rectangle", font_size: int = 20,
            col: Optional[int] = None, row: Optional[int] = None) -> str:
        """A labelled box.

        Position, in priority order: explicit ``x``/``y`` → grid ``col``/``row``
        (the box is centred in cell (col, row)) → automatic left-to-right cursor.
        Prefer ``col``/``row`` for anything with more than one row of boxes — it
        keeps a clean 2D layout and stops arrows from piling onto one line.

        ``shape`` is "rectangle", "ellipse" or "diamond". ``subtitle`` adds a
        smaller second line — good for a one-phrase description per box.
        """
        caption = label if not subtitle else f"{label}\n{subtitle}"
        _nlines = caption.count("\n") + 1
        if w is None:
            w = max(160.0, _text_width(caption, font_size) + 56)
        if h is None:
            h = max(72.0, _nlines * font_size * _LINE_HEIGHT + 40.0)
        # grid placement centres the box in cell (col, row). The provisional
        # position uses the fixed pitch; a render-time relayout (see _materialize)
        # later re-spaces grid boxes by their ACTUAL sizes so long labels never
        # let neighbours overlap.
        _grid_cell = None
        if (col is not None or row is not None) and x is None and y is None:
            _grid_cell = (col or 0, row or 0)
            _cx = self._origin_x + _grid_cell[0] * self._col_gap
            _cy = self._origin_y + _grid_cell[1] * self._row_gap
            x = _cx - w / 2
            y = _cy - h / 2
        if x is None:
            x = self._cursor_x
        if y is None:
            y = self._cursor_y

        box_id = self._next_id("box")
        bg = COLORS.get(color, color)
        roundness = {"type": 3} if shape == "rectangle" else None
        box_el = self._base(box_id, shape, x, y, w, h,
                            backgroundColor=bg, roundness=roundness)
        if _grid_cell is not None:
            box_el["_col"], box_el["_row"] = _grid_cell

        # Bound, centred label. Excalidraw CLIPS bound text to the stored
        # width/height and only re-measures when you enter/leave the text editor
        # — so a one-line height (or too-narrow width) renders multi-line/long
        # labels clipped until you double-click them. Size the text element to
        # the FULL caption (all lines, container-inner width) so it shows
        # complete on first load.
        _PAD = 5
        text_id = self._next_id("label")
        text_w = max(w - 2 * _PAD, 10)
        text_h = _nlines * font_size * _LINE_HEIGHT
        tx = x + _PAD
        ty = y + (h - text_h) / 2
        text_el = self._base(text_id, "text", tx, ty, text_w, text_h,
                             roundness=None)
        text_el.update({
            "text": caption,
            "fontSize": font_size,
            "fontFamily": _FONT_HAND,
            "textAlign": "center",
            "verticalAlign": "middle",
            "containerId": box_id,
            "originalText": caption,
            "lineHeight": _LINE_HEIGHT,
            "autoResize": False,
            "baseline": int(font_size * 0.9),
        })
        box_el["boundElements"] = [{"type": "text", "id": text_id}]

        self._elements.append(box_el)
        self._elements.append(text_el)
        # advance the auto-layout cursor to the right of this box. A generous
        # edge gap (~130px) leaves room for the arrow + its label and keeps the
        # overview from feeling cramped (matches the spacing the research
        # recommends for non-overlapping diagrams).
        self._cursor_x = x + w + 130
        self._cursor_y = y
        return box_id

    def _get(self, eid: str) -> dict:
        for el in self._elements:
            if el["id"] == eid:
                return el
        raise KeyError(f"no element {eid!r} in scene")

    def _route(self, a: dict, b: dict, exclude, gap: float):
        """Orthogonal (elbow) points from box ``a`` to box ``b``.

        Every arrow is routed in right angles, never a diagonal: a straight
        segment when the boxes share a row/column, an L-elbow when they're
        diagonal, and a U/Z detour through a local gutter when a box sits in the
        way. All candidates are scored against every box and we return the
        clearest, then fewest-corner, then shortest route — so arrows look like
        clean flowchart connectors and never run over a box.
        """
        acx, acy = a["x"] + a["width"] / 2, a["y"] + a["height"] / 2
        bcx, bcy = b["x"] + b["width"] / 2, b["y"] + b["height"] / 2
        obstacles = [e for e in self._elements
                     if e["type"] in ("rectangle", "ellipse", "diamond")
                     and e["id"] not in exclude
                     and not e["id"].startswith("zone")]

        def crossings(pts):
            return sum(1 for o in obstacles
                       for (x1, y1), (x2, y2) in zip(pts, pts[1:])
                       if _seg_hits_rect(x1, y1, x2, y2, o))

        def length(pts):
            return sum(math.hypot(p2[0] - p1[0], p2[1] - p1[1])
                       for p1, p2 in zip(pts, pts[1:]))

        # side-midpoint exit/entry points (offset by the small gap)
        aR, aL = (a["x"] + a["width"] + gap, acy), (a["x"] - gap, acy)
        aT, aB = (acx, a["y"] - gap), (acx, a["y"] + a["height"] + gap)
        bR, bL = (b["x"] + b["width"] + gap, bcy), (b["x"] - gap, bcy)
        bT, bB = (bcx, b["y"] - gap), (bcx, b["y"] + b["height"] + gap)
        tol = 8.0

        cands = []
        if abs(acy - bcy) <= tol:                       # same row -> straight
            cands.append([aR, bL] if bcx > acx else [aL, bR])
        elif abs(acx - bcx) <= tol:                     # same column -> straight
            cands.append([aB, bT] if bcy > acy else [aT, bB])
        else:                                           # diagonal -> two L-elbows
            _sh = aR if bcx > acx else aL               # horizontal leg first
            _ev = bT if acy < bcy else bB
            cands.append([_sh, (bcx, acy), _ev])
            _sv = aB if bcy > acy else aT               # vertical leg first
            _eh = bL if acx < bcx else bR
            cands.append([_sv, (acx, bcy), _eh])

        # if every simple candidate is blocked, add U/Z detours via a local gutter
        if all(crossings(c) for c in cands):
            hits = [o for o in obstacles
                    if any(_seg_hits_rect(p1[0], p1[1], p2[0], p2[1], o)
                           for c in cands for p1, p2 in zip(c, c[1:]))] or obstacles
            m = 34.0
            xr = max(o["x"] + o["width"] for o in hits) + m
            xl = min(o["x"] for o in hits) - m
            yb = max(o["y"] + o["height"] for o in hits) + m
            ya = min(o["y"] for o in hits) - m
            cands += [
                [aR, (xr, acy), (xr, bcy), bR],
                [aL, (xl, acy), (xl, bcy), bL],
                [aB, (acx, yb), (bcx, yb), bB],
                [aT, (acx, ya), (bcx, ya), bT],
            ]

        best = min(cands, key=lambda c: (crossings(c), len(c), length(c)))
        # If no elbow route is clean (only happens in very dense diagrams), a
        # straight diagonal that avoids the boxes beats an elbow that runs over
        # one. Crossing a box is the worst outcome; a rare diagonal is fine.
        if crossings(best) > 0:
            diag = [_edge_point(a, bcx, bcy, gap), _edge_point(b, acx, acy, gap)]
            if crossings(diag) < crossings(best):
                return diag
        return best

    def arrow(self, src: str, dst: str, label: Optional[str] = None,
              dashed: bool = False, font_size: int = 16) -> str:
        """Record a bound arrow from ``src`` to ``dst`` (built at render time).

        Deferred so the render-time relayout can space boxes by their real sizes
        BEFORE the arrow is routed against final positions. Endpoints land on box
        borders; if a third box sits on the path the arrow elbows around it. When
        several arrows share a target (or source), their endpoints are fanned
        apart so you can tell them apart instead of collapsing into one line
        (see _materialize). A label longer than the edge it sits on is lifted into
        a clear band above the boxes so the boxes never cover it.
        """
        arrow_id = self._next_id("arrow")
        label_id = self._next_id("arrowlabel") if label else None
        self._arrow_specs.append({
            "id": arrow_id, "src": src, "dst": dst, "label": label,
            "label_id": label_id, "dashed": dashed, "font_size": font_size,
            "start_shift": 0.0, "end_shift": 0.0,
        })
        return arrow_id

    def _emit_arrow(self, spec: dict) -> None:
        """Build the actual arrow (and its label) element from a spec, using the
        boxes' FINAL positions and any fan-out shift assigned in _materialize."""
        src, dst = spec["src"], spec["dst"]
        label, font_size = spec["label"], spec["font_size"]
        a, b = self._get(src), self._get(dst)
        _GAP = 4.0
        pts = self._route(a, b, exclude=(src, dst), gap=_GAP)
        # Fan shared endpoints apart: nudge the tail/tip perpendicular to their
        # adjacent segment so arrows into (or out of) one box don't stack.
        pts = _fan_endpoints(pts, spec["start_shift"], spec["end_shift"])
        sx, sy = pts[0]
        arrow_id = spec["id"]
        _rel = [(px - sx, py - sy) for px, py in pts]
        _xs = [p[0] for p in _rel]
        _ys = [p[1] for p in _rel]
        # Crisp elbow look: sharp corners (roundness None) and no hand-drawn
        # wobble (roughness 0), so the connectors read as clean right angles.
        el = self._base(arrow_id, "arrow", sx, sy,
                        (max(_xs) - min(_xs)) or 1.0, (max(_ys) - min(_ys)) or 1.0,
                        roundness=None, roughness=0,
                        strokeStyle="dashed" if spec["dashed"] else "solid")
        el.update({
            "points": [[round(x, 2), round(y, 2)] for x, y in _rel],
            "lastCommittedPoint": None,
            "startBinding": {"elementId": src, "focus": 0, "gap": 2},
            "endBinding": {"elementId": dst, "focus": 0, "gap": 2},
            "startArrowhead": None,
            "endArrowhead": "arrow",
            "elbowed": False,
        })
        # register the arrow on both boxes so bindings are honoured
        for box_id in (src, dst):
            self._get(box_id)["boundElements"].append({"type": "arrow", "id": arrow_id})
        self._elements.append(el)

        if label:
            label_id = spec["label_id"]
            lw = _text_width(label, font_size)
            lh = font_size * _LINE_HEIGHT
            (p1, p2) = _longest_segment(pts)
            seg_len = math.hypot(p2[0] - p1[0], p2[1] - p1[1])
            mx, my = (p1[0] + p2[0]) / 2, (p1[1] + p2[1]) / 2
            dx, dy = p2[0] - p1[0], p2[1] - p1[1]
            v_off = lh / 2 + 7   # clear gap above/below a horizontal line
            h_off = lw / 2 + 10  # clear gap beside a vertical/diagonal line
            # When the label is wider than the edge it labels, sitting it on the
            # line forces it over a neighbouring box. Instead LIFT it into a clear
            # band above both boxes (issue: short edge, long label).
            short_edge = lw > seg_len - 8
            above_y = min(a["y"], b["y"]) - lh / 2 - 16
            above = (mx, above_y)

            cands = []
            if short_edge:
                cands.append(above)
            if abs(dy) <= abs(dx) * 0.35:        # horizontal line -> above, then below
                cands += [(mx, my - v_off), (mx, my + v_off),
                          (mx, my - v_off - lh), (mx, my + v_off + lh)]
            elif abs(dx) <= abs(dy) * 0.35:      # vertical line -> to the side
                cands += [(mx + h_off, my), (mx - h_off, my),
                          (mx + h_off + lw / 2, my), (mx - h_off - lw / 2, my)]
            else:                                # diagonal line -> to the side
                cands += [(mx + h_off, my), (mx - h_off, my),
                          (mx, my - v_off), (mx, my + v_off)]
            cands += [above, (mx, above_y - lh)]   # always fall back to the lifted band

            def _overlaps(tlx, tly):
                for e in self._elements:
                    if e["type"] not in ("rectangle", "ellipse", "diamond", "text"):
                        continue
                    if e["id"].startswith("zone"):
                        continue
                    ix = min(tlx + lw, e["x"] + e["width"]) - max(tlx, e["x"])
                    iy = min(tly + lh, e["y"] + e["height"]) - max(tly, e["y"])
                    if ix > 1 and iy > 1:
                        return True
                return False

            cxl, cyl = cands[0]
            for _cx, _cy in cands:
                if not _overlaps(_cx - lw / 2, _cy - lh / 2):
                    cxl, cyl = _cx, _cy
                    break
            lbl = self._base(label_id, "text", cxl - lw / 2, cyl - lh / 2,
                             lw, lh, strokeColor="#495057", roundness=None)
            lbl.update({
                "text": label,
                "fontSize": font_size,
                "fontFamily": _FONT_HAND,
                "textAlign": "center",
                "verticalAlign": "middle",
                "containerId": None,
                "originalText": label,
                "lineHeight": _LINE_HEIGHT,
                "autoResize": True,
                "baseline": int(font_size * 0.9),
            })
            self._elements.append(lbl)

    def flow(self, boxes: list[str], dashed: bool = False,
             labels: Optional[list[str]] = None) -> None:
        """Connect a sequence of boxes with arrows: boxes[0]→[1]→…→[n]."""
        labels = labels or [None] * (len(boxes) - 1)
        for i in range(len(boxes) - 1):
            self.arrow(boxes[i], boxes[i + 1],
                       label=labels[i] if i < len(labels) else None,
                       dashed=dashed)

    def zone(self, boxes: Iterable[str], label: Optional[str] = None,
             color: str = "gray", pad: float = 28.0,
             stroke: Optional[str] = None) -> str:
        """A translucent rectangle drawn *behind* the given boxes to group them.

        Grouping is one of the things the overview does that a flat diagram
        can't — it says "these three belong together" at a glance. ``stroke`` sets
        the dashed border (and label) colour — use it when two sets of zones
        overlap (e.g. vertical "nodes" and horizontal "namespaces") so each set is
        told apart; vary ``pad`` too so their borders/labels don't coincide.

        Deferred (built at render) so it wraps the boxes' FINAL, re-spaced
        positions instead of their provisional ones.
        """
        zone_id = self._next_id("zone")
        self._zone_specs.append({
            "id": zone_id, "boxes": list(boxes), "label": label,
            "color": color, "pad": pad, "stroke": stroke,
        })
        return zone_id

    def _emit_zone(self, spec: dict) -> None:
        members = [self._get(b) for b in spec["boxes"]]
        pad, label = spec["pad"], spec["label"]
        min_x = min(m["x"] for m in members) - pad
        min_y = min(m["y"] for m in members) - pad - (24 if label else 0)
        max_x = max(m["x"] + m["width"] for m in members) + pad
        max_y = max(m["y"] + m["height"] for m in members) + pad
        bg = COLORS.get(spec["color"], spec["color"])
        sc = spec["stroke"] or "#868e96"
        el = self._base(spec["id"], "rectangle", min_x, min_y,
                        max_x - min_x, max_y - min_y,
                        backgroundColor=bg, fillStyle="hachure",
                        strokeStyle="dashed", strokeColor=sc, opacity=60)
        # insert at the front so it sits behind the boxes
        self._elements.insert(0, el)
        if label:
            self.text(label, x=min_x + 12, y=min_y + 6, font_size=16, color=sc)

    def note(self, content: str, near: Optional[str] = None,
             x: Optional[float] = None, y: Optional[float] = None) -> str:
        """A margin annotation. If ``near`` is a box id, it is placed below it.

        Deferred (built at render) so a ``near`` note follows its box to the box's
        final position after relayout.
        """
        note_id = self._next_id("note")
        self._note_specs.append({
            "id": note_id, "content": content, "near": near, "x": x, "y": y,
        })
        return note_id

    def _emit_note(self, spec: dict) -> None:
        near, x, y = spec["near"], spec["x"], spec["y"]
        if near is not None:
            b = self._get(near)
            x = b["x"] if x is None else x
            y = b["y"] + b["height"] + 24 if y is None else y
        x = 120.0 if x is None else x
        y = 380.0 if y is None else y
        # Wrap prose notes so one long line can't blow up scene width (which would
        # force fitted() to zoom far out into a short, cramped pane).
        _wrapped = _wrap_text(spec["content"], 16, 520.0)
        self.text(f"✎ {_wrapped}", x=x, y=y, font_size=16, color="#e8590c")

    # -- render-time layout -------------------------------------------------
    def _relayout_grid(self) -> None:
        """Re-space grid-placed boxes by their ACTUAL sizes, with guaranteed
        edge-to-edge gutters, so long labels can never make neighbours overlap.

        A column is as wide as its widest box; a row as tall as its tallest. The
        provisional fixed-pitch positions from ``box()`` are replaced here, once,
        before arrows/zones/notes are built against the final coordinates.
        """
        grid = [e for e in self._elements if "_col" in e]
        if not grid:
            return
        GUT_X, GUT_Y = 120.0, 90.0   # min gap between boxes (room for arrow + label)
        cols = sorted({e["_col"] for e in grid})
        rows = sorted({e["_row"] for e in grid})
        col_w = {c: max(e["width"] for e in grid if e["_col"] == c) for c in cols}
        row_h = {r: max(e["height"] for e in grid if e["_row"] == r) for r in rows}
        col_cx, row_cy, prev = {}, {}, None
        for c in cols:
            col_cx[c] = (self._origin_x + col_w[c] / 2 if prev is None
                         else col_cx[prev] + col_w[prev] / 2 + GUT_X + col_w[c] / 2)
            prev = c
        prev = None
        for r in rows:
            row_cy[r] = (self._origin_y + row_h[r] / 2 if prev is None
                         else row_cy[prev] + row_h[prev] / 2 + GUT_Y + row_h[r] / 2)
            prev = r
        for e in grid:
            nx = col_cx[e["_col"]] - e["width"] / 2
            ny = row_cy[e["_row"]] - e["height"] / 2
            dx, dy = nx - e["x"], ny - e["y"]
            e["x"], e["y"] = nx, ny
            for be in e.get("boundElements", []):
                if be.get("type") == "text":
                    t = self._get(be["id"])
                    t["x"] += dx
                    t["y"] += dy
                    break
            del e["_col"]
            del e["_row"]

    def _materialize(self) -> None:
        """Lay out boxes, then build the deferred arrows / zones / notes. Idempotent
        (guarded), so fitted() and to_dict() can both call it safely."""
        if self._materialized:
            return
        self._materialized = True
        self._relayout_grid()
        # fan out arrows that share a source or a target so they don't overlap
        from collections import defaultdict
        by_dst, by_src = defaultdict(list), defaultdict(list)
        for s in self._arrow_specs:
            by_dst[s["dst"]].append(s)
            by_src[s["src"]].append(s)
        SPREAD = 20.0
        for group, key in ((by_dst, "end_shift"), (by_src, "start_shift")):
            for _bid, lst in group.items():
                n = len(lst)
                if n < 2:
                    continue
                for i, s in enumerate(lst):
                    s[key] = (i - (n - 1) / 2) * SPREAD
        for s in self._arrow_specs:
            self._emit_arrow(s)
        for s in self._zone_specs:
            self._emit_zone(s)
        for s in self._note_specs:
            self._emit_note(s)

    # -- output -------------------------------------------------------------
    def _fit_appstate(self, view_w: float, view_h: float, pad: float) -> dict:
        """Compute an Excalidraw ``scrollX/scrollY/zoom`` that fits the whole scene
        into a ``view_w`` x ``view_h`` pane, content centred.

        The wigglystuff Excalidraw widget does NOT scroll-to-content on load, so a
        wide scene opens at 100% showing only its top-left corner — it looks
        "zoomed in" and you have to zoom out by hand. Setting zoom+scroll in the
        scene's appState (which the widget passes straight into Excalidraw's
        initialData) makes it open fitted. Excalidraw maps a scene point to the
        viewport as ``screen = (scene + scroll) * zoom``, so to centre content we
        solve scroll from that. Zoom is capped at 1.0 (never upscale — that is what
        makes it blurry) and floored at 0.2. Deterministic (rounded), so output
        stays diffable.
        """
        els = [e for e in self._elements
               if e["type"] in ("rectangle", "ellipse", "diamond", "text")]
        if not els:
            return {}
        min_x = min(e["x"] for e in els)
        min_y = min(e["y"] for e in els)
        max_x = max(e["x"] + e.get("width", 0.0) for e in els)
        max_y = max(e["y"] + e.get("height", 0.0) for e in els)
        content_w = max(max_x - min_x, 1.0)
        content_h = max(max_y - min_y, 1.0)
        zoom = min((view_w - 2 * pad) / content_w,
                   (view_h - 2 * pad) / content_h, 1.0)
        zoom = max(zoom, 0.2)
        scroll_x = (view_w / zoom - content_w) / 2 - min_x
        scroll_y = (view_h / zoom - content_h) / 2 - min_y
        return {
            "scrollX": round(scroll_x, 2),
            "scrollY": round(scroll_y, 2),
            "zoom": {"value": round(zoom, 4)},
        }

    def to_dict(self, fit: bool = True, view_w: float = 700.0,
                view_h: float = 560.0, pad: float = 48.0,
                zen: bool = False) -> dict:
        """The scene dict to hand to ``wigglystuff.Excalidraw(scene=...)``.

        Excalidraw paints elements in array order, so later = on top. Edge labels
        are floated to the END so they always sit ABOVE every arrow and line —
        otherwise a label added with its own arrow gets painted over by any arrow
        drawn afterwards and you can't read it.

        ``fit`` (default on) opens the scene zoomed/scrolled to show ALL of it,
        centred — the widget itself won't. Pass ``view_w``/``view_h`` to match the
        pane you render into (roughly the ``height=`` you give ``Excalidraw`` and
        the content column width) so the fit is tight. Set ``fit=False`` for the
        raw scene at 100%.

        ``zen`` (Excalidraw *zen mode*) opens with the shape-tool toolbar and side
        panels hidden — the sketch reads as a clean diagram, not an editor — while
        the canvas stays fully editable: the reader toggles the toolbar back with
        the bottom-left control or ``Alt+Z``. The widget forwards ``appState``
        straight into Excalidraw's ``initialData``, so this is just a flag there.
        """
        self._materialize()
        _labels = [e for e in self._elements if e["id"].startswith("arrowlabel")]
        _rest = [e for e in self._elements if not e["id"].startswith("arrowlabel")]
        app_state = {"viewBackgroundColor": "#ffffff", "gridSize": None}
        if zen:
            app_state["zenModeEnabled"] = True
        if fit:
            app_state.update(self._fit_appstate(view_w, view_h, pad))
        return {
            "type": "excalidraw",
            "version": 2,
            "source": "marimo-visual-explanation",
            "elements": _rest + _labels,
            "appState": app_state,
            "files": {},
        }

    def fitted(self, view_w: float = 1060.0, pad: float = 36.0,
               max_zoom: float = 1.0, min_zoom: float = 0.4,
               zen: bool = False, height_headroom: float = 1.3) -> tuple:
        """Return ``(scene_dict, height_px)`` sized to a ``view_w``-wide pane.

        This is the recommended way to embed a scene. It fits the zoom to the pane
        WIDTH (the real constraint) and then derives the pane HEIGHT from the
        content at that zoom — so the pane is just tall enough, with no empty band,
        and the diagram fills it. Pass the returned height to ``Excalidraw``:

            scene, h = s.fitted()
            mo.ui.anywidget(Excalidraw(scene=scene, height=h))

        Guessing a fixed height (the old way) is what made panes too tall and the
        drawing look tiny/lost. **``view_w`` must match your content column**: a
        marimo ``width="medium"`` app is **1110px** (default 1060 leaves margin);
        ``width="compact"``/``"normal"`` is 740px (pass ``view_w=700``);
        ``width="full"`` is the viewport (pass a large value). Fitting into 700 in
        a 1110 column is what left diagrams filling only ~60% of the pane. Zoom is
        capped at 1.0 (never upscale = never blurry).

        ``zen`` (default **off**) would open in Excalidraw zen mode — toolbar/side
        panels hidden, canvas still editable. It is off by default because zen mode's
        exit control overlays the bottom-left **zoom** buttons, making zoom in/out
        harder to hit; the plain toolbar is the lesser annoyance. The toolbar can't
        be repositioned (Excalidraw fixes it), so there is no clean way to tuck it
        away — leave ``zen=False`` unless you specifically want the decluttered look
        and don't need easy zooming.
        """
        self._materialize()
        els = [e for e in self._elements
               if e["type"] in ("rectangle", "ellipse", "diamond", "text")]
        if not els:
            return self.to_dict(fit=False), 400
        min_y = min(e["y"] for e in els)
        max_x = max(e["x"] + e.get("width", 0.0) for e in els)
        min_x = min(e["x"] for e in els)
        max_y = max(e["y"] + e.get("height", 0.0) for e in els)
        content_w = max(max_x - min_x, 1.0)
        content_h = max(max_y - min_y, 1.0)
        zoom = min((view_w - 2 * pad) / content_w, max_zoom)
        zoom = max(zoom, min_zoom)
        # height_headroom (default 1.3) makes the pane ~30% taller than the diagram
        # needs at fit-zoom, so there is room to ZOOM IN on the canvas without the
        # drawing being clipped by the pane edge.
        height = int(round(content_h * zoom * height_headroom + 2 * pad))
        return (self.to_dict(fit=True, view_w=view_w, view_h=height, pad=pad,
                             zen=zen), height)

    def to_json(self, indent: int = 2) -> str:
        return json.dumps(self.to_dict(), indent=indent)

    # -- validation ---------------------------------------------------------
    def validate(self) -> list[str]:
        """Return a list of structural problems (empty == looks good).

        This is the deterministic, browser-free check the skill relies on: the
        visual only renders in a browser, but a self-consistent scene (unique
        ids, bindings that point at real elements, bound text that names its
        container) is the strongest signal we can get headless.
        """
        self._materialize()
        problems: list[str] = []
        ids = [e["id"] for e in self._elements]
        if len(ids) != len(set(ids)):
            problems.append("duplicate element ids")
        idset = set(ids)
        required = {"id", "type", "x", "y", "width", "height", "seed", "version"}
        for e in self._elements:
            missing = required - e.keys()
            if missing:
                problems.append(f"{e['id']}: missing {sorted(missing)}")
            for b in (e.get("boundElements") or []):
                if b["id"] not in idset:
                    problems.append(f"{e['id']}: boundElements -> missing {b['id']}")
            if e["type"] == "arrow":
                for side in ("startBinding", "endBinding"):
                    bind = e.get(side)
                    if bind and bind["elementId"] not in idset:
                        problems.append(f"{e['id']}: {side} -> missing {bind['elementId']}")
            if e["type"] == "text" and e.get("containerId"):
                if e["containerId"] not in idset:
                    problems.append(f"{e['id']}: containerId -> missing {e['containerId']}")
        return problems


if __name__ == "__main__":
    # Build a representative scene and self-validate it.
    s = Scene(title="Request lifecycle (overview)")
    client = s.box("Client", color="blue", subtitle="browser / app")
    api = s.box("Core API", color="green", subtitle="auth + routing")
    data = s.box("Data", color="yellow", subtitle="Postgres + cache")
    s.flow([client, api, data], labels=["HTTPS", "query"])
    s.zone([api, data], label="our infra")
    s.note("cache miss = the slow path", near=data)

    problems = s.validate()
    print(f"elements: {len(s._elements)}")
    print(f"validation problems: {problems if problems else 'none ✓'}")
    print(s.to_json()[:400] + "\n...")
    assert not problems, problems
    print("\nOK: scene is self-consistent.")
