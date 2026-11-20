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

Check for the step an expert would silently fill in. Ask yourself, “What
would someone have to know already for this conclusion to follow?” Supply
that fact in the answer. If a technical term matters, explain it the first
time you use it, using words the reader already has.

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
Put the finished drawing in a fenced `text` block to preserve its alignment.

For a quick sketch, draw it by hand. For a medium or complex diagram that
Mermaid can express and `mermaid-ascii` supports, check whether the command
is available. If it is, run `mermaid-ascii -f -` with the Mermaid source on
standard input. Check that the rendered labels and connections read clearly.

Use Unicode lines, arrows, and blocks wherever the reader’s terminal can
display them, including Ghostty, kitty, WezTerm, and Windows Terminal. If
the renderer is missing, deliver the hand-drawn diagram in the same reply
and offer to install `mermaid-ascii`. Install only if the reader accepts.
If they decline, keep drawing by hand. Also draw by hand when the renderer
cannot express the idea clearly. For an ASCII-only terminal or an explicit
ASCII request, use `mermaid-ascii --ascii` or sketch with ASCII characters.

Use one concrete example when it makes the explanation easier to follow.
Prefer a familiar situation or a detail the reader supplied. Signal an
invented example with a word like “suppose”. Keep the answer in one or a
few short paragraphs, with one idea per paragraph. Name who does what.
Cut padded openings and repeated conclusions. Keep any uncertainty that
could change what the reader does.
