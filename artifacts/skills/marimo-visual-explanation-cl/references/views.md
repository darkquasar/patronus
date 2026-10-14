# The lenses — doctrine and worked examples

This skill helps someone **think through a decision, or understand how something
works**, during design, planning, and learning. The lenses are tools for that;
the value is in choosing the few that make *this* topic clearer, and in matching
each lens to the question it answers. This file is the reasoning plus two topics
drawn end-to-end: a decision, then a concept explanation.

## The rule: each lens serves the point

Before adding a lens, finish the sentence: *"This shows ___, which helps the user
understand/decide because ___."* If you can't, cut it. Three sharp lenses beat
six that overlap. (For a decision the payoff is a clearer choice; for a concept
it's a clearer mental model — same discipline either way.)

| Lens | The question it answers | Native shape |
|---|---|---|
| Markmap | What are the options and what does each involve? | nested outline (option space) |
| Mermaid | What's the *shape* of the tradeoff / logic / sequence? | the diagram type that fits |
| Weighted matrix | Which option wins, and how sensitive is that to priorities? | sliders → ranked bars |
| Comparison table | How do the options stack up criterion by criterion? | options × criteria grid |
| Excalidraw | What does the recommendation look like? | hand sketch of the choice |
| Tangle | What if this one assumption were different? | inline editable prose |
| Radar | How do the options' overall profiles differ? | overlapping shapes (5+ axes) |

## Match the shape to the question (especially Mermaid)

The most common mistake is forcing one shape onto everything. A tradeoff is a
**quadrant** or a **weighted matrix**, not a sequence diagram. Decision logic is
a **flowchart**. A rollout is a **timeline**. Interactions over time are a
**sequence**. Pick deliberately, and always honour an explicit request ("show me
a timeline"). Mermaid alone gives you `quadrantChart`, `flowchart`, `timeline`,
`sequenceDiagram`, `mindmap`, and `stateDiagram-v2` — see
[mermaid.md](mermaid.md).

## Don't over-render

A decision notebook is usually **3-4 lenses**, not all seven. A pure concept
explanation might be Markmap + a flowchart + an Excalidraw and no matrix at all.
A close quantitative call might be a weighted matrix + a radar + a table and no
Mermaid. Choose for the question.

## Worked example — "Postgres vs DynamoDB vs MongoDB for a new service"

The decision: pick a primary datastore for a new service expecting moderate scale
and a small team. Criteria that matter: strong consistency, scaling model,
operational burden, query flexibility, cost. Here are the lenses that fit — and
the ones that don't.

### Markmap — the option space

```
# Datastore choice
## Postgres
- strong consistency (ACID)
- rich queries (SQL, joins)
- vertical scaling, read replicas
- ops: managed RDS easy
## DynamoDB
- eventual (tunable) consistency
- key-value / single-table design
- seamless horizontal scale
- ops: serverless, no servers
## MongoDB
- tunable consistency
- flexible documents
- sharded horizontal scale
- ops: Atlas managed
```

> Lays out what each option *is* before judging it. A taxonomy, not a flow.

### Mermaid `quadrantChart` — the tradeoff at a glance

```
quadrantChart
    title Flexibility vs operational simplicity
    x-axis "More ops burden" --> "Less ops burden"
    y-axis "Rigid" --> "Flexible"
    quadrant-1 "Flexible & simple"
    quadrant-2 "Flexible, more ops"
    quadrant-3 "Rigid, more ops"
    quadrant-4 "Rigid & simple"
    Postgres: [0.45, 0.55]
    DynamoDB: [0.85, 0.30]
    MongoDB: [0.60, 0.75]
```

> Answers "where does each sit" instantly — the kind of 2×2 you'd draw on a
> whiteboard. Far better than a sequence diagram for a tradeoff.

### Weighted decision matrix — the interactive heart

Sliders for *consistency / scale / ops / flexibility / cost*; each option scored
1-5 on each; the weighted total re-ranks live as the user drags. The point isn't
the default ranking — it's letting the user discover "if I weight consistency
highly, Postgres wins; if scale dominates, Dynamo pulls ahead." See
[decision-lenses.md](decision-lenses.md) for the verified pattern.

### Excalidraw — the recommendation

```python
s = Scene(title="Recommendation: Postgres (RDS) to start")
app = s.box("Service", color="blue")
pg  = s.box("Postgres", color="green", subtitle="RDS, 1 replica")
s.flow([app, pg])
s.note("revisit if write throughput outgrows a single primary", near=pg)
```

> Commits to a recommendation and shows the shape of it, with the condition that
> would change the call.

### What was dropped (and why)

- *Radar*: the quadrant + weighted matrix already cover the multi-criteria
  comparison; a radar would repeat it.
- *Timeline / sequence*: there's no temporal or interaction story here — this is
  a tradeoff, not a process.

That restraint is the doctrine in action: four lenses, each pulling its weight.

## Worked example — "How does the OAuth2 authorization-code flow work?"

No options, no ranking — the job is a clear mental model of a protocol. The
weighted matrix and quadrant have nothing to rank here, so they drop out; the
lenses that fit are a sequence (the spine), a component sketch, and a
decomposition recap.

### Mermaid `sequenceDiagram` — the spine

```
sequenceDiagram
    participant U as User
    participant A as App
    participant Auth as Auth server
    participant R as Resource API
    U->>A: click "log in"
    A->>Auth: redirect with client_id, scope, PKCE challenge
    Auth->>U: prompt to authenticate + consent
    U->>Auth: approve
    Auth->>A: redirect back with authorization code
    A->>Auth: exchange code + PKCE verifier for tokens
    Auth->>A: access token (+ refresh token)
    A->>R: request with access token
    R->>A: protected resource
```

> A protocol is a conversation over time between actors — a sequence is the
> native shape. A quadrant or matrix would say nothing here.

### Excalidraw — who holds what

```python
s = Scene(title="OAuth2 actors")
u  = s.box("User / browser", col=0, row=0, color="blue")
ap = s.box("App", col=1, row=0, color="green", subtitle="holds tokens")
au = s.box("Auth server", col=2, row=0, color="yellow", subtitle="issues tokens")
rs = s.box("Resource API", col=1, row=1, color="green", subtitle="verifies tokens")
s.arrow(u, ap, label="log in")
s.arrow(ap, au, label="code ⇄ token")
s.arrow(ap, rs, label="token + request")
s.note("PKCE binds the code to this app", near=au)
```

> Commits the four actors and what each is responsible for to a single picture —
> the static counterpart to the sequence's motion.

### Markmap — the pieces, as a recap

The moving parts (grant types, token kinds, scopes, PKCE) as a hierarchy at the
end, so a reader can look up any term the flow referenced. A decomposition, not a
flow — the same closing-reference role Markmap plays in a decision.

### What was dropped (and why)

- *Weighted matrix / quadrant / radar*: nothing is being scored or compared.
  Forcing them would invent a ranking the topic doesn't have.
- *Timeline*: the sequence already carries the ordering; a timeline would repeat
  it with less detail.

Same discipline as the decision example: pick the shapes the topic actually has.

## Sequencing in the notebook

1. **The headline up front** — for a decision, the options and your tentative
   pick; for a concept, the one-sentence "what this is". Don't bury it.
2. **A concise explanation** — a few tight, concrete sentences or bullets. This
   is the default; only expand into a longer writeup if the user asks. The
   diagrams support the prose, they don't replace it.
3. **The diagrams** — the spine first (for a decision, the tradeoff: Mermaid,
   possibly two angles in tabs, then the weighted matrix; for a concept, the
   `sequenceDiagram`/`flowchart`/`stateDiagram`), then the **Excalidraw** sketch
   (the recommendation, or the component picture).
4. **Markmap last** — a full-space recap ("everything we weighed" for a decision,
   "all the pieces" for a concept). It closes the notebook rather than leading it.

Put a one-line `mo.md` before each lens saying what to look for and how it
informs the point.
