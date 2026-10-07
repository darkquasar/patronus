# Coordinator-owned shared services

These commands use only the executables delivered by the profile. They are an
operator runbook, not install hooks. The coordinator needs separate authority for
process launch, graph construction, logs and shutdown.

## SS-1: bind one root and snapshot

Record the canonical repository root, full revision, tracked diff hash, untracked
input policy and one protected runtime directory. Reject symlink aliases and a
root that differs from the selected Pi project. Record the expected endpoints:

- Serena: `http://127.0.0.1:9121/mcp`
- Graphify: `http://127.0.0.1:9122/mcp`

Measure resource headroom under FM-3 before starting Serena/LSP or building a
graph. A graph snapshot must have a completed provenance record before it can be
served as current evidence.

## SS-2: verify delivered executables

The published argv is bound to the immutable files and Git blob IDs in the
[source command contract](runtime-command-contract.json). After approved package
deployment, confirm that contract against the installed tools and record the
actual outputs of:

```sh
pi list
serena --version
serena start-mcp-server --help
graphify --version
graphify-mcp --help
```

The selected declarations must resolve to `pi-mcp-adapter@3.0.0`, Serena commit
`7a2968335f2198b966864de1ce3655c8e485a653`, and
`graphifyy[mcp]==0.9.31`. Both Graphify console scripts are required. Source
inspection and a version string do not prove installed bytes or platform
compatibility. Do not repair a mismatch by running `uvx`, `npx` or an unpinned
installer.

## SS-3: build Graphify once when separately authorized

From the selected root, run the reviewed Graphify build command and capture its
complete argv, cwd, mode, provider use, inputs, warnings, exit and output hashes.
A typical local extraction is:

```sh
cd /ABSOLUTE/APPROVED/PROJECT
graphify .
```

Do not copy this example without reviewing the selected package's help and
provider policy. Building may invoke workers or providers. It needs its own cost,
network and resource grant. Children never build or refresh the graph.

After the build settles, hash `graphify-out/graph.json`, copy that exact file into
the protected runtime directory under its digest, make the protected digest-named
copy read-only, and hash it again. Record the generated path and the served path in snapshot
provenance. The service must never read the mutable build output directly. A
subsequent build creates a new candidate snapshot and cannot mutate the file used
by a running service.

## SS-4: start exactly one process per service

Choose coordinator-owned log and process records outside the repository. Refuse
startup when either endpoint already has an unknown listener. Start Serena with an
explicit project and non-editing planning mode:

```sh
serena start-mcp-server \
  --transport streamable-http \
  --host 127.0.0.1 \
  --port 9121 \
  --project /ABSOLUTE/APPROVED/PROJECT \
  --context ide \
  --mode planning \
  --enable-web-dashboard false \
  --open-web-dashboard false
```

Start Graphify against the approved immutable snapshot:

```sh
graphify-mcp /ABSOLUTE/PROTECTED/CODE-INTEL-RUNTIME/graphify/<graph-sha256>.json \
  --transport http \
  --host 127.0.0.1 \
  --port 9122 \
  --path /mcp \
  --stateless
```

Run both under the coordinator's process supervisor so process identity, start
time, logs, descendants, exit and restart policy are recorded. Do not background
them with an untracked shell command. Loopback is local trust, not tenant
isolation. Do not change to a wildcard bind or remote address without a separate
authentication and network design.

## SS-5: reload clients and prove identity

Reload Pi after package and config changes. Use the adapter gateway to initialize
each exact shared entry, inspect its live tool schemas and complete the readiness
procedure. Serena must report the selected canonical root. Graphify responses must
bind to the recorded graph hash and revision. A second listener, unexpected root,
prototype alias or different graph blocks MCP dispatch.

The coordinator may then launch core roles with `async: true`. The strict `mcp`
allowlist makes background execution mandatory because only detached children load
ambient Pi extensions. A foreground launch fails before its first model turn; it
must not be treated as a source-read fallback. Each successfully started child
receives only the `mcp` gateway and installed code-intelligence guidance, queries
the shared HTTP entries, and receives no lifecycle authority or server command.

## SS-6: stop only after all consumers settle

Inventory and settle every parent, child and external client. Closing one HTTP
client does not stop either service. Ask the surviving client to perform a query
before shutdown when shared-service survival is part of qualification.

Stop through the recorded supervisor identity, not a guessed PID or port match.
Verify Serena, its language-server descendants and Graphify have exited. Preserve
logs, descriptor and snapshot provenance. Unknown clients or descendants retain
the processes and block cleanup pending owner disposition.
