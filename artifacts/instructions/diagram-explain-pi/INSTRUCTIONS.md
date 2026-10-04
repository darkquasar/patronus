# Diagram-explain-pi

The user's requested format and the task's output constraints win. This guidance
is advisory: omit diagrams when they conflict with those constraints.

When explaining anything non-trivial — an architecture, a control/data flow, a state
machine, or how a change moves through a system — include a small ASCII diagram
alongside the prose. The diagram is a complement to the explanation, not a
replacement: keep the words, add the picture.

Skip the diagram only when the answer is a single fact or a one-line change where a
drawing would add nothing.

## Charset & conventions

Use one consistent, portable charset so diagrams render the same in a terminal, a PR,
or an ADR:

- Nodes: `+---+` boxes (a box per component/service/module), label inside.
- Edges: `|` and `-` for connectors; arrowheads `>` `<` `^` `v` for direction.
- Sync call: `=>`   ·   async / event: `~>`
- Annotate edges with the protocol or trigger (`HTTP`, `gRPC`, `queue`, `event`).
- Tag platform- or scope-specific nodes in brackets, e.g. `[claude]`, `[CI]`.

Layout rules:
- Keep it ≤ 100 characters wide; never use tab characters (spaces only).
- Two spaces of separation between layers; align boxes so edges read cleanly.
- Pick the zoom level that fits the question: context (users ↔ system), container
  (services, DBs, queues), or component (modules, functions) — one level per diagram.

## Example

```
  +---------+   HTTP    +-----------+   =>    +----------+
  |  client | ========> |  api/web  | ======> |  service |
  +---------+           +-----------+         +----------+
                              |  ~> event           |
                              v                      v
                        +-----------+          +----------+
                        |  queue    |          |   db     |
                        +-----------+          +----------+
```

Every box labeled, every edge directional and annotated, width under 100 — that is the
bar each diagram should clear.

## Attribution and license

This Pi sibling adapts the authored Patronus instruction with explicit user-format
precedence and embeds this notice/license in its emitted entry. Only the ASCII charset and layout
conventions (box/edge characters, sync `=>` vs async `~>`, the ≤100-column / no-tabs
rule, and the context/container/component zoom levels) are borrowed from the upstream
`tools-visual-ascii-arch` skill. The prose and the instruction's behavior are original.

Conventions borrowed from: https://github.com/tjboudreaux/cc-visualization-skills
Commit:  4513e46b4a1bccea0f5177b16a8ee069a4d01b30
License: MIT

----------------------------------------------------------------------

MIT License

Copyright (c) 2026 TJ Boudreaux

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
