package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/scan"
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
func TestScanReportsDrift(t *testing.T) {
	f := fixtureRegistry(t)
	home := withRemoteEnv(t, f)

	// Install an artifact so there is a real state row to reconcile against.
	if _, errOut, err := runInstall(t, "fix-skill", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
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
	// but never install it — so there is NO state row. This is the team-research bug:
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
	if _, errOut, err := runInstall(t, "fix-skill", "--tool", "claude", "--local", "--deploy", "--yes"); err != nil {
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
// team-research drift in miniature — the installed skill said TeamCreate while the
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

	if _, errOut, err := runInstall(t, "fix-skill", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
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
// gap (pat-wkp3). A CLAUDE.md/AGENTS.md is a fold of many fenced sections from
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
	if _, errOut, err := runInstall(t, "fix-instruction", "fix-instruction-2", "--tool", "claude", "--global", "--deploy", "--yes"); err != nil {
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
