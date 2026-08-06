#!/usr/bin/env bash
# Smoke-drive the patronus CLI against a throwaway HOME.
#
# Patronus's defining behaviour is writing config files into a user's real
# ~/.claude, ~/.codex, and ~/.config/opencode. Driving it therefore needs a
# sandbox, not just a --dry-run: the interesting bugs live in the deploy path.
#
# Home is resolved through toolpath.HomeDir(os.LookupEnv), which prefers $HOME,
# so exporting HOME (plus the three per-tool overrides) redirects every write
# into SANDBOX. Nothing here touches the invoking user's real config.
#
# Usage:
#   .claude/skills/run-patronus/driver.sh            # full smoke: plan→deploy→scan→remove
#   .claude/skills/run-patronus/driver.sh build      # build the binary only
#   .claude/skills/run-patronus/driver.sh plan       # dry-run plan, writes nothing
#   .claude/skills/run-patronus/driver.sh profile    # deploy the whole core profile
#   .claude/skills/run-patronus/driver.sh guards     # the CI guards (gofmt/vet/placeholders/gate-intent)
#   .claude/skills/run-patronus/driver.sh shell      # print exports, then drop into the sandbox
set -uo pipefail

# The repo is found from the CALLER's cwd, not from this script's location: once
# installed as a skill this file lives under .claude/skills/, .agents/skills/, or
# .opencode/skills/ depending on the agent, and at global scope it is outside the
# checkout entirely. Walk up for the checkout markers instead.
find_repo() {
  local d="${PATRONUS_REPO:-$PWD}"
  while [ "$d" != "/" ]; do
    [ -d "$d/artifacts" ] && [ -d "$d/adapters" ] && [ -f "$d/go.mod" ] && { printf '%s' "$d"; return 0; }
    d="$(dirname "$d")"
  done
  return 1
}
REPO="$(find_repo)" || { echo "error: run this from inside the patronus checkout (or set PATRONUS_REPO)" >&2; exit 1; }
SANDBOX="$REPO/.patronus/smoke"   # under the gitignored .patronus/
BIN="$SANDBOX/patronus"
ITEM="${PATRONUS_SMOKE_ITEM:-go-style-uber}"

# Captured BEFORE any sandboxing, so the containment check has a real baseline.
REAL_HOME="$HOME"
REAL_CLAUDE_MTIME=$(stat -f %m "$REAL_HOME/.claude" 2>/dev/null || stat -c %Y "$REAL_HOME/.claude" 2>/dev/null || echo "")

pass=0; fail=0
# The `|| true` matters: `pass=$((pass+1))` evaluates to 0 on the first call and
# would make ok() return 1, which then leaks out as the whole script's exit code.
ok()   { printf '  \033[32mok\033[0m   %s\n' "$1"; pass=$((pass+1)) || true; }
bad()  { printf '  \033[31mFAIL\033[0m %s\n' "$1"; fail=$((fail+1)) || true; }
step() { printf '\n\033[1m== %s\033[0m\n' "$1"; }

# Fresh sandbox home. The project dir MUST live inside the repo: --local-registry
# walks up from cwd looking for artifacts/+adapters/, so a sandbox in /tmp fails
# with "not inside a patronus repo".
reset_sandbox() {
  rm -rf "$SANDBOX/home" "$SANDBOX/proj"
  mkdir -p "$SANDBOX/home" "$SANDBOX/proj"
}

# Every write-side env var patronus reads. Callers run this in a subshell.
sandbox_env() {
  export HOME="$SANDBOX/home"
  export XDG_CONFIG_HOME="$SANDBOX/home/.config"
  export CODEX_HOME="$SANDBOX/home/.codex"
  export OPENCODE_CONFIG_DIR="$SANDBOX/home/.config/opencode"
}

do_build() {
  step "build"
  mkdir -p "$SANDBOX"
  if go build -o "$BIN" ./cmd/patronus 2>&1; then ok "go build ./cmd/patronus"; else bad "go build"; return 1; fi
  "$BIN" --help >/dev/null 2>&1 && ok "binary runs" || bad "binary does not run"
}

do_plan() {
  step "plan (dry run — writes nothing)"
  reset_sandbox
  local out
  out=$( cd "$SANDBOX/proj" && ( sandbox_env; "$BIN" install "$ITEM" --target claude --global --local-registry 2>&1 ) )
  grep -q "dry run — no files were written" <<<"$out" && ok "declares itself a dry run" || bad "no dry-run banner"
  [ -z "$(find "$SANDBOX/home" -type f 2>/dev/null)" ] && ok "wrote nothing" || bad "dry run wrote files"
}

do_deploy() {
  step "deploy → scan → remove (sandboxed HOME)"
  reset_sandbox
  ( cd "$SANDBOX/proj" && sandbox_env
    "$BIN" install "$ITEM" --target claude --global --deploy --yes --local-registry 2>&1 | tail -3 ) >/dev/null
  local skill="$SANDBOX/home/.claude/skills/$ITEM/SKILL.md"
  [ -f "$skill" ] && ok "deployed $ITEM into sandbox home" || bad "artifact not written"
  [ -f "$SANDBOX/home/.patronus/state.json" ] && ok "recorded install state" || bad "no state.json"

  # Containment proof: the invoking user's real ~/.claude must not have been
  # touched by the deploy above. Compare its mtime across the write.
  if [ -n "${REAL_CLAUDE_MTIME:-}" ]; then
    local now; now=$(stat -f %m "$REAL_HOME/.claude" 2>/dev/null || stat -c %Y "$REAL_HOME/.claude" 2>/dev/null)
    [ "$now" = "$REAL_CLAUDE_MTIME" ] && ok "real ~/.claude untouched (mtime held)" || bad "ESCAPED THE SANDBOX: real ~/.claude changed"
  fi

  # scan takes NO --local-registry (unlike install): it always resolves the
  # remote catalog, so a locally-built item it has never published shows up as
  # ORPHANED-STATE. That verdict is expected here and is not a failure.
  local scanout
  scanout=$( cd "$SANDBOX/proj" && ( sandbox_env; "$BIN" scan 2>&1 ) )
  grep -q "Home:.*$SANDBOX/home" <<<"$scanout" && ok "scan resolves the sandbox home" || bad "scan read the wrong home"
  grep -q "$SANDBOX/home/.claude" <<<"$scanout" && ok "scan detects the sandbox install" || bad "scan missed the install"

  # remove works from tracked state alone: it takes NEITHER --yes NOR --local-registry.
  ( cd "$SANDBOX/proj" && sandbox_env; "$BIN" remove "$ITEM" --deploy 2>&1 | tail -3 ) >/dev/null
  [ ! -f "$skill" ] && ok "remove --deploy deleted the artifact" || bad "artifact survived remove"
}

do_profile() {
  step "profile core (multi-item deploy, no network)"
  reset_sandbox
  local out n
  out=$( cd "$SANDBOX/proj" && ( sandbox_env; "$BIN" install --profile core --target claude --global --deploy --yes --local-registry 2>&1 ) )
  n=$(find "$SANDBOX/home" -type f 2>/dev/null | wc -l | tr -d ' ')
  [ "$n" -gt 50 ] && ok "wrote $n files" || bad "expected >50 files, got $n"
  # Package installs are ADVISORY unless --allow-package-installs: never auto-run.
  grep -q "ADVISORY (run yourself)" <<<"$out" && ok "npm install left advisory" || bad "no advisory line"
}

do_guards() {
  step "CI guards"
  [ -z "$(gofmt -l . 2>/dev/null)" ] && ok "gofmt clean" || bad "gofmt: $(gofmt -l . | tr '\n' ' ')"
  go vet ./... >/dev/null 2>&1 && ok "go vet" || bad "go vet"
  go run ./cmd/patronus check-placeholders >/dev/null 2>&1 && ok "check-placeholders" || bad "check-placeholders"
  go run ./cmd/patronus check-gate-intent  >/dev/null 2>&1 && ok "check-gate-intent"  || bad "check-gate-intent"
}

do_shell() {
  reset_sandbox; do_build >/dev/null
  cat <<EOF
Sandbox ready. Paste these, then run the binary at $BIN:

  cd $SANDBOX/proj
  export HOME="$SANDBOX/home" \\
         XDG_CONFIG_HOME="$SANDBOX/home/.config" \\
         CODEX_HOME="$SANDBOX/home/.codex" \\
         OPENCODE_CONFIG_DIR="$SANDBOX/home/.config/opencode"
  $BIN install $ITEM --target claude --global --deploy --yes --local-registry
  find \$HOME -type f
EOF
}

cd "$REPO" || exit 1
case "${1:-all}" in
  build)   do_build ;;
  plan)    do_build >/dev/null && do_plan ;;
  deploy)  do_build >/dev/null && do_deploy ;;
  profile) do_build >/dev/null && do_profile ;;
  guards)  do_guards ;;
  shell)   do_shell; exit 0 ;;
  all)     do_build && do_plan && do_deploy && do_profile; do_guards ;;
  *)       echo "usage: $0 [all|build|plan|deploy|profile|guards|shell]" >&2; exit 2 ;;
esac

printf '\n\033[1m%d passed, %d failed\033[0m\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
