package recipe

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// Request is the input to Compute. It mirrors plan.Request's shape for recipes.
type Request struct {
	Recipe          *manifest.Recipe
	Adapters        map[string]*manifest.Adapter // keyed by tool, for the Mcp layout
	Resolver        toolpath.Resolver
	Tool            string // "claude"|"codex"|"opencode"|"all"|"" (=> recipe's wire.tools)
	Scope           string // "global"|"local"|"" (=> "global" for recipes)
	GOOS            string // host OS for asset resolution (defaults to runtime.GOOS)
	GOARCH          string // host arch (defaults to runtime.GOARCH)
	PreferSystemPkg bool   // --prefer-system-pkg (Phase-8 stub; warns + falls through)

	// Warnf, if set, receives non-fatal advisories (e.g. unresolved upstream,
	// --prefer-system-pkg not yet implemented). The cmd layer wires it to stderr.
	Warnf func(format string, args ...any)
}

// defaultInstallTo is the §2c floor placement directory when a recipe omits one.
const defaultInstallTo = "~/.patronus/bin/"

// Compute resolves a recipe install into FETCH + MERGE + EXEC diffs that feed the
// same change-set spine as artifacts (the brief's one-spine rule). It is
// read-only on disk: it reads existing config bytes (for MERGE classification)
// and stats the fetch destination (for FETCH SKIP detection), but downloads
// nothing — the applier does that.
//
// The productions, by wire mode (§4) and delivery source:
//   - deliver.source github-release -> one FETCH diff for the host asset.
//   - wire.mode mcp   -> one MERGE diff per tool (via MergeConfig).
//   - wire.mode run   -> one display-only EXEC diff per command×tool (Patronus-run).
//   - wire.mode self  -> one display-only EXEC diff per command×tool (self-managing).
func Compute(req Request) ([]diff.FileDiff, error) {
	rec := req.Recipe
	scope := req.Scope
	if scope == "" {
		scope = "global" // recipes default to global (binaries live in ~/.patronus/bin)
	}
	goos, goarch := req.GOOS, req.GOARCH
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	var diffs []diff.FileDiff

	// 1) FETCH — only for a github-release delivery. docker/cargo/script sources
	// and wire-only remote MCP (Delivery == nil) produce no download diff.
	installPath := ""
	if d, fetch := fetchDiff(req, goos, goarch); fetch != nil {
		installPath = d
		diffs = append(diffs, *fetch)
	}

	// 2) Wiring — dispatch on the single wire.mode discriminator: run/self EXEC
	// commands, mcp MERGEs the config. The shape (wire-only|fetch+wire|fetch+run)
	// is the computed display label, but the dispatch is mode, not shape.
	tools := resolveTools(req.Tool, rec)
	switch rec.Wire.Mode {
	case manifest.WireModeRun, manifest.WireModeSelf:
		diffs = append(diffs, execDiffs(rec, tools, scope)...)
	case manifest.WireModeMcp:
		merges, err := wireDiffs(req, tools, scope, installPath)
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, merges...)
	case "":
		// install-only: deliver a package and stop. A package-manager source has no
		// FETCH (npm/cargo resolve the host themselves), so surface the install
		// command as a display-only advisory row — Patronus never silently runs a
		// global package install; the user (or a future --prefer-system-pkg path)
		// runs it. Something else (a hook artifact) does the wiring.
		if d := installAdvisory(rec, scope); d != nil {
			diffs = append(diffs, *d)
		}
	}

	return diffs, nil
}

// installAdvisory builds the display-only EXEC row for an install-only recipe: a
// package-install command the user runs (marked self-managed so the applier skips
// it and remove reports it as not-auto-revertable). Returns nil when the source
// is not a package manager (it has its own FETCH path instead).
func installAdvisory(rec *manifest.Recipe, scope string) *diff.FileDiff {
	if rec.Delivery == nil {
		return nil
	}
	cmd := rec.Delivery.InstallCommand(rec.Name)
	if cmd == "" {
		return nil
	}
	return &diff.FileDiff{
		Path:     cmd,
		Action:   diff.Exec,
		Artifact: rec.Name,
		Type:     string(rec.Shape()),
		Role:     string(rec.Role),
		Tool:     "-", // a global package install is tool-agnostic
		Scope:    scope,
		Note:     "install: " + cmd,
		Exec:     &diff.ExecSpec{Command: strings.Fields(cmd), Display: cmd, SelfManaged: true, Advisory: true},
	}
}

// fetchDiff builds the FETCH diff for a github-release delivery, pre-classified
// against the destination on disk (matching sha -> SKIP). It returns the resolved
// install path (so wireDiffs can substitute {installPath}) and the diff, or
// ("", nil) when the recipe has no binary to fetch.
func fetchDiff(req Request, goos, goarch string) (string, *diff.FileDiff) {
	rec := req.Recipe
	if rec.Delivery == nil || rec.Delivery.Source != manifest.SourceGithubRelease {
		return "", nil
	}
	if req.PreferSystemPkg {
		warn(req, "--prefer-system-pkg is not yet implemented (Phase 8); using github-release floor")
	}

	asset, err := rec.Delivery.ResolveAsset(goos, goarch)
	if err != nil {
		// No pinned asset for this host (e.g. sandbox's TODO upstream): surface a
		// clear advisory and emit no FETCH rather than a fake download.
		warn(req, "%s: %v — skipping fetch", rec.Name, err)
		return "", nil
	}

	dest := resolveInstallPath(req.Resolver, rec)
	spec := &diff.FetchSpec{
		URL:        asset.URL,
		SHA256:     asset.SHA256,
		Dest:       dest,
		Archive:    asset.Archive,
		BinaryPath: asset.BinaryPath,
		Label:      fmt.Sprintf("%s (%s/%s)", rec.Name, goos, goarch),
	}

	d := diff.FileDiff{
		Path:     dest,
		Action:   classifyFetch(spec),
		Artifact: rec.Name,
		Type:     string(rec.Shape()),
		Role:     string(rec.Role),
		Tool:     "-", // a binary placement is tool-agnostic
		Scope:    "global",
		Note:     "fetch " + spec.Label,
		Fetch:    spec,
	}
	return dest, &d
}

// classifyFetch decides FETCH vs SKIP idempotently. The pinned sha256 verifies
// the *download* (the archive, or the raw binary), so it only equals the placed
// file's sha for a raw-binary asset. Therefore:
//   - raw binary: SKIP when the dest's sha matches the pinned digest.
//   - archive: the pinned sha is the archive's, not the extracted member's, so we
//     cannot recompute it from the placed binary; SKIP on dest presence (the
//     binary was sha-verified when first placed, and a re-fetch would re-verify).
//
// Absent dest -> FETCH in both cases. Kept here so the diff package stays free of
// filesystem + crypto.
func classifyFetch(spec *diff.FetchSpec) diff.Action {
	data, err := os.ReadFile(spec.Dest)
	if err != nil {
		return diff.Fetch // absent (or unreadable) -> needs fetching
	}
	if spec.Archive != "" {
		return diff.Skip // present; archive sha can't be rechecked against the binary
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) == normalizeHex(spec.SHA256) {
		return diff.Skip
	}
	return diff.Fetch
}

// resolveInstallPath resolves the absolute placement path for a recipe's binary:
// <installTo>/<binary>, with installTo defaulting to ~/.patronus/bin/ and binary
// defaulting to the recipe name.
func resolveInstallPath(res toolpath.Resolver, rec *manifest.Recipe) string {
	to := defaultInstallTo
	if rec.Delivery != nil && rec.Delivery.InstallTo != "" {
		to = rec.Delivery.InstallTo
	}
	bin := binaryName(rec)
	return filepath.Join(res.ExpandHome(strings.TrimSuffix(to, "/")), bin)
}

// binaryName is the installed filename: delivery.binary if set, else the recipe
// name. On Windows a ".exe" suffix is added when absent.
func binaryName(rec *manifest.Recipe) string {
	name := rec.Name
	if rec.Delivery != nil && rec.Delivery.Binary != "" {
		name = rec.Delivery.Binary
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
		name += ".exe"
	}
	return name
}

// wireDiffs builds one MCP-config MERGE per tool by driving adapter.MergeConfig
// (its first real caller). It substitutes {installPath} into the command and
// builds both `command` and `commandArray` so every tool's transport template
// resolves (OpenCode's stdio uses {commandArray}).
func wireDiffs(req Request, tools []string, scope, installPath string) ([]diff.FileDiff, error) {
	rec := req.Recipe
	wm := rec.Wire.Mcp
	var out []diff.FileDiff

	for _, tool := range tools {
		ad, ok := req.Adapters[tool]
		if !ok {
			return nil, fmt.Errorf("recipe %q: no adapter for tool %q", rec.Name, tool)
		}
		if ad.Layout.Mcp == nil {
			return nil, fmt.Errorf("recipe %q: tool %q has no Mcp layout", rec.Name, tool)
		}
		ft, err := ad.Layout.Mcp.ResolveTarget(scope)
		if err != nil {
			return nil, fmt.Errorf("recipe %q -> %s: %w", rec.Name, tool, err)
		}
		tr, ok := ad.Layout.Mcp.Transports[wm.Transport]
		if !ok {
			return nil, fmt.Errorf("recipe %q -> %s: no %q transport template", rec.Name, tool, wm.Transport)
		}

		path := req.Resolver.ResolveMarker(ft.File, tool, scope)
		before, _, err := readFile(path)
		if err != nil {
			return nil, fmt.Errorf("recipe %q: read %s: %w", rec.Name, path, err)
		}

		spec := serverSpec(rec.Name, wm, installPath, tool)
		after, err := adapter.MergeConfig(before, ft, tr, spec)
		if err != nil {
			return nil, fmt.Errorf("recipe %q -> %s: %w", rec.Name, tool, err)
		}

		out = append(out, diff.FileDiff{
			Path:     path,
			Action:   diff.Merge,
			Before:   before,
			After:    after,
			Artifact: rec.Name,
			Type:     string(rec.Shape()),
			Role:     string(rec.Role),
			Tool:     tool,
			Scope:    scope,
			Note:     "wire mcp: " + rec.Name,
		})
	}
	return out, nil
}

// toolContexts maps a Patronus tool id to the upstream "context"/client label a
// recipe's launch command wants when it differs from the bare tool name. It backs
// the {toolContext} token (e.g. Serena's `--context claude-code` vs `--context
// codex`). A tool absent from the map falls back to its own id, so the token is
// safe to use even where the value happens to equal the tool name.
var toolContexts = map[string]string{
	"claude":   "claude-code",
	"codex":    "codex",
	"opencode": "ide",
}

// toolContext resolves the {toolContext} substitution value for a tool.
func toolContext(tool string) string {
	if c, ok := toolContexts[tool]; ok {
		return c
	}
	return tool
}

// substTokens resolves the recipe wiring placeholders in a single string:
// {installPath} (the fetched binary's path) and {toolContext} (the per-tool
// client label, see toolContexts). Centralizing it keeps command and args in sync.
func substTokens(s, installPath, tool string) string {
	s = strings.ReplaceAll(s, "{installPath}", installPath)
	s = strings.ReplaceAll(s, "{toolContext}", toolContext(tool))
	return s
}

// serverSpec maps a WireMcp into the adapter.ServerSpec the MERGE primitive
// expects, resolving {installPath} + {toolContext} (per-tool, see toolContexts)
// and building command/commandArray for stdio.
func serverSpec(name string, wm *manifest.WireMcp, installPath, tool string) adapter.ServerSpec {
	vals := map[string]any{}
	switch wm.Transport {
	case "http":
		if wm.URL != "" {
			vals["url"] = wm.URL
		}
	case "stdio":
		cmd := substTokens(wm.Command, installPath, tool)
		if cmd != "" {
			vals["command"] = cmd
		}
		args := make([]string, len(wm.Args))
		for i, a := range wm.Args {
			args[i] = substTokens(a, installPath, tool)
		}
		if len(args) > 0 {
			vals["args"] = toAnySlice(args)
		}
		// OpenCode's stdio template uses command:[...] — build the array form from
		// the same resolved command + args so that tool's wiring resolves too.
		arr := append([]any{cmd}, toAnySlice(args)...)
		vals["commandArray"] = arr
	}
	return adapter.ServerSpec{Name: name, Transport: wm.Transport, Values: vals}
}

// execDiffs builds display-only EXEC rows for a run/self recipe: each wire.run
// command, with {tool} substituted, per targeted tool. The applier skips these;
// the cmd layer runs them on --deploy. mode: self flags them self-managing (which
// is provenance state records, not something the EXEC diff itself carries).
func execDiffs(rec *manifest.Recipe, tools []string, scope string) []diff.FileDiff {
	selfManaged := rec.Wire.Mode == manifest.WireModeSelf
	var out []diff.FileDiff
	for _, tool := range tools {
		for _, raw := range rec.Wire.Run {
			line := strings.ReplaceAll(raw, "{tool}", tool)
			argv := strings.Fields(line)
			if len(argv) == 0 {
				continue
			}
			out = append(out, diff.FileDiff{
				Path:     line, // display path = the command line
				Action:   diff.Exec,
				Artifact: rec.Name,
				Type:     string(rec.Shape()),
				Role:     string(rec.Role),
				Tool:     tool,
				Scope:    scope,
				Note:     "run: " + line,
				Exec:     &diff.ExecSpec{Command: argv, Display: line, SelfManaged: selfManaged},
			})
		}
	}
	return out
}

// resolveTools picks which tools to wire: an explicit --tool (other than "all"),
// else the recipe's wire.tools list.
func resolveTools(flag string, rec *manifest.Recipe) []string {
	if flag != "" && flag != "all" {
		return []string{flag}
	}
	return rec.Wire.Tools
}

// toAnySlice converts a []string to []any for JSON-array placeholder values.
func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// readFile reads a path, returning (nil,false,nil) when absent so callers treat
// a missing config as empty (a fresh MERGE).
func readFile(p string) ([]byte, bool, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return b, true, nil
}

func warn(req Request, format string, args ...any) {
	if req.Warnf != nil {
		req.Warnf(format, args...)
	}
}
