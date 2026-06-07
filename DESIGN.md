# Patronus — Design Plan

> A **meta-scaffolder** for AI coding environments. Patronus models the *layers* a modern agent
> environment needs (instructions, capabilities, memory, harness, observability, …) and installs them —
> as portable **artifacts** (skills/agents/commands authored once, translated per tool), **recipes**
> (external binaries like a memory layer or sandbox, fetched + wired), and **profiles** (curated bundles
> across all layers).
>
> Author/select once → CI builds per-tool scaffolds → users `patronus install` onto Claude Code, Codex, or
> OpenCode, at the **global** or **local-repo** level, on Linux / macOS / Windows.

---

## 0. Build constraints (hard rules)

1. **Do NOT author new artifacts/skills.** Use only what already exists in-repo: `team-research`, `team-implement` (tracked at `claude/skills/`). New skills/instructions/harnesses will be **sourced from specific upstreams later** — leave `TODO` markers, never invent content. *(Exception: per-artifact `patronus.yaml` manifests and adapter metadata are infrastructure, not artifact content — authoring those is in scope.)*
2. **Profiles ship as stubs.** `golang`, `python`, `cloudflare` are placeholders (`status: stub`) referencing items to be sourced later. They validate the schema + install flow, not content.
3. **Artifact-vs-recipe rule:** plain-markdown/words → built-in **artifact** (this repo); binary / Docker / bigger-than-a-file → **recipe** (fetched). (See §2b.)

---

## 1. Decisions locked in

| Decision | Choice | Rationale |
|---|---|---|
| Binary language | **Go** | Single static binary, trivial cross-compile to `linux/mac/windows × amd64/arm64`, native Windows (no WSL), distributable via `go install` + Homebrew + Scoop + `curl \| sh`. The whole premise is "download one binary that works anywhere" — Go's sweet spot. |
| Registry | **GitHub Releases** | CI attaches per-tool scaffold tarballs + `index.json` to a Release; binary fetches over HTTPS. Zero infra, free, versioned. (R2/CDN is a later upgrade path behind a config flag.) |
| Authoring model | **Portable core + adapter transforms** | Author one portable artifact; per-tool adapters mechanically reshape frontmatter/paths. Per-tool override block in the manifest for edge cases. |
| Memory layer | **NOT built by us — fetched as a recipe** | Default recipe = `akitaonrails/ai-memory` (Rust, SQLite+FTS5+optional vectors, git-markdown source of truth, MCP+CLI, explicitly multi-agent, self-wiring, 535★/v0.11). Fallback = `edg-l/engram` (binary-only, no Docker) for locked-down/airgapped cases. Patronus never authors a memory engine. See §5c. |
| Extensibility | **Tool recipes / templates** | A declarative manifest defines an external binary to fetch + verify + install + wire into each tool's MCP config. Memory layer and "secure agent sandbox" (openshell-style) are recipes. |
| Package management | **Self-contained floor + pluggable PM backends** | Patronus ships a minimal fetch-verify-shim core (works on every OS with zero prerequisites) as the default. System PMs (brew/scoop/winget/npm) are *opt-in accelerators* a recipe can declare — never a hard dependency. See §2c. |
| Product model | **Layers → installed via artifacts/recipes/profiles** | Patronus's thesis is a taxonomy of the layers a modern agent environment needs (§1A). Each layer is filled by an artifact or recipe; **profiles** bundle selections across all layers into one reproducible install. |
| Change model | **One `FileDiff` (before→after) spine** | Adapter, planner, renderer, and the (Phase-3) applier all speak `diff.FileDiff`. Dry-run, conflict classification, verbose unified diffs, JSON, and state/revert (§6d) derive from this one abstraction — no parallel code paths. See §6b. |
| Go dependencies | **Minimal & maintained** | `spf13/cobra` (CLI), `gopkg.in/yaml.v3` (manifests + agent frontmatter), `github.com/pelletier/go-toml/v2` (Codex TOML), `znkr.io/diff` v1.x (unified diffs — the maintained successor to the abandoned `hexops/gotextdiff`). No diff *algorithm* reinvented; equality is stdlib `bytes.Equal`. |

---

## 1A. The layer model (the product thesis)

Patronus is not "an installer" — it's an **opinionated model of what a complete AI coding environment is**. An agent is a loop: it *perceives* context, *decides*, *acts* on your system, and *produces* output, with you in the loop. Every layer either **feeds** that loop, **constrains** it, **observes** it, or **shares** it.

| # | Layer | Solves | Filled by | Tier |
|---|---|---|---|---|
| 1 | **Instructions / Identity** | Who the agent is; house rules & conventions (`CLAUDE.md`/`AGENTS.md`, output styles) | Artifact (author once → translate to 3 tools) | **T1** |
| 2 | **Capabilities** | What the agent can *do* — skills, subagents, slash commands | Artifact | **T1** (core) |
| 3 | **Memory** | Persistence across sessions/repos beyond the tool's built-in memory | Recipe (ai-memory / engram) | **T1** |
| 4 | **Context / Knowledge** | What the agent can *look up* — codebase index, docs/RAG, library refs, **curated design patterns** | **Artifact** (pattern skills, §9.1) *and* Recipe (code-index / docs MCP) | T2 |
| 5 | **Tools / Integrations** | The outside world — GitHub, DBs, browser, cloud, Jira MCP servers | Recipe (curated MCP bundles) | T2 |
| 6 | **Sandbox / Execution safety** | Constrain FS/network/exec — run agents securely | Recipe (openshell-style) | T2 |
| 7 | **Observability** | See what the agent *did* — traces, token/cost, session logs, audit | Recipe + hooks | T2 |
| 8 | **Evaluation / Harness** | Prove output is correct — test/typecheck/lint loops, eval suites, CI gates | Artifact (hooks/commands) + recipe | T2 |
| 9 | **Guardrails / Policy** | Hard rules — secret-scan, PII, permission policy, approval/blocking hooks | Artifact (hooks/settings) | T2 |
| 10 | **Orchestration** | Multi-agent coordination, parallel fan-out (`team-research`/`team-implement`) | Artifact (skills) — *no dedicated role/slot; realized as `role: capability` skills, lands in `capabilities:` (§5a)* | T2 (exists) |
| 11 | **Lifecycle / Reproducibility** | Pin/lock/share the whole env; bootstrap a new machine | **Profile** + lockfile | T3 (payoff) |

**Tier 1 (build first):** Instructions, Capabilities, Memory — these define the product and are highest-value/lowest-effort. **Tier 2:** the differentiators (Harness, Observability, Context, Sandbox, Guardrails). **Tier 3:** Lifecycle/reproducibility — the meta-payoff, delivered via **Profiles** (§5d).

### Instructions (L1) vs. Memory (L3) — a deliberate split
These are adjacent but architecturally different; conflating them is a common mistake:

| | Instructions (L1) | Memory (L3) |
|---|---|---|
| Nature | **Static, human-authored** | **Dynamic, agent-accumulated** |
| Source of truth | A file you write (`CLAUDE.md`/`AGENTS.md`) | A store the agent writes (ai-memory's SQLite + git-markdown) |
| Changes when | You edit it | The agent learns across sessions |
| Patronus installs it as | Translated **artifact** | Fetched **recipe** (owns its own store) |

They *reference* each other (an instruction may say "check project memory first") but are installed by different machinery. **Bridging them:** a `seed:` block on the memory recipe lets you author durable bootstrap facts ("deploys via X, staging DB is Y") that are *authored like instructions* but *pre-loaded into the memory store* at install time — so authored knowledge flows into L3 without merging the layers.

---

## 2. Two kinds of installable things

Patronus installs **artifacts** and **recipes**. Keeping these separate is the core design clarity.

### 2a. Artifacts (authored in-repo, portable)
Skills, subagents, slash commands, instruction snippets. Authored once in a tool-agnostic form; adapters translate them to each tool's on-disk shape. (**MCP servers are NOT artifacts** — they point at an external server and install via a config MERGE, often after fetching a binary; that's the recipe job. See §2b/§5c.)

### 2b. Tool recipes / templates (point at external binaries)
Declarative manifests that fetch a third-party binary/utility from the internet, verify it (checksum/signature), place it on `PATH` (or under `~/.patronus/bin`), and **wire it up** (register its MCP server / hooks into Claude/Codex/OpenCode config). Examples:
- **memory** — drop-in agent memory layer (ai-memory; engram fallback), MCP + CLI, global or local store.
- **sandbox** — secure agent execution sandbox (openshell-style), constrained FS/network.
- **observability**, **context/code-index**, **harness** — further recipes per the layer model (§1A).

Two recipe refinements the ai-memory evaluation surfaced:
- **Self-wiring recipes.** Some tools install *themselves* into agents (ai-memory ships `install-mcp --client <tool>` / `install-hooks --agent <tool>`). For these, Patronus orchestrates (fetch + start + invoke their installer) instead of doing the MCP-config merge itself. A recipe declares this via `wire.postInstall` vs `wire.mcp`.
- **Multiple recipes per capability.** A capability (e.g. `memory`) can be satisfied by more than one recipe (ai-memory *or* engram); the default is declared, alternatives selectable via `--recipe`.

Recipes are how Patronus grows without code changes: adding a new utility = adding a manifest, not editing Go.

**Canonical artifact-vs-recipe rule:** if a thing is **expressible in plain markdown/words → it's a built-in artifact** (stored in *this* repo). If it's a **binary, needs Docker, or is an apparatus bigger than a plain file → it's a recipe** (fetched from elsewhere). This resolves every layer's "filled by" column in §1A: e.g. a secret-scan *guardrail expressed as a hook instruction* is an artifact; a secret-scanning *binary* is a recipe; a *test-loop described in a command/hook* is an artifact; an *external eval-runner binary* is a recipe. It also explains why **Context/Knowledge (L4) is dual-filled**: *curated design patterns written in markdown* are an **artifact** (a pattern skill, §9.1), while a *code-index or docs-RAG binary/MCP* is a **recipe** — same layer, opposite sides of the rule.

### 2c. Package management: build vs. reuse (brew / bun / scoop / winget)

The decision differs by *what* is being installed:

- **Artifacts** → **no PM can help, ever.** Brew/bun/scoop install binaries to *their own* locations; none can reshape a SKILL.md into Codex TOML or merge an MCP entry into `~/.claude.json` at a chosen scope without clobbering. That translation + config-merge logic **is** Patronus's reason to exist and must be Patronus-native. No reuse option — this is the irreducible core.

- **Recipes (external binaries)** → **self-contained floor, with system PMs as opt-in accelerators.** Three models were weighed:

  | Model | Mechanism | Verdict |
  |---|---|---|
  | **A. Own fetch+verify** | Patronus downloads the GitHub-release asset, checks sha256, drops it in `~/.patronus/bin`, shims PATH | **Default/floor.** ~200 LOC. Zero prerequisites, identical UX on all OSes, works in CI/airgapped. We control versions. |
  | **B. OS package manager** | shell out: `brew install …`, `scoop install …`, `winget …`, `apt …` | **Opt-in accelerator.** Handles PATH/upgrade/uninstall for free — but fragmented across OSes, requires the tool to *be packaged* (engram isn't in brew), and Windows is weak. Cannot be the floor. |
  | **C. Cross-OS installer tool** (`ubi`/`eget`/`aqua`) | drive a tool that already does "fetch GH-release binary, any OS" | **Future pluggable backend.** Solves archive/checksum/PATH once across OSes, but adds a bootstrap dep. Add later as a backend, not now. |

  **Why bun is the wrong layer:** bun installs *npm packages*. The default memory layer (ai-memory; engram fallback) and a sandbox runner are **native binaries / Docker images**, not npm — bun only ever applies as one *optional backend* for the rare npm-delivered recipe (e.g. a TS reference MCP server), never the general mechanism.

  **Decision: don't reimplement a full PM, and don't hard-depend on one.** The floor must be self-contained (Model A) or Patronus dies on Windows / minimal Linux / CI — violating the "one binary, works anywhere" premise. But reusing a present system PM is strictly better (free upgrade/uninstall), so expose it as an **opt-in backend**, selected per-recipe via a `delivery` block:

  ```yaml
  # recipes/memory-engram.yaml — the engram FALLBACK recipe (binary-only, no Docker).
  # The DEFAULT memory recipe is ai-memory (§5c), which is self-wiring and Docker-first.
  # Engram is shown here because it's the recipe that exercises the github-release floor.
  delivery:
    primary: github-release        # Model A — always works, the floor
    fallbacks:                     # used only if present AND user opts in (--prefer-system-pkg)
      brew: false                  # engram is not (currently) in Homebrew — see §1/§2c prose
      scoop: false                 # aspirational; flip to a real ref once packaged
      winget: false
      npm: false                   # not an npm package
  ```

  Resolution order: `github-release` by default → if `--prefer-system-pkg` and a declared PM is detected, use it. **Never require** a system PM. Adding a backend (`ubi`, `aqua`, `bun`) is data + a small handler, not an engine rewrite — same philosophy as adapters and recipes.

---

## 3. Ground truth: per-tool on-disk layouts (researched)

This is the substrate the adapters target. Verified against official docs (2026).

### Claude Code
| Artifact | Global | Local (project) | Format |
|---|---|---|---|
| Skill | `~/.claude/skills/<name>/SKILL.md` | `.claude/skills/<name>/SKILL.md` | MD + YAML frontmatter; **dir name = command name** |
| Subagent | `~/.claude/agents/<name>.md` | `.claude/agents/<name>.md` | MD; frontmatter = config, **body = system prompt**; required `name`,`description` |
| Slash command | `~/.claude/commands/<name>.md` | `.claude/commands/<name>.md` | MD (legacy; merged into skills — prefer skills) |
| MCP (project) | — | `.mcp.json` (root) | JSON `{ "mcpServers": { name: {type:"stdio"\|"http", command/url, args, env, headers} } }` |
| MCP (user/local) | `~/.claude.json` | — | Same shape, different file (gotcha: not under `.claude/`) |
| Settings | `~/.claude/settings.json` | `.claude/settings.json` / `.claude/settings.local.json` | JSON (permissions, hooks, env) |
| **Markers** | `~/.claude/`, `~/.claude.json` | `.claude/`, `CLAUDE.md`, `.mcp.json` | |

### OpenAI Codex CLI (Rust; `~/.codex/`, TOML)
| Artifact | Global | Local (project) | Format |
|---|---|---|---|
| Instructions | `~/.codex/AGENTS.md` (`.override.md` wins) | `<repo>/AGENTS.md` (+ nested) | Markdown, concatenated root→leaf |
| Skill | `~/.codex/skills/<name>/SKILL.md` (also universal `~/.agents/skills/`) | `.agents/skills/<name>/SKILL.md` | dir + `SKILL.md`; required `name`,`description` |
| Subagent | `~/.codex/agents/*.toml` | `.codex/agents/*.toml` | **TOML**; required `name`,`description`,`developer_instructions` |
| Prompt/command | `~/.codex/prompts/*.md` (deprecated) | — | MD + YAML frontmatter |
| MCP | `~/.codex/config.toml` `[mcp_servers.<id>]` | `.codex/config.toml` (trusted) | **TOML**: stdio `command/args/env`; http `url/bearer_token_env_var/http_headers` |
| **Markers** | `~/.codex/`, `~/.codex/config.toml` | `.codex/`, `AGENTS.md` (weak/shared) | |

### OpenCode (`sst/opencode`; `~/.config/opencode/`, JSON/JSONC)
| Artifact | Global | Local (project) | Format |
|---|---|---|---|
| Config | `~/.config/opencode/opencode.json` | `opencode.json[c]` (root) | JSON/JSONC; `$schema: https://opencode.ai/config.json` |
| Instructions | `~/.config/opencode/AGENTS.md` | `AGENTS.md` / `CLAUDE.md` | MD; also `instructions: []` glob array in config |
| Subagent | `~/.config/opencode/agent/<name>.md` | `.opencode/agent/<name>.md` | MD + frontmatter (`mode`,`model`,`prompt`,`permission`); also inline `agent` key in config. Loader accepts `agent/` **and** `agents/` — **write singular** |
| Command | `~/.config/opencode/command/<name>.md` | `.opencode/command/<name>.md` | MD + frontmatter; accepts `command/`+`commands/` — write singular |
| Skill | `~/.config/opencode/skills/<name>/SKILL.md` | `.opencode/skills/<name>/SKILL.md` | **Natively Claude-SKILL.md-compatible**; also reads `.claude/skills/`, `.agents/skills/` |
| MCP | `opencode.json` `mcp` key | same | JSON: `type:"local"` (stdio, `command:[...]`,`environment`) or `type:"remote"` (`url`,`headers`,`oauth`). **Note: `local`/`remote`, not `stdio`/`http`** |
| **Markers** | `~/.config/opencode/` | `opencode.json[c]`, `.opencode/` | |

**Key cross-tool insight:** `SKILL.md` is already a near-universal format (Claude native, OpenCode native, Codex via `.agents/skills`) — the `Skill` kind is nearly passthrough. The hard translation work the adapters must own is the non-CREATE kinds: **subagents** (MD-body vs TOML vs MD-frontmatter), **MCP config** (3 file formats + the `stdio/http` vs `local/remote` split, plus Codex's shape-by-key-presence — §9.9), **hooks** (per-tool event-name + settings-file mapping), and **instructions** (`appendSection` into `CLAUDE.md`/`AGENTS.md`, vs OpenCode's `instructions: []` glob). See the kind table in §5a for the install-action each shape needs.

---

## 4. Repository layout

```
patronus/
├── artifacts/                      # AUTHOR ONCE — tool-agnostic source of truth
│   ├── skills/                      # naming: pattern classes are skills named pattern-<domain> (→ /pattern-<domain>)
│   │   ├── team-research/           # kind: Skill, role: capability (default)
│   │   │   ├── patronus.yaml        # manifest (kind, role, name, targets, overrides)
│   │   │   ├── SKILL.md             # portable body
│   │   │   └── files/...            # supporting files
│   │   ├── team-implement/          # kind: Skill, role: capability (default)
│   │   │   ├── patronus.yaml
│   │   │   ├── SKILL.md
│   │   │   └── files/...
│   │   └── pattern-cloudflare/      # MIGRATED pattern class as a Skill (role: pattern) — §9.1
│   │       ├── patronus.yaml        # kind: Skill, role: pattern, files: [patterns/]
│   │       ├── SKILL.md             # was cloudflare/cf-pattern-index.md (the routing/index body)
│   │       └── patterns/            # supporting dir named for the domain (not generic files/)
│   │           ├── pattern-001.md   # was cloudflare/pattern-001.md (loaded on demand)
│   │           └── ...              # pattern-002..007
│   ├── agents/
│   │   └── code-reviewer/
│   │       ├── patronus.yaml
│   │       └── agent.md             # portable: frontmatter + prompt body
│   └── commands/
│       └── ...
│
├── recipes/                        # TOOL RECIPES — external binaries to fetch+wire (per layer)
│   ├── memory-ai-memory.yaml        # default: ai-memory (docker/cargo, self-wiring)
│   ├── memory-engram.yaml           # fallback: engram (binary-only, no docker)
│   ├── sandbox.yaml                 # openshell-style secure agent runner
│   ├── observability-*.yaml         # L7
│   └── context-*.yaml               # L4
│
├── profiles/                       # PROFILES — curated bundles across all layers (§5d)
│   ├── golang.yaml                  # STUB — items sourced later
│   ├── python.yaml                  # STUB
│   └── cloudflare.yaml              # STUB
│
├── reference/                      # AUTHOR-FACING scaffolds — NOT installed onto users (§9.1)
│   └── templates/                   # was templates/ : pattern-template.md, pattern-index-template.md
│                                    #   (the shape every pattern-skill follows when authored)
│
├── adapters/                       # HOW each tool wants things laid out (data, not code)
│   ├── claude.yaml
│   ├── codex.yaml
│   └── opencode.yaml
│
├── registry/                       # CI OUTPUT (published to GitHub Releases, gitignored locally)
│   ├── index.json                  # catalog: artifacts + recipes, versions, checksums, per-tool tarballs
│   └── *.tar.gz
│
├── cmd/patronus/                   # the Go binary entrypoint
├── internal/
│   ├── manifest/                    # parse patronus.yaml / recipe.yaml / adapter.yaml (typed Layout schema)
│   ├── toolpath/                    # marker→abs-path resolution (env overrides, ~ expand); shared by scan+plan
│   ├── diff/                        # FileDiff (before→after) change-set abstraction; classify, Unified()
│   ├── adapter/                     # apply adapter transforms (artifact → per-tool FileDiffs), pure
│   ├── scan/                        # system scanner (detect Claude/Codex/OpenCode, std + non-std)
│   ├── plan/                        # compute change set (compose+classify); → diff.ChangeSet
│   ├── render/                      # dry-run summary table + tree + verbose unified diffs; JSON
│   ├── install/                     # Applier: atomic file writes, conflict prompts, Terraform-style partial-on-failure
│   ├── state/                       # record-only state.json (sha256 + pre-install bytes for Phase-8 revert)
│   ├── recipe/                      # fetch/verify/install external binaries + MCP wiring
│   └── registry/                    # fetch index.json + tarballs from GitHub Releases
├── .github/workflows/
│   ├── release.yml                  # cross-compile binary, build scaffolds, publish Release
│   └── build-registry.yml           # patronus build → tarballs + index.json
└── DESIGN.md
```

---

## 5. Manifest schemas

### 5a. Artifact manifest (`artifacts/**/patronus.yaml`)

**Two orthogonal axes.** Every artifact is described by `kind` (its *on-disk shape* — how the adapter installs it) and `role` (its *job* — which §1A layer it fills and which profile slot it lands in). Keeping these independent is what lets one shape serve many jobs (a `Hook` is both a guardrail and a harness) and one job span shapes — without duplicating adapter logic.

```yaml
apiVersion: patronus/v1

# ── AXIS 1: kind — ON-DISK SHAPE (selects the adapter layout block, §5b). CLOSED SET. ──
kind: Skill                 # Skill | Agent | Command | Hook | Instruction

# ── AXIS 2: role — JOB / §1A layer + profile slot (§5d). Defaults from kind; only set to override. ──
role: capability            # capability | pattern   (in use today)
                            # reserved for later layers: guardrail, harness, instruction

name: <artifact-name>       # metadata + state-tracking id; for file-drop kinds it's also the dir/file name
description: One-line summary used for catalog + agent selection.
version: 1.0.0
entry: <body-file>          # the portable body; filename depends on kind:
                            #   Skill→SKILL.md (pattern skill: the "load-first" index) · Agent→agent.md
                            #   Command→<name>.md · Instruction→INSTRUCTIONS.md · Hook→omit (config-only)
files: [files/]             # supporting dir(s) copied verbatim; may be named for the domain
                            #   (a pattern skill uses files: [patterns/] holding pattern-NNN.md)
targets: [claude, codex, opencode]   # which tools this supports
defaults:
  scope: project            # project | global  (user can override at install)
# Optional per-tool overrides for edge cases the adapter can't infer:
overrides:
  codex:
    # e.g. for an Agent, Codex needs TOML with developer_instructions
    developer_instructions_from: body
  opencode:
    mode: subagent
```

**`kind` — closed set, defined by *install action + path shape*** (each value = a hand-written `layout.<kind>` in all three adapters, so it grows only for a genuinely new shape):

| `kind` | Install action | Path shape | `{name}` in path? | Invocation |
|---|---|---|---|---|
| `Skill` | **CREATE** | `skills/{name}/SKILL.md` + files | yes | `/name` |
| `Agent` | **CREATE** | `agents/{name}.md` \| `.toml` | yes | delegated-to |
| `Command` | **CREATE** | `commands/{name}.md` | yes | `/name` |
| `Hook` | **MERGE** | `settings.json → hooks.{event}` | by `{event}` | event-fired |
| `Instruction` | **APPEND** | `CLAUDE.md` / `AGENTS.md` (fixed) | **no** | ambient (always read) |

> `Mcp` also appears in the §5b adapter `layout:` but is **not an artifact kind** — it's the **MERGE primitive the recipe engine calls** (§5c) to write an MCP config entry in each tool's format (same MERGE family as `Hook`). MCP servers are *recipes*, never artifacts. Artifact kinds = the 5 above.
> `Hook` earns a kind because no CREATE shape can install it (it merges a key into a settings file). `Instruction` earns one because neither CREATE (would clobber the user's house rules) nor MERGE (prose, not JSON/TOML) applies — it **appends a delimited section** to a concatenated doc (see §5b). *Per §0, `Hook`/`Instruction` shapes are defined now but no such artifacts ship until their layers are sourced upstream.*
>
> **Hooks arrive two ways** (parallel to MCP's self-wiring-vs-merged duality): (1) as a `kind: Hook` **artifact** Patronus merges via the §5b `Hook` layout, or (2) **recipe-wired** — a self-wiring recipe (e.g. ai-memory) installs its own hooks through `wire.postInstall`/`install-hooks` (§5c). The artifact path is for hooks *we* author (a guardrail/harness); the recipe path is for hooks an external tool installs for itself.

**`role` — open set, defined by *job / §1A layer*** (adding one = data + a profile-slot mapping, no adapter work). Defaults from `kind` so normal artifacts never set it:

| `role` | §1A layer | Default for kind(s) | Typical kind(s) | Status |
|---|---|---|---|---|
| `capability` | L2 Capabilities | `Skill`, `Agent`, `Command` | Skill/Agent/Command | ✅ active |
| `pattern` | L4 Context/Knowledge (artifact side) | — (override on a Skill) | Skill | ✅ active · **L4 also takes recipes (§5c `context`); dual-filled, see §5d** |
| `guardrail` | L9 Guardrails/Policy | `Hook` (default) | Hook | ⏳ reserved |
| `harness` | L8 Evaluation/Harness | — (override on a Hook) | Hook, Command | ⏳ reserved · **L8 also takes recipes (§5c `harness`); dual-filled, see §5d** |
| `instruction` | L1 Instructions/Identity | `Instruction` | Instruction | ⏳ reserved |

So today: **2 active roles** (`capability`, `pattern`); the rest are reserved for layers built later. `team-research`/`team-implement` are `kind: Skill` with the default `role: capability` (no `orchestration` role — L10 is filled by ordinary capability skills). A pattern skill is the one common override: `kind: Skill, role: pattern`.

### 5b. Adapter (`adapters/<tool>.yaml`) — declarative transform rules
```yaml
tool: claude
detect:                     # how the scanner positively IDs this tool
  global: ["~/.claude/", "~/.claude.json"]
  project: [".claude/", "CLAUDE.md", ".mcp.json"]
layout:
  Skill:
    global:  "~/.claude/skills/{name}/SKILL.md"
    project: ".claude/skills/{name}/SKILL.md"
    nameSource: dir         # command name comes from directory
    frontmatter: passthrough
  Agent:
    global:  "~/.claude/agents/{name}.md"
    project: ".claude/agents/{name}.md"
    bodyIs: systemPrompt    # portable body becomes the agent system prompt
    required: [name, description]
  Mcp:                      # MERGE primitive called by recipes (§5c), not a standalone installable
    project: { file: ".mcp.json", format: json, path: "mcpServers.{name}" }
    user:    { file: "~/.claude.json", format: json, path: "mcpServers.{name}" }
    transports:             # per-transport ORDERED key template (§9.9) — not a flat type field
      stdio: { keys: { type: "stdio", command: "{command}", args: "{args}", env: "{env}" } }
      http:  { keys: { type: "http", url: "{url}", headers: "{headers}" } }
  Hook:                     # MERGE family (like Mcp) — bind a command to a tool event
    global:  { file: "~/.claude/settings.json", format: json, path: "hooks.{event}", action: merge }
    project: { file: ".claude/settings.json",   format: json, path: "hooks.{event}", action: merge }
  Instruction:             # APPEND family — contribute a delimited, idempotent section to prose
    global:  { file: "~/.claude/CLAUDE.md", action: appendSection }
    project: { file: "CLAUDE.md",           action: appendSection }
    # appendSection inserts/replaces ONLY a fenced block keyed by {name}:
    #   <!-- patronus:start {name} --> … <!-- patronus:end {name} -->
    # re-install replaces that block in place; the user's other prose is never touched;
    # `remove` deletes the block and leaves the file. Codex relies on its native root→leaf
    # AGENTS.md concat; OpenCode may instead push a glob into the config `instructions: []`.
```
```yaml
# adapters/opencode.yaml (excerpt) — note the type-name translation + array command
tool: opencode
layout:
  Mcp:
    project: { file: "opencode.json", format: jsonc, path: "mcp.{name}" }
    transports:
      stdio: { keys: { type: "local",  command: "{commandArray}", environment: "{env}" } }  # type local + command:[...]
      http:  { keys: { type: "remote", url: "{url}", headers: "{headers}" } }                # type remote
```
> The ordered key template is the §9.9 fix: Codex's `transports.stdio.keys` simply **omits** `type` (so no type field is emitted), Claude/OpenCode carry a literal `type` value, and `{commandArray}` substitutes a JSON array. See §6c / §9.9.

### 5c. Recipe (`recipes/<name>.yaml`) — fetch external binary + wire it

**`capability` is to a recipe what `role` is to an artifact** (§5a): the routing field that maps the recipe to a §1A layer and therefore a profile slot (§5d). It's an open set; multiple recipes may share one capability (the default is declared, alternatives chosen via `--recipe`, §2b):

| `capability` | §1A layer | profile slot | example recipes | cardinality |
|---|---|---|---|---|
| `memory` | L3 | `memory:` | ai-memory (default), engram | single |
| `tools` | L5 | `tools:` | github, postgres-mcp, … (MCP servers) | list |
| `sandbox` | L6 | `sandbox:` | openshell-style runner | single |
| `observability` | L7 | `observability:` | (TBD) | list |
| `context` | L4 | `context:` | code-index / docs-RAG MCP | list · **L4 also takes `role: pattern` artifacts; dual-filled, see §5d** |
| `harness` | L8 | `harness:` | external eval-runner binary | list · **L8 also takes `role: harness` artifacts; dual-filled, see §5d** |

**Default memory recipe — ai-memory (self-wiring):**
```yaml
apiVersion: patronus/v1
kind: Recipe
name: memory-ai-memory
capability: memory                         # multiple recipes can satisfy "memory"; this is the default
summary: Multi-agent memory layer — MCP + CLI, SQLite+FTS5+optional vectors, git-markdown source of truth.
upstream: github.com/akitaonrails/ai-memory
license: MIT
delivery:
  primary: docker                          # blessed path: daemon on 127.0.0.1:49374 (multi-platform image)
  fallbacks:
    cargo: ai-memory                       # native binary build (Docker-free)
    aur:   ai-memory
scope:
  marker: ".ai-memory.toml"                # per-repo isolation marker; stable-UUID project scoping
  global: "~/.local/share/ai-memory"
# ai-memory wires ITSELF into each agent — Patronus just orchestrates its installers:
wire:
  selfWiring: true
  postInstall:
    - "ai-memory install-mcp   --client {tool} --apply"
    - "ai-memory install-hooks --agent  {tool} --apply"
  tools: [claude, codex, opencode]
# Optional: author durable bootstrap facts that pre-load into the store at install (L1→L3 bridge):
seed:
  - "Deploys via GitHub Actions to Cloudflare; staging DB is the `patronus-stg` D1 instance."
```

**Fallback memory recipe — engram (binary-only, no Docker):** `recipes/memory-engram.yaml` uses `delivery.primary: github-release` with per-OS/arch assets + sha256, `installTo: ~/.patronus/bin/`, and an explicit `wire.mcp` block (Patronus does the MCP-config merge itself, since engram is not self-wiring). Use when Docker is unavailable (locked-down Windows, airgapped CI).

**Sandbox recipe:** `recipes/sandbox.yaml` follows the same shape pointing at an openshell-style secure runner; `wire` may inject a hook/wrapper rather than an MCP server.

**MCP servers are recipes (L5 Tools), in two flavors — the only difference is whether there's a fetch step:**
- **Remote (http) MCP** — a hosted server at a URL. Nothing to download, so the recipe has **no `delivery:` block**; install = a single config MERGE (the `wire.mcp` block). This is the degenerate "wire-only" recipe.
- **Local (stdio) MCP** — a server binary/package that must be on disk first. The recipe has a `delivery:` block (fetch+verify via the §2c floor) **and** a `wire.mcp` block; install = **FETCH → MERGE**.
- **Self-wiring MCP** (e.g. ai-memory, above) — `wire.selfWiring: true`; Patronus fetches/starts it and runs the tool's own `install-mcp`.

```yaml
# recipes/github.yaml — REMOTE MCP: wire-only, no delivery
apiVersion: patronus/v1
kind: Recipe
name: github
capability: tools                  # L5 Tools/Integrations
summary: GitHub MCP server (hosted) — issues, PRs, code search.
wire:
  mcp:                             # adapter's Mcp MERGE writes this into each tool's config format
    transport: http               # http | stdio  (adapter maps http→remote for OpenCode, etc.)
    url: "https://api.githubcopilot.com/mcp/"
```
```yaml
# recipes/postgres-mcp.yaml — LOCAL MCP: fetch the binary, THEN wire it
delivery:
  primary: github-release          # fetch+verify the server binary (the floor, §2c)
wire:
  mcp:
    transport: stdio
    command: "{installPath}"       # config entry points at the fetched binary
    args: ["--dsn", "${DATABASE_URL}"]
```
A capability of `tools` can be satisfied by many such recipes; **curated MCP bundles** (a profile listing several under its `tools:`/L5 slot) are how a profile ships "GitHub + Postgres + browser" in one install.

> **Open: Windows path for ai-memory.** Upstream marks native Windows "experimental" (primary path is Docker/WSL2). Before making ai-memory the *unconditional* default on Windows, verify the Docker-free `cargo`/binary path works there; otherwise auto-fall-back to engram on Windows. (Tracked in §9.)

### 5d. Profile (`profiles/<name>.yaml`) — the meta-scaffolder payoff
A profile is the **third installable**: a curated bundle that selects items across *all* layers (§1A) so one command reproduces a whole opinionated environment. This is what makes Patronus a meta-scaffolder rather than an item installer.

**Starter profiles to ship — `golang`, `python`, `cloudflare` — as STUBS only.** We are *not* authoring new artifacts/skills now (see §0 constraint); profiles ship as placeholders that reference items to be **sourced later from specific upstreams**. A stub validates the schema + install flow without inventing content:
```yaml
apiVersion: patronus/v1
kind: Profile
name: golang
summary: "STUB — Go project agent environment. Items to be sourced (see TODO)."
status: stub                               # installer warns + no-ops content layers until populated
layers:                                    # keys are layer names from §1A
  instructions: []                         # TODO: source Go conventions artifact
  capabilities: [team-research, team-implement]   # ONLY existing in-repo artifacts
  memory:        memory-ai-memory          # capability default recipe
  # context / harness / observability / guardrails / sandbox: TODO — sourced later
todo:
  - "Source Go house-style + conventions instructions from <upstream-TBD>."
  - "Source Go harness (test/vet/lint loop) from <upstream-TBD>."
```
**Which profile slot an artifact lands in is decided by its `role` (§5a), not its `kind`** — this is what lets a `kind: Skill` go under `capabilities:` *or* `context:`:

| `role` | profile `layers:` key | §1A layer |
|---|---|---|
| `capability` | `capabilities:` | L2 |
| `pattern` | `context:` | L4 |
| `instruction` | `instructions:` | L1 |
| `guardrail` | `guardrails:` | L9 |
| `harness` | `harness:` | L8 |

This table covers **artifacts** (which carry a `role`). **Recipes** have no `role`; they slot by their `capability:` field instead (§5c) — e.g. `memory` → `memory:`, `tools` → `tools:`, `sandbox` → `sandbox:`. A profile's `layers:` map therefore mixes artifact slots (by role) and recipe slots (by capability), all keyed on the §1A layer name.

Slot **cardinality** follows the item type: single-recipe layers are scalars (`memory: memory-ai-memory`, `sandbox: …`), multi-item layers are lists (`capabilities: [...]`, `context: [...]`, `tools: [...]`) — see the §5c cardinality column.

**A single slot may hold BOTH artifacts and recipes** — this is how the dual-filled layers (§1A) work in practice. L4 Context, for instance, can list a `role: pattern` artifact *and* a `capability: context` docs-RAG recipe in the same `context:` array: `context: [pattern-cloudflare, docs-rag-mcp]`. The resolver dispatches each entry by **looking it up in the registry** — if it's an artifact, apply the adapter (CREATE/MERGE/APPEND); if it's a recipe, run the recipe engine (FETCH+wire). Names are unique across artifacts and recipes, so a bare name in a slot is unambiguous. (L8 Harness is dual-filled the same way: a `role: harness` Hook artifact + a `capability: harness` eval-runner recipe.)

`python.yaml` and `cloudflare.yaml` are near-identical stubs with their own `summary`/`todo` — **except** `cloudflare.yaml` populates its **`context:` slot with the real `pattern-cloudflare` skill** (a `role: pattern` artifact), so it ships partly real rather than fully stubbed:
```yaml
# profiles/cloudflare.yaml (excerpt)
layers:
  capabilities: [team-research, team-implement]   # role: capability
  context:      [pattern-cloudflare]              # role: pattern  ← the populated L4 slot
  memory:       memory-ai-memory
  # instructions / harness / guardrails / observability / sandbox: TODO — sourced later
```
This makes `cloudflare` the first profile to demonstrate a populated non-Tier-1 layer end-to-end. `patronus install --profile golang --tool claude --global` computes one combined change set across every *populated* item (full `--dry-run` tree + table) and warns about `status: stub`. A **lockfile** (`patronus.lock`, L11) pins exact versions/checksums of everything a profile resolved to, so a teammate or fresh machine reproduces the identical environment. Profiles compose (`extends: <name>`).

---

## 6. The Go binary — commands & behavior

```
patronus list [--artifacts] [--recipes] [--profiles] [--layers]   # catalog from registry index.json
patronus scan                                 # detect installed tools + existing configs
patronus install <name>... [flags]            # install artifact(s) / recipe(s)
patronus install --profile <name> [flags]     # install a whole profile (bundle across layers)
patronus update [<name>...]                   # refresh installed items
patronus remove <name>...                     # clean uninstall (tracked manifest)
patronus init --tool <t> [--global]           # scaffold a fresh project/global config
patronus lock                                 # write/refresh patronus.lock (pin versions+checksums)
```

**Install flags (the requested behaviors):**
```
--tool claude|codex|opencode|all   # target tool(s); default = auto-detect from scan
--global | --local                 # scope; default = prompt if ambiguous
--profile <name>                   # install a curated bundle across layers (§5d)
--recipe <name>                    # pick a specific recipe for a capability (e.g. memory-engram)
--prefer-system-pkg                # use brew/scoop/winget if present (else github-release floor)
--deploy                           # actually write changes to disk — the EXPLICIT write opt-in
--dry-run                          # explicitly plan only (this is the default; no-op without --deploy)
--verbose, -v                      # also show per-artifact unified diffs above the summary table
--yes                              # assume yes (CI/non-interactive); still respects --no-overwrite
--force                            # overwrite existing (otherwise per-file confirm)
```
> **Safe by default.** `install` is a **dry run unless `--deploy` is passed** — the absence of `--deploy` (or an explicit `--dry-run`) writes nothing, so no one deploys live by mistake. `--deploy` and `--dry-run` are mutually exclusive.
>
> **Status:** `--tool`, `--global/--local`, `--dry-run`, `--deploy`, `--verbose`, `--force`, `--yes`, `--recipe`, `--prefer-system-pkg` are live. The plan is always computed and rendered; `--deploy` then **writes it** (atomic, idempotent), performs FETCH downloads, runs self-wiring EXEC commands, and records state (§6c/§6d). Default and `--dry-run` do nothing. `--prefer-system-pkg` currently warns and falls through to the github-release floor (real system-PM backends are Phase 8). `--profile` arrives with the profile engine (Phase 5).

### 6a. System scanner (`internal/scan`)
- Detects each tool at **global** and **local** scope using the `detect:` markers in adapters.
- Detects **non-standard but unambiguous** configs: e.g. a `skills/<x>/SKILL.md` tree outside the standard dir, a stray `config.toml` containing `[mcp_servers.*]` (Codex), an `opencode.json` with `$schema: opencode.ai`. Heuristic = content signature, not just path. Honors `CODEX_HOME`, `OPENCODE_CONFIG_DIR`, `XDG_CONFIG_HOME`.
- Emits a structured inventory the planner consumes.

### 6b. Diff abstraction, planner + dry-run (`internal/diff`, `internal/plan`, `internal/render`)

**The diff abstraction (`internal/diff`) is the spine.** Every layer — the adapter transform engine, the planner, the dry-run renderer, and the (Phase 3) applier — speaks in **`FileDiff`s**: a target path with its `Before` and `After` bytes plus metadata (`Artifact`, `Capability`, `Tool`, `Scope`, `Role`, `Note`). One type flows from *compute* to *apply*, so conflict classification, SKIP detection, rendering, JSON output, and eventual disk writes all derive from a single source of truth — and `remove`/`update` become straightforward later.

- `diff.Classify(intended, before, after, exists)` assigns the terminal action: `CREATE` → absent→CREATE / identical→SKIP / differs→CONFLICT; `APPEND`/`MERGE` fold the existing content into `After` by construction, so they are non-destructive and never CONFLICT (equal→SKIP, else keep the action).
- `FileDiff.Unified()` renders a `---/+++/@@` unified diff of Before vs After for the verbose view, via **`znkr.io/diff`** (v1.x, maintained, the actively-supported successor to the stale `hexops/gotextdiff`).
- `diff.Applier` is the Phase-3 write interface; Phase 2 is **dry-run only** (no `Apply` implementation ships).

**The planner (`internal/plan.Compute`)** takes the registry catalog + scan inventory + selected names + flags, resolves tools (a specific `--tool` must be targeted; `all`/empty → the artifact's targets **detected** at scope, ordered claude→opencode→codex, falling back to all targeted if none detected) and scope (flag, else artifact default; `project`→`local`), drives the adapter engine per (artifact×tool×scope), **composes** diffs that land on the same path (e.g. codex+opencode both appending to a shared project `AGENTS.md` → one combined `After`), classifies each against the real filesystem, and returns a `diff.ChangeSet`. It performs read-only fs access only.

**Dry-run output (`internal/render.PrintPlan`)** — fixed order in every case: the **artifact-centric summary table first**, then the **ASCII tree**, then (only with `--verbose`) the **per-artifact unified diffs**, then a footer tally. Example (`patronus install pattern-cloudflare --tool claude --global`):

```
┌────────────────────┬───────────────────────────────────────────────┬───────────┬────────────┬────────┬────────┐
│ Artifact           │ Impacted path(s)                              │ Operation │ Capability │ Tool   │ Scope  │
├────────────────────┼───────────────────────────────────────────────┼───────────┼────────────┼────────┼────────┤
│ pattern-cloudflare │ ~/.claude/skills/pattern-cloudflare/ (8 files)│ CREATE    │ pattern    │ claude │ global │
└────────────────────┴───────────────────────────────────────────────┴───────────┴────────────┴────────┴────────┘

~/
└── .claude/
    └── skills/
        └── pattern-cloudflare/
            ├── SKILL.md          (new)  # CREATE — role: pattern
            └── patterns/
                ├── pattern-001.md (new) # CREATE — role: pattern
                └── … pattern-007.md (new)

Plan: 8 CREATE
(dry run — no files were written)
```

The summary table columns are exactly the user-facing view: **Artifact** (the patronus item being installed) · **Impacted path(s)** (the local file/folder it lands on; many files sharing a root collapse to `<dir>/ (N files)`) · **Operation** · **Capability** (what's added: skill/pattern/instruction/agent/command/mcp/hook) · **Tool** · **Scope**. `--verbose` appends the full `---/+++/@@` body per artifact **after the tree**. `--json` emits the `ChangeSet` (paths + actions + metadata; `Before`/`After` bytes are intentionally excluded).

Actions: `CREATE` (write a new file), `APPEND` (insert/replace a delimited section in prose — §5b, never touches the user's other text), `MERGE` (config edit — never blind overwrite), `FETCH` (download+verify a recipe binary — Phase 4), `CONFLICT` (a CREATE whose target exists & differs → needs confirm/`--force`), `SKIP` (identical, idempotent). A multi-tool install (`--tool all`) repeats the applicable rows per tool, except where the same absolute path is shared, which composes into one row labeled `tool-a+tool-b`.

### 6c. Apply path — atomic, never-overwrite-unconfirmed (`internal/install`, Phase 3 ✅)
- The **`install.Applier`** (`internal/install`) consumes the same `diff.ChangeSet` the planner produced and the dry run displayed — one change model from compute to apply, no parallel path. `--deploy` runs it; without `--deploy` (or with `--dry-run`) nothing is written.
- **Atomic per file:** `WriteFileAtomic` writes to a temp file in the target dir, fsyncs, and `os.Rename`s over the target (atomic on POSIX and Windows), so a crash never leaves a half-written config. Parent dirs are created as needed.
- **Terraform-style on failure:** on the first write error the applier stops, returns what already succeeded (recorded in state), and surfaces the error — **no whole-set rollback**. Re-running is idempotent (done files classify as `SKIP`).
- Config merging stays the **pure functions** in `internal/adapter` (`MergeConfig`): the planner classifies MERGE/CONFLICT/SKIP from the computed bytes, and the applier writes those same bytes. Files (`.mcp.json`, `config.toml`, `opencode.json`, `~/.claude.json`) are **merged**, not replaced — parse, set the one dotted key (`setDotted`, preserving siblings), re-serialize. JSON via `encoding/json` (deterministic); TOML via **`github.com/pelletier/go-toml/v2`**. JSONC comments are stripped to parse and re-emitted as plain JSON (lossless comment round-trip is a later refinement).
- The MCP transport object is built from the adapter's **per-transport ordered key templates** (§9.9 fix): literal templates (`type: "stdio"`/`"local"`/`"remote"`) emit verbatim, placeholder templates (`{command}`, `{args}`, `{commandArray}`, …) substitute the recipe value with its native type, an absent key is omitted — exactly how Codex carries **no** `type` field.
- A file that already exists and **differs** from a CREATE → `CONFLICT`: interactive per-file prompt (overwrite / skip / show diff via `FileDiff.Unified()`), `--force` to overwrite, `--yes` for non-interactive (conflicts skipped, **never** silently overwritten). Identical content → `SKIP` (idempotent).
- After applying, Patronus records what it wrote in a **state file** (§6d).

### 6d. State file — Terraform-style tracking & revert (record-only in Phase 3; revert in Phase 8)
Patronus needs durable knowledge of *what it installed* to support clean `remove`, idempotent `update`, and **revert to a prior state** — the role Terraform's state file plays.
- **Format:** plain **JSON via stdlib `encoding/json`** (deterministic, git-diffable, zero new deps, same family as `index.json`/`patronus.lock`). State is kilobytes written once per command, so format performance is irrelevant; readability and transparency are the criteria. (SQLite/binary formats rejected as over-engineering at this scale.)
- **Location:** `~/.patronus/state.json` (global-scope installs) + `<project>/.patronus/state.json` (local), mirroring the scope split. A single `install` run groups its applied diffs by scope and writes each scope's file.
- **Content** (`internal/state`): per installed item the artifact name + version, tool, scope, install timestamp, and per file the path, action, and a **sha256 checksum of the bytes Patronus wrote** (not the user's surrounding prose). `Merge` upserts by (artifact, tool, scope) so re-install replaces rather than duplicates.
- **Why a checksum, not full content:** lets a later `scan`/`plan` distinguish *unchanged* (matches checksum → safe to remove/update), *user-edited* (differs → warn), and *orphaned* (recorded but source gone → offer cleanup).
- **Forward-compat captured now (read in Phase 8):** for APPEND the fenced **section name**, and for APPEND/MERGE the **pre-install file bytes** (`Prior`), are recorded at apply time — these are the only revert inputs that cannot be reconstructed later (they exist only before the write). CREATE records no prior (its revert is a delete). Phase 3 **writes** these fields but never reads them.
- **Revert (Phase 8):** because everything flows through the `diff.ChangeSet` spine (§6b), revert is the **inverse** of a recorded apply — delete CREATEd files, remove APPENDed sections by marker, restore MERGEd files to `Prior`. No new machinery, just a read of state back into the same shape.
- **Relationship to the lockfile (L11, §5d):** `patronus.lock` pins *what a profile resolved to* (versions + checksums, for reproducibility across machines); `state.json` records *what is actually installed here* (for local lifecycle). Lock is the desired spec, state is the realized fact.

---

## 7. Build & distribution pipeline (CI/CD)

1. **`build-registry.yml`** — on change to `artifacts/` or `recipes/`: run `patronus build` (a CI subcommand) → for each artifact × target tool, apply the adapter → produce per-tool scaffold tarballs; for each recipe, resolve upstream release URLs + checksums; emit `registry/index.json`.
2. **`release.yml`** — on tag: cross-compile the binary (`linux,darwin,windows × amd64,arm64`), generate checksums, and publish a **GitHub Release** with the binaries + scaffold tarballs + `index.json` attached.
3. **End-user fetch** — `patronus` reads `index.json` from the latest Release (or a pinned version), downloads only the needed tarball, verifies sha256, installs per the plan.

**Install paths for the binary itself:** `curl -fsSL https://.../install.sh | sh` (Unix), `iwr ... | iex` (Windows PowerShell), plus Homebrew tap and Scoop manifest.

---

## 8. Phased delivery

**Status legend:** ✅ Done · 🔵 In progress · ⚪ Planned. Updated as part of each phase's PR.

| Phase | Status | Deliverable |
|---|---|---|
| **0** | ✅ Done | Repo restructure: move **existing** `team-research`/`team-implement` from `claude/skills/` (the git-tracked copies — *not* the gitignored `.claude/skills/` working copy) into `artifacts/skills/` (no new artifact *content*); make those skills **self-contained** by inlining the team-lifecycle protocol they currently borrow from the swarm `CLAUDE.md` (see §9.1); migrate the `cloudflare/` pattern set into `artifacts/skills/pattern-cloudflare/` as a `role: pattern` skill (index→`SKILL.md`, `pattern-00N.md`→`patterns/`; invoked `/pattern-cloudflare`); move `templates/` → `reference/templates/`; write `adapters/{claude,codex,opencode}.yaml`; add `golang`/`python`/`cloudflare` profile **stubs**. |
| **1** | ✅ Done | Go skeleton: `cmd/patronus`, `manifest`, `scan`, `list`. `patronus scan` + `patronus list` working against a **local** registry. |
| **2** | ✅ Done | **Diff spine + adapter + plan + dry-run.** New `internal/diff` (FileDiff before→after, Classify, Unified via `znkr.io/diff`); `internal/toolpath` (extracted path resolution); typed adapter `Layout` schema with polymorphic decode + per-transport ordered key templates (resolves §9.9); `internal/adapter` transform engine — Skill (passthrough) + Instruction (appendSection) drive end-to-end, Command + Agent (claude/opencode/codex reshape) + full **MCP MERGE** (json/jsonc/toml, all 3 tools incl. Codex shape-by-key) implemented as pure in-memory fns (unit-tested; no shipping driver yet); `internal/plan.Compute` (tool/scope resolution, cross-tool path compose, fs classification); `internal/render` dry-run = **summary table + tree + footer** (in that order), `--verbose` appends per-artifact unified diffs after the tree, `--json` emits the ChangeSet; real `patronus install` with `--tool/--global/--local/--deploy/--dry-run/--verbose`. **Safe by default: dry run unless `--deploy`; `--deploy` currently shows the plan then refuses (apply is Phase 3) so nothing is ever written this phase.** State-file & revert designed (§6d), built in Phase 3. |
| **3** | ✅ Done | **`install --deploy` apply path + record-only state.** `internal/install.Applier` writes the Phase-2 change set with **atomic per-file writes** (temp + rename) and **Terraform-style partial-on-failure** (no rollback; re-run is idempotent → SKIP); CONFLICT prompts (overwrite/skip/diff), `--force`, `--yes` (never silent overwrite). `internal/state` records what was written to `~/.patronus/state.json` (global) + `<project>/.patronus/state.json` (local): per-file sha256 of the bytes we wrote, plus the **pre-install bytes + section name for APPEND/MERGE** so Phase-8 revert is a pure read. **Record-only this phase — no `remove`/revert yet.** `--deploy` flips from "refuse" to a real write; default stays dry-run. |
| **4** | ✅ Done | **`recipe` engine: fetch+verify+wire, on the one change-set spine.** New `diff` actions **FETCH** (download+sha256-verify+place a binary, archive-aware) and **EXEC** (self-wiring post-install command); standalone `internal/archive` tar.gz/zip extractor (reused by the Phase-6 remote registry); typed `Delivery.Assets` + per-GOOS/GOARCH `ResolveAsset`; `McpLayout.ResolveTarget` (routes Claude global MCP → `~/.claude.json` user target); `internal/recipe.Compute` is the **first real caller of `adapter.MergeConfig`** — it produces FETCH + MCP MERGE (incl. OpenCode `commandArray` + `{installPath}` substitution) + EXEC rows; `install.Applier` gains an injectable `Fetcher` and a FETCH case (Terraform-style on verify failure); `plan.Finalize` is the shared compose/classify/sort tail so artifact + recipe diffs converge on one `ChangeSet`; cmd dispatches by registry lookup, runs EXEC on `--deploy` via an injectable runner, and records recipes in `state.json` (FETCH binary sha + `selfWired`/`postInstall`). `--recipe`/`--prefer-system-pkg` flags added (`--prefer-system-pkg` is a Phase-8 warn-and-fall-through stub). **Shipped end-to-end:** `memory-engram` (real github-release fetch+verify+extract+wire across all 3 tools, idempotent), `memory-ai-memory` (docker self-wiring EXEC), `github` (remote http MCP, wire-only MERGE). `sandbox` plans honestly but stays unfetched until its upstream (§9.4) is pinned. *(Engram upstream corrected to `Gentleman-Programming/engram` — the stale `edg-l/engram` ref does not exist.)* |
| **5** | ⚪ Planned | **Profiles** (§5d): combined cross-layer change set + `patronus.lock`. Ship 1–2 starter profiles. |
| **6** | ⚪ Planned | CI: `build-registry.yml` + `release.yml` → GitHub Releases. Install scripts, Homebrew tap, Scoop manifest. |
| **7** | ⚪ Planned | Tier-2 layers as recipes/artifacts: observability, harness, context, guardrails, sandbox hardening. |
| **8** | ⚪ Planned | `update` / `remove` / **revert** (reads the §6d state recorded since Phase 3 — delete CREATEs, un-APPEND sections, restore MERGE `Prior`), non-standard-config detection hardening, R2/CDN registry option, `--prefer-system-pkg`. |

**Tier-1 focus (per your call):** Instructions (L1) + Capabilities (L2) + Memory (L3) land in Phases 0–4. Tier-2 layers (harness, observability, context, …) are Phase 7 recipes/artifacts — designed now, built after the core proves out.

---

## 9. Open questions to resolve before Phase 1

1. ~~Existing content migration~~ **RESOLVED.** The current repo (a "patterns for LLMs" repo per its README) migrates into the new structure as follows:
   - **Skills.** `team-research` + `team-implement` are tracked at **`claude/skills/`** (the dotted `.claude/skills/` is a gitignored working copy that is byte-identical — *ignore it; the `claude/skills/` copies are source of truth*). These move to `artifacts/skills/`.
   - **Decouple from `CLAUDE.md`.** Both skills currently say "Follow CLAUDE.md Section 2A exactly" / "Read CLAUDE.md", borrowing the team-lifecycle protocol from `claude/claude-agent-factory-swarm/CLAUDE.md`. A portable artifact cannot depend on a sibling instructions file that isn't installed with it. **Fix in Phase 0:** inline the operative team-lifecycle steps (TeamCreate → manual worktrees → spawn with `team_name`/`name` → merge → cleanup) into each `SKILL.md` and drop the external "Section 2A" references, so each skill is self-contained. (The swarm `CLAUDE.md` may *also* be sourced later as an L1 Instructions artifact, but the skills must not require it.)
   - **Patterns → installable artifacts (one skill per pattern class).** A pattern set is *already* shaped like a skill: `cf-pattern-index.md` opens with "**Load this file first.** Use it to decide which full pattern file(s) to fetch" and carries a routing table + dependency graph — i.e. a progressive-disclosure entry doc that points at supporting files loaded on demand. That is exactly the `SKILL.md` + supporting-files model the existing skills already use ([RESEARCHER-TEMPLATE.md] etc.). So each pattern set becomes a **skill artifact** (§2a, §5a) — portable across all three tools via the same adapters, no new machinery.
     - **Granularity decision: one skill per pattern class, NOT one mega-`patterns` skill.** This is what preserves the profile model (§5d) — `cloudflare.yaml → context: [pattern-cloudflare]` selects *only* CF patterns; a Go profile never installs them. It also makes each class independently versioned/locked, keeps contributions collision-free (a contributor commits one self-contained skill dir, not an edit to a shared mega-index), and avoids loading every class's routing table into context on every session. (A single `patterns/` skill with class subfolders was rejected: all-or-nothing install, one shared version, merge-conflict index, and — since *dir name = invocation name* — only one invocable `/patterns` command regardless of task.)
     - **Naming convention: every pattern skill is named `pattern-<domain>`.** The directory name *is* the invocation/command name (§3), so this yields `/pattern-cloudflare`, `/pattern-react`, `/pattern-python`, … — a single discoverable, tab-completable namespace. Authoring a new pattern class = adding a new `artifacts/skills/pattern-<domain>/` skill, never a folder inside an existing one.
     - `cloudflare/cf-pattern-index.md` → `artifacts/skills/pattern-cloudflare/SKILL.md` (the index *is* the routing body; add frontmatter `name: pattern-cloudflare` + `description`).
     - `cloudflare/pattern-00N.md` → `artifacts/skills/pattern-cloudflare/patterns/pattern-00N.md` (supporting files, loaded on demand exactly as the index already directs; the supporting dir is named `patterns/` rather than the generic `files/` — declared via `files: [patterns/]` in `patronus.yaml`, §5a).
     - The "Load this file first" + "fetch the full pattern file(s)" mechanic maps 1:1 onto how an agent reads a SKILL.md then pulls its referenced files — so installing the skill gives the agent **a pattern-reading capability**, which is the natural framing.
     - `claude/mcp-server-configuration.md` and `mcp/README.md` → become a `pattern-mcp` skill (`kind: Skill, role: pattern`), the same shape as `pattern-cloudflare` — they are "load-first reference" docs, so they fit the pattern model exactly. Consistent and agent-loadable; no per-set special-casing.
   - **`templates/`** (`pattern-template.md`, `pattern-index-template.md`) → keep as **authoring templates** under `reference/templates/`. These are author-facing scaffolds for writing *new* pattern skills — not installed onto a user's machine, so they are not artifacts themselves — but they define the shape every pattern skill follows.
   - **Pattern role (NOT a new kind):** since patterns are skills-that-deliver-knowledge rather than skills-that-drive-behavior, model them as `kind: Skill` with `role: pattern` (§5a) — a *role* on the existing Skill kind, not a new kind. They adapt and install **byte-identically** to any other skill (same `layout.Skill` block); `role` only changes which §1A layer/profile slot they fill (L4 / `context:`) and how `patronus list` groups them. *(Authoring these means writing frontmatter + a `patronus.yaml` per set — covered by the §0.1 infrastructure carve-out; the pattern **content** is migrated verbatim, not newly authored.)*
   - **Root `README.md`** is updated to describe Patronus (the meta-scaffolder), retaining a pointer to the pattern skills.
2. **`name`/module path** for the Go module. Proposed: `github.com/darkquasar/patronus` (matches repo name + current git owner) — confirm.
3. **Recipe trust model:** pin upstream versions + checksums in-repo (reproducible, but manual bumps) vs resolve-latest at build time (auto, less reproducible). Recommend pinned.
4. **Sandbox recipe target:** confirm the specific "openshell"-style tool you want as the secure-runner reference (exact upstream repo).
5. **ai-memory on Windows:** verify the Docker-free `cargo`/native path works on Windows; if not, auto-fall-back to engram there (§5c). Affects whether ai-memory is the unconditional cross-OS default.
6. ~~Layer↔installable mapping~~ **RESOLVED:** plain-markdown → artifact, binary/Docker/bigger-than-file → recipe (§0.3, §2b).
7. ~~First profiles~~ **RESOLVED:** `golang`, `python`, `cloudflare` — shipped as stubs (§0.2, §5d).
8. **Upstream sources for content:** the specific repos/locations to source instructions, harnesses, and profile content from (the `TODO`/`<upstream-TBD>` markers). Needed before profiles move from stub → populated.
9. **Adapter MCP schema is too thin for Codex.** ✅ **RESOLVED (Phase 2).** The §5b adapter modeled stdio-vs-http as a flat `typeField` map, which couldn't express Codex TOML's "shape-by-key-presence" (no `type` field; stdio vs http distinguished by *which keys are present* — `command/args/env` vs `url/bearer_token_env_var/http_headers`). **Fix shipped:** each transport now declares an **ordered key template** (`transports.<t>.keys`) decoded into an order-preserving `OrderedMap` (`internal/manifest/layout.go`). `buildTransportObject` (`internal/adapter/mcp.go`) renders it — literal values emit verbatim (Claude `type:"stdio"`, OpenCode `type:"local"/"remote"`), placeholders substitute typed values (OpenCode `command:{commandArray}` → JSON array), and a key absent from the template is never emitted (Codex omits `type`). Unit-tested across all three tools, both transports.
```
