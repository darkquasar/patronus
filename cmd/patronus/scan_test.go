package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
)

// runScan drives the real cobra scan command and captures both streams —
// mirroring runInstall/runList/runLock.
func runScan(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := newScanCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

// fakeLister returns canned `plugin list --json` bytes per tool, so scan
// reconciles against a fixture without spawning a process.
type fakeLister struct{ out map[string][]byte }

func (f fakeLister) List(_ context.Context, tool string) ([]byte, bool) {
	b, ok := f.out[tool]
	return b, ok
}

// detectedInv builds an inventory reporting the named tools as detected globally.
func detectedInv(tools ...string) *scan.Inventory {
	inv := &scan.Inventory{}
	for _, t := range tools {
		inv.Tools = append(inv.Tools, scan.ToolStatus{
			Tool:   t,
			Global: &scan.Detection{Scope: scan.Scope("global"), Detected: true},
		})
	}
	return inv
}

func TestReconcilePluginLockFlipsVerified(t *testing.T) {
	wd := t.TempDir()
	lockPath := filepath.Join(wd, "patronus.lock")

	// A lock tracking one plugin as unverified intent.
	if err := lock.Save(lockPath, &lock.Lock{Version: lock.Version, Entries: []lock.Entry{
		{Name: "superpowers", Kind: "plugin", Source: "registry", Status: lock.StatusUnverified},
	}}); err != nil {
		t.Fatal(err)
	}

	// A catalog mapping the entry name to its claude-code id.
	cat := &registry.Catalog{Plugins: []registry.PluginEntry{{Manifest: &manifest.Plugin{
		Meta:    manifest.Meta{Family: manifest.FamilyPlugin, Name: "superpowers"},
		Sources: map[string]manifest.PluginSource{"claude-code": {Kind: "marketplace", Marketplace: "claude-plugins-official", Plugin: "superpowers"}},
	}}}}

	// Override the catalog loader (scanCatalog reaches the network otherwise).
	prev := scanCatalogFn
	scanCatalogFn = func(context.Context, string, func(string, ...any)) *registry.Catalog { return cat }
	defer func() { scanCatalogFn = prev }()

	// claude reports superpowers@claude-plugins-official installed.
	lister := fakeLister{out: map[string][]byte{
		"claude": []byte(`[{"name":"superpowers","marketplace":"claude-plugins-official"}]`),
	}}

	reconcilePluginLock(context.Background(), wd, detectedInv("claude"), lister, func(string, ...any) {})

	got, err := lock.Load(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries[0].Status != lock.StatusVerified {
		t.Errorf("status = %q, want verified", got.Entries[0].Status)
	}
}

func TestReconcilePluginLockFlipsMissing(t *testing.T) {
	wd := t.TempDir()
	lockPath := filepath.Join(wd, "patronus.lock")
	if err := lock.Save(lockPath, &lock.Lock{Version: lock.Version, Entries: []lock.Entry{
		{Name: "superpowers", Kind: "plugin", Source: "registry", Status: lock.StatusVerified},
	}}); err != nil {
		t.Fatal(err)
	}
	cat := &registry.Catalog{Plugins: []registry.PluginEntry{{Manifest: &manifest.Plugin{
		Meta:    manifest.Meta{Family: manifest.FamilyPlugin, Name: "superpowers"},
		Sources: map[string]manifest.PluginSource{"claude-code": {Kind: "marketplace", Marketplace: "claude-plugins-official", Plugin: "superpowers"}},
	}}}}
	prev := scanCatalogFn
	scanCatalogFn = func(context.Context, string, func(string, ...any)) *registry.Catalog { return cat }
	defer func() { scanCatalogFn = prev }()

	// claude is reachable but reports an empty plugin list -> missing.
	lister := fakeLister{out: map[string][]byte{"claude": []byte(`[]`)}}
	reconcilePluginLock(context.Background(), wd, detectedInv("claude"), lister, func(string, ...any) {})

	got, err := lock.Load(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries[0].Status != lock.StatusMissing {
		t.Errorf("status = %q, want missing", got.Entries[0].Status)
	}
}

func TestReconcilePluginLockNoLockIsNoop(t *testing.T) {
	wd := t.TempDir() // no patronus.lock written
	// Must not panic or create a lock; catalog loader must not even be consulted.
	reconcilePluginLock(context.Background(), wd, detectedInv("claude"), fakeLister{}, func(string, ...any) {})
	if _, err := lock.Load(filepath.Join(wd, "patronus.lock")); err != nil {
		t.Fatalf("Load of absent lock should be empty, not error: %v", err)
	}
}

// driftRow is one finding parsed out of scan's drift table — as opposed to the
// legend, whose explanatory lines mention every verdict word and would make a naive
// strings.Contains(out, "USER-EDITED") match even when nothing was flagged.
type driftRow struct {
	verdict string
	path    string
}

// parseDriftRows extracts the finding rows from scan's output: the lines under the
// "Drift:" header and its column header, up to the blank line that precedes the
// legend. Asserting against these — not the raw output — is what keeps a test from
// passing on a legend word.
func parseDriftRows(out string) []driftRow {
	lines := strings.Split(out, "\n")
	var rows []driftRow
	inTable := false
	for _, ln := range lines {
		switch {
		case strings.HasPrefix(ln, "Drift:"):
			inTable = true
		case !inTable:
			continue
		case strings.HasPrefix(ln, "VERDICT"):
			continue // column header
		case strings.TrimSpace(ln) == "":
			return rows // blank line ends the table (legend follows)
		default:
			fields := strings.Fields(ln)
			if len(fields) >= 3 {
				// VERDICT ITEM PATH DETAIL...; PATH is field 2 (absolute, no spaces).
				rows = append(rows, driftRow{verdict: fields[0], path: fields[2]})
			}
		}
	}
	return rows
}

// hasDrift reports whether the finding rows (never the legend) contain a row with
// this verdict at this path.
func hasDrift(out, verdict, path string) bool {
	for _, r := range parseDriftRows(out) {
		if r.verdict == verdict && r.path == path {
			return true
		}
	}
	return false
}

// hasVerdict reports whether any finding row carries this verdict.
func hasVerdict(out, verdict string) bool {
	for _, r := range parseDriftRows(out) {
		if r.verdict == verdict {
			return true
		}
	}
	return false
}

// serveFixtureFrom builds a fixture catalog root and serves it from memory, the way
// fixtureRegistry does — but takes the root as a PARAMETER so the caller keeps a
// handle on it and can mutate the SOURCE. That is what the STALE verdict needs: it
// is the only condition where the source must move while the deployed bytes hold
// still.
//
// ORDERING (do not reorder): build runs while cwd is the fixture root, BEFORE the
// caller invokes withRemoteEnv — withRemoteEnv t.Chdir's into a dir where
// DiscoverRoot fails by design (that is what selects the Remote registry).
func serveFixtureFrom(t *testing.T, root string) *servingFetcher {
	t.Helper()
	outDir := t.TempDir()
	t.Chdir(root)
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build fixture registry: %v", err)
	}
	f := serveTree(t, outDir)
	f.bodies[fixRawURL] = fixRawBinary
	f.bodies[fixArchiveURL] = fixArchiveTarGz(t)
	f.bodies[fixMcpURL] = fixMcpTarGz(t)
	return f
}

// TestScanReportsDrift is the acceptance gate for R7. The conditions must be
// reported DISTINCTLY, because they mean opposite things:
//
//	STALE            -> our source moved on; install should re-deploy
//	USER-EDITED      -> the user changed it; report, NEVER silently overwrite
//	UNMANAGED SHADOW -> a file sits where we would deploy, and we never wrote it
//	                    (THE DEFECT THAT MOTIVATED THIS — invisible to a
//	                     state.json-only check, because it has no state row)
//	ORPHANED STATE   -> a state row for an item the catalog no longer has (e.g. bd)
//
// Class A: it asserts Patronus's BEHAVIOR, so it binds to the fixture catalog, never
// to the real one.
// TestScanInstallPathWiredRecipeNotUserEdited is the {installPath} drift-tolerance
// regression test. It answers the question that had to be settled before absolutizing
// {installPath} into more wired surfaces: does an absolute path baked into a wired
// config make a CLEAN install read back as a machine-specific-bytes false positive
// (USER-EDITED / STALE)? It does NOT — the on-disk config is compared against the
// checksum recorded at install, which already embeds the absolute path, so the bytes
// are self-consistent. That is what keeps path normalization out of scope.
func TestScanNoDriftOnInstallPathWiredRecipe(t *testing.T) {
	root := fixtureCatalog(t)
	outDir := t.TempDir()
	t.Chdir(root)
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	f := serveTree(t, outDir)
	f.bodies[fixMcpURL] = fixMcpTarGz(t)
	withRemoteEnv(t, f)

	// Install a fetch+merge recipe: places a binary AND merges an MCP entry into
	// .claude.json. A clean install must scan with NO drift finding for that entry.
	if _, e, err := runInstall(t, "fix-mcp-bin", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, e)
	}

	out, _, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if hasVerdict(out, "ORPHANED-STATE") {
		t.Errorf("clean install of a wired recipe reported ORPHANED-STATE drift:\n%s", out)
	}
}

func TestScanInstallPathWiredRecipeNotUserEdited(t *testing.T) {
	root := fixtureCatalog(t)
	outDir := t.TempDir()
	t.Chdir(root)
	if _, err := runBuild(t, "--out", outDir, "--base-url", testRegistryBase); err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	f := serveTree(t, outDir)
	f.bodies[fixMcpURL] = fixMcpTarGz(t)
	withRemoteEnv(t, f)

	// Install the {installPath}-wired recipe: places the binary AND merges an MCP
	// entry whose command is the binary's absolute path.
	if _, e, err := runInstall(t, "fix-mcp-bin", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, e)
	}

	out, _, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	// A clean install of a wired recipe is fully in sync — no drift of any kind,
	// including the ORPHANED-STATE false positive a recipe MERGE row once produced.
	if hasVerdict(out, "ORPHANED-STATE") {
		t.Errorf("wired recipe reported ORPHANED-STATE (recipe MERGE regression):\n%s", out)
	}
	for _, r := range parseDriftRows(out) {
		if r.verdict == "USER-EDITED" || r.verdict == "STALE" {
			t.Errorf("install-path-wired config read as a machine-specific false positive: %s at %s\n%s", r.verdict, r.path, out)
		}
	}
}

func TestScanReportsDrift(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	// Install an artifact so there is a real state row to reconcile against.
	if _, errOut, err := runInstall(t, "fix-skill", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}

	skill := filepath.Join(home, ".claude", "skills", "fix-skill", "SKILL.md")
	if _, err := os.Stat(skill); err != nil {
		t.Fatalf("precondition: fix-skill was not deployed: %v", err)
	}

	// (a) USER-EDITED: change the deployed file behind Patronus's back.
	if err := os.WriteFile(skill, []byte("the user typed this\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// (b) UNMANAGED SHADOW: put a file where Patronus WOULD deploy fix-skill-claude,
	// but never install it — so there is NO state row. This is the research-team bug:
	// placed by hand or by another tool, invisible to any state.json-only check.
	shadow := filepath.Join(home, ".claude", "skills", "fix-skill-claude", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(shadow), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shadow, []byte("placed by hand or another tool\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, _, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if !hasDrift(out, "USER-EDITED", skill) {
		t.Errorf("scan did not report the hand-edited file as USER-EDITED:\n%s", out)
	}
	if !hasDrift(out, "UNMANAGED-SHADOW", shadow) {
		t.Errorf("scan did not report the unmanaged shadow — this is the defect that "+
			"motivated the guard, and it is INVISIBLE to a state.json-only check:\n%s", out)
	}
}

// TestScanReportsCrossScopeShadow is a regression test for a bug the fixture missed
// until the guard was run against a real machine: an artifact installed at ONE scope
// still has an unmanaged shadow if a copy is hand-placed at ANOTHER scope. The first
// cut skipped pass 2 for any installed NAME, so a project-scope install blinded the
// global shadow. The gate must be per-PATH, not per-name.
func TestScanReportsCrossScopeShadow(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	// Install fix-skill at PROJECT (local) scope only.
	if _, errOut, err := runInstall(t, "fix-skill", "--target", "claude", "--local", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}

	// Hand-place a copy at the GLOBAL scope, which Patronus never wrote. The name is
	// installed (at project scope), but THIS path has no state row -> a shadow.
	shadow := filepath.Join(home, ".claude", "skills", "fix-skill", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(shadow), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shadow, []byte("hand-placed at global; installed only at project\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, _, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !hasDrift(out, "UNMANAGED-SHADOW", shadow) {
		t.Errorf("a global shadow was missed because the name is installed at project "+
			"scope — the pass-2 gate must be per-path, not per-name:\n%s", out)
	}
}

// TestScanReportsStale proves the STALE verdict: the deployed copy is exactly what
// Patronus wrote, but the SOURCE moved on and nothing re-deployed it. This is the
// research-team drift in miniature — the installed skill said TeamCreate while the
// source said Agent, and every status reported "installed".
//
// It is the one verdict that needs a MUTABLE source, so it builds from a fixture root
// it holds onto, installs, then publishes a NEW VERSION of the skill and rebuilds.
//
// The version bump is not incidental — it is how a source actually moves on. A remote
// artifact is cached under an IMMUTABLE name-version key (registry.Materialize:
// "cache hit (immutable version key -> never stale)"), so republishing different bytes
// under the SAME version is not a thing the registry models. Drift is what happens
// when the catalog advances to v1.1.0 and the deployed copy is still v1.0.0.
func TestScanReportsStale(t *testing.T) {
	root := fixtureCatalog(t)
	manifestPath := filepath.Join(root, "artifacts", "skills", "fix-skill", "patronus.yaml")
	src := filepath.Join(root, "artifacts", "skills", "fix-skill", "SKILL.md")

	f := serveFixtureFrom(t, root)
	home := withRemoteEnv(t, f)

	if _, errOut, err := runInstall(t, "fix-skill", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}
	skill := filepath.Join(home, ".claude", "skills", "fix-skill", "SKILL.md")
	deployed := mustRead(t, skill)

	// The SOURCE moves on — a new version, with new bytes. Nothing re-deploys it.
	bumped := strings.Replace(string(mustRead(t, manifestPath)), "version: 1.0.0", "version: 1.1.0", 1)
	if !strings.Contains(bumped, "version: 1.1.0") {
		t.Fatal("precondition: could not bump the fixture skill's version")
	}
	if err := os.WriteFile(manifestPath, []byte(bumped), 0o644); err != nil {
		t.Fatal(err)
	}
	moved := "---\nname: fix-skill\ndescription: fixture skill\n---\nThe source says something NEW.\n"
	if err := os.WriteFile(src, []byte(moved), 0o644); err != nil {
		t.Fatal(err)
	}
	// Re-serve the advanced catalog at the same URLs (withRemoteEnv's fetchers are
	// swapped in place, so rebuilding the served tree is what makes scan see it).
	f2 := serveFixtureFrom(t, root)
	fetcherForCommands, registryFetcher, fetcherForDeploy = f2, f2, f2
	t.Chdir(t.TempDir()) // back to a no-catalog cwd so scan resolves the REMOTE registry

	// The install above warmed the remote index cache with the OLD (1.0.0) catalog.
	// A real client sees a new version only after that cache refreshes; clearing it is
	// how "the published catalog advanced to 1.1.0" reaches this scan. Without this,
	// scan reads the stale cached index and correctly reports no drift.
	if err := os.RemoveAll(filepath.Join(home, ".patronus", "cache")); err != nil {
		t.Fatal(err)
	}

	// The deployed bytes are untouched — so this is NOT user-edited.
	if got := mustRead(t, skill); !bytes.Equal(got, deployed) {
		t.Fatalf("precondition: the deployed file must be untouched, got:\n%s", got)
	}

	out, _, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !hasDrift(out, "STALE", skill) {
		t.Errorf("scan did not report the un-redeployed skill as STALE — the source moved "+
			"on and the deployed copy did not:\n%s", out)
	}
	if hasVerdict(out, "USER-EDITED") {
		t.Errorf("an untouched deployed file must NEVER be reported USER-EDITED:\n%s", out)
	}
}

// TestScanReconcilesAtRecordedScope is the acceptance gate for recorded-scope
// reconciliation. An artifact whose MANIFEST defaults to project scope, installed --global, must
// reconcile against the scope it was RECORDED at.
//
// Before the fix, scan collected installed artifacts into a name-only set and
// replanned each with an empty Scope, so plan.Compute fell back to the manifest
// default. The global copy then had no would-deploy source, drift.Classify saw
// hasSource=false, and every such install was reported ORPHANED-STATE. On a real
// machine that was ~78 false rows.
func TestScanReconcilesAtRecordedScope(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	// fix-skill-project defaults to PROJECT scope; install it at GLOBAL.
	if _, errOut, err := runInstall(t, "fix-skill-project", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}
	skill := filepath.Join(home, ".claude", "skills", "fix-skill-project", "SKILL.md")
	if _, err := os.Stat(skill); err != nil {
		t.Fatalf("precondition: the artifact was not deployed at GLOBAL scope: %v", err)
	}

	out, _, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if hasDriftItem(out, "ORPHANED-STATE", "fix-skill-project", skill) {
		t.Errorf("a clean --global install of a project-default artifact was reported "+
			"ORPHANED-STATE: reconciliation replanned at the MANIFEST default instead of "+
			"the RECORDED scope:\n%s", out)
	}
	// Nor may suppressing the false verdict trade it for a different false one.
	for _, verdict := range []string{"STALE", "USER-EDITED", "MISSING"} {
		if hasDrift(out, verdict, skill) {
			t.Errorf("clean install reported %s at %s:\n%s", verdict, skill, out)
		}
	}
}

// TestScanReconcilesAtRecordedScopeReverse is the mirror direction: an artifact
// whose manifest defaults to GLOBAL, installed --local. Both directions matter,
// because a fix that hardcoded either scope would pass only one of them.
func TestScanReconcilesAtRecordedScopeReverse(t *testing.T) {
	f := fixtureRegistry(t)
	withRemoteEnv(t, f)

	if _, errOut, err := runInstall(t, "fix-instruction-global", "--target", "claude", "--local", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}

	out, _, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	for _, r := range parseDriftRows(out) {
		if r.verdict == "ORPHANED-STATE" {
			t.Errorf("a clean --local install of a global-default artifact was reported "+
				"ORPHANED-STATE at %s (reverse scope direction):\n%s", r.path, out)
		}
	}
}

// TestScanReconcilesEachIdentitySeparately proves the unit of reconciliation is the
// {artifact, tool, scope} TRIPLE, not the name. The same artifact installed at two
// scopes has two placements, and each must be judged against its own recorded scope
// — which is exactly what a --scope flag could not express, since one state file
// holds both rows at once.
func TestScanReconcilesEachIdentitySeparately(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	// The same artifact at BOTH scopes. Its manifest default is project, so the
	// global row is the one a manifest-default replan gets wrong.
	if _, errOut, err := runInstall(t, "fix-skill-project", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install global: %v\n%s", err, errOut)
	}
	if _, errOut, err := runInstall(t, "fix-skill-project", "--target", "claude", "--local", "--deploy", "--yes"); err != nil {
		t.Fatalf("install local: %v\n%s", err, errOut)
	}

	globalSkill := filepath.Join(home, ".claude", "skills", "fix-skill-project", "SKILL.md")

	// Only the GLOBAL copy is hand-edited. Each identity must answer for its own
	// path: the edited one is USER-EDITED, the untouched one stays silent.
	if err := os.WriteFile(globalSkill, []byte("the user typed this\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, _, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !hasDrift(out, "USER-EDITED", globalSkill) {
		t.Errorf("the hand-edited GLOBAL copy was not reported; each identity must "+
			"reconcile against its own recorded scope:\n%s", out)
	}
	if hasVerdict(out, "ORPHANED-STATE") {
		t.Errorf("two identities of one artifact produced a false ORPHANED-STATE:\n%s", out)
	}
}

// TestScanDriftIsDeterministic guards the map-ordering trap: identities are collected
// into a MAP, and a map's iteration order is random. Two identities converging on
// one path must not make the reported verdict depend on which one the runtime
// happened to visit last.
func TestScanDriftIsDeterministic(t *testing.T) {
	f := fixtureRegistry(t)
	withRemoteEnv(t, f)

	// Several artifacts across tools and scopes, including two instructions folding
	// into one composed CLAUDE.md and one artifact at two scopes.
	if _, errOut, err := runInstall(t, "fix-instruction", "fix-instruction-2", "fix-skill-project", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install global: %v\n%s", err, errOut)
	}
	if _, errOut, err := runInstall(t, "fix-skill-project", "--target", "claude", "--local", "--deploy", "--yes"); err != nil {
		t.Fatalf("install local: %v\n%s", err, errOut)
	}

	first, _, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	for i := 0; i < 5; i++ {
		out, _, err := runScan(t)
		if err != nil {
			t.Fatalf("scan %d: %v", i, err)
		}
		if out != first {
			t.Fatalf("scan output varies between runs — a map-ordered winner is leaking "+
				"into the report.\nfirst:\n%s\nrun %d:\n%s", first, i, out)
		}
	}
}

// TestScanWarnsOnUnreplayableToolLabel guards the label-vs-selector split. state stores an ATTRIBUTION
// label, and plan.Request.Tool takes a SELECTOR; "agnostic" is a recipe-row label
// with no honest artifact expansion ("" and "all" both invent targets). Such a row
// must WARN, never be dropped in silence — a swallowed identity turns a false
// ORPHANED into a silent one, which is the worse failure.
func TestScanWarnsOnUnreplayableToolLabel(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	if _, errOut, err := runInstall(t, "fix-skill", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}

	// Corrupt the recorded tool label into the recipe-only "agnostic" placeholder,
	// which is invalid on an artifact row.
	statePath := filepath.Join(home, ".patronus", "state.json")
	st, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for i := range st.Items {
		if st.Items[i].Artifact == "fix-skill" {
			st.Items[i].Tool = "agnostic"
			found = true
		}
	}
	if !found {
		t.Fatal("precondition: no state row for fix-skill")
	}
	if err := state.Save(statePath, st); err != nil {
		t.Fatal(err)
	}

	_, errOut, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !strings.Contains(errOut, "fix-skill") || !strings.Contains(errOut, "agnostic") {
		t.Errorf("an unreplayable tool label was handled SILENTLY; it must warn:\n%s", errOut)
	}
}

// TestScanWarnsOnMissingRecordedScope guards the empty-scope case. An empty Scope is not neutral:
// it re-triggers the manifest-default fallback, which is the bug. A legacy row
// without a scope must have one INFERRED from the state file that held it, and the
// inference must be announced rather than applied in silence.
func TestScanWarnsOnMissingRecordedScope(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	// fix-skill-project defaults to PROJECT; installed --global, then its recorded
	// scope is erased the way a pre-scope state file would have left it.
	if _, errOut, err := runInstall(t, "fix-skill-project", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}
	statePath := filepath.Join(home, ".patronus", "state.json")
	st, err := state.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	for i := range st.Items {
		st.Items[i].Scope = ""
	}
	if err := state.Save(statePath, st); err != nil {
		t.Fatal(err)
	}

	out, errOut, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !strings.Contains(errOut, "no scope") {
		t.Errorf("a legacy row with no recorded scope was reconciled silently:\n%s", errOut)
	}
	// The inference is from the HOME state file, so it must be global — which is
	// where the artifact actually lives, so the row still reads clean.
	skill := filepath.Join(home, ".claude", "skills", "fix-skill-project", "SKILL.md")
	if hasDrift(out, "ORPHANED-STATE", skill) {
		t.Errorf("the inferred scope did not match where the file actually is:\n%s", out)
	}
}

// hasDriftItem is hasDrift plus the ITEM column: a composed CLAUDE.md carries one
// row per fenced section, all sharing the file path, so the section's OWNING
// artifact is the only thing that tells them apart.
func hasDriftItem(out, verdict, item, path string) bool {
	for _, ln := range strings.Split(out, "\n") {
		f := strings.Fields(ln)
		if len(f) >= 3 && f[0] == verdict && f[1] == item && f[2] == path {
			return true
		}
	}
	return false
}

// TestScanReportsComposedSectionDrift is the acceptance gate for the composed/APPEND
// gap. A CLAUDE.md/AGENTS.md is a fold of many fenced sections from
// different artifacts; whole-file Classify cannot judge it, because the file never
// equals any single source and every contributor records the same whole-file
// checksum. So drift reconciles PER SECTION.
//
// This installs TWO instructions into ONE CLAUDE.md, moves ONLY the first one's
// source, and asserts the scan reports STALE for that section's artifact while the
// untouched section stays silent — proving the verdict is per section, not per file.
//
// Class A: it asserts Patronus's BEHAVIOR, so it binds to the fixture catalog.
func TestScanReportsComposedSectionDrift(t *testing.T) {
	root := fixtureCatalog(t)
	manifestPath := filepath.Join(root, "artifacts", "instructions", "fix-instruction", "patronus.yaml")
	src := filepath.Join(root, "artifacts", "instructions", "fix-instruction", "INSTRUCTIONS.md")

	f := serveFixtureFrom(t, root)
	home := withRemoteEnv(t, f)

	// Both instructions fold into ONE global CLAUDE.md as distinct fenced sections.
	if _, errOut, err := runInstall(t, "fix-instruction", "fix-instruction-2", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}
	claudeMd := filepath.Join(home, ".claude", "CLAUDE.md")
	cb := string(mustRead(t, claudeMd))
	for _, want := range []string{"patronus:start fix-instruction", "patronus:start fix-instruction-2"} {
		if !strings.Contains(cb, want) {
			t.Fatalf("precondition: CLAUDE.md missing %q:\n%s", want, cb)
		}
	}

	// ONLY fix-instruction's source moves on (new version, new body).
	bumped := strings.Replace(string(mustRead(t, manifestPath)), "version: 1.0.0", "version: 1.1.0", 1)
	if !strings.Contains(bumped, "version: 1.1.0") {
		t.Fatal("precondition: could not bump fix-instruction's version")
	}
	if err := os.WriteFile(manifestPath, []byte(bumped), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("# Fixture instruction\n\nThe source section says something NEW.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f2 := serveFixtureFrom(t, root)
	fetcherForCommands, registryFetcher, fetcherForDeploy = f2, f2, f2
	t.Chdir(t.TempDir())
	if err := os.RemoveAll(filepath.Join(home, ".patronus", "cache")); err != nil {
		t.Fatal(err)
	}

	// The deployed CLAUDE.md is untouched — nothing re-folded it.
	if got := string(mustRead(t, claudeMd)); got != cb {
		t.Fatalf("precondition: deployed CLAUDE.md must be untouched")
	}

	out, _, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if !hasDriftItem(out, "STALE", "fix-instruction", claudeMd) {
		t.Errorf("scan did not report the moved-on SECTION as STALE — composed files "+
			"were the known gap, reconciled per section now:\n%s", out)
	}
	// The untouched section's source did NOT move, so it must not be flagged. A
	// whole-file compare would have wrongly flagged BOTH (or neither).
	if hasDriftItem(out, "STALE", "fix-instruction-2", claudeMd) {
		t.Errorf("scan wrongly reported the UNCHANGED section as STALE — the verdict is "+
			"not per-section:\n%s", out)
	}
	if hasVerdict(out, "USER-EDITED") {
		t.Errorf("an untouched composed file must NEVER be reported USER-EDITED:\n%s", out)
	}
}

// TestScanComposedMcpConfigIsClean is the MERGE-side twin of
// TestScanReportsComposedSectionDrift. Two MCP recipes fold into ONE .claude.json;
// with both server blocks present exactly as installed, neither contributor may be
// reported. A whole-file compare reports a permanent false STALE here, because the
// composed file equals neither contributor's standalone bytes.
func TestScanComposedMcpConfigIsClean(t *testing.T) {
	root := fixtureCatalog(t)
	f := serveFixtureFrom(t, root)
	home := withRemoteEnv(t, f)

	if _, errOut, err := runInstall(t, "fix-mcp-bin", "fix-mcp-two", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}
	cfg := filepath.Join(home, ".claude.json")
	body := string(mustRead(t, cfg))
	for _, want := range []string{"fix-mcp-bin", "fix-mcp-two"} {
		if !strings.Contains(body, want) {
			t.Fatalf("precondition: both servers must be wired; %q missing:\n%s", want, body)
		}
	}

	out, _, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	// PASS 1: neither contributor is STALE against a config that matches them both.
	for _, item := range []string{"fix-mcp-bin", "fix-mcp-two"} {
		if hasDriftItem(out, "STALE", item, cfg) {
			t.Errorf("composed MCP config wrongly reported STALE for %s; the whole-file "+
				"compare is back:\n%s", item, out)
		}
	}
	// PASS 2: the same file must not then be reported as an unmanaged shadow. This
	// is the `recorded`-set half: excluding a path from `rows` without marking it
	// recorded trades a false STALE for a false shadow.
	for _, verdict := range []string{"UNMANAGED-SHADOW", "USER-EDITED", "MISSING"} {
		if hasDrift(out, verdict, cfg) {
			t.Errorf("composed MCP config wrongly reported %s:\n%s", verdict, out)
		}
	}
}

// TestScanComposedMcpReportsGenuineEdit guards against over-suppressing: silencing
// the false STALE must not silence a real one. A user edit to ONE contributor's own
// dotted path is still that contributor's drift.
func TestScanComposedMcpReportsGenuineEdit(t *testing.T) {
	root := fixtureCatalog(t)
	f := serveFixtureFrom(t, root)
	home := withRemoteEnv(t, f)

	if _, errOut, err := runInstall(t, "fix-mcp-bin", "fix-mcp-two", "--target", "claude", "--global", "--deploy", "--yes"); err != nil {
		t.Fatalf("install: %v\n%s", err, errOut)
	}
	cfg := filepath.Join(home, ".claude.json")

	// The user rewrites fix-mcp-two's server block by hand.
	var doc map[string]any
	if err := json.Unmarshal(mustRead(t, cfg), &doc); err != nil {
		t.Fatal(err)
	}
	doc["mcpServers"].(map[string]any)["fix-mcp-two"] = map[string]any{"type": "http", "url": "https://USER-EDITED/"}
	edited, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, edited, 0o644); err != nil {
		t.Fatal(err)
	}

	out, _, err := runScan(t)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if !hasDriftItem(out, "STALE", "fix-mcp-two", cfg) {
		t.Errorf("a hand-edited server block was not reported for its owner:\n%s", out)
	}
	// The untouched contributor stays silent: the verdict is per setting, not per file.
	if hasDriftItem(out, "STALE", "fix-mcp-bin", cfg) {
		t.Errorf("the UNTOUCHED contributor was wrongly reported; the verdict is not "+
			"per-setting:\n%s", out)
	}
}
