#!/bin/sh
# worker-precheck.sh <base-ref> [--global]
#
# Exit 0 only when the complete OpenCode worker working set is present:
#   1. agents/patronus-writer-oc.md            (frontmatter mode: subagent)
#   2. plugins/patronus-hook-runner.ts
#   3. patronus/hooks.json                     (patronusId second segment block-secrets-oc)
#      and                                     (patronusId second segment gitleaks-guard-oc)
#   4. patronus/hooks/block-secrets-oc.py      (executable)
#   5. patronus/hooks/gitleaks-guard-oc.py     (executable)
#   6. opencode.json                           (permission.edit carries "*", "*.env", ".env.*")
#
# Without --global the identities must be tracked at <base-ref> in the current
# repository under .opencode/ (and opencode.json at the root): a fresh attempt
# worktree sees tracked files only. With --global they must exist under
# ${OPENCODE_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/opencode}.
# Every missing identity is named on stderr. POSIX sh; python3 for JSON.
set -eu

usage() {
  echo "usage: $0 <base-ref> [--global]" >&2
  exit 2
}

[ $# -ge 1 ] && [ $# -le 2 ] || usage
base=$1
mode=tracked
if [ $# -eq 2 ]; then
  [ "$2" = "--global" ] || usage
  mode=global
fi
[ -n "$base" ] || usage

missing=0
miss() {
  echo "worker-precheck: missing: $1" >&2
  missing=$((missing + 1))
}

# content_of <identity-path> : prints the bytes of the identity or fails.
# is_exec <identity-path>    : succeeds when the identity is executable.
if [ "$mode" = global ]; then
  R=${OPENCODE_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/opencode}
  agent="$R/agents/patronus-writer-oc.md"
  runner="$R/plugins/patronus-hook-runner.ts"
  hooks="$R/patronus/hooks.json"
  guard1="$R/patronus/hooks/block-secrets-oc.py"
  guard2="$R/patronus/hooks/gitleaks-guard-oc.py"
  config="$R/opencode.json"
  present() { [ -f "$1" ]; }
  content_of() { cat "$1"; }
  is_exec() { [ -x "$1" ]; }
  where="global root $R"
else
  git rev-parse --verify --quiet "$base^{commit}" >/dev/null 2>&1 || {
    echo "worker-precheck: $base is not a commit in this repository" >&2
    exit 2
  }
  agent=".opencode/agents/patronus-writer-oc.md"
  runner=".opencode/plugins/patronus-hook-runner.ts"
  hooks=".opencode/patronus/hooks.json"
  guard1=".opencode/patronus/hooks/block-secrets-oc.py"
  guard2=".opencode/patronus/hooks/gitleaks-guard-oc.py"
  config="opencode.json"
  listing=$(git ls-tree -r --name-only "$base")
  present() { printf '%s\n' "$listing" | grep -qxF -- "$1"; }
  content_of() { git show "$base:$1"; }
  is_exec() { [ "$(git ls-tree "$base" -- "$1" | cut -d' ' -f1)" = "100755" ]; }
  where="tracked at $base"
fi

# 1. agent with mode: subagent
if present "$agent"; then
  content_of "$agent" | python3 -c '
import sys
text = sys.stdin.read().replace("\r\n", "\n")
ok = False
if text.startswith("---\n"):
    end = text.find("\n---", 4)
    block = text[4:end] if end >= 0 else ""
    for line in block.splitlines():
        key, sep, rest = line.partition(":")
        if sep and key.strip() == "mode" and rest.strip().strip("\"'"'"'") == "subagent":
            ok = True
sys.exit(0 if ok else 1)
' || miss "$agent frontmatter mode: subagent"
else
  miss "$agent"
fi

# 2. hook runner plugin
present "$runner" || miss "$runner"

# 3. hooks.json registering both guards (second colon-separated patronusId segment)
if present "$hooks"; then
  content_of "$hooks" | python3 -c '
import json, sys
try:
    doc = json.load(sys.stdin)
except ValueError:
    sys.exit(3)
found = set()
def walk(node):
    if isinstance(node, dict):
        pid = node.get("patronusId")
        if isinstance(pid, str):
            parts = pid.split(":")
            if len(parts) >= 2:
                found.add(parts[1])
        for value in node.values():
            walk(value)
    elif isinstance(node, list):
        for value in node:
            walk(value)
walk(doc)
want = sys.argv[1:]
missing = [w for w in want if w not in found]
for w in missing:
    print(w)
sys.exit(1 if missing else 0)
' block-secrets-oc gitleaks-guard-oc > "${TMPDIR:-/tmp}/worker-precheck.$$" 2>/dev/null || {
    status=$?
    if [ "$status" -eq 3 ]; then
      miss "$hooks is not valid JSON"
    elif [ "$status" -eq 1 ] && [ -s "${TMPDIR:-/tmp}/worker-precheck.$$" ]; then
      while IFS= read -r guard || [ -n "$guard" ]; do
        miss "$hooks patronusId entry for $guard"
      done < "${TMPDIR:-/tmp}/worker-precheck.$$"
    else
      miss "$hooks validator failed"
    fi
  }
  rm -f "${TMPDIR:-/tmp}/worker-precheck.$$"
else
  miss "$hooks"
fi

# 4./5. executable guard scripts
for guard in "$guard1" "$guard2"; do
  if present "$guard"; then
    is_exec "$guard" || miss "$guard executable bit"
  else
    miss "$guard"
  fi
done

# 6. opencode.json permission.edit keys
if present "$config"; then
  content_of "$config" | python3 -c '
import json, sys
try:
    doc = json.load(sys.stdin)
except ValueError:
    sys.exit(3)
permission = doc.get("permission") if isinstance(doc, dict) else None
edit = permission.get("edit") if isinstance(permission, dict) else None
if not isinstance(edit, dict):
    print("permission.edit object")
    sys.exit(1)
missing = [k for k in ("*", "*.env", ".env.*") if k not in edit]
for k in missing:
    print("permission.edit key %s" % k)
sys.exit(1 if missing else 0)
' > "${TMPDIR:-/tmp}/worker-precheck.$$" 2>/dev/null || {
    status=$?
    if [ "$status" -eq 3 ]; then
      miss "$config is not valid JSON"
    elif [ "$status" -eq 1 ] && [ -s "${TMPDIR:-/tmp}/worker-precheck.$$" ]; then
      while IFS= read -r item || [ -n "$item" ]; do
        miss "$config $item"
      done < "${TMPDIR:-/tmp}/worker-precheck.$$"
    else
      miss "$config validator failed"
    fi
  }
  rm -f "${TMPDIR:-/tmp}/worker-precheck.$$"
else
  miss "$config"
fi

if [ "$missing" -ne 0 ]; then
  echo "worker-precheck: FAIL: $missing missing identit(y/ies) $where" >&2
  exit 1
fi
echo "worker-precheck: OK: complete worker working set $where"
exit 0
