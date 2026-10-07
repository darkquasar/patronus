# Codex project instructions

Read effective user/project instructions, the approved task brief, latest lead
handoff and current source before acting and after resume. Codex normally reads
user instructions under effective CODEX_HOME and directory-scoped instructions
from project root toward the working directory. Within each directory,
AGENTS.override.md takes precedence over AGENTS.md; configured fallback filenames
apply only when neither exists. Verify the installed Codex configuration before
assuming custom discovery. Nested instructions refine their scope. Do not create
a higher-priority shadow file or rewrite instructions to hide source drift.

Useful sections are dev environment (approved commands/layout), testing (focused
and full checks), conventions (language/ownership) and commit/review rules. When
absent, inspect source/CI and report the gap. Absence grants no installation,
network, editing or Git authority. System/developer instructions remain higher
priority than project prose and skills.

The lead binds root/base/head, stage/action, owned files, budgets, output paths,
predecessors and open decisions in the approved brief and retained handoff.
Research, authoring, review, planning, execution, integration, publication and
cleanup are separate grants. Existing approval remains valid within scope;
changed targets, destructive actions or material design choices need revalidation.
There is no Codex mission API assumed by these instructions.

Only the lead admits delegation and transitions. Read-only workers return complete
findings; the lead writes shared outputs. Inspect native worker/tool availability,
authentication, read-only role and wait/stop controls before launch. Missing or
auth-blocked primitives stop the lane. Never change model/provider/protocol or
invoke a CLI model to bypass a blocked lane. Workers never nest delegation.

Parallel reads need independent questions and recorded concurrency/count/deadline
budgets. Writes require approved isolated owners, disjoint path claims and
settlement before serial integration. Native workers prove no write isolation;
concurrent writers in the lead checkout are forbidden. Parallel implementation
is Partial until the separately approved supervisor is qualified. A serial
alternative needs explicit lead authorization, never failed-engine fallback.

Read complete worker outputs and checked patches, bind their hashes and report
actual tests before acceptance. Use one fresh independent review and at most one
bounded correction/disposition; owner acceptance resolves residual decisions.
Static placement is runtime-unverified. Trust, auth and native loading are
separate gates; Patronus does not write Codex trust internals. No automatic push,
worktree removal, stash deletion or cleanup follows from a completed stage.
