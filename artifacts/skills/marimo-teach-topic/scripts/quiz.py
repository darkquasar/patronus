#!/usr/bin/env python3
"""Check-your-understanding helpers for marimo-teach-topic lessons.

Multiple-choice questions with instant, *teaching* feedback: a wrong answer still
explains why. Two functions, used across two cells because of marimo reactivity:

  cell A:  q1_pick = mcq(mo, "question?", ["a", "b", "c"])   # return the widget
           q1_pick
  cell B:  check(mo, q1_pick.value, "b", "because ...")      # reads .value

marimo rules this respects:
  - The widget (`mcq`) is a plain `mo.ui.radio`, so `.value` works downstream.
  - Every question needs a UNIQUE variable name across the whole notebook
    (q1_pick, q2_pick, ...), or you get MultipleDefinitionError at load.
  - `check` re-runs whenever the learner changes the selection.
"""
from __future__ import annotations


def mcq(mo, question, options, label=None):
    """A multiple-choice question. Returns an `mo.ui.radio` — assign it a UNIQUE
    name and return it as the cell's last expression so `.value` is readable in
    the downstream check cell. `options` is a list of answer strings; `.value` is
    the selected string (or None before the learner picks).
    """
    return mo.ui.radio(options=list(options), label=label or f"**❓ {question}**")


def check(mo, selected, correct, explanation, hint=None):
    """Feedback for an `mcq`. Reads the selected value; compares to `correct`.

    - nothing picked yet -> a neutral nudge (and an optional hint)
    - correct   -> green callout + the explanation (reinforce why it's right)
    - incorrect -> red callout + the explanation (so a wrong answer still teaches)
    """
    if not selected:
        body = "_Pick an option above to check your answer._"
        if hint:
            body += f"\n\n_Hint: {hint}_"
        return mo.callout(mo.md(body), kind="neutral")
    if selected == correct:
        return mo.callout(
            mo.md(f"**✅ Correct.**\n\n{explanation}"), kind="success"
        )
    return mo.callout(
        mo.md(
            f"**❌ Not quite.** You chose **{selected}**; the answer is "
            f"**{correct}**.\n\n{explanation}"
        ),
        kind="danger",
    )


def reveal(mo, prompt, answer):
    """Open-ended check: pose a question, hide a model answer behind a toggle.
    Use when there's no clean set of options (design / 'in your own words')."""
    return mo.vstack([
        mo.md(f"**✍️ {prompt}**"),
        mo.accordion({"Show a model answer": mo.md(answer)}),
    ])
