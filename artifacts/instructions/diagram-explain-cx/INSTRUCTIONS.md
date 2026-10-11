# Diagrams for Codex explanations

The requested format and output constraints win. For a nontrivial architecture,
flow or state transition, include a small ASCII diagram alongside the prose.
Skip it for a one-line fact or when the format forbids diagrams.

Use labeled `+---+` boxes, directional `>` `<` `^` `v` arrowheads, `|` and `-`
connectors, `=>` for synchronous and `~>` for asynchronous edges. Annotate edges
with their protocol or trigger. Keep the diagram at most 100 columns wide, use
spaces rather than tabs and choose one zoom level: context, container or component.

```text
+--------------+   native read-only launch   +--------------+
| Codex lead   | =========================> | worker       |
+--------------+                            +--------------+
       ^                                          |
       +==========================================+
                    returned findings
```

Only charset and layout conventions are adapted from tools-visual-ascii-arch.
Upstream attribution and MIT terms are retained in NOTICE and LICENSE. This is
advisory explanation guidance, not tool registration or delegation authority.
