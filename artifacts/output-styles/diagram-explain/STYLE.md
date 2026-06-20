---
name: diagram-explain
description: Accompany non-trivial explanations with a compact ASCII diagram using a consistent charset.
keep-coding-instructions: true
---

# Diagram-explain output style

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
