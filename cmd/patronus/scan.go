package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/darkquasar/patronus/internal/adapter/builtin"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/plugin"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/render"
	"github.com/darkquasar/patronus/internal/scan"
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
			root, err := registry.DiscoverRoot(wd)
			if err != nil {
				return err
			}
			adapters, err := loadAdapters(filepath.Join(root, "adapters"))
			if err != nil {
				return err
			}
			inv, err := scan.Scan(scan.Options{ProjectDir: wd, Adapters: adapters})
			if err != nil {
				return err
			}
			if jsonOutput {
				return render.JSON(cmd.OutOrStdout(), inv)
			}
			render.PrintInventory(cmd.OutOrStdout(), inv)

			// Reconcile the lock's plugin statuses against installed reality. Scan
			// reads tools; it never writes tool state. A missing lock, an
			// unavailable registry, or an unreachable tool CLI degrades gracefully
			// (statuses that cannot be confirmed stay unverified).
			warnf := func(f string, a ...any) { fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+f+"\n", a...) }
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
