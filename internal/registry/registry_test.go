package registry

import (
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
)

func TestCatalogHoldsPlugins(t *testing.T) {
	cat := &Catalog{
		Plugins: []PluginEntry{
			{Manifest: &manifest.Plugin{Meta: manifest.Meta{Name: "superpowers", Family: manifest.FamilyPlugin}}},
		},
	}
	if len(cat.Plugins) != 1 {
		t.Fatalf("plugins = %d, want 1", len(cat.Plugins))
	}
	if cat.Plugins[0].Manifest.Name != "superpowers" {
		t.Errorf("name = %s, want superpowers", cat.Plugins[0].Manifest.Name)
	}
}
