# Pi sandbox kit

This package installs a standalone sbx kit at
`~/.patronus/packages/pi-sandbox/` on Darwin/arm64. Patronus manages the
host kit files. You launch the sandbox separately from your workspace:

```sh
sbx run "$HOME/.patronus/packages/pi-sandbox/" .
```

Running the kit requires an installed sbx runtime with kit schema v2 support
and separately configured Anthropic authentication. Follow sbx's provider
authentication instructions for an Anthropic API key or its supported stored
OAuth login. The upstream kit gives an API key precedence when available.
Patronus installs successfully without sbx, checks only PATH for readiness,
and performs no runtime installation, login, OAuth refresh or credential writes.

The static spec retains upstream proxy-managed credential placeholders and
the runtime destination `~/.pi/agent/auth.json`. That file is created by sbx
when applicable; it is not a bundled source file. Likewise, `AGENTS.md` names
sbx's runtime instruction profile. The `sandbox.entrypoint` and `setup.install`
fields are instructions for sbx when you launch, including npm proxy setup.
Patronus only delivers these bytes.

## Pinned contents and validation

The kit derives from [docker/sbx-kits-contrib/pi at
869c83997680a252ed2b35671b3fd0d9adc2d487](https://github.com/docker/sbx-kits-contrib/tree/869c83997680a252ed2b35671b3fd0d9adc2d487/pi).
Its image is pinned to:

```text
docker.io/sbx/pi-image@sha256:a2c3ac953dcef579f89a1a4c20f993e3b9325d58633b7f672694f64b0a067a67
```

The index selects Linux/arm64 manifest
`sha256:0f09d1c977e08e64889fe345397171a963b4c63de01cc392bd20ecd00fa43603`
for the VM. Read-only registry and immutable layer inspection verified
`@earendil-works/pi-coding-agent` **0.87.1**. Registry build provenance links
the image to the upstream commit above; its signature was not independently
verified. The separate **pi-subagents 0.71.0** release is not installed in this
image. Pi's bundled subagent example does not establish that extension's presence.

This package follows the upstream Anthropic kit. It does not reproduce a
custom OpenAI environment. Static acceptance checks the reviewed kit semantics
against upstream's normative [SPEC-v2.md](https://github.com/docker/sbx-kits-contrib/blob/869c83997680a252ed2b35671b3fd0d9adc2d487/spec/SPEC-v2.md),
pins and payload inventory. Upstream provides no JSON schema. No VM launch,
provider authentication or end-to-end runtime validation was performed for
this package. License and modification details are in `LICENSE` and `NOTICE`.

## Separate example: a Codex mixin

sbx also supports adding a mixin to a built-in agent using `--kit`. This is
a separate usage mode. The following illustrative mixin directory must already
exist; the Pi package does not install it:

```sh
sbx create --name codex-work --clone --cpus 2 --memory 4g \
  --skills off --pull missing \
  --kit "$HOME/my-sbx-kits/codex-mixin" \
  codex "$PWD"
sbx run --name codex-work
```

`--clone` creates a separate checkout in the sandbox. Its edits need to be
retrieved and integrated deliberately; they do not automatically change your
host checkout or open a pull request. Consult sbx's documented clone retrieval
workflow in the [sbx documentation](https://docs.docker.com/ai/sandboxes/)
for your installed version. Without clone isolation, a shared workspace can
expose host files to edits; check your runtime's workspace mode before launch.

Reusing a sandbox name can reattach to an existing VM. Updating the host kit
does not update that VM or its image. Choose creation or recreation separately
using sbx's lifecycle instructions, preserving any work first. Herdr tab
integration remains experimental and requires separate validation.

## Updates, conflicts and removal

Keep personal files and runtime output outside the package directory.
Unknown content, including `.DS_Store`, blocks replacement even with
`--force`; relocate the reported paths before retrying. An existing unowned
root cannot be adopted. Edits to owned files require explicit consent:

```sh
patronus update pi-sandbox --deploy
patronus update pi-sandbox --deploy --force
patronus remove pi-sandbox --deploy
```

Use force only when you intend to replace your owned-file edits. `--yes`
does not bypass these conflicts. Removal preserves unknown content and edited
owned files unless you explicitly use removal's `--force` for owned files.

The ownership receipt is `~/.patronus/package-state/pi-sandbox.json`;
recovery evidence lives under
`~/.patronus/package-state/transactions/pi-sandbox/`. If recovery reports a
conflict, preserve that evidence, reconcile the listed paths and retry the
deployment. Read-only plans do not recover by writing. An error after a durable
commit can mean cleanup or discovery repair remains; retry after resolving
the reported problem. Install, update and remove affect host kit files only;
VMs, sessions, credentials and repository work remain outside Patronus ownership.
