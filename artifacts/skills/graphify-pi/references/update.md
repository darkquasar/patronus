# Freshness and updates

Children inspect provenance; only the coordinator may refresh a snapshot under
explicit indexing-stage and resource/provider budget authority. A changed HEAD,
tracked diff, relevant untracked file, missing file or newly excluded document
requires a coverage warning, not a child rebuild. HEAD timestamps and tracked-file
status are heuristics: external edits, shell-mediated writes and untracked inputs
can escape them. Compare the recorded included inventory and hashes as well.

Label a divergent graph historical and source-check the affected region. Missing
coverage cannot prove code absence. Preserve the existing snapshot and partial
failure evidence; do not delete stale flags or overwrite it with an empty result.

For a separately admitted coordinator update, record the prior graph hash/root,
revision and dirty state, exact changed/deleted/excluded inputs, command/mode/tool
and model versions, environment policy, limits and output location before effects.
Deliberately select qualified code-only/provider-free extraction when appropriate;
non-code semantic extraction or clustering can entail other processes/providers.
No mode or provider substitution is implicit when the requested path fails.

An incremental merge must preserve direction and source identity, distinguish
re-extracted from deleted files, record failed chunks, and not stamp failed inputs
as processed. Compare new/old inventory, warnings and hashes before publishing the
new provenance. Cluster-only also writes outputs and needs the same admission;
it is not a query or a cheap child repair. Do not restart clients/services without
accounting for dependent work and explicit lifecycle authority.
