#!/bin/bash
# block-secrets — a PreToolUse guardrail that blocks an agent from WRITING an
# obvious, high-confidence secret into a file (Write/Edit/MultiEdit).
#
# THREAT MODEL (be honest about scope):
#   (a) agent WRITES an obvious secret  -> this hook addresses it (best-effort).
#   (b) agent READS an existing secret and leaks it -> NOT covered here (needs
#       read-path / egress controls; no write-time regex can stop it).
#   (c) agent COMMITS a secret          -> covered by the gitleaks commit-guard.
# This hook reduces (a) only. It is a guardrail, not a guarantee: a determined or
# novel secret format can evade these conservative patterns. Patterns are kept
# high-confidence on purpose, to avoid blocking legitimate edits on false hits.

INPUT=$(cat)

# The text the tool is about to write differs by tool: Write uses .content,
# Edit/MultiEdit use .new_string (or per-edit new_string). Concatenate whatever
# is present so one check covers all three.
CONTENT=$(printf '%s' "$INPUT" | jq -r '
  [ .tool_input.content,
    .tool_input.new_string,
    ( .tool_input.edits // [] | .[]?.new_string )
  ] | map(select(. != null)) | join("\n")
')

if [ -z "$CONTENT" ]; then
  exit 0
fi

# High-confidence secret patterns. Conservative by design — each matches a format
# that is almost never a false positive.
PATTERNS=(
  '-----BEGIN [A-Z ]*PRIVATE KEY-----'        # PEM private keys
  'AKIA[0-9A-Z]{16}'                           # AWS access key id
  'ASIA[0-9A-Z]{16}'                           # AWS temporary access key id
  'gh[pousr]_[A-Za-z0-9]{36,}'                 # GitHub tokens (pat/oauth/user/server/refresh)
  'glpat-[A-Za-z0-9_-]{20,}'                   # GitLab personal access token
  'xox[baprs]-[A-Za-z0-9-]{10,}'               # Slack tokens
  'sk-[A-Za-z0-9]{32,}'                        # OpenAI-style secret keys
  'AIza[0-9A-Za-z_-]{35}'                      # Google API key
)

for pattern in "${PATTERNS[@]}"; do
  if printf '%s' "$CONTENT" | grep -qE "$pattern"; then
    echo "BLOCKED: the content you are about to write contains what looks like a secret (pattern: ${pattern}). Writing credentials into files is not allowed — use a secret store or an environment variable reference instead." >&2
    exit 2
  fi
done

exit 0
