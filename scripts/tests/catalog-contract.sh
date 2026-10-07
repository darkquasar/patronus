#!/usr/bin/env bash
# Catalog contract gate: structural and public-CLI checks against the REAL
# catalog source of this checkout.
#
# Usage: scripts/tests/catalog-contract.sh --out DIR --base REV
#
# Writes only below DIR:
#   DIR/bin/              patronus and catalog-check binaries
#   DIR/registry/         the public build output that this gate checked
#   DIR/source/           scratch copy of the checkout for public-CLI lock cases
#   DIR/home/<case>/      isolated HOME per case
#   DIR/closures/         <profile>--<target>.lock public lock output per case
#   DIR/logs/             per-step output
#
# The caller owns any resource lock and the Go environment. This script never
# acquires a lock, never fetches a registry URL (locks use --local-registry and
# the remote URL points at an unreachable local port), never writes the real
# HOME or the checkout, and never runs artifact programs, harnesses, native
# CLIs or SDKs.
set -euo pipefail

usage() { echo "usage: $0 --out DIR --base REV" >&2; exit 2; }
out='' base=''
while [ $# -gt 0 ]; do
  case $1 in
    --out) [ $# -ge 2 ] || usage; out=$2; shift 2;;
    --base) [ $# -ge 2 ] || usage; base=$2; shift 2;;
    *) usage;;
  esac
done
[ -n "$out" ] && [ -n "$base" ] || usage

root=$(cd "$(dirname "$0")/../.." && pwd)
mkdir -p "$out"
out=$(cd "$out" && pwd)
case "$out/" in "$root"/*) echo "catalog gate: --out must be outside the checkout ($root)" >&2; exit 2;; esac
[ -z "$(ls -A "$out")" ] || { echo "catalog gate: --out $out must be empty" >&2; exit 2; }
mkdir -p "$out/bin" "$out/home" "$out/closures" "$out/logs"

export PYTHONDONTWRITEBYTECODE=1
export PATRONUS_REGISTRY_URL=http://127.0.0.1:9/patronus-catalog-gate-never-fetches
step() { echo "== catalog gate: $*"; }
fail() { echo "catalog gate: FAIL: $*" >&2; exit 1; }
cd "$root"

# check-versions silently degrades on a missing base; refuse that here.
git rev-parse --verify --quiet "$base^{commit}" >/dev/null || fail "--base $base is not a commit in this clone (fetch full history)"

step "build public CLI"
go build -mod=readonly -o "$out/bin/patronus" ./cmd/patronus
patronus=$out/bin/patronus

step "catalog-check module: gofmt, vet, test, build (nested module, no go.work)"
[ ! -e go.work ] || fail "go.work must not fold the catalog tool into the root module"
(
  cd tools/catalog-check
  export GOWORK=off
  unformatted=$(gofmt -l .)
  [ -z "$unformatted" ] || { echo "$unformatted"; fail "tools/catalog-check is not gofmt-clean"; }
  if go list -mod=readonly -deps ./... | grep -q '^github.com/darkquasar/patronus\(/\|$\)'; then
    fail "tools/catalog-check imports the application module"
  fi
  go vet -mod=readonly ./...
  go test -mod=readonly ./...
  go build -mod=readonly -o "$out/bin/catalog-check" .
)
check=$out/bin/catalog-check

step "public build into $out/registry (base URL is recorded, never fetched)"
"$patronus" build --out "$out/registry"
index=$out/registry/catalog/index.json
[ -f "$index" ] || fail "build produced no $index"

step "public placeholder, gate-intent and version checks"
"$patronus" check-placeholders
"$patronus" check-gate-intent
"$patronus" check-versions --base "$base"

step "publication and package-delivery contract suites (offline fakes)"
bash scripts/tests/check-catalog-publication-test.sh
bash scripts/tests/publish-packages-test.sh

# Retain content/qualification fixture definitions, but do not run shipped prose
# inventories or artifact/native qualification in this structural release gate.
step "invented evidence-format contracts (no native, model or harness calls)"
python3 -m unittest discover -s scripts/tests -p 'test_pi_evidence.py'

step "catalog-check structure: source vs public index"
"$check" --source "$root" --index "$index"

step "enumerate admitted profile/target cases"
"$check" --source "$root" --index "$index" --profile-cases > "$out/profile-cases.json"
python3 - "$out/profile-cases.json" "$out/cases.tsv" "$out/lane-only.txt" <<'PY'
import json, re, sys
cases = json.load(open(sys.argv[1]))
name = re.compile(r'^[a-z0-9]+(-[a-z0-9]+)*$')
if not isinstance(cases, list) or not cases:
    sys.exit('catalog gate: FAIL: --profile-cases produced no cases')
targets = {}
for case in cases:
    if not isinstance(case, dict) or set(case) != {'profile', 'target'}:
        sys.exit('catalog gate: FAIL: malformed case %r' % (case,))
    p, t = case['profile'], case['target']
    if not (isinstance(p, str) and isinstance(t, str) and name.match(p) and name.match(t)):
        sys.exit('catalog gate: FAIL: unsafe case name %r' % (case,))
    if t in targets.setdefault(p, set()):
        sys.exit('catalog gate: FAIL: duplicate case %s/%s' % (p, t))
    targets[p].add(t)
with open(sys.argv[2], 'w') as f:
    for case in cases:
        f.write('%s\t%s\n' % (case['profile'], case['target']))
# Lane-only profiles (admitted only for codex and/or pi) must refuse claude.
with open(sys.argv[3], 'w') as f:
    for p in sorted(targets):
        if targets[p] <= {'codex', 'pi'}:
            f.write(p + '\n')
PY

step "scratch source copy for isolated public-CLI locks"
mkdir -p "$out/source"
tar -C "$root" --exclude=./.git -cf - . | tar -C "$out/source" -xf -
for dir in artifacts adapters recipes profiles packages docs/compatibility; do
  [ ! -e "$root/$dir" ] || diff -r "$root/$dir" "$out/source/$dir" >/dev/null || fail "scratch copy of $dir differs from source"
done

run_lock() { # profile target -> exit status; logs and project under the case name
  local p=$1 t=$2 c=$1--$2 home project
  home=$out/home/$c
  project=$out/source/.patronus-cases/$c
  mkdir -p "$home" "$project"
  (cd "$project" && env HOME="$home" XDG_CONFIG_HOME="$home/.config" XDG_CACHE_HOME="$home/.cache" \
     XDG_DATA_HOME="$home/.local/share" CODEX_HOME="$home/.codex" \
     "$patronus" lock --profile "$p" --target "$t" --local-registry \
     >"$out/logs/lock-$c.out" 2>"$out/logs/lock-$c.err")
}

step "public-CLI lock for every admitted case"
count=0
while IFS=$'\t' read -r p t; do
  c=$p--$t
  if ! run_lock "$p" "$t"; then
    cat "$out/logs/lock-$c.err" >&2; fail "lock $c refused"
  fi
  # A documented stub status notice is allowed; any other warning (an
  # unresolved item or a `without` that matched nothing) fails the case.
  if grep '^warning: ' "$out/logs/lock-$c.err" | grep -v '^warning: profile "[^"]*" is a stub: ' >&2; then
    fail "lock $c has unresolved profile warnings"
  fi
  [ -f "$out/source/.patronus-cases/$c/patronus.lock" ] || fail "lock $c wrote no patronus.lock"
  cp "$out/source/.patronus-cases/$c/patronus.lock" "$out/closures/$c.lock"
  count=$((count + 1))
done < "$out/cases.tsv"
[ "$count" -gt 0 ] || fail "no lock cases ran"
echo "catalog gate: $count admitted lock cases passed"

step "lane-only profiles refuse the claude target"
while read -r p; do
  [ -n "$p" ] || continue
  c=$p--claude
  if run_lock "$p" claude; then fail "lane-only profile $p accepted --target claude"; fi
  [ ! -e "$out/source/.patronus-cases/$c/patronus.lock" ] || fail "refused case $c wrote a lock"
  echo "refused as expected: $c ($(head -n 1 "$out/logs/lock-$c.err"))"
done < "$out/lane-only.txt"

step "catalog-check ledger coverage against public closure locks"
"$check" --source "$root" --index "$index" --closures "$out/closures"

echo "catalog gate: PASS (registry $out/registry)"
