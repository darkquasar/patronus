# Repository tools

Run tools from the repository root. Product source stays in `cmd/` and `internal/`;
artifact-local helpers remain with their owning artifact.

| Location | Purpose |
|---|---|
| `install.sh`, `install.ps1` | Download and checksum-verify the binary |
| `distribution/` | Capture release receipts and dispatch authorized distribution |
| `distribution/reference/` | Historical package shapes, not live catalogue definitions |
| `publish-packages.sh`, `check-*.py`, `check-*.sh`, `r2-audit.sh` | Publication checks and owner-operated maintenance |
| `tests/` | Application/catalog gates and script regression suites |
| `catalog-check/` | Standalone Go catalogue validation module |
| `qualification/` | Opt-in procedures and retained evidence; not automatic acceptance |
| `vms/` | Packer/Tart definitions for native package checks |

See [CONTRIBUTING.md](../CONTRIBUTING.md) for required gates and
[vms/README.md](vms/README.md) for VM setup. Moving tools does not authorize
publication, VM execution or native qualification.

```text
+------------------+  => invoke  +------------------+  => report  +------------------+
| Contributor/CI   | =========> | Repository tools | ==========> | Checks/evidence  |
+------------------+            +------------------+             +------------------+
```

The `qualification/skill-provenance/*.md` archives retain their original bytes,
including historical `scripts/` references. The index records their current
locations and original hashes. Current consumers use `tools/`.
