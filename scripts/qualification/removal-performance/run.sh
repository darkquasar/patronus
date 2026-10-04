#!/usr/bin/env bash
# Invoke ONLY through the owner's run-heavy.sh; do not nest that wrapper.
set -euo pipefail
out=${1:?usage: run.sh OUTPUT_DIR TEMP_PARENT CPU BENCH_REGEX [timing|profile]}
temp_parent=${2:?}
cpu=${3:?}
bench=${4:?}
mode=${5:-timing}
[[ "$mode" == timing || "$mode" == profile ]] || exit 64
[[ "$out" == /* && "$temp_parent" == /* ]] || { echo 'use absolute paths' >&2; exit 64; }
mkdir "$out" # Refuse to overwrite evidence.
work=$(mktemp -d "$temp_parent/patronus-removal.XXXXXXXX")
mkdir "$out/build-temp"
trap 'rm -rf -- "$work" "$out/build-temp"' EXIT
# /dev/shm may be noexec: only fixture data belongs there, never Go executables.
export TMPDIR="$work" GOTMPDIR="$out/build-temp" GOMAXPROCS=1
{
  printf 'source_head='; git rev-parse HEAD
  printf 'source_status:\n'; git status --short
  printf 'go='; go version
  printf 'cpu=%s\nmode=%s\nfixture_parent=%s\nTMPDIR=%s\nGOTMPDIR=%s\n' "$cpu" "$mode" "$temp_parent" "$TMPDIR" "$GOTMPDIR"
  printf 'page_size='; getconf PAGESIZE
  printf 'GOTOOLCHAIN=%s GOPROXY=%s GOSUMDB=%s GOMAXPROCS=%s GOMEMLIMIT=%s\n' "${GOTOOLCHAIN:-}" "${GOPROXY:-}" "${GOSUMDB:-}" "$GOMAXPROCS" "${GOMEMLIMIT:-}"
  uname -a
  stat -f -c 'filesystem=%T block_size=%S' "$work"
  df -Pk "$work"
  grep -m 1 'model name' /proc/cpuinfo || true
  grep Cpus_allowed_list /proc/self/status
} > "$out/environment.txt"
args=(go test -p 1 -count=1 -parallel=1 -cpu=1 -run '^$' -bench "$bench" -benchtime=1x -benchmem -timeout=590s -json)
if [[ "$mode" == profile ]]; then
  args+=(-cpuprofile "$out/cpu.pprof" -memprofile "$out/alloc.pprof" -o "$out/packagedelivery.test")
fi
args+=(./internal/packagedelivery)
printf '%q ' taskset -c "$cpu" "${args[@]}" > "$out/command.txt"
printf '\n' >> "$out/command.txt"
set +e
taskset -c "$cpu" "${args[@]}" > "$out/go-test.jsonl" 2> "$out/stderr.log"
status=$?
set -e
printf '%s\n' "$status" > "$out/exit-status.txt"
python3 scripts/qualification/removal-performance/summarize.py "$out/go-test.jsonl" > "$out/timings.json"
cat "$out/timings.json"
exit "$status"
