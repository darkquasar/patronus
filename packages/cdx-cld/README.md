# cdx-cld

An sbx mixin for the built-in `codex` agent (the base). It adds Claude Code and
deploys the Patronus harness for both agents with the patronus CLI, pinned to
v2.7.0 through the repo installer (`install.sh --version`).

| agent | harness | authentication |
|-------|---------|----------------|
| `codex` (base) | `core-profile-cx` | host OpenAI credential via the sandbox proxy (`sbx secret set openai --oauth`), or `codex login` |
| `claude` | `core` | its own OAuth `/login` inside the VM |

Do not set `ANTHROPIC_API_KEY` in the sandbox: Claude Code prefers it over OAuth.

## Run it

```sh
sbx run codex --kit ~/.patronus/packages/cdx-cld/ .

# or create now, attach later
sbx create codex --kit ~/.patronus/packages/cdx-cld/ --name cdx-cld
sbx run --name cdx-cld
```

## Network

Egress is deny-by-default. The allowlist covers Anthropic login and API, the
Patronus registry, the harness's hosted MCP servers (GitHub, Context7), and a
research set: code hosting, package registries, docs, Q&A and search. Add hosts
one at a time in `spec.yaml`. Widening to `**` lets agent-authored code reach
any host.
