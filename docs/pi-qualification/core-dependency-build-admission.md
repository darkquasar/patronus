# Core dependency build admission

> **Historical / superseded.** This document records the former receipt/static-payload design. It is retained as provenance, not current package installation, update, removal or rollback guidance. See `docs/pi-delivery.md` for the Pi-owned npm lifecycle.

Dated 2026-10-01. QP-02 / pat-g4zf. **Procedure, not an acquisition, script,
staging or operational grant.** The candidate dossiers are incomplete. This
procedure precedes CP-05: it must not wait for CP-03 or QP-03 to authorize research
about the closure that those tasks will consume.

Required core consists of the external Pi host plus Patronus-delivered tk,
pi-subagents and pi-web-access. The latter two require separate self-contained
payloads; web is required by owner amendment E-03, not an optional overlay.
Optional MCP adapter, Serena and Graphify remain operator-provisioned. No new
installer, service controller or runtime permission boundary is introduced.

## 1. Coordinator admission before acquisition (C evidence)

CP-05 requests a concrete resource/credential/publication plan from the
coordinator **before effects**. The owner-approved acquisition record must name:

- Exact upstream identities/versions/immutable refs from the candidate records;
  registry metadata, source, archive, checksum and attestation URLs; allowed
  HTTPS origins and redirect destinations; retrieval time and byte limits.
  Candidate URLs are leads, not a grant to fetch them. Initially request the
  selected npm tarballs from registry.npmjs.org and pinned source from
  github.com/raw.githubusercontent.com for nicobailon/pi-subagents and
  nicobailon/pi-web-access; retain the existing wedow/ticket commit/script pin.
  Every newly discovered transitive archive/source origin needs explicit inclusion
  in a bounded grant before retrieval. No latest-tag resolution as a substitute
  for the selected pins, no unreviewed VCS dependency/submodule acquisition.
- Disposable absolute source/download/build/output/evidence roots, allowed
  writes, selected platform, pinned Node/npm/build tools and container image
  digest, exact planned argv, bounded network and owner of cleanup. HOME,
  PI_CODING_AGENT_DIR and package caches must be explicit fixture paths; no
  ambient npm tree, live harness root, host home or project auth/config mutation.
- Separate acquisition and build/script policy. Deny lifecycle scripts by
  default, including transitive scripts. Record effective npm configuration and
  reviewed script-suppression argv (for an approved npm acquisition, explicitly
  use --ignore-scripts rather than relying on Pi). Do not execute downloaded
  entrypoints to inspect them. Lock resolution and extraction are acquisition
  effects, not deployment. Pi npm/git installation does **not** promise
  --ignore-scripts. Never use the legacy `npx pi-subagents` installer.
- Memory admission: MemFree and MemAvailable each >=750 MiB with >=500 MB
  reserve; refuse unknown/insufficient headroom. All Go builds/tests and Docker
  runs acquire `/home/agent/.local/state/patronus-code-intel/pi-implementation-test.lock`.
  Go uses GOTOOLCHAIN=local, GOPROXY=off, GOSUMDB=off, GOMAXPROCS=2 and -p 1.
  Containers are nonprivileged, bounded CPU/RAM/PIDs, no management socket or
  whole-home mount, read-only inputs and only assigned writable roots.
- Credential-free acquisition first. Any necessary secret is separately approved
  by the coordinator with exact source/destination/command, supplied only as an
  ephemeral secret file, never printed, committed or baked into images. Existing
  account access is not authority to spend quota or publish. Record expiry or
  revalidation triggers, owner verification and interruption/settlement policy.

Stop for an ungranted source, script, peer repair, changed pin or resource failure;
retain partial evidence. No implicit npm/uv/git repair, upgrade, protocol/model
fallback or connect-time package resolution is permitted.

## 2. Quarantined acquisition and Q closure review (C evidence)

CP-05 records exact downloaded bytes, registry integrity and recomputed SHA-256,
immutable source commit, retrieval URLs/timestamps and extraction inventory.
Registry integrity is not publisher authentication. Record what source-to-package
mapping was actually established; the web published entrypoint was historically
`./dist`, not the repository's `./index.ts`. Reverify against the acquired bytes.

Before **build execution or claiming closure admission**, the Q owner reviews:

1. A full locked runtime tree (including optional/platform/peer decisions), every
   archive digest, source-lock, SBOM and dependency edges; licenses/notices and
   redistribution obligations for all included files. The old SDK spike lock is
   historical evidence of one combined install, not an approved production lock.
2. Root and transitive install/lifecycle/prepare/build/publish scripts, native
   bindings, generated code and subprocess/network behavior. Root script absence
   is insufficient. If necessary, a script exception needs a new owner grant
   naming exact package/version/digest/script/argv, reason, filesystem/network
   limits, expected outputs and review. Run only in the approved isolated build;
   no exception permits scripts during static Patronus deployment.
3. Provenance subject matching the acquired digest, signature verification,
   trusted signing identity, builder/workflow/ref and source linkage. Record
   absent/unverifiable attestations as unknown, not clean. SLSA advertisement and
   npm signatures alone are not verified provenance. No attestation from an
   unrelated version can satisfy this review.
4. Reachable advisory review against the **locked** closure, dated databases and
   source/release-note analysis, including intervening security/compatibility
   fixes since the selected pin. A failed/incomplete audit, policy 404 or zero
   matches cannot establish safety. Obtain a separately authorized bounded
   currency refresh; never upgrade while collecting evidence.
5. Upstream tests/CI tied to the selected commit and exact local build tools;
   platform-specific dependency resolution and the actual installed-byte
   comparison required later. Historical published-file equality is narrower
   than complete installed-tree equality and is not a fresh audit.

The reviewer signs an exact input-hash disposition: accept **for this build**,
conditional/defer or reject. Unknown high-impact closure/scripts/licenses block
build admission; provenance/advisory gaps must remain explicit and block their
qualified operational claims. If required evidence cannot be obtained, defer the
surface and escalate rather than fabricate a clean value. Acquiring unknown
inputs for review does not preapprove them. QP-02 records today's gaps; CP-05
obtains the post-acquisition Q review through the parent before proceeding.

## 3. Reproducible C construction (CP-05; separately granted build)

Use only the reviewed locks and acquired roots. Never copy the ambient installed
tree. Bundle the complete exact runtime closure of each extension, but exclude
Pi host peers (pi-agent-core/pi-ai/pi-coding-agent/pi-tui/typebox as applicable),
including compatibility aliases. Record how peers resolve from the selected
external host; broad ranges are not compatibility proof. Subagents' observed
undici 8.10.0 needs Node >=22.19; test the selected Node 22.22.1 rather than infer
that every Node or architecture works.

Keep **recipe version, Patronus payload version and upstream version/ref**
separate in package.yaml, source-lock and evidence. Use the existing deterministic
package builder/decoder, outer reserved Patronus inventory, nested `pi/` runtime
root, normalized regular-file paths/modes and exact declared member inventory.
Reject symlinks, traversal, undeclared files and collisions. Include NOTICE,
LICENSES.md, source-lock.json and SBOM.json. Current default archive ceilings are
32 MiB compressed, 128 MiB expanded, 16 MiB per file and 4096 entries; read the
current decoder before construction and do not silently raise limits. The old
subagents inventory estimate (~1692 files/~16 MB) is feasibility only.

Build twice from independently clean approved roots with identical input hashes,
pinned tools and normalized environment. Capture argv, times, tool/image hashes,
logs, exits, member inventories and both archive SHA-256 values; compare exact
bytes, decode each archive and verify inventory/hashes/modes. Investigate any
mismatch. No assumed reproducibility from a successful build. Tests that import
third-party payloads run only under the separate deployment qualification grant,
not ordinary application regression suites. Payload-owned build checks may
validate the acquired closure without creating per-catalog-item Go tests.

## 4. Controlled HTTPS staging and real-pin acceptance

Staging is a **publication effect**, even if described as temporary. CP-05 must
stop for explicit coordinator confirmation of owner authorization for the release
operator, exact asset bytes/digests, destination account/origin/path, access
policy, immutable retention and safe failure/cleanup policy. Local build approval
is insufficient. No placeholder production URL/hash, mutable `latest` object,
public asset upload or release command under this document alone.

After that separate grant, the release operator stages the exact reviewed bytes
without rebuilding. Fetch the real immutable HTTPS endpoint under the approved
verification grant, recompute its digest and decode its content; record redirect
chain, final URL, archive identity and hash. C alone edits recipes to the actual
verified URL/digest. Recheck recipe/payload/source-lock/SBOM/license consistency
and independent review of final hashes. An unavailable endpoint or mismatch
blocks real-pin acceptance; retain the failed observation, never substitute a
fixture URL. Integrity proves correspondence to the selected bytes, not trust.

QP-03 then qualifies **those staged bytes**, static placement and explicit native
local-path registration/loading at the approved effective roots. QP-04 controls
release promotion, final endpoint verification and precise support claims; no
publication or operational approval is inherited from this authoring task.
Changed URL, payload, closure, source, host, environment or config invalidates
applicable evidence and requires fresh hash-bound review/qualification.

## 5. Runtime, ownership, update and rollback gates

QP-03 must record exact Linux ARM64 environment, Node 22.22.1, external Pi 0.87.1,
provider/config hashes and all peers. For selected code-intel it additionally
requires Go 1.26.0/gopls 0.20.0 and exact Python interpreter **patch versions**,
executable hashes, per-service virtualenv/lock/SBOM (including extras/native
wheels and language servers). A path containing python3.14 is not an exact Python
environment. Other platforms/versions remain unqualified.

Historically, global tk files and receipt-owned subagents/web payloads were the explicit static
ownership exceptions (P-08 plus E-03). Pi host, optional packages/services,
caches, sessions, missions, outputs, logs, graph caches, worktrees and credentials
remain operator-owned. Patronus does not own arbitrary homes or native package
inventories. Local removal preserves all globals. Before any global mutation,
obtain all-consumer inventory/quiescence evidence; there is no profile refcount
or automatic discovery of manual consumers.

Before extension update/remove/rollback: settle detached work and descendants,
retain outputs, preview effective Pi inventory/config and duplicate npm/local
sources, verify receipts and drift, and manually deregister the exact
formerly receipt-derived native local source under a separate activation/config grant.
Unknown consumers, ownership, drift or process state means preserve/escalate.
Replace/remove only owned payloads, never native cache/config directories.
Register the replacement explicitly only after preflight succeeds and reload or
restart consumers; existing sessions may retain old code. Test read-only payloads
with all caches/logs/output outside them; any required in-tree write fails.

Retain old/new locks, source diffs, digests, archives, config priors, review and
qualification records. Rollback is a separately approved owned-payload operation
plus manual registration/config restoration and fresh loading checks, not an
activation transaction journal. A later failure does not imply earlier static
changes were rolled back; record partial effects and settlement before cleanup.
External Pi update and package update are distinct: at the inspected host,
`pi update` updates Pi, while `pi update --extensions` reconciles packages. Do
not run either automatically. Retention/deletion of external data requires its
owner's explicit sensitivity/archive/expiry/deletion policy.

For required web, explicitly apply the reviewed inert template in a clean sandbox
before activation: DDG-only, eager tools, maxInlineContentChars 8000, workflow
none, raw fetch, no summary/background/paid fallback and verified disabled
browser/cookie/clone/media surfaces; sanitize both cookie-enabling environment
variables. Test all four tools and long/paginated/error responses. The 8000 value
is a presentation setting, not a network-byte, aggregate-output, call, cost or
egress limit. Missing stronger enforcement (pat-n6za) blocks
web-bounded-unattended, not an independently evidenced functional core claim.

## Validation layers and stop conditions

- **M:** existing invented checker/provisioning vectors establish record consistency
  and refusal, not actual npm script denial or installer behavior. D owns invented
  mechanism tests; do not modify siblings to make real content pass.
- **C:** candidate structure/digests, generic catalog validation, package closure
  checks and human source/license/provenance review. These can complete while
  operational qualification is withheld.
- **I:** separately authorized QP-03 real-resource deployment/activation/lifecycle
  observations. Neither fixture success nor upstream CI replaces them.

Run `python3 scripts/check-pi-evidence.py record --root . --file <candidate>`
for each record, then the existing offline `test_pi_*.py` suites. Requested
operational claims must still refuse these incomplete candidates. The checker
checks declared hashes/consistency, not grant authenticity, log truth, security
or publication authority. Owner approval and independent review remain necessary.
