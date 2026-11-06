# T3 Windows native validation

This is one Windows x64 staging job for the `feat/package-distribution` PR into
`darkquasar/patronus:main`. It runs on `pull_request`, so it does not need a
default-branch `workflow_dispatch` entry ([GitHub's dispatch registration rule](https://docs.github.com/en/actions/how-tos/manage-workflow-runs/manually-run-a-workflow)).
It has a read-only token, credential-free
checkouts, a 45-minute bound and no architecture matrix. Dependencies are pinned
in `package-windows-pins.json`; Python validation uses the accepted tap's hash lock.

```text
  +-----------------+  => render  +----------------------+  => validate  +------------------+
  | T3 source [tap] | ==========> | Two synthetic apps   | ===========> | WinGet native    |
  +-----------------+             +----------------------+              +------------------+
                                             |
                                  => loopback transport
                                             v
                                  +----------------------+  => results  +------------------+
                                  | Normal user [CI]     | ===========> | Evidence artifact|
                                  | Scoop + WinGet       |              +------------------+
                                  +----------------------+
```

The coordinator script prepares binaries and snapshots, checks canonical Scoop
and WinGet candidates with the actual T3 validators, and retains exact hashes.
Only the local-install copies replace download URLs with loopback URLs. Both sets
are preserved; the local copies are never admitted as publishable release identity.
The binaries report the two fixture versions; neither is a released app.

The runner administrator registers pinned WinGet and enables `LocalManifestFiles`
with the machine policy, so it reaches the separate test user's SID. A newly created account belongs only to Users and runs
the acceptance script through a limited scheduled task. The script verifies the
actual token is non-admin, lacks symlink privilege, and Developer Mode is off.
The disposable account receives the batch-logon right required to start its task;
existing assignees are preserved and an explicit denial is never removed. This is
CI host setup, not a package consumer prerequisite.
The hosted image's Developer Mode is explicitly disabled before starting that
account; the user phase verifies it remains off.
WinGet and Scoop installs then run under that identity. No policy-bypass flags are
used; Windows PowerShell's normal `RemoteSigned` policy is set explicitly and asserted
for the acceptance identity. The pinned Scoop source is
installed with the official bootstrap's setup functions, without its mutable-master
download routine. The harness calls the PowerShell entry directly: the upstream
CMD/shell launchers contain policy overrides and are never invoked.

Required observations: both apps discovered in one local bucket; x86 denied with
no installed app or shim; public local Git bucket registration and discovery; x64 Scoop
install and command shim; persisted User PATH and exact versions in a new process;
native `winget validate --manifest` for both canonical
and local manifest sets using schema 1.10.0; ordinary-user WinGet install and alias
resolution and x86 denial; the existing `install-checksum-test.ps1` regression in a
separate normal-policy process. The fresh process loads persisted User/Machine PATH
explicitly and verifies resolution under the expected manager directory. This proves
registry PATH resolution in a new process; it does not claim a new-logon observation.
WinGet may expose commands through Links or its matching per-user portable package
directory; either must resolve the frozen x64 binary hash and version. Scoop's pinned code
prints its unsupported-architecture error but exits zero; absence of installed
files and the specific diagnostic establish rejection, not exit status alone.

The job records runner image/version, OS, client/schema revisions, source and
candidate hashes, actual native logs and structured exit records. It retains WinGet
diagnostic logs, phase timing and task status, and stops an overrun task before upload.
Its artifact omits downloaded
bootstrap archives and installed runtime files. The disposable runner expires
normally; this procedure does not remove existing local users, VMs or worktrees.

ARM Windows execution, actual x86 hardware, live releases, default-source WinGet
availability, and composed upgrade/uninstall sentinel acceptance remain unrun.
The Scoop uninstall here only frees command names before the WinGet check and
does not claim the later C/P lifecycle acceptance. A raw-portable alias failure
returns to the owner; do not add lifecycle hooks, elevation or different release
assets to make the check pass.

## Activation and plan amendment

The owner approved the harness review, local commits, branch pushes and draft PR
activation. Completion of the review correction and native Windows evidence remains
required before T3 acceptance:

1. Make the exact accepted tap commit
   `212e2f8eb066561b4bd8a2b825c494145b43f53b` available in
   `darkquasar/homebrew-tap` on `distribution-control`. That repository is currently
   initially empty; the authorized initial push established `distribution-control` as its default branch.
   This exposes T3 tooling, not published formulae or packages.
2. Commit the reviewed harness in Patronus, push `feat/package-distribution`,
   and open a draft PR to `main`. The workflow checks the PR merge revision and
   the immutable tap commit. Retain run/attempt/artifact links and download the
   evidence to the durable T3 execution folder before artifact retention expires.
3. Inspect complete native results. Leave T3 open for failures or missing evidence;
   obtain the owner's milestone check before starting T4.

The canonical plan's T3 completion and host-coverage requirements remain in force.
The owner's Windows-job handoff supplements T3 ownership with these test files and
this PR-only workflow, and replaces T4's blanket statement “No Windows validation
job ... is added” only for this bounded T3 acceptance procedure. It adds no T4
publication workflow, recurring matrix or new credential boundary. The approved
plan/spec bytes are unchanged; accept this explicit supplement with the remote
activation decision. The earlier T3 Claude review/correction applies to the tap
bytes. The owner-approved, separate harness review completed after a same-session
usage-limit interruption. Its original findings are preserved; bounded corrections
are parent-verified and are not automatically independently re-reviewed. The newly
identified native WinGet schema-header issue received the owner's separate tap
correction approval. The renderer now emits each official schema header;
canonical-byte validation rejects missing or modified headers.

Testing strategy: `focused-postcheck`, because this change adds native test
procedures and workflow wiring without changing package or installer behavior.
Check Python fixture generation with real locked validators, inspect canonical
versus adapted outputs and denial results, parse both PowerShell scripts, and
validate workflow structure. No manufactured RED or test-of-test suite is added.
Native runtime acceptance requires the actual Windows job.
