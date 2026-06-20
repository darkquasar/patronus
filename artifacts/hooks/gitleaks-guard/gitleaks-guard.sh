#!/bin/bash
# gitleaks-guard — a PreToolUse Bash hook that scans the STAGED diff for secrets
# at COMMIT time only. gitleaks is a commit-/repo-state scanner, so it is wired at
# its design altitude (a commit guard) rather than per-edit: the hook fires on
# every Bash command, but this script does real work only when the command is a
# `git commit`. See the threat model in block-secrets — gitleaks covers case (c),
# committing a secret, which is its core competency.
#
# Requires the `gitleaks` binary on PATH (installed by the `gitleaks` recipe into
# ~/.patronus/bin). If gitleaks is not found, the guard fails OPEN (exit 0) with a
# warning rather than blocking every commit — a missing scanner must not wedge the
# workflow.

INPUT=$(cat)
COMMAND=$(printf '%s' "$INPUT" | jq -r '.tool_input.command')

# Only act on git commits; let every other Bash command through untouched.
if ! printf '%s' "$COMMAND" | grep -qE '\bgit\b.*\bcommit\b'; then
  exit 0
fi

if ! command -v gitleaks >/dev/null 2>&1; then
  echo "gitleaks-guard: gitleaks binary not found on PATH; skipping secret scan (install the gitleaks recipe, ensure ~/.patronus/bin is on PATH)." >&2
  exit 0
fi

# Scan only what is about to be committed: the staged diff. --exit-code 1 makes
# gitleaks return non-zero on a finding, which we surface as a block (exit 2).
if ! git diff --cached -U0 | gitleaks stdin --no-banner --exit-code 1 >/dev/null 2>&1; then
  echo "BLOCKED: gitleaks detected a likely secret in the staged changes. Unstage or remove it before committing (run 'git diff --cached | gitleaks stdin -v' to see the finding)." >&2
  exit 2
fi

exit 0
