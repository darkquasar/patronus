package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/profile"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func TestCodexUsesSharedInstallSpine(t *testing.T) {
	root := newRootCmd()
	for _, cmd := range root.Commands() {
		for _, name := range append([]string{cmd.Name()}, cmd.Aliases...) {
			if name == "install-codex" {
				t.Fatal("Codex must use ordinary install")
			}
		}
	}
	// Check actual calls, not comments that happen to name the shared functions.
	f, err := parser.ParseFile(token.NewFileSet(), "install.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	calls := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if receiver, ok := selector.X.(*ast.Ident); ok {
			calls[receiver.Name+"."+selector.Sel.Name] = true
		}
		return true
	})
	for _, name := range []string{"profile.Resolve", "plan.Compute", "recipe.Compute", "plan.Finalize", "plan.AdmitSettings", "app.Apply"} {
		if !calls[name] {
			t.Errorf("ordinary install lost shared call %s", name)
		}
	}
}

func TestCodexFlowEmitsSharedChangeSet(t *testing.T) {
	for _, scope := range []string{"local", "global"} {
		t.Run(scope, func(t *testing.T) {
			home, project, source := t.TempDir(), t.TempDir(), t.TempDir()
			res := toolpath.New(func(k string) (string, bool) {
				if k == "HOME" {
					return home, true
				}
				return "", false
			}, home, project)
			cat := &registry.Catalog{}
			for _, name := range []string{"invented-one-cx", "invented-two-cx"} {
				dir := filepath.Join(source, name)
				if err := os.Mkdir(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "INSTRUCTIONS.md"), []byte("# "+name+"\nInvented guidance\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				cat.Artifacts = append(cat.Artifacts, registry.ArtifactEntry{Manifest: &manifest.Artifact{Meta: manifest.Meta{Family: manifest.FamilyArtifact, Name: name, Version: "1.0.0"}, Type: manifest.TypeInstruction, Entry: "INSTRUCTIONS.md", Targets: []string{"codex"}}, Source: registry.Source{LocalDir: dir}})
			}
			rec, err := manifest.DecodeRecipe([]byte("apiVersion: patronus/v2\nfamily: recipe\nrole: tools\nname: invented-mcp\nversion: 1.0.0\nwire:\n  method: merge\n  actor: patronus\n  tools: [codex]\n  mcp: {transport: http, url: 'https://example.invalid/mcp'}\n"))
			if err != nil {
				t.Fatal(err)
			}
			cat.Recipes = []registry.RecipeEntry{{Manifest: rec}}
			cat.Profiles = []registry.ProfileEntry{{Manifest: &manifest.Profile{Meta: manifest.Meta{Family: manifest.FamilyProfile, Name: "invented-profile-cx"}, Layers: manifest.ProfileLayers{Instructions: manifest.StringList{"invented-one-cx", "invented-two-cx"}, Tools: manifest.StringList{"invented-mcp"}}}}}
			resolved, err := profile.Resolve(cat, "invented-profile-cx", "codex")
			if err != nil || len(resolved.Warnings) != 0 {
				t.Fatalf("resolve: %+v, %v", resolved, err)
			}
			ad, err := manifest.LoadAdapter(filepath.Join("..", "..", "adapters", "codex.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			// Exercise the actual mixed artifact/recipe dispatcher, which reaches
			// plan.Compute and plan.Finalize. No paid calls or config writes.
			cs, err := computePlan(planInputs{cat: cat, adapters: map[string]*manifest.Adapter{"codex": ad}, res: res, names: resolved.Names(), tool: "codex", scope: scope})
			if err != nil {
				t.Fatal(err)
			}
			if !cs.DryRun || len(cs.Diffs) != 2 {
				t.Fatalf("shared ChangeSet = %+v", cs)
			}
			var sections, merges int
			for _, d := range cs.Diffs {
				if d.Tool != "codex" || d.Scope != scope {
					t.Fatalf("foreign diff: %+v", d)
				}
				if d.Section != nil {
					sections++
					if len(d.Contrib) != 1 || !strings.Contains(string(d.After), "invented-one-cx") || !strings.Contains(string(d.After), "invented-two-cx") {
						t.Fatalf("Finalize did not compose sections: %+v", d)
					}
				}
				if d.Setting != nil {
					merges++
					if d.Action != diff.Merge {
						t.Fatalf("MCP ownership evidence not MERGE: %+v", d)
					}
				}
				if _, err := os.Stat(d.Path); !os.IsNotExist(err) {
					t.Fatalf("preview wrote %s: %v", d.Path, err)
				}
			}
			if sections != 1 || merges != 1 {
				t.Fatalf("sections=%d structured merges=%d", sections, merges)
			}
		})
	}
}

// A synthetic profile reaches the ordinary shared resolver and lock command.
func TestCodexCoreProfileLockUsesSharedSpine(t *testing.T) {
	root := fixtureCatalog(t)
	profilePath := filepath.Join(root, "profiles/fix-all.yaml")
	codexWrite(t, profilePath, append(mustRead(t, profilePath), []byte("\n  tools: [fix-mcp-bin, fix-mcp-two]\n")...))
	cat, err := registry.NewLocalRegistry(root).Catalog(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := profile.Resolve(cat, "fix-all", "codex")
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Warnings) != 0 {
		t.Fatalf("unresolved cx closure: %v", resolved.Warnings)
	}
	l, err := lock.FromResolved(cat, resolved, "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if l.Profile != "fix-all" || len(l.Entries) != len(resolved.Items) {
		t.Fatalf("lock differs from resolver: %+v", l)
	}
	path := filepath.Join(t.TempDir(), "patronus.lock")
	if err := lock.Save(path, l); err != nil {
		t.Fatal(err)
	}
	got, err := lock.Load(path)
	if err != nil || len(got.Entries) != len(l.Entries) {
		t.Fatalf("lock roundtrip: %+v %v", got, err)
	}
	for _, e := range got.Entries {
		if e.Source != "registry" || e.Version == "" || !strings.HasPrefix(e.SHA256, "sha256:") {
			t.Errorf("incomplete pin: %+v", e)
		}
	}
	isolated := root
	t.Chdir(isolated)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CODEX_HOME", t.TempDir())
	out, warnings, err := runLock(t, "--profile", "fix-all", "--target", "codex", "--local-registry")
	if err != nil || warnings != "" {
		t.Fatalf("ordinary lock command: %s %s %v", out, warnings, err)
	}
	cliLock, err := lock.Load(filepath.Join(isolated, "patronus.lock"))
	if err != nil || len(cliLock.Entries) != len(l.Entries) {
		t.Fatalf("CLI lock closure: %+v %v", cliLock, err)
	}
}

func TestCodexCoreProfilePreviewKeepsRecipes(t *testing.T) {
	root := fixtureCatalog(t)
	profilePath := filepath.Join(root, "profiles/fix-all.yaml")
	codexWrite(t, profilePath, append(mustRead(t, profilePath), []byte("\n  tools: [fix-mcp-bin, fix-mcp-two]\n")...))
	cat, err := registry.NewLocalRegistry(root).Catalog(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := profile.Resolve(cat, "fix-all", "codex")
	if err != nil || len(resolved.Warnings) != 0 {
		t.Fatalf("resolve: %+v %v", resolved, err)
	}
	t.Chdir(root)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CODEX_HOME", t.TempDir())
	for _, scope := range []string{"--global", "--local"} {
		t.Run(scope, func(t *testing.T) {
			out, warnings, err := runInstall(t, "--profile", "fix-all", "--target", "codex", scope, "--dry-run", "--local-registry")
			if err != nil {
				t.Fatalf("preview: %s %s %v", out, warnings, err)
			}
			if strings.Contains(warnings, "not resolvable") {
				t.Fatalf("unresolved warnings: %s", warnings)
			}
			for _, name := range []string{"fix-bin", "fix-archive-bin", "fix-mcp-bin", "fix-mcp-two", "dry run"} {
				if !strings.Contains(out, name) {
					t.Errorf("recipe preview missing %s: %s", name, out)
				}
			}
		})
	}
}
