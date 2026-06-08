package builtin

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEmbeddedMatchesRoot guards against drift: the embedded adapter yaml must be
// byte-identical to the canonical copies authored at the repo root adapters/.
// If this fails, re-copy the root adapters into this package (the root is the
// authoring source).
func TestEmbeddedMatchesRoot(t *testing.T) {
	for _, tool := range []string{"claude", "codex", "opencode"} {
		name := tool + ".yaml"
		embedded, err := files.ReadFile(name)
		if err != nil {
			t.Fatalf("embedded %s: %v", name, err)
		}
		rootPath := filepath.Join("..", "..", "..", "adapters", name)
		root, err := os.ReadFile(rootPath)
		if err != nil {
			t.Fatalf("root %s: %v", rootPath, err)
		}
		if string(embedded) != string(root) {
			t.Errorf("%s drifted from %s — re-copy the root adapter into internal/adapter/builtin/", name, rootPath)
		}
	}
}

func TestAdaptersParse(t *testing.T) {
	ads, err := Adapters()
	if err != nil {
		t.Fatal(err)
	}
	if len(ads) != 3 {
		t.Fatalf("got %d adapters, want 3", len(ads))
	}
	seen := map[string]bool{}
	for _, a := range ads {
		seen[a.Tool] = true
	}
	for _, tool := range []string{"claude", "codex", "opencode"} {
		if !seen[tool] {
			t.Errorf("missing embedded adapter for %s", tool)
		}
	}
}
