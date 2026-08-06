---
name: run-patronus
description: Build, run, and smoke-test the Patronus CLI — install/scan/remove/profile against a sandboxed HOME, plus the CI guards. Use when asked to run patronus, build the binary, test the installer, deploy a profile, check what install writes to disk, or reproduce CI locally.
---

# Run Patronus

Patronus is a Go CLI that installs artifacts, recipes, and profiles onto Claude Code,
Codex, and OpenCode. Its defining behaviour is **writing config files into a real
`~/.claude`, `~/.codex`, and `~/.config/opencode`**, which is what makes it awkward to
drive: the interesting code path is the one that mutates your actual environment.

The driver solves that by redirecting `HOME`. **Run it from the repo root** — it locates
the checkout itself, and `--local-registry` resolves from the working directory.

```bash
bash {skillDir}/driver.sh          # full smoke: build→plan→deploy→scan→remove→profile→guards
```

Exits 0 on success, 1 if any check failed, 2 on a bad subcommand.

## Run (agent path)

`driver.sh` below is shorthand for `bash {skillDir}/driver.sh`. **Invoke it via `bash`,
not directly:** skill sidecar files install as `0644`, so `{skillDir}/driver.sh` alone
fails with permission denied. (Only hook scripts get `0755` —
`internal/adapter/hook.go:244`; the skill adapter has no equivalent.) Running the copy
in the checkout directly is fine — that one keeps its exec bit.

| Command | What it does |
|---|---|
| `driver.sh` / `driver.sh all` | Everything below, in order. ~40s. |
| `driver.sh build` | `go build -o .patronus/smoke/patronus ./cmd/patronus` |
| `driver.sh plan` | Dry-run install; asserts **nothing** is written |
| `driver.sh deploy` | install → scan → remove round-trip in the sandbox |
| `driver.sh profile` | Deploys the whole `core` profile (77 files, no network) |
| `driver.sh guards` | gofmt, `go vet`, check-placeholders, check-gate-intent |
| `driver.sh shell` | Prints the exports, then leaves a ready sandbox for manual poking |

Override the test artifact with `PATRONUS_SMOKE_ITEM=<name> driver.sh deploy`.

Expected output of a clean run:

```
16 passed, 0 failed
```

### How the sandbox works

Not an OS sandbox — **environment redirection**. `internal/toolpath/resolver.go:38`
resolves home by preferring `$HOME`, and every write path reaches it via
`toolpath.HomeDir(os.LookupEnv)` (`cmd/patronus/install.go:100,246`). So exporting four
vars redirects every write into `.patronus/smoke/home`:

```bash
export HOME="$SANDBOX/home" \
       XDG_CONFIG_HOME="$SANDBOX/home/.config" \
       CODEX_HOME="$SANDBOX/home/.codex" \
       OPENCODE_CONFIG_DIR="$SANDBOX/home/.config/opencode"
```

That set is lifted from the project's own `cmd/patronus/integration_test.go:96-108`,
which sandboxes identically with `t.Setenv`. The driver asserts containment by
comparing the real `~/.claude` mtime across the deploy.

**Limits, since this is redirection and not a jail:** a hardcoded absolute path in a
future recipe would bypass it, and `--allow-package-installs` would run a real global
`npm install`. The driver never passes that flag and instead asserts the advisory line
appears.

## Manual invocation

```bash
bash {skillDir}/driver.sh shell   # prints exports + a ready sandbox
```

Then, from `.patronus/smoke/proj` with those exports set:

```bash
$SANDBOX/patronus install go-style-uber --target claude --global --deploy --yes --local-registry
$SANDBOX/patronus scan
$SANDBOX/patronus remove go-style-uber --deploy
find "$HOME" -type f
```

## Run (human path)

`go build -o /tmp/patronus ./cmd/patronus`, then `/tmp/patronus install <item> --target claude`.
Safe without `--deploy` — install is a dry run by default. **With `--deploy` and no
sandbox it writes to your real config.** Prefer the driver.

## Test

```bash
go test -race ./...              # what CI runs; ~13s in cmd/patronus
go test ./internal/plan/ -run TestCompute -v
```

## Gotchas

Things that cost me a cycle each and that you cannot guess from `--help`:

- **`--local-registry` resolves from cwd, not the repo.** It walks up looking for
  `artifacts/`+`adapters/`, so a sandbox project dir in `/tmp` dies with
  `not inside a patronus repo`. The sandbox project **must** live inside the checkout —
  hence `.patronus/smoke/proj`. (`.patronus/` is already gitignored.)
- **The flags are not uniform across subcommands.** `install` takes `--yes` and
  `--local-registry`; **`remove` takes neither** (it works from tracked state alone) and
  **`scan` takes neither** either. Passing them is a hard `unknown flag` error.
- **`scan` always resolves the REMOTE catalog.** A locally-built item it has never
  published is reported as `ORPHANED-STATE`. That verdict is expected in the sandbox and
  is not a failure; there is no flag to point `scan` at the local checkout.
- **`--target` is required** for anything wiring to a runtime. Omitting it errors.
- **Scope changes the delivery shape.** `go-style-uber` at `--local` is an `APPEND` into
  `./CLAUDE.md`; the same artifact at `--global` is a `CREATE` of
  `~/.claude/skills/go-style-uber/SKILL.md`. Assert against the right one.
- **Package installs are advisory.** A `core` deploy prints
  `ADVISORY (run yourself): npm install -g ccusage` and reports `1 skipped` rather than
  running it. Only `--allow-package-installs` changes that.
- **Skill sidecar files lose the exec bit.** `files:` entries install `0644`, so an
  installed `driver.sh` cannot be executed directly. Hooks avoid this because
  `internal/adapter/hook.go:244` hardcodes `Mode: 0o755`; there is no skill equivalent.
  Invoke through `bash <path>`.
- **Don't read `$?` after a pipe.** `go vet ./... | head; echo $?` reports *head's*
  status and shows a green vet on a broken tree. This bit me twice; the driver captures
  status directly.

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `Error: --local-registry: not inside a patronus repo` | cwd is outside the checkout. Run from `.patronus/smoke/proj`. |
| `Error: unknown flag: --yes` on remove | `remove` has no `--yes`. Drop it. |
| `Error: unknown flag: --local-registry` on remove/scan | Neither accepts it. Drop it. |
| `permission denied: .../driver.sh` | Installed sidecars are `0644`. Run `bash {skillDir}/driver.sh`. |
| `error: run this from inside the patronus checkout` | The driver walks up from cwd for `artifacts/`+`adapters/`+`go.mod`. `cd` into the checkout or set `PATRONUS_REPO=/path/to/patronus`. |
| `vet: ... imported and not used` | A stray import (an editor auto-import can add one). `git checkout <file>`, re-run `driver.sh guards`. |
| `ORPHANED-STATE` in sandbox scan | Expected — `scan` diffs against the remote catalog. Not a failure. |
| Driver reports `0 failed` but exits 1 | Fixed: `pass=$((pass+1))` returns 1 on the first call. The helpers end in `|| true`. |
