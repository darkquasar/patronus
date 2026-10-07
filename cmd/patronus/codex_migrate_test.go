package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/state"
)

type codexMigrateFixture struct {
	home, project, codexHome string
}

func newCodexMigrateFixture(t *testing.T, codexHome string) codexMigrateFixture {
	t.Helper()
	home, project := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	if codexHome != "" {
		codexHome = filepath.Join(home, codexHome)
	}
	t.Setenv("CODEX_HOME", codexHome)
	t.Chdir(project)
	return codexMigrateFixture{home: home, project: project, codexHome: codexHome}
}

func (f codexMigrateFixture) root(scope string) string {
	if scope == "global" {
		return f.home
	}
	return f.project
}

// seed writes an invented recorded legacy skill with sidecars and records it.
func (f codexMigrateFixture) seed(t *testing.T, scope, legacyRoot, name string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{
		"SKILL.md":               []byte("---\nname: " + name + "\ndescription: invented migration fixture\n---\nbody\n"),
		"references/guide.md":    []byte("invented sidecar\n"),
		"scripts/run-fixture.sh": []byte("#!/bin/sh\necho invented\n"),
	}
	var diffs []diff.FileDiff
	for rel, raw := range files {
		path := filepath.Join(legacyRoot, name, rel)
		codexWrite(t, path, raw)
		diffs = append(diffs, diff.FileDiff{Path: path, Action: diff.Create, After: raw, Artifact: name, Type: "skill", Tool: "codex", Scope: scope, Version: "1.0.0"})
	}
	sp := filepath.Join(f.root(scope), ".patronus/state.json")
	s, err := state.Load(sp)
	if err != nil {
		t.Fatal(err)
	}
	state.Merge(s, state.FromChangeSet(diffs, "invented-time"))
	if err := state.Save(sp, s); err != nil {
		t.Fatal(err)
	}
	return files
}

func (f codexMigrateFixture) run(scope string, deploy bool, hooks codexMigrateOpts) (string, error) {
	var out bytes.Buffer
	hooks.home, hooks.project, hooks.scopes, hooks.deploy = f.home, f.project, []string{scope}, deploy
	err := runCodexMigrate(&out, hooks)
	return out.String(), err
}

func codexStateFiles(t *testing.T, path string) map[string]string {
	t.Helper()
	s, err := state.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, it := range s.Items {
		for _, f := range it.Files {
			out[f.Path] = it.Artifact
		}
	}
	return out
}

func TestCodexOwnedRootMigrationRetiresOnlyVerifiedSources(t *testing.T) {
	for _, tc := range []struct{ name, scope, codexHome, legacy string }{
		{"global-default", "global", "", ".codex/skills"},
		{"global-codex-home", "global", "custom-codex", "custom-codex/skills"},
		{"local", "local", "", ".codex/skills"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCodexMigrateFixture(t, tc.codexHome)
			legacy := filepath.Join(f.root(tc.scope), tc.legacy)
			files := f.seed(t, tc.scope, legacy, "fixture-cx")
			other := "local"
			if tc.scope == "local" {
				other = "global"
			}
			otherState := filepath.Join(f.root(other), ".patronus/state.json")
			codexWrite(t, otherState, []byte("{\"version\":2,\"items\":[]}\n"))
			otherBefore := mustRead(t, otherState)
			sp := filepath.Join(f.root(tc.scope), ".patronus/state.json")
			stateBefore := mustRead(t, sp)

			out, err := f.run(tc.scope, false, codexMigrateOpts{})
			if err != nil || !strings.Contains(out, "MIGRATE fixture-cx") || !strings.Contains(out, "Preview only") {
				t.Fatalf("preview: %v\n%s", err, out)
			}
			if !bytes.Equal(mustRead(t, sp), stateBefore) {
				t.Fatal("preview changed state")
			}
			if _, err := os.Stat(filepath.Join(f.root(tc.scope), ".agents/skills/fixture-cx")); !os.IsNotExist(err) {
				t.Fatalf("preview wrote destination: %v", err)
			}

			var order []string
			if _, err := f.run(tc.scope, true, codexMigrateOpts{beforeWrite: func(d diff.FileDiff) error {
				order = append(order, string(d.Action))
				return nil
			}, saveState: func(p string, s *state.State) error {
				order = append(order, "SAVE")
				return state.Save(p, s)
			}}); err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(order, ","); got != "CREATE,CREATE,CREATE,SAVE,DELETE,DELETE,DELETE,SAVE" {
				t.Fatalf("destination verification/persistence must precede retirement: %s", got)
			}
			dest := filepath.Join(f.root(tc.scope), ".agents/skills/fixture-cx")
			owned := codexStateFiles(t, sp)
			for rel, raw := range files {
				if got := mustRead(t, filepath.Join(dest, rel)); !bytes.Equal(got, raw) {
					t.Fatalf("%s not relocated", rel)
				}
				if owned[filepath.Join(dest, rel)] != "fixture-cx" || owned[filepath.Join(legacy, "fixture-cx", rel)] != "" {
					t.Fatalf("ownership not moved for %s: %v", rel, owned)
				}
			}
			if _, err := os.Stat(filepath.Join(legacy, "fixture-cx")); !os.IsNotExist(err) {
				t.Fatalf("verified legacy dir not retired: %v", err)
			}
			if !bytes.Equal(mustRead(t, otherState), otherBefore) {
				t.Fatal("unselected scope changed")
			}
			if out, err := f.run(tc.scope, false, codexMigrateOpts{}); err != nil || !strings.Contains(out, "no recorded legacy") {
				t.Fatalf("rerun not idempotent: %v %s", err, out)
			}
		})
	}
}

func TestCodexOwnedRootMigrationPreservesUnownedAndEditedFiles(t *testing.T) {
	cases := map[string]func(t *testing.T, f codexMigrateFixture, legacy string) string{
		"edited-source": func(t *testing.T, f codexMigrateFixture, legacy string) string {
			codexWrite(t, filepath.Join(legacy, "fixture-cx/references/guide.md"), []byte("user edit\n"))
			return "edited source"
		},
		"unowned-sidecar": func(t *testing.T, f codexMigrateFixture, legacy string) string {
			codexWrite(t, filepath.Join(legacy, "fixture-cx/notes.md"), []byte("user notes\n"))
			return "unowned file"
		},
		"destination-collision": func(t *testing.T, f codexMigrateFixture, legacy string) string {
			codexWrite(t, filepath.Join(f.project, ".agents/skills/fixture-cx/SKILL.md"), []byte("---\nname: fixture-cx\ndescription: user\n---\n"))
			return "collision"
		},
		"global-same-name-unowned": func(t *testing.T, f codexMigrateFixture, legacy string) string {
			codexWrite(t, filepath.Join(f.home, ".agents/skills/fixture-cx/SKILL.md"), []byte("---\nname: fixture-cx\ndescription: user\n---\n"))
			return "collision"
		},
		"symlinked-sidecar": func(t *testing.T, f codexMigrateFixture, legacy string) string {
			target := filepath.Join(legacy, "fixture-cx/references/guide.md")
			if err := os.Remove(target); err != nil {
				t.Fatal(err)
			}
			outside := filepath.Join(f.home, "outside.md")
			codexWrite(t, outside, []byte("invented sidecar\n"))
			if err := os.Symlink(outside, target); err != nil {
				t.Fatal(err)
			}
			return "symlink"
		},
		"missing-source": func(t *testing.T, f codexMigrateFixture, legacy string) string {
			if err := os.Remove(filepath.Join(legacy, "fixture-cx/scripts/run-fixture.sh")); err != nil {
				t.Fatal(err)
			}
			return "missing source"
		},
		"foreign-overlap-other-scope": func(t *testing.T, f codexMigrateFixture, legacy string) string {
			s := &state.State{Version: state.Version, Items: []state.Item{{Artifact: "other", Tool: "pi", Scope: "global", Files: []state.FileState{{Path: filepath.Join(f.project, ".agents/skills/fixture-cx/SKILL.md"), Action: "CREATE", Checksum: "sha256:00"}}}}}
			if err := state.Save(filepath.Join(f.home, ".patronus/state.json"), s); err != nil {
				t.Fatal(err)
			}
			return "incompatible ownership"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := newCodexMigrateFixture(t, "")
			legacy := filepath.Join(f.project, ".codex/skills")
			f.seed(t, "local", legacy, "fixture-cx")
			want := mutate(t, f, legacy)
			sp := filepath.Join(f.project, ".patronus/state.json")
			before := mustRead(t, sp)
			snapshot := codexTree(t, legacy)
			if _, err := f.run("local", true, codexMigrateOpts{}); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("want refusal %q, got %v", want, err)
			}
			if !bytes.Equal(mustRead(t, sp), before) || codexTree(t, legacy) != snapshot {
				t.Fatal("refusal changed ownership or legacy files")
			}
			if name != "destination-collision" {
				if _, err := os.Stat(filepath.Join(f.project, ".agents/skills/fixture-cx")); !os.IsNotExist(err) {
					t.Fatalf("refusal wrote destination: %v", err)
				}
			}
		})
	}
	t.Run("unowned-legacy-skill-not-adopted", func(t *testing.T) {
		f := newCodexMigrateFixture(t, "")
		codexWrite(t, filepath.Join(f.home, ".codex/skills/user-cx/SKILL.md"), []byte("---\nname: user-cx\ndescription: user\n---\n"))
		if out, err := f.run("global", true, codexMigrateOpts{}); err != nil || !strings.Contains(out, "no recorded legacy") {
			t.Fatalf("unowned skill adopted: %v %s", err, out)
		}
		if _, err := os.Stat(filepath.Join(f.home, ".codex/skills/user-cx/SKILL.md")); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("deploy-needs-scope-and-no-force", func(t *testing.T) {
		newCodexMigrateFixture(t, "")
		for _, args := range [][]string{{"--deploy"}, {"--local", "--deploy", "--force"}} {
			cmd := newMigrateCodexSkillsCmd()
			cmd.SetArgs(args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			if err := cmd.Execute(); err == nil {
				t.Fatalf("%v accepted", args)
			}
		}
	})
}

func TestCodexOwnedRootMigrationFailureRetainsOldOwnership(t *testing.T) {
	injected := errors.New("invented failure")
	for _, mode := range []string{"destination-write", "ownership-save", "retirement"} {
		t.Run(mode, func(t *testing.T) {
			f := newCodexMigrateFixture(t, "")
			legacy := filepath.Join(f.home, ".codex/skills")
			files := f.seed(t, "global", legacy, "fixture-cx")
			sp := filepath.Join(f.home, ".patronus/state.json")
			creates, deletes := 0, 0
			hooks := codexMigrateOpts{beforeWrite: func(d diff.FileDiff) error {
				switch d.Action {
				case diff.Create:
					creates++
					if mode == "destination-write" && creates == 2 {
						return injected
					}
				case diff.Delete:
					deletes++
					if mode == "retirement" && deletes == 2 {
						return injected
					}
				}
				return nil
			}}
			if mode == "ownership-save" {
				hooks.saveState = func(string, *state.State) error { return injected }
			}
			if _, err := f.run("global", true, hooks); err == nil || !strings.Contains(err.Error(), "retained") {
				t.Fatalf("failure not reported with retention: %v", err)
			}
			owned := codexStateFiles(t, sp)
			for rel, raw := range files {
				src := filepath.Join(legacy, "fixture-cx", rel)
				got, err := os.ReadFile(src)
				if os.IsNotExist(err) {
					if mode != "retirement" {
						t.Fatalf("%s source retired before verified ownership", rel)
					}
					if owned[src] != "" {
						t.Fatalf("retired source still recorded: %s", src)
					}
					continue
				}
				if !bytes.Equal(got, raw) || owned[src] != "fixture-cx" {
					t.Fatalf("old data/ownership lost for %s: %v", rel, owned)
				}
			}
			if mode == "ownership-save" && deletes != 0 {
				t.Fatal("retirement ran after failed ownership persistence")
			}
			if mode == "ownership-save" {
				return // unrecorded destinations stay unresolved; resume refuses below is not attempted
			}
			// Verified facts were persisted; a fresh run resumes to completion.
			if _, err := f.run("global", true, codexMigrateOpts{}); err != nil {
				t.Fatalf("resume after %s: %v", mode, err)
			}
			for rel, raw := range files {
				if !bytes.Equal(mustRead(t, filepath.Join(f.home, ".agents/skills/fixture-cx", rel)), raw) {
					t.Fatalf("resume lost %s", rel)
				}
			}
			if _, err := os.Stat(filepath.Join(legacy, "fixture-cx")); !os.IsNotExist(err) {
				t.Fatalf("resume left legacy dir: %v", err)
			}
		})
	}
}

func codexTree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		b.WriteString(path + " " + info.Mode().String())
		if info.Mode().IsRegular() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			b.Write(raw)
		}
		b.WriteByte('\n')
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}
