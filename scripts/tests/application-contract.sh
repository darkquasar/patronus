#!/usr/bin/env bash
# Application contract gate: formatting, vet, optional lint and the race test
# suite of the root Go module. With --without-catalog the checks run in a
# scratch copy of the checkout that has NO artifacts/ and NO profiles/ tree, so
# a passing run proves application tests use invented catalogs only.
#
# Usage: scripts/tests/application-contract.sh --out DIR [--without-catalog]
#
# The caller owns any resource lock and the Go environment (GOPROXY, GOFLAGS,
# GOTOOLCHAIN). Local runs set GOPROXY=off; CI may download pinned modules.
# This script never acquires a lock and never runs artifact programs,
# harnesses, native CLIs or SDKs.
set -euo pipefail

usage() { echo "usage: $0 --out DIR [--without-catalog]" >&2; exit 2; }
out='' without_catalog=false
while [ $# -gt 0 ]; do
  case $1 in
    --out) [ $# -ge 2 ] || usage; out=$2; shift 2;;
    --without-catalog) without_catalog=true; shift;;
    *) usage;;
  esac
done
[ -n "$out" ] || usage

root=$(cd "$(dirname "$0")/../.." && pwd)
mkdir -p "$out"
out=$(cd "$out" && pwd)
case "$out/" in "$root"/*) echo "application gate: --out must be outside the checkout ($root)" >&2; exit 2;; esac
[ -z "$(ls -A "$out")" ] || { echo "application gate: --out $out must be empty" >&2; exit 2; }

export PYTHONDONTWRITEBYTECODE=1

subject=$root
if [ "$without_catalog" = true ]; then
  subject=$out/source
  mkdir -p "$subject"
  tar -C "$root" --exclude=./.git --exclude=./artifacts --exclude=./profiles -cf - . | tar -C "$subject" -xf -
  if [ -e "$subject/artifacts" ] || [ -e "$subject/profiles" ]; then
    echo "application gate: scratch copy still contains a production catalog tree" >&2; exit 1
  fi
  echo "application gate: scratch source without artifacts/ and profiles/: $subject"
fi
cd "$subject"

step() { echo "== application gate: $*"; }

step gofmt
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
  echo "These files are not gofmt-clean:"; echo "$unformatted"; exit 1
fi

step "go vet ./..."
go vet -mod=readonly ./...

if command -v golangci-lint >/dev/null 2>&1; then
  step "golangci-lint run"
  golangci-lint run
else
  echo "SKIP golangci-lint: not on PATH (CI runs it in the lint job)"
fi

# The default build excludes the artifactbehavior tag, so retained guard and
# supervisor fixture executions are NOT RUN here.
step "go test -race ./..."
go test -mod=readonly -race -p "${GO_TEST_P:-2}" ./...

echo "application gate: PASS ($subject)"
