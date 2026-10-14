# Deliverable Templates

The Team Lead synthesizes one deliverable — the consolidated `<slug>-research.md` — from the raw
complete researcher responses retained by the lead as `*-findings.md` files, and writes the
folder's `meta.yaml` streams skeleton. Read-only researchers return text, not files.
The `<stream>-spec.md` and `<stream>-plan.md` are authored downstream by `spec-writing-cl` and
`plan-writing-cl`; research-team-cl does not write them.

---

## Deliverable: `<slug>-research.md`

The consolidated research document.

```markdown
# <Domain Name> — Research

**Date**: <date>
**Status**: Complete
**Authors**: Team <team-name> (AI-assisted research)

## Problem Statement

<what we set out to understand>

## Scope

**In scope**: <what was investigated>
**Out of scope**: <what was explicitly excluded>

## Key Findings

### <Finding Area 1>

<synthesized findings from relevant streams, with evidence>

### <Finding Area 2>

<synthesized findings>

...

## Constraints & Hard Limits

<all discovered constraints, consolidated and deduplicated>

## Trade-off Analysis

<comparison of approaches where multiple options exist, with recommendations>

## Open Questions

<unresolved items that need future investigation or user decisions>

## Appendix: Stream Findings

- [<stream-1>-findings.md](<relative-path>) — <one-line summary>
- [<stream-2>-findings.md](<relative-path>) — <one-line summary>
...
```

---

## Manifest: `meta.yaml` streams skeleton

Write one `research:` entry and one stream per independent work stream the research identified,
leaving `spec:` and `plan:` **null** — the downstream skills fill the field they own.

```yaml
slug: NN-slug
intent: "One line: what this investigation was."
created: <today, YYYY-MM-DD>     # from context; do not invent
updated: <today, YYYY-MM-DD>

research: <slug>-research.md     # ONE per folder; naming it is what marks research done

streams:
  - slug: <stream>              # THE name: <stream>-spec.md, <stream>-plan.md, --tags <stream>
    intent: "One line: what this stream is."
    spec: null                  # spec-writing-cl fills this in
    plan: null                  # plan-writing-cl fills this in
    epic: null                  # optional reference only when tk is selected
    tasks: []                   # optional references, never mirrored mutable status
```
