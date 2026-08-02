#!/usr/bin/env bash
# Reference form of the serena-refs-hint PreToolUse nudge (the manifest inlines this
# into `hook.command`). NOT placed on disk by Patronus — there is no `files:`/`script:`
# entry, so placeHookScript never runs. Fails OPEN (exit 0) always.
set -euo pipefail

input="$(cat)"
file="$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty' 2>/dev/null || true)"

# Only nudge when editing a source file (by extension); stay silent for docs,
# config, data. The nudge points at serena's caller lookup — a symbol edit that
# ignores its call sites is the mistake this catches.
case "$file" in
  *.go|*.py|*.ts|*.tsx|*.js|*.jsx|*.rs|*.java|*.rb|*.c|*.cc|*.cpp|*.h|*.hpp)
    echo "hint: before editing a shared symbol, check its callers with serena find_referencing_symbols." >&2
    ;;
esac
exit 0
