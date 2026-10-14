#!/usr/bin/env bash
# Claude SessionStart activation. Scope comes only from this installed hook's
# location; cwd, hook input, HOME and alternate skill roots cannot redirect it.
# Missing selected skill is visible but does not wedge session start.
# Adapted from obra/superpowers hooks/session-start; see NOTICE.
set -euo pipefail

HOOK_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
SKILL="${HOOK_DIR}/../skills/skills-dispatch-cl/SKILL.md"
if [ -f "$SKILL" ]; then
  content="You have an installed skill library. The exact skills-dispatch-cl body from this hook's scope follows. Load applicable skills on demand.

$(cat "$SKILL")"
else
  content="Patronus skills-dispatch-cl activation unavailable: selected skill missing at ${SKILL}. No alternate scope was read."
fi

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
printf '{\n  "hookSpecificOutput": {\n    "hookEventName": "SessionStart",\n    "additionalContext": "%s"\n  }\n}\n' "$escaped"
