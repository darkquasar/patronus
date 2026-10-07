package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/registry"
	"github.com/darkquasar/patronus/internal/toolpath"
)

// Use the ordinary named-item planning spine and invented home/project roots.
// No live CLI/backend calls, deployment or native activation are involved.
func TestCodexRemainingSkillPortsPlaceSidecars(t *testing.T) {
	root := fixtureSkillBundle(t)
	catalog, err := registry.NewLocalRegistry(root).Catalog(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	ad, err := manifest.LoadAdapter(filepath.Join(root, "adapters/codex.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	names := []string{"fix-router", "fix-review"}
	for _, scope := range []string{"local", "global"} {
		t.Run(scope, func(t *testing.T) {
			home, project := t.TempDir(), t.TempDir()
			res := toolpath.New(func(key string) (string, bool) {
				if key == "HOME" {
					return home, true
				}
				return "", false
			}, home, project)
			cs, err := computePlan(planInputs{cat: catalog, adapters: map[string]*manifest.Adapter{"codex": ad}, res: res, names: names, tool: "codex", scope: scope})
			if err != nil {
				t.Fatal(err)
			}
			if !cs.DryRun {
				t.Fatal("ordinary preview must not deploy")
			}
			emitted := map[string][]byte{}
			for _, d := range cs.Diffs {
				if d.Tool != "codex" || d.Scope != scope {
					t.Fatalf("foreign diff: %+v", d)
				}
				emitted[filepath.Clean(d.Path)] = d.After
				if _, err := os.Stat(d.Path); !os.IsNotExist(err) {
					t.Fatalf("preview wrote %s: %v", d.Path, err)
				}
			}
			// Scope roots are selected by the actual adapter, not asserted as
			// native qualification. Root migration has a separate owner.
			marker := strings.ReplaceAll(ad.Layout.Skill.ForScope(scope).Path, "{name}", "invented-cx")
			skillRoot := filepath.Dir(filepath.Dir(res.ResolveMarker(marker, "codex", scope)))
			bodyRoot := skillRoot
			if scope == "local" {
				bodyRoot = res.RelativeTo(skillRoot)
			}
			for _, name := range names {
				dir := filepath.Join(root, "artifacts/skills", name)
				m, err := manifest.LoadArtifact(filepath.Join(dir, "patronus.yaml"))
				if err != nil {
					t.Fatal(err)
				}
				for _, file := range append([]string{m.Entry}, m.Files...) {
					source, err := os.ReadFile(filepath.Join(dir, file))
					if err != nil {
						t.Fatal(err)
					}
					path := filepath.Join(skillRoot, name, file)
					body, ok := emitted[path]
					if !ok {
						t.Fatalf("sidecar not planned: %s", path)
					}
					if bytes.Contains(body, []byte("{skillDir}")) || bytes.Contains(body, []byte("{skillsDir}")) {
						t.Errorf("unresolved source placeholder: %s", path)
					}
					if file == "LICENSE" || file == "NOTICE" {
						if !bytes.Equal(source, body) {
							t.Errorf("license/provenance bytes changed in transform: %s", path)
						}
					}
					for _, route := range regexp.MustCompile(`\{skillDir\}/([a-zA-Z0-9./_-]+)`).FindAllStringSubmatch(string(source), -1) {
						dest := filepath.Join(skillRoot, name, route[1])
						if emitted[dest] == nil || !strings.Contains(string(body), filepath.Join(bodyRoot, name, route[1])) {
							t.Errorf("local sidecar route not placed/resolved: %s -> %s", path, dest)
						}
					}
					for _, route := range regexp.MustCompile(`\{skillsDir\}/([a-z0-9-]+)/SKILL\.md`).FindAllStringSubmatch(string(source), -1) {
						dest := filepath.Join(bodyRoot, route[1], "SKILL.md")
						if !strings.Contains(string(body), dest) {
							t.Errorf("companion route not resolved: %s -> %s", path, dest)
						}
					}
				}
			}
		})
	}
}
