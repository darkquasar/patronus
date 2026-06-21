#!/usr/bin/env bash
# skills-heartbeat — a UserPromptSubmit hook that re-asserts the skill-dispatch
# rule on every human turn. The one-time SessionStart keystone injection decays as
# a long session fills context (the "lost in the middle" problem); this re-injects
# a SHORT reminder at the recent, high-attention end on each turn, plus the names
# of the installed skills so the agent knows what is available without re-reading
# any skill body (bodies still lazy-load via the Skill tool only when one applies).
#
# Deliberately tiny: it pays its cost on EVERY turn, so it lists names, not bodies.
# Fails open — no skills directory means no reminder (exit 0), never a wedge.

set -euo pipefail

SKILLS_DIR="${HOME}/.claude/skills"

# Collect installed skill names (one dir each). Absent dir -> empty list.
names=""
if [ -d "$SKILLS_DIR" ]; then
  for d in "$SKILLS_DIR"/*/; do
    [ -d "$d" ] || continue
    n="$(basename "$d")"
    names="${names:+$names, }$n"
  done
fi

reminder="Before acting on this turn: if any installed skill might apply (even a 1% chance), invoke it via the Skill tool FIRST — this holds across the whole installed set, not one family."
if [ -n "$names" ]; then
  reminder="${reminder} Installed skills: ${names}."
fi

# Escape for embedding in a JSON string (single C-level passes).
escape_for_json() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  printf '%s' "$s"
}
escaped=$(escape_for_json "$reminder")

# Claude Code reads hookSpecificOutput.additionalContext for UserPromptSubmit.
printf '{\n  "hookSpecificOutput": {\n    "hookEventName": "UserPromptSubmit",\n    "additionalContext": "%s"\n  }\n}\n' "$escaped"

exit 0
