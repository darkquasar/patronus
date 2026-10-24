# pi-cld

An sbx mixin for a pi sandbox. It adds Claude Code and deploys the Patronus
harness for pi (`core-profile-pi`) with the patronus CLI, pinned to v2.7.0
through the repo installer (`install.sh --version`).

## Run it

Compose it with a pi base kit, such as the `pi-sandbox` package:

```sh
sbx create ~/.patronus/packages/pi-sandbox/ --kit ~/.patronus/packages/pi-cld/ --name <name> --skills off
sbx run --name <name>
```

- `pi` reaches Anthropic through the sandbox proxy.
- `claude` is a wrapper that unsets pi's `ANTHROPIC_API_KEY` sentinel, so Claude
  Code uses its own OAuth `/login`.

## Harness

Two setup steps install it: one runs `install.sh --version v2.7.0`, which
verifies the binary against the release checksums, the other runs
`patronus install --profile core-profile-pi --target pi --global --deploy`.
Patronus v2.7.0 supports pi 0.87.1, the version in the pinned pi image. To repair
or update the harness, re-run the install line with `~/.local/bin` and
`~/.patronus/bin` on `PATH`.

## Network

Egress is allow-all (`**`), so the deploy can reach GitHub and package
registries. Real keys stay on the host. Narrow it to a host allowlist to restore
the egress boundary.
