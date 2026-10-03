# Repository acquisition and merged snapshots

A repository URL is not child permission to clone, install dependencies, execute
checkout code or create a graph. Children use the coordinator's existing snapshot
and report missing repositories/coverage.

Coordinator acquisition/merge requires explicit stage, network and resource budget
authority. Record each source URL, immutable revision, canonical root, tracked
changes and untracked policy, license/provenance review, input hashes and ownership.
Do not resolve an unpinned branch as though it were a qualified version. No build,
lifecycle script or provider invocation is implied by acquiring source bytes.

For separately admitted multi-root graphs, preserve per-node repository identity,
source locations and edge direction. Use disjoint intermediate/output locations;
do not let successive subfolder scans clobber another snapshot. Record component
hashes and merge command/version, exclusions and unresolved edges in the final
provenance. A cross-repository inferred edge is not a live caller observation.

A writer's local changes are not in the shared main graph just because both paths
refer to the same repository. Never retarget a running shared service to repair
that mismatch. Ask for separate provisioning or use current local source reads.
