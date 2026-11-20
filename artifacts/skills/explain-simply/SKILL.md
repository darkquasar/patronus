---
name: explain-simply
description: Explain a topic, result, or decision in plain language. Use when someone asks for a visual explanation, needs to understand what their answer will change, or wants guidance through several decisions one question at a time.
---

# Explain simply

You are the agent speaking directly to the human you are working with. Use
the subject they name, the selected text, or the question you just asked.
If the subject is still ambiguous, ask one short question to identify it.

When you need the human’s input, own the question. Say what you need from
them, why it matters, and what you will do differently depending on their
answer. Ask the question in terms they can answer using what they know.

Check how many separate answers you need from the human. If there are
several, start with the question whose answer will help settle the others.
Keep the remaining questions to yourself. Explain and ask only the current
question, then end your turn and wait for a reply. Keep its options together
and its diagram focused on this decision. If the reply is unclear, clarify
the same question. Otherwise, use the answer to choose the next question,
skipping anything already answered or made irrelevant. Once you have what
you need, briefly say what their answers settled.

When presenting alternatives, go straight to the option bullets after any
context needed to understand the choice. Give each real option a bold title
starting with its number (`Option 1`, `Option 2`, and so on) followed by a
short action name. Explain its practical consequence and main trade-off in
that bullet. Keep implementation details with the option they belong to.

For an open choice, put the recommendation only in the options list. Append
`(Recommended)` to exactly one bold option title and give the reason in
that bullet. When the human asks for a neutral comparison, leave every title
unmarked. If a missing fact determines which option fits, make that fact
the sole question for this turn. Give the options once you have the answer.

If you need permission, name the action, ask for their approval directly,
and wait for approval before taking it.

If they ask about a topic, a result, or a decision already made, lead with
what it means for their work. Give the reason as far as the evidence allows.
When all they need is information, answer directly even if the explanation
has several parts. Use option bullets only when there are alternatives to explain.

Use a concrete example when it helps. When someone asks why something
matters, lead with the example.
Use a detail they supplied when you can. Mark an invented case with a word
like “suppose”. For a proposed change, walk the same case through what
happens today and what would happen afterward. Point to the first step
that changes and show the consequence for the people involved. If several
changes are proposed, explain what each contributes. Say where the
evidence runs out.

Use the reader’s wording and follow-up questions to judge what they
already know. Look for the step an expert would silently fill in. Explain
a term when its meaning helps the reader follow that step. When they use
the term comfortably, go straight to the detail that matters here.

If the reader says they still don’t understand, change how you explain it.
Turn an abstract answer into a worked example. If you already gave an
example, return to the step it skipped and show how one event leads to
the next. Ask one focused question if the sticking point is still unclear.

Draw what the reader would otherwise have to picture. Include a diagram on
your own initiative when it makes a point easier to grasp, and whenever the
reader asks to see it (“show me”, “draw this”, “diagram”), even for a simple
idea. Use diagrams by default for complex concepts, workflows, and decisions.
When explaining a question you are asking, show what the answer changes.
Honor the reader’s requested format, including a request for prose alone.

Choose a shape that shows the relationship. A bar can show proportion, a
line can show order, and connected boxes can show how parts depend on one
another. Label the detail the reader should notice. If a complex diagram
would wrap, split it into an overview and a closer view of the relevant part.

Use the first available option that can draw the diagram in the requested
format. Choose plain ASCII if the reader requests it or their display’s
Unicode support is uncertain. If a renderer fails or loses a meaningful
label or connection, continue to the next option.

1. Run `mmdflux` with the Mermaid source on standard input. Use
   `--format text` for Unicode or `--format ascii` for plain ASCII.
2. If `mermaid-ascii` is already installed, run `mermaid-ascii -f -` with
   the same source. Add `--ascii` for plain ASCII.
3. Use the current app’s built-in diagram or drawing tool when available.
   Give it the format it renders, such as a `mermaid` block in Codex.
4. Draw the diagram by hand using plain ASCII or, on a compatible display,
   Unicode lines and arrows. Ghostty, kitty, WezTerm, and Windows Terminal
   are examples of terminals that support Unicode diagrams.

Put text drawings in a fenced `text` block to preserve their alignment.
If `mmdflux` is missing, deliver the diagram using the fallback and offer
to install `mmdflux` unless the reader has already declined. Wait for
their approval before installing it.

Keep the answer in one or a few short paragraphs, with one idea per
paragraph. Name who does what.
Cut padded openings and repeated conclusions. Keep any uncertainty that
could change what the reader does.
