package recipe

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// Request is the input to Compute. It mirrors plan.Request's shape for recipes.
type Request struct {
	Recipe   *manifest.Recipe
	Adapters map[string]*manifest.Adapter // keyed by tool, for the Mcp layout
	Resolver toolpath.Resolver
	Tool     string // "claude"|"codex"|"opencode"|"all"|"" (=> recipe's wire.tools)
	Scope    string // "global"|"local"|"" (=> "global" for recipes)
	GOOS     string // host OS for asset resolution (defaults to runtime.GOOS)
	GOARCH   string // host arch (defaults to runtime.GOARCH)

	// PlacedDigest reports the sha256 Patronus RECORDED for the binary it placed at
	// a dest (from state.json). classifyFetch needs it to tell "the binary we placed
	// and verified" from "some other file that happens to be here": an ARCHIVE's pin
	// is the tarball's digest, not the extracted member's, so the placed file cannot
	// be checked against the pin. Nil means no record is available, and classifyFetch
	// then FETCHes rather than trusting an unhashed file — fail closed.
	PlacedDigest PlacedDigestFunc

	// Warnf, if set, receives non-fatal advisories (e.g. an unresolved upstream
	// with no pinned asset for this host). The cmd layer wires it to stderr.
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
// The productions, by wire method (§4) and delivery via:
//   - deliver.via fetch               -> one FETCH diff for the host asset/artifact.
//   - deliver.via package-manager     -> one advisory EXEC carrying the ordered candidates.
//   - wire.method merge               -> one MERGE diff per tool (via MergeConfig).
//   - wire.method exec, actor patronus -> one EXEC diff per command×tool (Patronus-run).
//   - wire.method exec, actor external -> one display-only EXEC diff per command×tool.
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

	// 1) DELIVERY — obtaining the payload, independent of wiring:
	//   - via: fetch          -> a FETCH diff for the host asset/artifact.
	//   - via: package-manager -> an advisory EXEC carrying the ordered install
	//     candidates. This fires for EVERY package-manager delivery, not only
	//     install-only recipes: a recipe that installs a CLI via uv AND wires its
	//     MCP server (graphify) needs both the install advisory and the merge. The
	//     manager resolves the host itself, so there is no FETCH for this path.
	//     Patronus never silently runs the install — the consent layer (cmd) decides.
	installPath := ""
	if d, fetch := fetchDiff(req, goos, goarch); fetch != nil {
		installPath = d
		diffs = append(diffs, *fetch)
	}
	if d := installAdvisory(rec, scope); d != nil {
		diffs = append(diffs, *d)
	}

	// 2) Wiring — dispatch on the wire.method discriminator: exec runs commands,
	// merge MERGEs the config. The actor axis (patronus|external) decides whether an
	// exec is advisory, not which branch we take. WireNone wires nothing (the
	// delivery above was the whole job, or a hook artifact does the wiring).
	tools := resolveTools(req.Tool, rec)
	switch rec.Wire.Method {
	case manifest.WireExec:
		diffs = append(diffs, execDiffs(rec, tools, scope)...)
	case manifest.WireMerge:
		merges, err := wireDiffs(req, tools, scope, installPath)
		if err != nil {
			return nil, err
		}
		diffs = append(diffs, merges...)
	case manifest.WireNone:
		// Nothing to wire: a package/binary was delivered above, and something else
		// (a hook artifact, or the user) does any wiring.
	}

	return diffs, nil
}

// installAdvisory builds the advisory EXEC row for a via:package-manager
// install-only recipe. It sorts the candidate list by preference (lower first),
// carries the whole ordered list on the ExecSpec (so the consent layer can detect
// managers on PATH and pick one), and displays the most-preferred command. It is
// marked self-managed + advisory so the applier skips it by default. Returns nil
// when the recipe does not install via a package manager (it has its own FETCH
// path) or no candidate renders a command.
func installAdvisory(rec *manifest.Recipe, scope string) *diff.FileDiff {
	if rec.Delivery == nil || rec.Delivery.Via != manifest.ViaPackageManager {
		return nil
	}
	cands := append([]manifest.InstallCandidate(nil), rec.Delivery.Install...)
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].Preference < cands[j].Preference })
	specs := candidateSpecs(cands, rec.Name)
	if len(specs) == 0 {
		return nil
	}
	cmd := specs[0].Command
	return &diff.FileDiff{
		Path:     cmd,
		Action:   diff.Exec,
		Artifact: rec.Name,
		Type:     string(rec.Shape()),
		Role:     string(rec.Role),
		Tool:     "-", // a package install is tool-agnostic
		Scope:    scope,
		Note:     "install: " + cmd,
		Exec:     &diff.ExecSpec{Command: strings.Fields(cmd), Display: cmd, SelfManaged: true, Advisory: true, Candidates: specs},
	}
}

// candidateSpecs renders each install candidate to a (manager, command) spec for
// the consent layer, dropping any candidate whose manager has no install template.
func candidateSpecs(cands []manifest.InstallCandidate, recipeName string) []diff.InstallCandidateSpec {
	var out []diff.InstallCandidateSpec
	for _, c := range cands {
		if cmd := c.InstallCommand(recipeName); cmd != "" {
			out = append(out, diff.InstallCandidateSpec{Manager: string(c.Manager), Command: cmd})
		}
	}
	return out
}

// fetchDiff builds the FETCH diff for a via:fetch delivery, pre-classified against
// the destination on disk (matching sha -> SKIP). A fetch is one of two sub-shapes,
// distinguished by which field is set: a per-OS/arch asset MATRIX (Assets) or a
// single pinned URL artifact (URL). It returns the resolved install path (so
// wireDiffs can substitute {installPath}) and the diff, or ("", nil) when the
// recipe has no binary to fetch — including when this host has no pinned artifact,
// which is an advisory, not an error. docker/package-manager/script deliveries have
// no fetcher.
func fetchDiff(req Request, goos, goarch string) (string, *diff.FileDiff) {
	rec := req.Recipe
	if rec.Delivery == nil || rec.Delivery.Via != manifest.ViaFetch {
		return "", nil // docker/package-manager/script or wire-only: no fetcher
	}

	var spec *diff.FetchSpec
	if rec.Delivery.URL != "" {
		// One pinned artifact for every supported host — no per-OS/arch matrix.
		// Platforms gates the hosts it can run on (tk is bash: POSIX only), and an
		// unsupported host takes the same seam as a missing asset: warn, emit no
		// FETCH. Archive stays empty, so classifyFetch verifies the placed file's sha
		// against the pin on every run.
		pin, err := rec.Delivery.ResolveURL(goos)
		if err != nil {
			warn(req, "%s: %v — skipping fetch", rec.Name, err)
			return "", nil
		}
		spec = &diff.FetchSpec{
			URL:    pin.URL,
			SHA256: pin.SHA256,
			Label:  fmt.Sprintf("%s (%s)", rec.Name, goos),
		}
	} else {
		asset, err := rec.Delivery.ResolveAsset(goos, goarch)
		if err != nil {
			// No pinned asset for this host (e.g. sandbox's TODO upstream): surface a
			// clear advisory and emit no FETCH rather than a fake download.
			warn(req, "%s: %v — skipping fetch", rec.Name, err)
			return "", nil
		}
		spec = &diff.FetchSpec{
			URL:        asset.URL,
			SHA256:     asset.SHA256,
			Archive:    asset.Archive,
			BinaryPath: asset.BinaryPath,
			Label:      fmt.Sprintf("%s (%s/%s)", rec.Name, goos, goarch),
		}
	}

	spec.Dest = resolveInstallPath(req.Resolver, rec)
	d := diff.FileDiff{
		Path:     spec.Dest,
		Action:   classifyFetch(spec, req.PlacedDigest),
		Artifact: rec.Name,
		Type:     string(rec.Shape()),
		Role:     string(rec.Role),
		Tool:     "-", // a binary placement is tool-agnostic
		Scope:    "global",
		Note:     "fetch " + spec.Label,
		Fetch:    spec,
	}
	return spec.Dest, &d
}

// PlacedDigestFunc reports the sha256 (lowercase hex, no "sha256:" prefix) that
// Patronus RECORDED for the binary it placed at dest, and whether such a record
// exists. It is threaded in from the caller so this package stays free of
// internal/state (no new package edge).
type PlacedDigestFunc func(dest string) (string, bool)

// classifyFetch decides FETCH vs SKIP idempotently, and it NEVER reports a file as
// verified without hashing it.
//
//   - raw binary: the pin IS the placed file's digest. SKIP when they match.
//   - archive: the pin is the ARCHIVE's sha; the extracted member is what lands on
//     disk, and you cannot recompute one from the other. So compare the file against
//     the digest Patronus RECORDED when it placed the binary (FetchSpec.PlacedSHA256,
//     stamped by install/apply.go, persisted by internal/state). Match -> SKIP.
//     Mismatch, or no record -> FETCH.
//
// It used to SKIP an archive on MERE PRESENCE, unhashed. That made every re-run
// unverified: once ANY file existed at the dest, Patronus reported "verified, up to
// date" forever, and gitleaks-guard EXECUTES one of these binaries on every commit.
// A file we have never hashed is NOT a file we have verified. Do not restore that
// branch.
//
// Absent dest -> FETCH in every case. Kept here so the diff package stays free of
// filesystem + crypto.
func classifyFetch(spec *diff.FetchSpec, placed PlacedDigestFunc) diff.Action {
	data, err := os.ReadFile(spec.Dest)
	if err != nil {
		return diff.Fetch // absent (or unreadable) -> needs fetching
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])

	if spec.Archive != "" {
		if placed == nil {
			return diff.Fetch // no way to check -> re-fetch and re-verify
		}
		want, ok := placed(spec.Dest)
		if !ok {
			return diff.Fetch // never recorded -> never verified -> fetch
		}
		if got == normalizeHex(want) {
			return diff.Skip
		}
		return diff.Fetch
	}

	if got == normalizeHex(spec.SHA256) {
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
// the {toolContext} placeholder (e.g. Serena's `--context claude-code` vs `--context
// codex`). A tool absent from the map falls back to its own id, so the placeholder is
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

// substPlaceholders resolves the recipe wiring placeholders in a single string:
// {installPath} (the fetched binary's path) and {toolContext} (the per-tool
// client label, see toolContexts). Centralizing it keeps command and args in sync.
func substPlaceholders(s, installPath, tool string) string {
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
		cmd := substPlaceholders(wm.Command, installPath, tool)
		if cmd != "" {
			vals["command"] = cmd
		}
		args := make([]string, len(wm.Args))
		for i, a := range wm.Args {
			args[i] = substPlaceholders(a, installPath, tool)
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

// execDiffs builds EXEC rows for an exec-method recipe: each wire.run command, with
// {tool} substituted, per targeted tool. The applier always skips these; the cmd
// layer runs them on --deploy UNLESS they are advisory.
//
//   - actor: patronus — Patronus runs the commands we specified (auto-run on --deploy).
//   - actor: external — something outside Patronus wires it via its OWN installer
//     (e.g. ai-memory install-hooks). That presupposes the tool's CLI is already on
//     $PATH — something Patronus did not deliver (ai-memory ships via Docker/cargo,
//     not a fetched binary). Auto-running it therefore errors on any machine where
//     the user hasn't installed the tool yet. So an external EXEC is ADVISORY:
//     Patronus DISPLAYS the wiring command but never executes it. SelfManaged is
//     the provenance state records; Advisory is what keeps a missing binary from
//     failing the install. Both bits are derived from actor == external.
func execDiffs(rec *manifest.Recipe, tools []string, scope string) []diff.FileDiff {
	external := rec.Wire.Actor == manifest.ActorExternal
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
				Exec:     &diff.ExecSpec{Command: argv, Display: line, SelfManaged: external, Advisory: external},
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
