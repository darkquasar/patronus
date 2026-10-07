# Codex guard delivery and pending native acceptance

Status: source behavior checked with invented payloads and a fake scanner. Native
Codex event shape, matcher vocabulary, TOML acceptance and trust are
**runtime-pending**, not established by these tests. SessionStart is Partial/N/A.
No Codex CLI/backend or real credentials are involved in source tests.

## Delivery contract

The existing shared hook adapter places one executable `hook.script` and merges
one owned matcher group into `hooks.PreToolUse` in `config.toml`. Each Codex guard
is a self-contained Python 3 file, not a wrapper around an undeployed sibling.
Global scripts go to `$CODEX_HOME/hooks/` (default `~/.codex/hooks/`); project
scripts go to `.codex/hooks/`. Global config follows CODEX_HOME. Missing script
or registration surfaces error rather than silently skipping selected scripts.
An unchanged reinstall may correctly SKIP already installed bytes.

```
  +------------------+  shared transform =>  +-------------------+
  | Codex hook source| ====================> | script + TOML      |
  +------------------+                       +-------------------+
                                                       | invented event =>
                                                       v
                                             +-------------------+
                                             | Python guard      |
                                             +-------------------+
```

This is the static matcher-group shape under test, not observed native evidence:

```toml
[[hooks.PreToolUse]]
matcher = "Write|Edit|MultiEdit|apply_patch"
patronusId = "<owned artifact identity>"
[[hooks.PreToolUse.hooks]]
type = "command"
command = '"<placed block-secrets-cx.py path>"'
```

`gitleaks-guard-cx` instead uses the `Bash` matcher and its own placed script.
Patronus does not write Codex trust internals. `core-profile-cx`1.1.0 selects
these source guards and its ledger retains Partial/runtime-pending outcomes.
Selection and emitted registration do not prove native guard activation.

## Invented input and failure contracts

`block-secrets-cx` accepts a JSON object with `tool_name` and `tool_input`:

- `Write`: string `content`; `Edit`: string `new_string`.
- `MultiEdit`: array `edits`, each with string `new_string`.
- `apply_patch`: freeform string `tool_input`, or an object containing exactly
  one string field `patch` or `input`. The native tool name takes precedence over
  a matcher alias such as `Write`.

The supported patch envelope is `*** Begin Patch` / `*** End Patch`, with
Add/Update/Delete File sections, optional Update Move to / End of File markers
and `@@` hunks. Only added `+` lines are scanned. Removed and unchanged context
lines are not newly written text. Malformed or unsupported matched inputs exit
2 with a diagnostic, rather than silently claiming protection. High-confidence
PEM/AWS/GitHub/GitLab/Slack/OpenAI-style/Google patterns block with exit 2;
benign content exits 0. Diagnostics do not print payload/token contents.

`gitleaks-guard-cx` accepts `tool_name: Bash` with string
`tool_input.command`. Optional top-level absolute `cwd` selects the event's
working directory; otherwise the script's cwd applies. Literal `git commit`
commands (including simple command sequences, `git -C <literal repo>`,
`git -c user.name=...`, `git -c user.email=...`, and literal `cd` sequences) scan
the current staged diff. Quoted text passed to `echo` is not a commit. Unsupported
repository overrides on recognized commits block with an actionable diagnostic.
It never executes the submitted shell command.

The scanner resolves `~/.patronus/bin/gitleaks` first, then PATH. The manifest
requires the `gitleaks` recipe; Python 3 and Git must also be available. Git diff
uses `--cached -U0 --no-ext-diff --no-textconv`. Gitleaks receives that diff on
stdin with `--exit-code 1 --redact`. Scanner exit 1 blocks a likely secret;
other failures, missing scanner/Git, unavailable staged diff and bounded
execution timeouts block with prerequisite/error diagnostics. Noncommits bypass
the scanner. These fail-closed rules are Codex-specific; the shared legacy
Claude/OpenCode hook bytes and behavior are unchanged.

## Threat limits and dispositions

Neither guard prevents reading and leaking an existing secret, recognizes every
secret, or constitutes a filesystem/egress sandbox. The commit guard scans the
index as it exists **before** the submitted command: it does not predict later
`git add`, `commit -a`, pathspec commits, hooks or other index changes. Shell
wrappers, aliases, variable expansion and dynamic/subshell commands are outside
the literal-command recognition contract. Use a repository pre-commit/CI scan
for broader commit enforcement. No native protection claim follows from these
source checks.

| Core outcome | Disposition in this stage |
|---|---|
| Secret write guard | Port: `block-secrets-cx`, invented behavior checked; native runtime-pending |
| Staged secret commit guard | Port: `gitleaks-guard-cx`, fake scanner checked; native runtime-pending |
| Language detection | Partial/N/A: no SessionStart vocabulary qualified here |
| Skill dispatch | Partial/N/A: no native activation hook qualified here |
| Heartbeat | N/A: no Codex-native SessionStart equivalent delivered here |
| Work-state regrounding | Partial/N/A: no native lifecycle hook qualified here |

## Later native trust procedure (not executed)

Requires a separate explicit runtime/native/auth/trust grant and a qualified
Codex version/platform. The existing CLI 401 stop remains in force. Do not retry
auth, change global trust/config, or infer readiness from a static TOML fixture.

1. In an expressly authorized disposable project, use an invented nonce such as
   `codex-hook-fixture-2026` and nonsecret content. Record CLI/version, scoped
   CODEX_HOME, exact config/script hashes and observed event bytes, redacted.
2. Through Codex's native, scoped consent flow only, authorize those exact script
   bytes. Record the observed trust outcome without copying internal trust data
   or credentials. Patronus must not synthesize trust entries.
3. Observe a benign edit, an invented-token write and a staged fake-scanner
   commit event. Verify matched payload shape, exit behavior and nonce effects;
   compare Write alias and native apply_patch. No invented fixture substitutes
   for this observation.
4. Change only invented script/nonce content under the granted test scope and
   observe whether native trust requires renewed consent. A missing event,
   stale trust or unsupported matcher keeps qualification pending.
5. Record SessionStart vocabulary separately or retain Partial/N/A. Stop on
   auth/trust failure; cleanup needs its own exact scope grant.

## Source checks

Historical (pre-decoupling) Go cases executed actual placed scripts or adapter
transforms. The executing cases are retained behind the `artifactbehavior` build
tag and are NOT RUN in the current application or catalog gates; inert-script
delivery tests cover placement and registration:
`TestCodexHookPlacesScriptAndRegistration`,
`TestCodexScriptHookCannotBeSkippedSilently`,
`TestCodexBlockSecretsApplyPatchPayload`, and
`TestCodexGitleaksGuardBashEvent`. Additional tests cover both guard artifacts in
both scopes and ordinary shared-installer placement/reinstall of the write guard.
Scratch Git fixtures and fake scanner processes were the only runtime inputs.
These results are not current native acceptance. Full observed logs and hashes are retained in the writer's run-bound handoff.
