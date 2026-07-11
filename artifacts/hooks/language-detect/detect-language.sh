#!/usr/bin/env bash
# language-detect — a SessionStart hook (startup|clear|compact) that inspects the
# repo root for language manifests and tells the agent which idioms the codebase
# actually follows, before it writes its first line.
#
# It emits a POINTER, not the idiom body: naming "Go (go.mod)" costs a line, while
# inlining a style guide would spend the context budget the idiom instruction itself
# already owns. If the matching instruction is installed the agent reads it; if not,
# the pointer still biases it toward the language's conventions.
#
# Conditional by design (same contract as work-state-reground): it only names a
# language whose manifest is actually present, so it never asserts an idiom for a
# language the repo does not use. A repo with no recognized manifest emits nothing.
# Fails open (exit 0).

set -uo pipefail

root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)

hits=""

# --- Go
if [ -f "$root/go.mod" ]; then
  hits="${hits}
- Go (go.mod) — follow Go idioms; apply the go-style-uber instruction if installed."
fi

# --- Rust
if [ -f "$root/Cargo.toml" ]; then
  hits="${hits}
- Rust (Cargo.toml) — follow Rust idioms (clippy-clean, Result-based errors)."
fi

# --- JavaScript / TypeScript
if [ -f "$root/package.json" ]; then
  hits="${hits}
- JavaScript/TypeScript (package.json) — follow the project's existing module and typing conventions."
fi

# --- Python
if [ -f "$root/pyproject.toml" ] || [ -f "$root/requirements.txt" ] || [ -f "$root/setup.py" ]; then
  hits="${hits}
- Python (pyproject.toml/requirements.txt) — follow PEP 8 and the project's typing conventions."
fi

# Nothing recognized — stay silent rather than emit an empty instruction.
if [ -z "$hits" ]; then
  printf '{}\n'
  exit 0
fi

msg="Project language(s) detected — match the codebase's existing idioms rather than importing another language's style:${hits}"

escape_for_json() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  printf '%s' "$s"
}
escaped=$(escape_for_json "$msg")

printf '{\n  "hookSpecificOutput": {\n    "hookEventName": "SessionStart",\n    "additionalContext": "%s"\n  }\n}\n' "$escaped"

exit 0
