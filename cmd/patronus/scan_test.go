package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/scan"
)

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
