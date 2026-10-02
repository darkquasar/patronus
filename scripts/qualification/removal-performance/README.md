# Removal performance and preservation contract

These tests use real `Service.Remove`, `Service.Recover`, receipt/journal storage,
file unlink, fsync, and caller-held package locks. No persistence override or
no-op sync is installed. They depend on public service behavior, not journal
fields. Linux/macOS only; the runner's affinity and instrumentation require Linux.

## Invented workload (not a web/full-profile install)

* N = 100, 1000, 4014 **distinct regular files**, each 128 bytes, mode 0644.
  Payload totals: 12,800 / 128,000 / 513,792 bytes.
* One package root and ceil(N/100) immediate directories, at most 100 files each.
  Names are `d000/f000000`, etc.; data and SHA-256 are invented, not upstream pins.
* Setup directly creates/syncs files and writes a validated synthetic receipt.
  It does not call Replace, download, decode an archive, install a real package,
  repair discovery, or run a CLI. Archive digest is synthetic metadata.
* Setup, lock acquisition/release, postconditions, discovery acknowledgement and
  fixture cleanup are outside the benchmark timer and allocation measurement.
  Real Remove includes its initial Recover, scans, hash checks and persistence.
* Recovery-only starts at an interrupted first unlink intent, prepared outside the
  timer using a returned fault error. Interrupted-total includes that Remove plus
  a fresh Service.Recover, but **not** OS process launch or operator delay.
  Neither restart timing is a SIGKILL timing; independent tests exercise SIGKILL.
* `ns/op`, `B/op`, `allocs/op` are Go benchmark measurements. `user-ns/op` and
  `sys-ns/op` are process getrusage deltas around the operation (including minor
  benchmark timer bookkeeping), not whole `go test` process measurements.

## Same commands on baseline and candidate

Use an isolated checkout with these harness files. Do not benchmark a shared HOME.
Run from repository root. All heavy invocations go through the owner wrapper;
`run.sh` does **not** invoke/nest it. It refuses to overwrite an output directory.
Python is the preinstalled stdlib-only JSON parser; there are no new dependencies.

```sh
E=/home/agent/workspace/patronus-removal-optimization-evidence
H=$E/run-heavy.sh
# Set these uniquely per checkout/run; the integrator owns E/integrate.
OUT=$E/integrate/candidate-overlay-1
FIXTURES=$E/integrate
bash "$H" 600 bash scripts/qualification/removal-performance/run.sh \
  "$OUT" "$FIXTURES" 0 '^BenchmarkRemovalRealPersistence/N(100|1000|4014)$'

# Baseline N4014: ONE <=600s invocation only, not repeated multi-minute loops.
# Candidate: bounded repeat, e.g. candidate-overlay-1, -2, -3 with same command.
# Recovery has its own production timeout: do NOT run baseline recovery N4014
# after the 184s Remove establishes that it would exceed the baseline 60s budget.
bash "$H" 180 bash scripts/qualification/removal-performance/run.sh \
  "$E/integrate/candidate-restart" "$FIXTURES" 0 \
  '^BenchmarkRemoval(Recovery|InterruptedTotal)RealPersistence/N(100|1000)$'

bash "$H" 300 go test -p 1 -count=1 -parallel=1 \
  ./internal/packagestate ./internal/packagedelivery
bash "$H" 300 go vet -p 1 ./internal/packagedelivery

# Identical binary flags/toolchain for comparison; no additional ldflags.
bash "$H" 300 env CGO_ENABLED=0 go build -p 1 -trimpath -buildvcs=false \
  -o "$E/integrate/patronus-candidate" ./cmd/patronus
stat -c '%s' "$E/integrate/patronus-candidate"
sha256sum "$E/integrate/patronus-candidate"
```

The runner uses `taskset -c 0`, `GOMAXPROCS=1`, `-cpu=1`, `-parallel=1`, `-p 1`,
`-count=1`, `-benchtime=1x`, and `-timeout=590s`. Keep the CPU and fixture filesystem
identical when comparing revisions. Its private `TMPDIR` contains fixtures;
private executable `GOTMPDIR` under OUTPUT_DIR contains Go build intermediates.
Both are removed on normal exit. Metadata/environment, exact command, raw
`go-test.jsonl`, stderr, status and parsed `timings.json` remain. No cache flushing.
Compilation and whole test wall time in JSON are **not** removal timings.

### Filesystem constraint on this VM

The recorded VM is linux/arm64, Go 1.26.0, 16KiB pages, two available CPUs.
`/dev/shm` is 64MiB/noexec. 4014 tiny files alone occupy ~62.72MiB of page-backed
storage; baseline journals and temporary replacements cannot safely fit alongside.
An initial run failed **before test execution** because Go put its executable in
noexec TMPDIR. The GOTMPDIR correction was approved, and the owner approved the
full N100/1000/4014 comparison on private E/bench **overlay**, without remounting,
using secrets mounts, hardlinks, empty files, or changing semantics.

Optional tmpfs100/1000: change only TEMP_PARENT to `/dev/shm` and keep OUTPUT_DIR
on executable overlay. Check free capacity first. Tmpfs4014's <=5s objective is
**untested**, not passed or failed. Overlay is a separately labelled filesystem
case, not a dedicated physical-device benchmark and not proof of power durability.
If integration provisions an already-approved larger disposable tmpfs, record it
as a new series; do not compare it directly to the overlay baseline.

## Diagnostics separate from headline timing

```sh
bash "$H" 180 bash scripts/qualification/removal-performance/run.sh \
  "$E/integrate/candidate-profile-1000" "$FIXTURES" 0 \
  '^BenchmarkRemovalRealPersistence/N1000$' profile
bash "$H" 120 go tool pprof -top \
  "$E/integrate/candidate-profile-1000/packagedelivery.test" \
  "$E/integrate/candidate-profile-1000/cpu.pprof"
# alloc.pprof: use go tool pprof -top -alloc_space with the same test binary.
# Profiles cover the whole test, INCLUDING fixture setup; not operation-only CPU.

bash "$H" 180 env PATRONUS_REMOVAL_IO=1 TMPDIR="$FIXTURES" GOMAXPROCS=1 \
  taskset -c 0 go test -p 1 -count=1 -parallel=1 -cpu=1 \
  -run '^TestRemovalPerformanceLogicalIO$' -json ./internal/packagedelivery
bash "$H" 60 python3 scripts/qualification/removal-performance/test_summarize.py

# Optional actual old-reader gate; reuse a prebuilt baseline, never acquire one.
bash "$H" 180 env PATRONUS_REMOVAL_LEGACY_BINARY="$E/bench/patronus-baseline" \
  TMPDIR="$FIXTURES" go test -p 1 -count=1 -parallel=1 \
  -run '^TestRemovalContractLegacyBinaryRefusesPending$' -v ./internal/packagedelivery
```

The opt-in Linux I/O test uses `/proc/self/io`: `wchar` is logical bytes written by
this process, `syscw` write syscalls. No payload writes are requested inside Remove,
so this is useful metadata-growth evidence, but it is **not** physical device bytes,
atomic-replacement counts or fsync counts. No production instrumentation is used. The candidate diagnostic includes N4014;
the original baseline diagnostic measured only N100/1000.
A near-linear write count alone is insufficient; also compare byte and allocation
growth and inspect profiles. No hard wall-clock budget is imposed on normal tests.

## Independent restart/preservation assertions

`TestRemovalContract*` covers edits plus unknown siblings, force limited to owned
paths, cancellation before work and during pending work, retry, and actual child
SIGKILL at public service fault boundaries. Every child has a 20s parent deadline,
is killed if necessary, and is waited/reaped. Exit status must identify SIGKILL;
existing `os.Exit(97)` tests are not relabelled. Fresh parent Service/lock acquisition
checks restart and lock release. SIGKILL is process interruption, not a power cut.

The second `after-unlink-intent` occurrence proves the first path completed; both
identical and changed recreations must survive recovery and force. A post-force
pending edit must block recovery without deleting later pending paths. The core restart tests do not
read Removed/PendingRemove/cursor/manifest fields or fabricate transaction JSON.
The opt-in legacy-binary test additionally launches the prebuilt baseline CLI with
an isolated HOME, no inherited tool overrides, and a 10s deadline. It verifies that
`remove --global --force --deploy` refuses the new marker, preserves every payload
file and the journal bytes, and that current recovery then succeeds.

Integration tasks beyond these stable seams:

* Preserve semantic `after-unlink-intent`, `after-unlink` (after parent sync), and
  `after-removal-commit` service fault points; the first is also used by benchmarks.
* Manifest-publication-before/after, the gap between unlink and parent sync,
  checkpoint-write boundaries, receipt-publication gaps and discovery-clear failure
  need protocol/storage fault tests. Baseline has no stable public callbacks for
  all of these; this lane does not add production hooks or infer boundaries by
  polling a transient JSON format.
* Corrupt/truncated/out-of-range/mismatched progress, old-reader rejection, old
  journal replay, ENOSPC/sync injection and hostile type/symlink substitution remain
  protocol/integration coverage responsibilities. This suite does not claim them.

## Baseline observed (unmodified production bb89d47)

CPU0, overlay, one operation each; no profile in headline figures:

| N | Remove seconds | user seconds | system seconds | allocated bytes | allocations |
|---:|---:|---:|---:|---:|---:|
|100|0.268370416|0.091909|0.043703|52,996,248|172,114|
|1000|10.062930671|8.163847|0.534763|4,931,091,768|10,528,288|
|4014|183.971242296|166.044259|5.633599|85,781,921,488|159,612,894|

Restart-only100/1000: 0.243053626 / 9.995243213 seconds.
Interrupted-total100/1000: 0.255615958 / 9.851197296 seconds (different samples,
not a subtraction/overhead estimate). Tmpfs100/1000: 0.307991750 / 10.023605796s.

Logical write bytes100/1000: 11,261,612 / 1,083,285,027 (~96.2x for 10x files);
write syscalls402 /4007. Separate profile1000: transaction writing84.35% cumulative
CPU, JSON MarshalIndent42.06%, transaction validation37.62% (overlapping stacks,
not additive). Removal accounts for97.34% of that whole-test profile. This supports
metadata rewriting/revalidation as major costs, rather than a model-only assertion.
Single baseline samples and shared VM scheduling limit statistical claims.

Static baseline binary: 14,577,584 bytes, CGO_ENABLED=0, `-p 1 -trimpath
-buildvcs=false`, Go1.26.0 linux/arm64, SHA-256
`fb15409f13d3e16defb4780f3c46e40f3b4daebd0bca6f2619c655bade262999`.
Candidate growth >3MiB is an owner review trigger. All raw evidence is under
`/home/agent/workspace/patronus-removal-optimization-evidence/bench`.

## Integrated candidate observed (2026-10-03)

The combined protocol and harness were measured on the same CPU0/GOMAXPROCS1
private overlay filesystem and unchanged invented workload. Three independent
one-operation runs, no profiling in these figures:

| N | candidate seconds (three runs) | median seconds | baseline / median |
|---:|---|---:|---:|
|100|0.086532208 / 0.075459124 / 0.087118000|0.086532208|3.10x|
|1000|0.743148334 / 0.776716250 / 0.781062625|0.776716250|12.96x|
|4014|3.048896044 / 3.188796834 / 3.103506668|3.103506668|59.28x|

The large workload achieves seconds-scale removal and exceeds the 10x goal;
N100 does **not** achieve 10x. Tmpfs4014 remains untested. These small shared-VM
samples are observations, not universal latency guarantees.

Candidate restart-only N100/1000/4014: 0.080624042 / 0.770554459 / 3.077009710s.
Interrupted-total: 0.082800958 / 0.773659625 / 3.107122752s. No baseline4014
recovery was repeated. Candidate logical write bytes: 139,552 / 1,376,486 /
5,516,143; write syscalls: 205 / 2,005 / 8,032. This is approximately linear
metadata growth (9.86x bytes for 10x files), unlike the baseline's ~96.2x.
Candidate removal allocations are approximately 4.46 / 44.9 / 181.1 MB per
operation, also near-linear instead of baseline 53 / 4,931 / 85,782 MB.

The separate N1000 candidate profile has only 250ms of sampled CPU, includes
fixture setup, and shows 64% flat syscall time. Serialization/validation no longer
dominates the way it did at baseline. The implementation retains both durable
per-file checkpoints, ancestry/type/hash checks and all syncs; no additional
safety tradeoff was made to chase the N100 ratio.

Candidate static binary: 14,655,143 bytes, **+77,559 bytes (+0.532%)**, identical
build flags/toolchain and dependency metadata; SHA-256
`cd09e7158a5be61ad11f695a5de3819a332ea7f2a881d81256ba60fe4143427f`.
Integrated state/delivery/SIGKILL/scan/CLI tests, full repository tests, vet, parser
regressions and actual baseline-binary refusal passed. Race validation remains
unavailable in the existing CGO-disabled environment. Raw commands, timings,
profiles, binary and source provenance are in
`/home/agent/workspace/patronus-removal-optimization-evidence/integrate`.

No actual installed web payload is measured here. A future isolated-copy case must
copy read-only original receipt plus tree into its own `/state/home`, rebind the
receipt's canonical Root through validated storage, verify contents/modes, and use
that independent namespace only. Original manifests/grants must not be reused as
instructions. That is a separate synthetic-install-state/real-bytes case, not a
full-profile deployment claim; no Docker framework is introduced here.
