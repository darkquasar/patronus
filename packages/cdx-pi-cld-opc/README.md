# cdx-pi-cld-opc

An sbx mixin for the built-in `codex` agent. It adds Claude Code, pi and
opencode, and deploys the Patronus harness for all four agents with the
patronus CLI, pinned to v2.7.0 through the repo installer (`install.sh
--version`). Codex, pi and opencode reach OpenAI through the sandbox proxy with
no key in the VM.

| agent | provider | authentication |
|-------|----------|----------------|
| `codex` 0.162.1 | OpenAI | base agent; brokered by the proxy (host `openai` secret) |
| `pi` 1.1.0 | OpenAI or Anthropic | bare `pi` uses brokered OpenAI; `--provider anthropic` to switch |
| `opencode` 1.19.0 | OpenAI or Anthropic | brokered OpenAI from the shipped config; Anthropic via its own auth |
| `claude` 2.1.296 | Anthropic | its own `/login` on first run |

## How brokering works

Only an agent kit may declare an oauth credential, so the kit builds on the
codex base, which binds the host `openai` secret. pi and opencode do not read
the proxy sentinel from the environment. Their shipped configs point at the
ChatGPT Codex backend with a placeholder bearer, and the proxy swaps in the real
token at egress:

```
  codex / pi / opencode
         |  openai-responses, Bearer oai-oat01-proxy-managed
         v
  sbx proxy (gateway.docker.internal:3128)  -- injects the real token -->
         |
         v
  https://chatgpt.com/backend-api/codex  ->  gpt-6.1-sol
```

The subscription token authorizes `gpt-6.1-sol` (default) and `gpt-6-astra`,
so both configs allow only those. `api.openai.com` rejects it because it lacks
API-platform scopes.

## Prerequisite

A stored OpenAI OAuth secret on the host:

```sh
sbx secret set openai --oauth
```

## Run it

```sh
sbx create codex --kit ~/.patronus/packages/cdx-pi-cld-opc/ --name <name> --skills off
sbx exec -it <name> bash      # starts in pi; exit pi for a shell
```

`~/workspace/QUICKSTART.md` lists the per-agent commands. Prefer the shell
attach: `sbx run --name <name>` attaches the base agent directly, so Ctrl+C ends
the session.

For opencode's own OpenAI OAuth (the full model catalog), publish port 1455,
which its `redirect_uri` requires, then run `opencode auth login` inside. That
token is stored in the VM.

```sh
sbx ports <name> --publish 1455:1455
```

## Harness

| agent | profile |
|-------|---------|
| `codex` | `core-profile-cx` |
| `claude` | `core` |
| `opencode` | `core-profile-oc` |
| `pi` | `core-profile-pi` |

The pi harness installs native pi packages, which needs a patronus release that
accepts pi 1.1.0.

## Layout

```
 cdx-pi-cld-opc/
 |-- spec.yaml
 |-- README.md
 `-- files/home/
     |-- .pi/agent/models.json            pi: brokered openai-codex provider
     |-- .config/opencode/opencode.json   opencode: brokered openai + anthropic
     |-- .pi-autolaunch.sh                `pi` wrapper; starts interactive shells in pi
     `-- workspace/QUICKSTART.md          per-agent commands
```

## Network

Egress is allow-all (`**`) so installers and the deploy can reach GitHub and
package registries. Narrow it to a host allowlist to restore the egress boundary.
