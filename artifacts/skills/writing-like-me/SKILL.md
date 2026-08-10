---
name: writing-like-me
description: >
  Write in your own voice, editorially clean first. Runs the writing-editorial tiers over a draft (or
  over a faceless base draft it composes), then a voice pass in which a Claude subagent and a second
  model from a different family each apply your exemplar corpus independently, then merges the two
  and shows you where they disagreed. Use WHENEVER the user asks to "make this sound like me", "write
  this in my voice", "draft this the way I would", or hands over a draft and asks to have it voiced.
  Ships with EMPTY exemplar files by design: it does nothing useful until you supply a corpus at
  ~/.claude/patronus/voice/. Requires the writing-editorial skill, which its manifest pulls in
  automatically.
---

# Write like me

**Editorial controls first, personal voice second.** An editor works span by span; a composer may
restructure. This pipeline puts every span-local pass ahead of the one pass allowed to rewrite
freely, so the voice pass works on prose whose surface problems are already gone.

```
  [1] writing-editorial          all four tiers, per its dispatch choice
       |                         (skipped with a warning if not installed)
       v  clean, de-slopped draft + PRESERVE list
       |
  [2] +------------------------+        +--------------------------+
      | Claude subagent        |        | codex over MCP           |
      | fresh context          |        | sandbox: read-only       |
      | exemplars: voice only  |        | same context             |
      | + weights.md           |        | codex-reply to refine    |
      | + PRESERVE + ledger    |        |                          |
      | + original             |        |                          |
      +------------------------+        +--------------------------+
       |                                 |
       v  voiced draft A                 v  voiced draft B
       |                                 |
  [3] main agent merges <----------------+
       |
       v
      1. merged draft
      2. where the two disagreed, verbatim, both versions
      3. 2-4 threadable variations
```

## Entry modes

This skill is both an editor and a composer, so decide first which one you are. Choose from what the
user supplies:

| Mode | Input | Stage 1 |
|---|---|---|
| **Edit** | an existing draft | run the editorial tiers over it, then voice |
| **Compose** | a request, no draft | write a meaning-first base draft **with no corpus in context**, then the editorial tiers, then voice |
| **Voice-only** | a draft the user says is already edited | skip the tiers, go straight to the voice stage, and say so |

Voice-only and a missing sibling skill both skip tier-1, so no ledger is emitted. Do not treat that
as an empty slot: an unexamined draft may already carry a live correction, and assuming `remaining:
1` would license a second. Apply tier-1.3's detection to the incoming draft first, open the ledger
from what you find, and say that its state was inferred rather than carried.

Compose mode's faceless first draft is the point. A draft written before any voice is in context
reaches for different material than one written to sound like someone, and that difference is worth
more than the head start.

## Stage 1: the editorial tiers

Run all four tiers of the sibling `writing-editorial` skill, honoring its dispatch question. Carry
two things forward, because everything downstream is bound by both: its **PRESERVE list** (from
tier-2) and its **contrast ledger** (from tier-1.3).

**The sibling resolves by path, not by name.** The tier files are at
`{skillsDir}/writing-editorial/tier-0.md` through `tier-3.md`, with the router at
`{skillsDir}/writing-editorial/SKILL.md`. `{skillsDir}` is substituted at install time to the
directory holding every installed skill, so the path is correct on each agent's layout without this
file knowing which one it is on. Where the host exposes a skill-invocation mechanism by name, using
it is equivalent and preferred, because it honors the router's own dispatch question directly. Where
it does not, read the router and the four tier files from those paths and apply them in order. Both
routes reach the same files, which is what makes the absence check below meaningful.

**If the tier files are absent from that path**, this is corruption rather than a supported mode. The
`requires:` edge means `writing-editorial` is present under every supported install path, so an
absence means a manually deleted directory, a damaged install, or a hand-copied skill. Report the
expected sibling location, state that stage 1 was skipped and the draft is unedited, and run stages 2
and 3 anyway. Do not go hunting for the skill across agent layouts.

## Stage 2: the voice pass

Both models receive exactly the same six context items:

1. the cleaned draft from stage 1;
2. the exemplar pool, and what to read from it versus take from the draft (below);
3. `{skillDir}/weights.md`;
4. the **PRESERVE list**;
5. the **contrast ledger**;
6. the **pre-editorial original**.

Neither receives the editorial tier files. The voice pass is not there to re-litigate decisions the
editorial pass already made.

**Why the original matters.** The never-inject rule says add nothing the author did not write, and
that is unenforceable without a fixed reference: a fresh model cannot tell an author's first-person
aside from one an earlier pass introduced. With the original in context, the constraint is checkable.
In voice-only mode there is no original, so the rule narrows to "add nothing not present in the draft
you were given", which is weaker, and the output says so.

**PRESERVE is binding here too.** This is the one stage permitted to rewrite freely, which makes it
the stage where a protected span is most at risk. A voice pass that flattens a coined term
immediately after tier-2 protected it defeats the whole mechanism. Override an entry only with a
stated reason, naming which entry and why.

**The contrast ledger binds this stage as well**, and for the same reason it binds tier-3: a pass
that writes new sentences is not given a fresh allowance. Applying a voice is where an antithesis is
most tempting, because mirrored shapes read as punchy and voice work rewards punch. With
`remaining: 0`, voice the draft in the positive. Displacing the retained correction is allowed with
a stated reason; adding a second is not.

### What the corpus is for

**The corpus supplies a voice, never a format.** This is the distinction the whole stage turns on,
and getting it backwards produces the most common failure: a long draft chopped into the shape of
the exemplars, so a design doc comes back reading like a thread of posts.

Two things live in any exemplar, and they travel differently:

| Read from the corpus | Take from the draft |
|---|---|
| sentence rhythm and the length spread within a paragraph | how long the finished piece is |
| diction, register, and the words this author reaches for | what the piece has to cover |
| how a paragraph opens, turns, and lands | how many paragraphs there are |
| punctuation habits, contractions, spelling conventions | the section structure and headings |
| stance: hedged or blunt, first person or impersonal | the order of the argument |
| the moves, such as a concrete case before the claim | the scope of each section |

The left column is the voice and it projects onto any length. The right column belongs to the piece
you were given, and the voice pass has no business changing it.

**A corpus of short pieces still teaches long-form voice.** Rhythm is a property of consecutive
sentences, not of a word count, so it is fully present in a 120-word post and it scales without
distortion. What a short corpus cannot teach is long-form architecture: how this author sustains an
argument over ten paragraphs, when they summarize, how they hand off between sections. Take that
from the draft and from tier-3's work, which already settled it.

### Measure the corpus; do not imagine it

**Short-form pieces are not made of short sentences, and assuming they are is the failure mode this
section exists to stop.** A corpus of posts routinely averages 17 words a sentence with a fifth of
them past 26, which is an ordinary spread for any register. A voice pass that reads "short-form" and
reaches for clipped, punchy sentences is applying a stereotype of the format rather than the voice in
front of it, and it will flatten an author who does not write that way.

So before voicing anything, take the corpus's actual numbers:

```
- mean and median sentence length
- the share of sentences past 26 words
- the longest sentence in the pool
- paragraph lengths: shortest, typical, longest
```

Those are the target. Match that distribution in the output, and check the result against it: if the
draft you produce has a lower mean, fewer long sentences, or a smaller spread than the corpus, you
have imposed brevity the author did not write. Say what you measured, so the reader can see which
distribution you were aiming at.

This matters more, not less, when projecting onto a long piece. Sustained argument needs the long
sentences, and they are the first thing lost when a model treats "short-form corpus" as a licence to
chop.

Concretely, when the corpus is short-form and the target is long:

- **project** the sentence-level habits across every paragraph of the long piece;
- **do not** shorten paragraphs toward the exemplar length;
- **do not** add the punchy one-line turns that suit a post and read as manufactured rhythm at
  length, which is the tier-2.3 anti-rule;
- **do not** cut material to reach a post-sized piece. The draft's coverage is fixed.

### Corpus resolution

Resolve the pool for the target form. First hit wins:

1. `$PATRONUS_VOICE_DIR/<genre>.md`, when the environment variable is set;
2. `~/.claude/patronus/voice/<genre>.md`, the default user-owned location;
3. the shipped stub, which contains no exemplars and triggers the empty-pool path.

Create neither the directory nor the files. On a first run with no corpus, print the resolved path
you looked for and what to put there, then continue in degraded mode. Corpus setup stays an explicit
user act, and upgrades stay incapable of touching it.

**The pool matching the target form wins whenever it has exemplars.** Long target with a populated
`long-form.md` uses `long-form.md`, and the short pool is not consulted, because a pool in the target
form carries architecture as well as voice. The other pool is used only when the matching one is
empty, and then it is projected rather than copied.

| Target | long-form.md | short-form.md | Behaviour |
|---|---|---|---|
| long | populated | either | **use `long-form.md`.** Matching form, so voice and architecture both come from it |
| long | empty | populated | **use `short-form.md`, projected.** A supported path, not a degraded one, so it needs no permission. Voice from the corpus, length and structure from the draft |
| short | either | populated | **use `short-form.md`** |
| short | populated | empty | **use `long-form.md`, projected.** Same rule in reverse: take rhythm and diction, not the long piece's architecture |
| any | empty | empty | skip the voice stage, run the editorial tiers only, and say the pipeline ran in editorial-only mode |
| any | unreadable | unreadable | report the path and the error, treat as empty, do not fail the run |

Say which pool was used either way. Where the pools differ in what they can teach, the reader should
know which one shaped the output.

### The Claude side

A fresh subagent, with the six context items above.

### The codex side

Reach codex over MCP, not by driving its CLI. Codex ships an MCP server on stdio:

```
codex mcp-server
```

Register it once as an MCP server named `codex`, then call its tools. It exposes two:

| Tool | Use |
|---|---|
| `codex` | start a session. Required argument: `prompt`. Useful optional arguments: `sandbox`, `cwd`, `model`, `base-instructions` |
| `codex-reply` | continue that session, with `threadId` and the next `prompt` |

Call `codex` with `sandbox: "read-only"`, and with `prompt` naming the paths of all six items: the
draft, exemplars, weights, PRESERVE list, contrast ledger, and original. Codex reads them from disk
itself.

**The draft never goes into a shell string, and over MCP it never goes into an argv either.** The
prompt is a JSON string field, so quoting, backticks, newlines, and command-length limits stop being
failure modes: the transport carries arbitrary bytes without escaping. Passing paths rather than
pasted content keeps the prompt small and lets codex read exactly what stage 2 wrote.

**The transport owns the timeout and the process.** The MCP client bounds the call, returns a typed
error on expiry, and manages the server process, so this file specifies no signals, no reaping, and
no fallback timer. Where the host lets you set a per-call timeout, allow around three minutes: a
voice pass over a long draft is slower than a typical tool call.

**The result is the reply, not a delimited region of stdout.** An MCP tool call returns structured
content, so there is no envelope to agree on and nothing to parse out of console chatter. Take the
returned text as the voiced draft. Where codex returns commentary alongside it, ask for the draft
alone via `codex-reply` rather than guessing which span is the draft.

`codex-reply` is what makes this better than a one-shot call. If the first pass drifts from the
corpus, over-corrects, or flattens a PRESERVE span, continue the same thread and say so, instead of
restarting with a longer prompt.

**Degrade, never block.** If the codex MCP server is not registered, the call errors, or it times
out, continue with the Claude draft alone and say which of those happened, naming the error. The
cross-model stage is an optional check, not a dependency.

**Why a second model rather than a second Claude.** Same-family judges are biased toward
low-perplexity text, and low perplexity is the signature of generic machine prose, so a same-family
critic is biased in the wrong direction for voice work. One model from a different family captures
nearly all the available benefit; a larger panel does not.

## Stage 3: the merge

**The main agent merges**, not a third subagent. Produce three things:

1. The merged draft: one version, taking the stronger choice at each point of divergence.
2. The disagreements, verbatim. Where the two drafts diverged materially, show both versions.
   Disagreement goes to the human rather than to a vote: with two voters there is no majority, and
   the disagreements are the most informative output of the stage.
3. Two to four threadable variations, concrete alternatives the user can splice in, each named
   for what it changes: open on the concrete case rather than the claim; keep or drop a hedge; add or
   remove a coined term; hand off at the ending rather than summarizing.

Report any PRESERVE entry that either model overrode, with the reason it gave.

**Recompute the ledger from the merged draft.** The two voice passes run in parallel, so each may
have displaced the retained correction independently. Taking the stronger half of each can therefore
carry both replacements through and leave two live corrections where the ledger allows one. Never
union the two branches' ledgers: read the merged text and count what is actually in it, then report
that single reconciled ledger and any reallocation it records.

## The run trail

Every run writes its own directory under `docs/writing/<slug>/`, where `<slug>` names the piece.
Whether that path is committed or ignored is the user's call: add it to `.gitignore` for private
drafting, leave it tracked where the prose is a team deliverable.

Ask once, at the start:

```
Keep full drafts for diffing? [Y/n]
```

Enter keeps them. Running unattended keeps them too, so the default is the same either way and a
trail is never silently thinner than it looks. Drafts are what make the trail checkable: without
them a reader has only each stage's own account of what it did, and a stage that fails to report an
edit is indistinguishable from one that made none.

```
docs/writing/<slug>/
  manifest.yaml         always
  EDITS.md              always, per stage
  PRESERVE.md           always
  draft-00-original.md  when drafts are kept
  draft-0N-<stage>.md   when drafts are kept
```

`manifest.yaml` records what each stage did and what it carried:

```yaml
run: deleuze-three-contributions
mode: compose
drafts_kept: true
stages:
  - stage: tier-1
    rules_fired: [tier-1.3]
    edits: 3
    contrast_ledger:
      retained: "actualization is not resemblance but invention"
      remaining: 0
  - stage: tier-3
    rules_fired: [tier-3.7, tier-3.8, tier-3.11]
    edits: 4
    preserve_overrides: []
    ledger_reallocations: []
voice:
  corpus: ~/.claude/patronus/voice/short-form.md
  exemplars_supplied: 12
  model_reported_influences:      # the model's own account, not verified provenance
    - "consequence-first turn opening on a bare connective"
```

Two things about that shape are deliberate. **The drafts are what make it checkable**, not the
manifest: a stage's account of its own edits is a claim, and the draft beside it is the evidence.
Diff two stages and you see what actually changed, including what a stage failed to mention. The
manifest carries no checksums, because a model writing a hash it did not compute records a number
that proves nothing.

And **`model_reported_influences` is labelled as self-reported**, because a model cannot reliably say
afterwards which exemplar shaped a structure it synthesized across a dozen. The corpus path and the
exemplar count are knowable; the attribution is a claim.

## Known limits, stated rather than hidden

- Register mismatch. Informal-register imitation verifies far less reliably than formal registers,
  so a corpus of casual posts gives weaker signal than one of worked prose. This is about how
  reliably the voice transfers, not about which lengths are allowed: a short corpus projecting onto
  a long piece is the supported path, and the constraint is that casual registers are harder to
  imitate at all.
- A short corpus cannot teach long-form architecture. Sentence rhythm and diction scale from a
  120-word post to a 4000-word essay, but how this author sustains an argument across ten
  paragraphs is not in the corpus to learn. That comes from the draft and from tier-3, and the
  voice pass leaves it alone.
- Style strength and content preservation trade off against each other. They are competing
  objectives, not a tuning failure. This pipeline picks a point on that curve by putting
  content-shaping first and voice second.
- Scrubbing a widely circulated word list is itself becoming detectable, which is a reason the
  editorial tiers lean on structural rules rather than on lexical substitution alone.
