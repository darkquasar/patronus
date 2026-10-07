# ADR-0003 validator

This bundled Go command checks folder metadata without a model, credentials or network access. It parses YAML using `gopkg.in/yaml.v3` v3.0.1, pinned by the adjacent `go.mod` and `go.sum`. Go 1.25+ and an already populated dependency cache are prerequisites. Missing tools/cache block the check; these instructions grant no package installation or online download.

Set `SPEC_FOLDER` to the actual absolute research-effort folder, then run under the project's approved resource lock:

```sh
(cd "{skillDir}/scripts" && GOPROXY=off GOTOOLCHAIN=local go run -mod=readonly . "$SPEC_FOLDER")
```

Success prints `ADR-0003 OK` and exits 0. A violation prints a diagnostic on stderr and exits 1. Bad argument count exits 2. The command reads files only; it never edits metadata, writes planning outputs or calls a host API. Go may write its ordinary build cache.

The command reads only `research` and `streams[].spec` / `streams[].plan` values, ignoring YAML comments and unrelated project fields. Null/omitted document fields represent unfinished work. Non-null references must be local `.md` filenames resolving to regular files. It rejects missing referenced research/spec/plan files and every unreferenced `*-spec.md` / `*-plan.md` directory entry. Nested/traversing paths and referenced symlinks are refused. It is a two-invariant folder lint, not a semantic spec review, work scheduler, slug/stream uniqueness validator or native runtime qualification.

The repository's `TestCodexSpecFolderMetaInvariants` invokes this exact installed command interface on invented fixtures, including missing references, orphan specs/plans and filenames appearing only in comments. No local planning state or personal credentials are test inputs.

`main.go` is authored for Patronus under the repository GPL-3.0 license bundled at the skill root. The YAML dependency is external (not vendored), MIT licensed: https://github.com/go-yaml/yaml/tree/v3.0.1 . Its checksum pins are retained; this command does not bootstrap it.
