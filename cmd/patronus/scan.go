package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/adapter/builtin"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/drift"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/plan"
	"github.com/darkquasar/patronus/internal/plugin"
	"github.com/darkquasar/patronus/internal/recipe"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/render"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Detect installed AI coding tools and their configs",
		Long:  "Detects Claude Code, Codex, and OpenCode at global and local scope using the repo's adapter detect: markers (honoring CODEX_HOME, OPENCODE_CONFIG_DIR, XDG_CONFIG_HOME).",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			// A checkout is a bonus, not a requirement: scan is the diagnostic you
			// run on YOUR machine to see what is deployed into ~/.claude, and that
			// is exactly where there is no artifacts/+adapters/ above the cwd. An
			// absent root leaves adaptersDir empty, which is precisely the case
			// loadAdapters already documents a fallback for (the embedded adapters).
			var adaptersDir string
			if root, err := registry.DiscoverRoot(wd); err == nil {
				adaptersDir = filepath.Join(root, "adapters")
			}
			adapters, err := loadAdapters(adaptersDir)
			if err != nil {
				return err
			}
			inv, err := scan.Scan(scan.Options{ProjectDir: wd, Adapters: adapters})
			if err != nil {
				return err
			}

			warnf := func(f string, a ...any) { fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+f+"\n", a...) }

			// Reconcile every DEPLOYED file against both the sha we recorded writing
			// and the source the catalog holds now. It never fails the scan: an
			// unreachable catalog degrades to reporting nothing.
			findings := reconcileDrift(cmd.Context(), wd, inv, adapters, warnf)

			if jsonOutput {
				return render.JSON(cmd.OutOrStdout(), struct {
					*scan.Inventory
					Drift []drift.Finding `json:"drift,omitempty"`
				}{inv, findings})
			}
			render.PrintInventory(cmd.OutOrStdout(), inv)
			render.PrintDrift(cmd.OutOrStdout(), findings)

			// Reconcile the lock's plugin statuses against installed reality. Scan
			// reads tools; it never writes tool state. A missing lock, an
			// unavailable registry, or an unreachable tool CLI degrades gracefully
			// (statuses that cannot be confirmed stay unverified).
			reconcilePluginLock(cmd.Context(), wd, inv, pluginListerForScan, warnf)
			return nil
		},
	}
}

// pluginLister runs `<bin> plugin list --json` and returns its stdout. The real
// impl shells out; tests inject a fake so scan reconciles against a fixture
// without spawning a process. A test hook, not a user knob.
type pluginLister interface {
	List(ctx context.Context, tool string) ([]byte, bool)
}

// pluginListerForScan is the package-level seam scan uses to read each tool's
// installed plugins. Production shells out via execPluginLister; tests override it.
var pluginListerForScan pluginLister = execPluginLister{}

// execPluginLister is the production pluginLister: it runs `<bin> plugin list
// --json` and returns stdout, with ok=false when the tool has no plugin CLI on
// this machine (missing binary, or the command errors).
type execPluginLister struct{}

var _ pluginLister = execPluginLister{}

func (execPluginLister) List(ctx context.Context, tool string) ([]byte, bool) {
	bin, ok := plugin.Bin(tool)
	if !ok {
		return nil, false
	}
	if _, err := exec.LookPath(bin); err != nil {
		return nil, false
	}
	out, err := exec.CommandContext(ctx, bin, "plugin", "list", "--json").Output()
	if err != nil {
		return nil, false
	}
	return out, true
}

// reconcilePluginLock reads each detected, plugin-capable tool's installed plugins
// and flips the lock's plugin entries to verified/missing (unreachable tools leave
// them unverified). It is a no-op when there is no patronus.lock at wd, or the lock
// tracks no plugins. The catalog is loaded to map each lock entry's name to its
// per-tool "<plugin>@<marketplace>" id; if the catalog is unavailable, plugins are
// left unverified (their ids cannot be resolved). It never fails the scan.
func reconcilePluginLock(ctx context.Context, wd string, inv *scan.Inventory, lister pluginLister, warnf func(string, ...any)) {
	lockPath := filepath.Join(wd, "patronus.lock")
	l, err := lock.Load(lockPath)
	if err != nil || len(l.Entries) == 0 {
		return // no lock (or empty) → nothing to reconcile
	}
	if !hasPluginEntry(l.Entries) {
		return
	}

	cat := scanCatalogFn(ctx, wd, warnf)
	if cat == nil {
		return // cannot map entry names to ids; leave statuses as-is (unverified)
	}

	// For each detected, plugin-capable tool with a reachable CLI, read its
	// installed ids and map them back to lock-entry names.
	idsByTool := map[string]map[string]bool{}
	for _, ts := range inv.Tools {
		tool := ts.Tool
		eco, capable := plugin.EcosystemFor(tool)
		if !capable || !detected(ts) {
			continue
		}
		out, ok := lister.List(ctx, tool)
		if !ok {
			continue // CLI unreachable → this tool contributes no reachability
		}
		installed, err := plugin.DetectInstalled(tool, bytes.NewReader(out))
		if err != nil {
			warnf("scan: parsing %s plugin list: %v", tool, err)
			continue
		}
		names := map[string]bool{}
		for _, e := range l.Entries {
			if e.Kind != "plugin" {
				continue
			}
			p := findPlugin(cat, e.Name)
			if p == nil {
				continue
			}
			id, ok := plugin.Ref(p.Manifest, eco)
			if ok && installed[id] {
				names[e.Name] = true
			}
		}
		idsByTool[tool] = names
	}

	reconciled := plugin.Reconcile(l.Entries, idsByTool)
	l.Entries = reconciled
	if err := lock.Save(lockPath, l); err != nil {
		warnf("scan: writing reconciled lock: %v", err)
	}
}

// hasPluginEntry reports whether any entry is a tracked plugin.
func hasPluginEntry(entries []lock.Entry) bool {
	for _, e := range entries {
		if e.Kind == "plugin" {
			return true
		}
	}
	return false
}

// detected reports whether a tool was found at either scope.
func detected(ts scan.ToolStatus) bool {
	return (ts.Global != nil && ts.Global.Detected) || (ts.Local != nil && ts.Local.Detected)
}

// scanCatalogFn loads the catalog scan uses to map lock-entry names to per-tool
// ids. A package var so tests inject a fixture catalog without a network round
// trip — a test hook, not a user knob.
var scanCatalogFn = scanCatalog

// scanCatalog loads the catalog for id-mapping, returning nil (not an error) when
// the registry is unavailable — reconciliation degrades to leaving statuses as-is.
func scanCatalog(ctx context.Context, wd string, warnf func(string, ...any)) *registry.Catalog {
	reg, _, err := resolveRegistry(ctx, wd, registrySel{}, homeDir(), warnf)
	if err != nil {
		return nil
	}
	cat, err := reg.Catalog(ctx)
	if err != nil {
		return nil
	}
	return cat
}

// reconcileDrift compares every deployed file against BOTH the sha Patronus
// recorded when it wrote the file AND the source bytes the catalog holds now,
// reporting the conditions distinctly — they mean opposite things.
//
// It takes TWO passes, and the second one is the whole point:
//
//	PASS 1 (state -> disk):   catches STALE, USER-EDITED, MISSING, ORPHANED-STATE
//	PASS 2 (catalog -> disk): catches UNMANAGED SHADOW — a file occupying a path we
//	                          WOULD deploy to, with NO state row, because Patronus
//	                          never wrote it.
//
// You CANNOT find an unmanaged shadow by walking state.json: by definition it has no
// state row. That is exactly how the stale research-team skill hid — placed by hand
// or by another tool, Patronus had no record of it, and every check that walked
// state.json reported nothing wrong while the agent executed a protocol from a
// deleted era.
//
// Neither pass re-derives where an item lands. plan.Compute — the same function
// install drives — already returns, for every artifact, the path Patronus WOULD
// deploy to AND the bytes it WOULD write (diff.FileDiff.Path/.After). Two
// implementations of "where does this item land" is precisely the class of
// divergence this guard exists to catch, so there is only one.
//
// It also does NOT download the catalog to look at it. A remote artifact's source
// is only fetched (materialized) when it is actually INSTALLED — i.e. it has a state
// row, so its bytes are needed for the STALE comparison. An unmanaged shadow needs
// no source at all: the verdict turns on "a file is here and we have no record of
// writing it", which is decided before Classify ever looks at the source. So a scan
// of a machine with nothing installed downloads nothing.
//
// An unreachable catalog is not a failure: it yields no source, so the drift report
// degrades to silence rather than to a false verdict.
// It never fails the scan: every way of not-knowing (no registry, no catalog, an
// artifact whose source cannot be fetched) degrades to reporting less, never to an
// error — so drift is pure additive signal on top of the inventory.
func reconcileDrift(ctx context.Context, wd string, inv *scan.Inventory, adapters []*manifest.Adapter, warnf func(string, ...any)) []drift.Finding {
	reg, _, err := resolveRegistry(ctx, wd, registrySel{}, homeDir(), warnf)
	if err != nil {
		return nil // no registry -> nothing to compare against -> stay quiet
	}
	cat := scanCatalogFn(ctx, wd, warnf)
	if cat == nil {
		return nil
	}

	// Read state FIRST: it names the items that are actually installed, and those
	// are the only ones whose SOURCE we need (to tell STALE from OK). Fetching the
	// rest would download content just to browse it.
	type recordedFile struct {
		key      artifactInstallKey
		checksum string
	}
	// One recorded APPEND section of a composed file (CLAUDE.md/AGENTS.md). Several
	// of these can share a path — one per contributing artifact — which is exactly
	// why they cannot be keyed by path like the whole-file rows.
	type recordedSection struct {
		key     artifactInstallKey
		path    string
		section string
	}
	// The catalog's artifact names. Only these go through the adapter spine — a
	// recipe (tk, gitleaks, …) FETCHes a binary with no source to diff, so feeding
	// its name to plan.Compute would just raise "unknown artifact".
	isArtifact := map[string]bool{}
	for i := range cat.Artifacts {
		isArtifact[cat.Artifacts[i].Manifest.Name] = true
	}

	// recordedMerge is one contributor's setting-level claim on a shared config: the
	// dotted key it owns and the value it would write there. A composed config has N
	// of these and one whole-file checksum that matches none of them, so it is
	// reconciled per contributor (pass 1c) rather than whole-file.
	type recordedMerge struct {
		key  artifactInstallKey
		path string
		edit *diff.SettingEdit
	}

	rows := map[string]recordedFile{}
	var sectionRows []recordedSection
	appendPath := map[string]bool{} // paths reconciled per-section, not whole-file
	var mergeRows []recordedMerge
	mergePath := map[string]bool{} // paths reconciled per-setting, not whole-file
	// Two sets, because they answer different questions. installedIdents is WHERE
	// each artifact was installed — the recorded {artifact, tool, scope} triples we
	// must replan at, one plan.Request each. installedNames is WHICH sources to
	// fetch, once per artifact however many identities it has.
	installedIdents := map[artifactInstallKey]struct{}{}
	installedNames := map[string]struct{}{}
	installedRecipes := map[string]bool{} // recipe items whose MERGE rows need a computed source
	// The scope to assume for a legacy row that records none. Which state file held
	// the row is the only evidence of scope left in that case.
	scopeOfDir := map[string]string{}
	if inv.Home != "" {
		scopeOfDir[inv.Home] = toolpath.ScopeGlobal
	}
	if inv.ProjectDir != "" {
		scopeOfDir[inv.ProjectDir] = toolpath.ScopeLocal
	}
	for _, dir := range []string{inv.Home, inv.ProjectDir} {
		if dir == "" {
			continue
		}
		s, err := state.Load(filepath.Join(dir, ".patronus", "state.json"))
		if err != nil {
			continue // unreadable state: nothing recorded that we can reconcile
		}
		for _, it := range s.Items {
			// The identity this item's file rows reconcile against. Scope is taken
			// from the row, never left empty: an empty Scope is not neutral, it makes
			// plan.Compute fall back to the artifact's manifest default, which is
			// precisely the false-ORPHANED bug.
			key := artifactInstallKey{artifact: it.Artifact, tool: it.Tool, scope: it.Scope}
			if key.scope == "" {
				key.scope = scopeOfDir[dir]
				warnf("drift: %s (%s) records no scope; assuming %q from %s", it.Artifact, it.Tool, key.scope, dir)
			}
			if isArtifact[it.Artifact] {
				installedNames[it.Artifact] = struct{}{}
				installedIdents[key] = struct{}{}
			}
			if catalogHasRecipe(cat, it.Artifact) {
				installedRecipes[it.Artifact] = true
			}
			for _, f := range it.Files {
				// A FETCH row is a binary Patronus downloaded, not an artifact it
				// rendered from a source file; recipe.classifyFetch already
				// re-verifies those against their pin on every run.
				if f.Action == string(diff.Fetch) {
					continue
				}
				// A composed APPEND (CLAUDE.md/AGENTS.md) is a fold of many sources
				// into one file; its whole-file checksum never matches any single
				// source, so it is reconciled PER SECTION below, not here.
				if f.Action == string(diff.Append) && f.Section != "" {
					sectionRows = append(sectionRows, recordedSection{key: key, path: f.Path, section: f.Section})
					appendPath[f.Path] = true
					continue
				}
				// A composed MERGE (settings.json, .claude.json) is a fold of many
				// setting edits from different artifacts; its whole-file checksum
				// never matches any single contributor, so it is reconciled PER
				// SETTING below, not here.
				if f.Action == string(diff.Merge) && f.Setting != nil {
					mergeRows = append(mergeRows, recordedMerge{key: key, path: f.Path, edit: f.Setting})
					mergePath[f.Path] = true
					continue
				}
				rows[f.Path] = recordedFile{key: key, checksum: f.Checksum}
			}
		}
	}
	// A path that ANY contributor appended to or merged into is a composed file;
	// never also whole-file reconcile it (a non-section, non-setting row for the
	// same file would otherwise re-introduce the false whole-file compare).
	for p := range appendPath {
		delete(rows, p)
	}
	for p := range mergePath {
		delete(rows, p)
	}

	// Materialize ONLY the installed artifacts, so their source bytes are on disk for
	// the content comparison. Keyed by NAME, not by identity: one artifact installed
	// at two identities still has one source to fetch. A no-op for a local checkout
	// (LocalDir is already set).
	names := make([]string, 0, len(installedNames))
	for name := range installedNames {
		names = append(names, name)
	}
	sort.Strings(names)
	if err := materializeSelected(ctx, reg, cat, names); err != nil {
		warnf("drift: cannot fetch installed sources: %v", err)
	}

	// What the INSTALLED artifacts would deploy now, replanned AT THE RECORDED
	// IDENTITY: for each of their paths, the bytes the catalog source would write
	// there. Pass 1 reads this to tell STALE from OK. Non-installed artifacts are
	// pass 2's job (and need no source).
	idents := make([]artifactInstallKey, 0, len(installedIdents))
	for k := range installedIdents {
		idents = append(idents, k)
	}
	sortInstallKeys(idents)
	would, wouldSection := wouldDeploy(cat, inv, adapters, wd, idents, warnf)

	// Recipe MERGE rows (MCP entries) are ownable bytes Patronus computes, but
	// wouldDeploy is artifact-only — so add them here or Pass 1 misreads every
	// installed MCP recipe as ORPHANED-STATE.
	recNames := make([]string, 0, len(installedRecipes))
	for name := range installedRecipes {
		recNames = append(recNames, name)
	}
	sort.Strings(recNames)
	if len(recNames) > 0 {
		res := toolpath.New(os.LookupEnv, toolpath.HomeDir(os.LookupEnv), wd)
		for path, w := range recipeMergeSources(cat, adapterMap(adapters), res, recNames, warnf) {
			// A recipe has no artifact identity to plan at; its rows are found by the
			// recipe NAME the state row carries, which is what lookupWould falls back to.
			would[wouldKey{artifact: w.item, path: path}] = w
		}
	}

	var findings []drift.Finding
	recorded := map[string]bool{}

	// PASS 1: state -> disk. Every file we RECORDED writing, reconciled against what
	// is there now and against what the source says now. The lookup is per IDENTITY:
	// the same artifact installed at two tool/scope identities has two placements,
	// and a path-only lookup would let one silently answer for the other.
	paths := make([]string, 0, len(rows))
	for path := range rows {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		row := rows[path]
		recorded[path] = true
		current, exists := readIfExists(path)
		w, hasSource := lookupWould(would, row.key, path)
		v := drift.Classify(current, exists, row.checksum, w.source, hasSource)
		if v == drift.OK {
			continue
		}
		findings = append(findings, drift.Finding{
			Path:    path,
			Item:    row.key.artifact,
			Verdict: v,
			Detail:  driftDetail(v),
		})
	}

	// PASS 1b: composed APPEND files, reconciled PER SECTION. A CLAUDE.md/AGENTS.md
	// is a fold of many fenced sections from different artifacts; whole-file Classify
	// cannot judge it (§ drift.ClassifySection). For each recorded section, compare
	// the body inside its fence on disk against the body the source would fold now.
	for _, sr := range sectionRows {
		recorded[sr.path] = true
		current, exists := readIfExists(sr.path)
		var onDisk []byte
		present := false
		if exists {
			onDisk, present = adapter.SectionBody(current, sr.section)
		}
		src, hasSource := lookupWouldSection(wouldSection, sr.key, sr.path, sr.section)
		v := drift.ClassifySection(onDisk, present, src, hasSource)
		if v == drift.OK {
			continue
		}
		findings = append(findings, drift.Finding{
			Path:    sr.path,
			Item:    sr.key.artifact,
			Verdict: v,
			Detail:  driftDetail(v),
		})
	}

	// PASS 1c: composed MERGE files, reconciled PER SETTING. A settings.json or
	// .claude.json is a fold of many scalar sets and list-appends from different
	// artifacts; whole-file Classify cannot judge it, and keying by path kept only
	// one contributor, so every edit was attributed to whichever row survived. For
	// each recorded edit, ask the narrow question: is MY dotted path present with
	// MY value?
	//
	// recorded[path] is set for every row whatever the verdict, exactly as pass 1b
	// does. It is what tells pass 2 the occupancy is explained; without it a
	// composed config trades a false STALE for a false shadow.
	for _, mr := range mergeRows {
		recorded[mr.path] = true
		current, exists := readIfExists(mr.path)
		if !exists {
			findings = append(findings, drift.Finding{
				Path: mr.path, Item: mr.key.artifact, Verdict: drift.Missing, Detail: driftDetail(drift.Missing),
			})
			continue
		}
		v := classifySettingEdit(current, mr.edit)
		if v == drift.OK {
			continue
		}
		findings = append(findings, drift.Finding{
			Path: mr.path, Item: mr.key.artifact, Verdict: v, Detail: driftDetail(v),
		})
	}

	// PASS 2: catalog -> disk. The ONLY pass that can see an unmanaged shadow,
	// because an unmanaged shadow has no state row to walk.
	//
	// A shadow is a directory (or file) that exists on disk where an artifact would
	// live, but that Patronus never wrote. So the hunt is occupancy-driven: for each
	// catalog artifact that is NOT installed, resolve where it WOULD land — from its
	// NAME alone, no source needed (candidateMarker) — and look there. Only when that
	// spot is occupied do we materialize that one artifact and Transform it to learn
	// its real, possibly multi-file, deploy paths. A skill's paths are data-dependent
	// (it deploys SKILL.md plus a whole subtree), so the paths must come from
	// Transform, not from a second path-deriver that would silently miss the subtree.
	//
	// The occupancy gate is what keeps this cheap: a clean machine has no occupied
	// spots, so it materializes nothing — honoring "don't download content just to
	// browse". Only a machine that already has a shadow pays to confirm it.
	env := os.LookupEnv
	res := toolpath.New(env, toolpath.HomeDir(env), wd)
	eng := adapter.New(res)
	byTool := adapterMap(adapters)

	for i := range cat.Artifacts {
		art := cat.Artifacts[i].Manifest
		// NOTE: we do NOT skip artifacts that are installed. An artifact installed at
		// one scope can still have an unmanaged shadow at ANOTHER — brainstorming
		// installed at project scope, with a hand-placed copy in ~/.claude/skills too.
		// The per-path `recorded` check below is the real gate: a path pass 1 already
		// reconciled is skipped; a path it did not is a shadow, whoever owns the name.
		for _, tool := range detectedTools(inv) {
			ad, ok := byTool[tool]
			if !ok {
				continue
			}
			for _, scope := range []string{toolpath.ScopeGlobal, toolpath.ScopeLocal} {
				marker := candidateMarker(art, ad, tool, scope, res)
				if marker == "" {
					continue
				}
				if _, err := os.Stat(marker); err != nil {
					continue // the spot this artifact would occupy is empty: not drift
				}
				// If pass 1 already recorded a file at (or under) this exact spot, the
				// occupancy is explained — this scope's copy is managed. Skip the fetch;
				// a shadow at a DIFFERENT scope has its own marker and is checked there.
				if markerIsRecorded(marker, recorded) {
					continue
				}
				// Occupied and unexplained. Materialize just this one and ask Transform
				// for the real paths, so a skill's whole subtree is covered — not just
				// its entry.
				e := &cat.Artifacts[i]
				if err := materializeSelected(ctx, reg, cat, []string{art.Name}); err != nil {
					warnf("drift: cannot fetch %s to confirm a shadow at %s: %v", art.Name, marker, err)
				}
				diffs, err := eng.Transform(art, ad, scope, e.Source.LocalDir, func(p string) ([]byte, bool, error) {
					b, ok := readIfExists(p)
					return b, ok, nil
				})
				if err != nil {
					warnf("drift: cannot resolve %s paths at %s: %v", art.Name, marker, err)
					continue
				}
				for _, d := range diffs {
					if d.IsDir || recorded[d.Path] {
						continue
					}
					if _, exists := readIfExists(d.Path); !exists {
						continue
					}
					recorded[d.Path] = true // don't double-report across tools/scopes
					findings = append(findings, drift.Finding{
						Path:    d.Path,
						Item:    art.Name,
						Verdict: drift.UnmanagedShadow,
						Detail:  driftDetail(drift.UnmanagedShadow),
					})
				}
			}
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Verdict != findings[j].Verdict {
			return findings[i].Verdict < findings[j].Verdict
		}
		return findings[i].Path < findings[j].Path
	})
	return findings
}

// markerIsRecorded reports whether pass 1 already reconciled a file at this
// candidate spot — the marker path itself, or (for a skill's directory marker) any
// file underneath it. When it did, this scope's copy is managed and pass 2 has
// nothing to add; a shadow at another scope resolves to a different marker.
func markerIsRecorded(marker string, recorded map[string]bool) bool {
	if recorded[marker] {
		return true
	}
	prefix := marker + string(filepath.Separator)
	for p := range recorded {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// detectedTools returns the tools scan positively identified, so the shadow hunt
// only looks under layouts for tools that are actually present.
func detectedTools(inv *scan.Inventory) []string {
	var out []string
	for _, ts := range inv.Tools {
		if detected(ts) {
			out = append(out, ts.Tool)
		}
	}
	return out
}

// candidateMarker returns a path whose existence on disk is a cheap, NAME-ONLY
// proxy for "this artifact may already occupy its spot" — the parent directory for
// a directory-shaped artifact (a skill, which owns ~/.claude/skills/{name}/), or the
// target file itself for a file-shaped one (agent/command/output-style). It reads no
// source: the layout template plus the artifact name is all a marker needs, which is
// what lets the shadow hunt gate on occupancy BEFORE deciding to fetch anything.
//
// It returns "" for types that do not own a per-artifact spot — instructions,
// settings, hooks, and MCP all MERGE into a file shared with other artifacts, so a
// file being present there is never evidence of one artifact's shadow.
func candidateMarker(art *manifest.Artifact, ad *manifest.Adapter, tool, scope string, res toolpath.Resolver) string {
	resolve := func(tmpl string) string {
		// Mirror adapter.Engine.resolvePath exactly: substitute {name}, then resolve
		// the marker for this tool+scope. Same two steps, so the spot we probe is the
		// spot Transform would place into.
		return res.ResolveMarker(strings.ReplaceAll(tmpl, "{name}", art.Name), tool, scope)
	}
	switch art.Type {
	case manifest.TypeSkill:
		if ad.Layout.Skill != nil {
			if t := ad.Layout.Skill.ForScope(scope); t.OK() {
				// A skill owns its directory; the directory is the marker.
				return filepath.Dir(resolve(t.Path))
			}
		}
	case manifest.TypeAgent:
		if ad.Layout.Agent != nil {
			if t := ad.Layout.Agent.ForScope(scope); t.OK() {
				return resolve(t.Path)
			}
		}
	case manifest.TypeCommand:
		if ad.Layout.Command != nil {
			if t := ad.Layout.Command.ForScope(scope); t.OK() {
				return resolve(t.Path)
			}
		}
	case manifest.TypeOutputStyle:
		if ad.Layout.OutputStyle != nil {
			if t := ad.Layout.OutputStyle.ForScope(scope); t.OK() && t.Action != "appendSection" {
				return resolve(t.File)
			}
		}
	}
	return ""
}

// driftDetail is the one-line "so what" for a verdict.
func driftDetail(v drift.Verdict) string {
	switch v {
	case drift.Stale:
		return "source moved on; re-run install"
	case drift.UserEdited:
		return "changed since we wrote it; we will not overwrite"
	case drift.UnmanagedShadow:
		return "not installed by Patronus"
	case drift.OrphanedState:
		return "recorded, but no longer in the catalog"
	case drift.Missing:
		return "recorded, but gone from disk"
	}
	return ""
}

// artifactInstallKey is one RECORDED install identity: an artifact, the tool label
// state stored for it, and the scope it was installed at. It is the unit
// reconciliation replans at, because it is the unit install recorded. Reconciling by
// name alone loses the scope, and plan.Compute then falls back to the artifact's
// MANIFEST DEFAULT — which reports every non-default-scope install as ORPHANED-STATE
// .
//
// tool holds the label verbatim as state stored it, which is an ATTRIBUTION and not
// a selector: it can be a composite ("claude+opencode") for a shared path, or the
// "agnostic" placeholder. replayTools translates it into the concrete selectors
// plan.Request accepts.
type artifactInstallKey struct {
	artifact string
	tool     string
	scope    string
}

// sortInstallKeys orders identities by artifact, then tool, then scope. Planning
// order comes from a map, and a map's order is random; two identities can converge
// on one path, so an unsorted plan makes the winner differ between runs.
func sortInstallKeys(keys []artifactInstallKey) {
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.artifact != b.artifact {
			return a.artifact < b.artifact
		}
		if a.tool != b.tool {
			return a.tool < b.tool
		}
		return a.scope < b.scope
	})
}

// replayTools turns the tool label state RECORDED into the concrete tool selectors
// plan.Request accepts. The two vocabularies are not the same one:
//
//   - A concrete tool ("claude") is already a selector; it passes through.
//   - A composite ("claude+opencode") is what plan.mergeTool writes when several
//     tools share one path. It replays as EXACTLY those tools. Mapping it to "all"
//     would be wrong in both directions: "all" is detection-sensitive, so it can
//     plan for a tool the install never touched, or drop one it did.
//   - "agnostic" is a RECIPE-row label (internal/recipe). On an artifact row it is
//     invalid state, and there is no honest expansion: "" and "all" both invent
//     targets. It yields no tools, and the caller warns.
func replayTools(label string) []string {
	if label == "" || label == recipe.TargetAgnostic {
		return nil
	}
	return strings.Split(label, "+")
}

// deployDiffs runs each RECORDED IDENTITY through the install spine (plan.Compute)
// to learn where it WOULD land and what bytes it WOULD write. This is the single
// source of truth for artifact placement — scan asks the same question install
// answers, via the same code, at the same tool and scope install used.
//
// It takes an explicit identity list rather than the whole catalog on purpose: pass 1
// only needs the artifacts that are actually INSTALLED (the ones with a state row),
// and computing diffs for the rest would both waste work and try to read source that
// a remote catalog never fetched. Artifacts only: a recipe FETCHes a binary (verified
// against its pin by classifyFetch on every run), and a plugin is registered through
// the tool's own CLI — neither is a file Patronus renders from a source it can diff.
//
// It computes ONE IDENTITY AT A TIME. A single artifact that cannot be transformed
// must not blind the drift check for every other one; per-identity isolation degrades
// to "we could not read that one" instead of "we saw nothing".
func deployDiffs(cat *registry.Catalog, inv *scan.Inventory, adapters []*manifest.Adapter, wd string, idents []artifactInstallKey, warnf func(string, ...any)) map[artifactInstallKey][]diff.FileDiff {
	env := os.LookupEnv
	req := plan.Request{
		Catalog:   cat,
		Inventory: inv,
		Adapters:  adapterMap(adapters),
		Resolver:  toolpath.New(env, toolpath.HomeDir(env), wd),
	}

	out := make(map[artifactInstallKey][]diff.FileDiff, len(idents))
	for _, k := range idents {
		tools := replayTools(k.tool)
		if len(tools) == 0 {
			// No honest selector to replan at. Say so: a swallowed identity here would
			// turn a false ORPHANED into a silent one, which is the worse failure.
			if warnf != nil {
				warnf("drift: %s records tool %q, which is not an artifact target; cannot reconcile it", k.artifact, k.tool)
			}
			continue
		}
		for _, tool := range tools {
			r := req
			r.Names = []string{k.artifact}
			r.Tool = tool
			r.Scope = k.scope
			cs, err := plan.Compute(r)
			if err != nil {
				// Not fatal: this identity simply cannot be compared. Every other one
				// still can, and a partial drift report beats a silent one.
				if warnf != nil {
					warnf("drift: cannot resolve %s (%s/%s): %v", k.artifact, tool, k.scope, err)
				}
				continue
			}
			// Compute classifies Action against the real filesystem but never rewrites
			// After — After stays the bytes the catalog WOULD write, which is exactly
			// the side of the comparison drift needs.
			out[k] = append(out[k], cs.Diffs...)
		}
	}
	return out
}

// wouldWrite is what the catalog would put at one path: the bytes, and the
// artifact that owns them (so an unmanaged shadow can be reported by name).
type wouldWrite struct {
	source []byte
	item   string
}

// wouldKey identifies one would-deploy result by the IDENTITY that produced it plus
// the path it lands on. Keying by path alone is the trap this closes: a
// second identity landing on the same path would silently answer for the first, and
// with a map source the winner varied run to run.
type wouldKey struct {
	artifact string
	tool     string
	scope    string
	path     string
}

// sectionKey identifies one fenced APPEND section within a shared composed file,
// under the identity that would fold it in.
type sectionKey struct {
	artifact string
	tool     string
	scope    string
	path     string
	section  string
}

// wouldDeploy maps each identity's deploy paths -> what it would write there (for
// whole-file STALE/OK), AND each composed APPEND section -> the body the catalog
// would fold in there now (for per-section reconciliation of CLAUDE.md/AGENTS.md).
// deployDiffs computes one identity at a time, so each artifact's own diff carries
// ITS section's Body — which is exactly the per-section source a fold cannot recover
// from the merged whole file.
func wouldDeploy(cat *registry.Catalog, inv *scan.Inventory, adapters []*manifest.Adapter, wd string, idents []artifactInstallKey, warnf func(string, ...any)) (map[wouldKey]wouldWrite, map[sectionKey][]byte) {
	byIdent := deployDiffs(cat, inv, adapters, wd, idents, warnf)
	out := map[wouldKey]wouldWrite{}
	sections := map[sectionKey][]byte{}
	// Walk idents (sorted) rather than the map, so a later identity overwriting an
	// earlier one on the same key is deterministic.
	for _, k := range idents {
		for _, d := range byIdent[k] {
			if d.IsDir {
				continue
			}
			out[wouldKey{artifact: k.artifact, tool: k.tool, scope: k.scope, path: d.Path}] = wouldWrite{source: d.After, item: d.Artifact}
			// Capture the section body whenever the diff carries one — including when
			// plan.Compute has already downgraded the APPEND to SKIP because the section
			// is present and unchanged (the reclassify keeps Section intact, only Action
			// changes). Gating on Action==Append would miss exactly the OK case.
			if d.Section != nil {
				sections[sectionKey{artifact: k.artifact, tool: k.tool, scope: k.scope, path: d.Path, section: d.Section.Name}] = d.Section.Body
			}
		}
	}
	return out, sections
}

// lookupWould finds what a recorded row's OWN identity would write at its path.
// Recipe rows are added under their name with no tool/scope (a recipe is planned
// once, not per artifact identity), so a miss falls back to the name-only key.
func lookupWould(would map[wouldKey]wouldWrite, k artifactInstallKey, path string) (wouldWrite, bool) {
	if w, ok := would[wouldKey{artifact: k.artifact, tool: k.tool, scope: k.scope, path: path}]; ok {
		return w, true
	}
	w, ok := would[wouldKey{artifact: k.artifact, path: path}]
	return w, ok
}

// lookupWouldSection is lookupWould for one fenced section of a composed file.
func lookupWouldSection(sections map[sectionKey][]byte, k artifactInstallKey, path, section string) ([]byte, bool) {
	b, ok := sections[sectionKey{artifact: k.artifact, tool: k.tool, scope: k.scope, path: path, section: section}]
	return b, ok
}

// recipeMergeSources computes the would-deploy bytes for each installed recipe's
// MERGE rows (an MCP entry folded into a tool's own config, e.g. .claude.json).
// scan's artifact-only wouldDeploy never sees these, so without them a recipe's
// MERGE row reaches drift.Classify with hasSource=false and is misread as
// ORPHANED-STATE. FETCH/EXEC rows are skipped: a fetched binary has no
// diffable source (classifyFetch verifies it against its pin) and an EXEC is an
// advisory Patronus never ran.
func recipeMergeSources(cat *registry.Catalog, adapters map[string]*manifest.Adapter, res toolpath.Resolver, names []string, warnf func(string, ...any)) map[string]wouldWrite {
	out := map[string]wouldWrite{}
	for _, name := range names {
		rec := findRecipe(cat, name)
		if rec == nil {
			continue
		}
		diffs, err := recipe.Compute(recipe.Request{
			Recipe:   rec.Manifest,
			Adapters: adapters,
			Resolver: res,
			Tool:     "all", // every wired tool; per-path dest keys keep tools distinct
			Scope:    "global",
			Warnf:    warnf,
		})
		if err != nil {
			warnf("drift: cannot compute recipe %q source: %v", name, err)
			continue
		}
		for _, d := range diffs {
			if d.Action == diff.Merge {
				// Accumulate: two recipes merging into one config each contribute, and
				// keeping only the last made the comparison source one recipe's
				// standalone bytes. Fold each onto the previous result.
				if prev, ok := out[d.Path]; ok && d.Setting != nil {
					if folded, err := adapter.ApplySettingEdit(prev.source, d.Setting); err == nil {
						out[d.Path] = wouldWrite{source: folded, item: prev.item}
						continue
					}
				}
				out[d.Path] = wouldWrite{source: d.After, item: d.Artifact}
			}
		}
	}
	return out
}

// classifySettingEdit reconciles ONE contributor's edit against a composed config:
// is its dotted path present, and does it hold the value the edit would write? A
// sibling artifact's key on the same file is none of this contributor's business,
// which is what makes this correct where a whole-file checksum is not.
func classifySettingEdit(current []byte, e *diff.SettingEdit) drift.Verdict {
	refolded, err := adapter.ApplySettingEdit(current, e)
	if err != nil {
		// An unparseable config is not this contributor's drift to report.
		return drift.OK
	}
	if bytes.Equal(normalizeJSON(refolded), normalizeJSON(current)) {
		// Re-applying our edit changes nothing: our claim is already satisfied.
		return drift.OK
	}
	return drift.Stale
}

// normalizeJSON round-trips bytes through unmarshal/marshal so key ordering and
// whitespace do not read as drift. Both sides of the compare need it, not just
// one: ApplySettingEdit returns bytes that went through serializeConfig, while
// the on-disk side is whatever is there. After a Patronus write those agree, but
// after any user reformat they would not, and an unnormalized compare would then
// report a permanent false STALE for exactly the reformat case this reconcile
// exists to stop reporting. Bytes that do not parse as JSON pass through
// unchanged (a TOML config compares literally, which is what it already did).
func normalizeJSON(b []byte) []byte {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return b
	}
	out, err := json.Marshal(v)
	if err != nil {
		return b
	}
	return out
}

// readIfExists returns a file's bytes and whether it is there. An unreadable file
// counts as absent — we cannot classify bytes we cannot read.
func readIfExists(path string) ([]byte, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return b, true
}

// loadAdapters reads the adapter definitions from a checkout's adapters/ dir. When
// that dir is absent or empty (the installed-binary case, which has no checkout),
// it falls back to the adapters embedded in the binary — kept in lockstep with the
// adapter engine via internal/adapter/builtin.
func loadAdapters(dir string) ([]*manifest.Adapter, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return builtin.Adapters()
	}
	var out []*manifest.Adapter
	for _, path := range matches {
		ad, err := manifest.LoadAdapter(path)
		if err != nil {
			return nil, err
		}
		out = append(out, ad)
	}
	return out, nil
}
