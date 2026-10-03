# Ingestion and watchers

Neither URL ingestion nor watching belongs to child query work. Do not fetch a
URL into the corpus, start a background watcher, or auto-update after a fetch.
Report a missing source to the coordinator and continue with bounded source
reads when allowed.

Coordinator ingestion requires explicit stage, network and resource budgets:
record origin, exact bytes/hash, content type, author/capture provenance, reviewed
input location, sensitive-data exclusions, and allowed follow-on processing.
Media/transcription and semantic providers need their own admitted budgets; a URL
is not permission for arbitrary redirects, downloads or model calls.

A watcher requires a separate lifecycle grant and one owner, measured memory,
concurrency/deadline limits, known process identities and a settlement procedure.
Debouncing file changes is not a cost or process bound. Code-only changes and
non-code changes can trigger different runtime behavior at different pins; verify
it rather than assuming a deterministic watcher. Unknown descendants mean retain
and escalate, not broad process killing. No watcher is installed by this profile.
