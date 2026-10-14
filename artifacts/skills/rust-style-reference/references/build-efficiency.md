# Rust compilation under parallel development

Read when concurrent writers/builds, expensive test processes or cache reuse are
actually in scope. Ordinary single-task work does not need a resource ceremony.

## Two distinct limits

Keep independent readers/writers moving while bounding heavy operations separately.
The coordinator owns machine-wide admission using an existing project mechanism.
Workers follow its allocation, not a private semaphore invented for each lane.
Include Cargo compilation/tests, linkers, coordinator integration checks and
benchmarks. Internal parallelism multiplies outer invocations: cargo -j or
CARGO_BUILD_JOBS controls compilation, not every test harness or service fixture.
Also account for nextest/test-thread settings and resource-heavy child processes.

For example, six writers may read/edit while two heavy slots admit Cargo runs
with three compiler jobs each. Six potential compile jobs are a budgeting input,
not a guarantee of memory use. Slot/job counts are illustrative: select from
measured aggregate RAM/CPU/disk and project constraints, including linkers. Avoid
fixed universal lane counts. Benchmarks needing stable resources run in a reserved
window. Existing TDD slices retain their intended red/green, subject to admission.

If no shared resource mechanism exists, agree a simple coordinator-owned schedule
or serialize heavy checks under the existing budget. Do not build a scheduler,
new ledger or permission pipeline just to run Cargo. Independent edits need not
wait while a different lane owns a build slot.

## Cache reuse without false freshness

Reuse supported dependency/download/compiler caches or a validated project
baseline adapter. Keep mutable per-worktree outputs isolated, commonly with a
per-lane CARGO_TARGET_DIR; one shared mutable target can serialize work through
Cargo locks and is not writer isolation. Do not clean another lane's target.

A prepared baseline requires matching toolchain/target, lockfile resolution,
features/profile, compiler/linker flags and relevant build-script/environment
inputs, plus a quiescent source. Keep the baseline revision in the existing
handoff. Warm artifacts accelerate checks; they do not prove current code passes.
Do not manually hardlink Cargo workspace artifacts as a universal shortcut.
Artifact hardlink reuse needs a validated project-specific adapter; it is not a
general Cargo guarantee. Use a cold isolated target/check if freshness is unclear,
without deleting shared caches or silently installing cache tooling.

## Scope and evidence

During a task, check affected packages and their consumers with the project's
authorized commands and relevant features, negative cases and legacy regressions.
Treat fmt/clippy/test success as different claims. Manifest changes may warrant
existing dependency/license checks; nextest, cargo-deny and alternate toolchains
are not prerequisites unless the project requires them. Report missing required
tools instead of silently substituting or installing them.

Reuse prior evidence only while checked bytes, command/options, relevant
dependencies/environment, acceptance scope and readable logs/exits still match.
Integration changes those inputs: run the required aggregate checks on the
combined result, including relevant feature matrices, CLI behavior or multiprocess
cases. Finishing consumes that result when final inputs still match, not another
identical full suite automatically. A compile error is not an intended behavioral
TDD red. Do not weaken project review or safety gates to increase throughput.

Capture commands/results once in the existing report. Useful observations are
setup/queue time, aggregate memory including linkers, disk growth and accepted
outcomes, not compiler count alone. No separate compilation-attestation ledger.
