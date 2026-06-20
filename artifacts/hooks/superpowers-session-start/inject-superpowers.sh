#!/usr/bin/env bash
# superpowers-session-start — a SessionStart hook that injects the
# superpowers-bootstrap skill body as session context, so the skill-first
# discipline is active from the first turn without the agent having to discover
# it. This is the keystone's activation (pairs with the superpowers-bootstrap
# skill that core installs).
#
# Adapted from obra/superpowers hooks/session-start: the upstream reads the skill
# from a plugin root via CLAUDE_PLUGIN_ROOT/run-hook.cmd indirection. Patronus
# installs the skill at a known path instead, so this reads it directly. If the
# skill is not found, the hook emits no context (exit 0) rather than failing — a
# missing skill must not wedge session start.

set -euo pipefail

# The bootstrap skill is installed alongside the agent's other skills. Resolve the
# global location; a project-scoped install would also be found by the agent, but
# the global copy is the stable one for a SessionStart injection.
SKILL="${HOME}/.claude/skills/superpowers-bootstrap/SKILL.md"
if [ ! -f "$SKILL" ]; then
  exit 0
fi
content=$(cat "$SKILL")

# Escape the skill body for embedding in a JSON string (single C-level passes).
escape_for_json() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  printf '%s' "$s"
}
escaped=$(escape_for_json "$content")

context="<EXTREMELY_IMPORTANT>\nYou have superpowers. The full content of your 'superpowers-bootstrap' skill — your introduction to using skills — follows. For all other skills, use the Skill tool.\n\n${escaped}\n</EXTREMELY_IMPORTANT>"

# Claude Code reads hookSpecificOutput.additionalContext for SessionStart.
printf '{\n  "hookSpecificOutput": {\n    "hookEventName": "SessionStart",\n    "additionalContext": "%s"\n  }\n}\n' "$context"

exit 0
