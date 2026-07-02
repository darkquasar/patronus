package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/render"
)

func TestPluginStatusFromLock(t *testing.T) {
	wd := t.TempDir()
	if err := lock.Save(filepath.Join(wd, "patronus.lock"), &lock.Lock{Version: lock.Version, Entries: []lock.Entry{
		{Name: "superpowers", Kind: "plugin", Status: lock.StatusVerified},
		{Name: "team-research", Kind: "artifact"},
	}}); err != nil {
		t.Fatal(err)
	}
	got := pluginStatusFromLock(wd)
	if got["superpowers"] != lock.StatusVerified {
		t.Errorf("superpowers status = %q, want verified", got["superpowers"])
	}
	if _, ok := got["team-research"]; ok {
		t.Errorf("artifact entry must not appear in plugin status map: %v", got)
	}
}

func TestPluginStatusFromLockNoLock(t *testing.T) {
	if got := pluginStatusFromLock(t.TempDir()); got != nil {
		t.Errorf("no lock should yield nil status map, got %v", got)
	}
}

// TestListPluginsShowsStatus proves the plugin section renders each plugin's
// reconciliation status when a lock is present.
func TestListPluginsShowsStatus(t *testing.T) {
	cat := &registry.Catalog{Plugins: []registry.PluginEntry{{Manifest: &manifest.Plugin{
		Meta: manifest.Meta{Family: manifest.FamilyPlugin, Name: "superpowers", Description: "power tools"},
	}}}}
	view := render.CatalogView{Plugins: true, PluginStatus: map[string]string{"superpowers": lock.StatusVerified}}
	var b bytes.Buffer
	render.PrintCatalog(&b, cat, view)
	out := b.String()
	if !strings.Contains(out, "superpowers") || !strings.Contains(out, "verified") {
		t.Errorf("expected plugin name + status in output, got:\n%s", out)
	}
	if !strings.Contains(out, "STATUS") {
		t.Errorf("expected a STATUS column header, got:\n%s", out)
	}
}
