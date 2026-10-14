package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/harness"
	"github.com/darkquasar/patronus/internal/manifest"
	"github.com/darkquasar/patronus/internal/scan"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func managedReferenceCopy(t *testing.T, root, tool, scope, name string) state.Item {
	t.Helper()
	owner := state.Item{Artifact: name, ItemVersion: "1.0.0", Type: "skill", Tool: tool, Scope: scope}
	for rel, raw := range map[string][]byte{
		"SKILL.md": []byte("---\nname: " + name + "\ndescription: invented reference\n---\nRead this scope's body.md.\n"),
		"body.md":  []byte("invented shared guidance\n"),
	} {
		path := filepath.Join(root, name, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0644); err != nil {
			t.Fatal(err)
		}
		owner.Files = append(owner.Files, state.FileState{Path: path, Action: "CREATE", Checksum: state.Checksum(raw)})
	}
	return owner
}

func TestManagedReferenceComputedMaskWiring(t *testing.T) {
	in, _ := maskFixture(t)
	in.tool, in.harness = "opencode", testRegistry(t)
	in.cat.Artifacts[0].Manifest.Targets = []string{"opencode"}
	ad, err := manifest.DecodeAdapter([]byte("apiVersion: patronus/v2\nfamily: adapter\ntool: opencode\nlayout:\n  setting:\n    project: {file: opencode.json, format: jsonc}\n"))
	if err != nil {
		t.Fatal(err)
	}
	in.adapters = map[string]*manifest.Adapter{"opencode": ad}
	home, project := in.res.ExpandHome("~"), in.res.ResolveMarker(".", "opencode", "local")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_CONTENT", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	name := "decision-clarity-reference"
	local := managedReferenceCopy(t, filepath.Join(project, ".claude/skills"), "claude", "local", name)
	global := managedReferenceCopy(t, filepath.Join(home, ".config/opencode/skills"), "opencode", "global", name)
	for root, owner := range map[string]state.Item{project: local, home: global} {
		if err := state.Save(filepath.Join(root, ".patronus/state.json"), &state.State{Version: state.Version, Items: []state.Item{owner}}); err != nil {
			t.Fatal(err)
		}
	}
	workflow := filepath.Join(project, ".claude/skills/workflow-cl/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(workflow), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workflow, []byte("---\nname: workflow-cl\ndescription: invented\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cs, err := computePlan(in)
	if err != nil {
		t.Fatalf("computed mask refused matching managed references: %v", err)
	}
	if len(cs.Diffs) != 1 || cs.Diffs[0].Setting == nil || cs.Diffs[0].Setting.Dotted != `permission.skill."*-cl"` {
		t.Fatalf("wrong computed mask: %+v", cs.Diffs)
	}
	got, exists, err := adapter.ReadDotted(cs.Diffs[0].After, manifest.FileTarget{File: "opencode.json", Format: "jsonc"}, `permission.skill."*-cl"`)
	if err != nil || !exists || got != "deny" {
		t.Fatalf("actual computed mask: %q %v", cs.Diffs[0].After, err)
	}
	for _, fault := range []string{"unmanaged global", "unmanaged local", "version", "partial", "edited sidecar", "extra file"} {
		t.Run(fault, func(t *testing.T) {
			localOwner, globalOwner := local, global
			globalEntry := filepath.Join(home, ".config/opencode/skills", name, "SKILL.md")
			localEntry := filepath.Join(project, ".claude/skills", name, "SKILL.md")
			sidecar := filepath.Join(filepath.Dir(globalEntry), "body.md")
			localItems, globalItems := []state.Item{localOwner}, []state.Item{globalOwner}
			switch fault {
			case "unmanaged global":
				globalItems = nil
			case "unmanaged local":
				localItems = nil
			case "version":
				globalItems[0].ItemVersion = "2.0.0"
			case "partial":
				globalItems[0].Files = nil
				for _, file := range globalOwner.Files {
					if file.Path != sidecar {
						globalItems[0].Files = append(globalItems[0].Files, file)
					}
				}
			case "edited sidecar":
				if err := os.WriteFile(sidecar, []byte("edited\n"), 0644); err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := os.WriteFile(sidecar, []byte("invented shared guidance\n"), 0644); err != nil {
						t.Error(err)
					}
				}()
			case "extra file":
				extra := filepath.Join(filepath.Dir(globalEntry), "unmanaged.txt")
				if err := os.WriteFile(extra, []byte("unmanaged\n"), 0644); err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := os.Remove(extra); err != nil {
						t.Error(err)
					}
				}()
			}
			for root, items := range map[string][]state.Item{project: localItems, home: globalItems} {
				if err := state.Save(filepath.Join(root, ".patronus/state.json"), &state.State{Version: state.Version, Items: items}); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := computePlan(in); err == nil || !strings.Contains(err.Error(), globalEntry) || !strings.Contains(err.Error(), localEntry) {
				t.Fatalf("unsafe computed reference %s: %v", fault, err)
			}
		})
	}
}

func TestManagedReferencePiAdmissionWiring(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(home, ".pi/agent"))
	t.Setenv("PI_PACKAGE_DIR", "")
	t.Setenv("PI_OFFLINE", "true")
	name := "rust-style-reference"
	local := managedReferenceCopy(t, filepath.Join(project, ".pi/skills"), "pi", "local", name)
	global := managedReferenceCopy(t, filepath.Join(home, ".agents/skills"), "codex", "global", name)
	for root, owner := range map[string]state.Item{project: local, home: global} {
		if err := state.Save(filepath.Join(root, ".patronus/state.json"), &state.State{Version: state.Version, Items: []state.Item{owner}}); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(project, ".pi/skills", name, "SKILL.md")
	raw := mustRead(t, path)
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Path: path, Type: "skill", Tool: "pi", Scope: "local", Artifact: name, Version: "1.0.0", Action: diff.Skip, Before: raw, After: raw}}}
	reg := testRegistry(t)
	d, _ := reg.Lookup("pi")
	res := toolpath.New(os.LookupEnv, home, project)
	if _, err := admitTarget(reg, d, cs, res, home, project, nil); err != nil {
		t.Fatalf("Pi admission refused matching managed Codex/Pi references: %v", err)
	}
	for _, fault := range []string{"unmanaged global", "unmanaged local", "version", "partial", "edited sidecar"} {
		t.Run(fault, func(t *testing.T) {
			localItems, globalItems := []state.Item{local}, []state.Item{global}
			globalEntry := filepath.Join(home, ".agents/skills", name, "SKILL.md")
			sidecar := filepath.Join(filepath.Dir(globalEntry), "body.md")
			switch fault {
			case "unmanaged global":
				globalItems = nil
			case "unmanaged local":
				localItems = nil
			case "version":
				globalItems[0].ItemVersion = "2.0.0"
			case "partial":
				globalItems[0].Files = nil
				for _, file := range global.Files {
					if file.Path != sidecar {
						globalItems[0].Files = append(globalItems[0].Files, file)
					}
				}
			case "edited sidecar":
				if err := os.WriteFile(sidecar, []byte("edited\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			for root, items := range map[string][]state.Item{project: localItems, home: globalItems} {
				if err := state.Save(filepath.Join(root, ".patronus/state.json"), &state.State{Version: state.Version, Items: items}); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := admitTarget(reg, d, cs, res, home, project, nil); err == nil || !strings.Contains(err.Error(), globalEntry) || !strings.Contains(err.Error(), path) {
				t.Fatalf("unsafe Pi reference %s: %v", fault, err)
			}
		})
	}
}

func TestManagedReferencePiForeignInventoryWiring(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	env := func(key string) (string, bool) {
		if key == "PI_OFFLINE" {
			return "true", true
		}
		return "", false
	}
	name := "decision-clarity-reference"
	local := managedReferenceCopy(t, filepath.Join(project, ".pi/skills"), "pi", "local", name)
	foreignRoot := filepath.Join(home, ".codex/skills")
	foreign := managedReferenceCopy(t, foreignRoot, "codex", "global", name)
	path := filepath.Join(project, ".pi/skills", name, "SKILL.md")
	raw := mustRead(t, path)
	reg := testRegistry(t)
	d, _ := reg.Lookup("pi")
	in := &harness.Admission{
		Home: home, Project: project, Resolver: toolpath.New(env, home, project), Env: env,
		Owners:  []state.Item{local, foreign},
		Roots:   []scan.SkillRoot{{Path: foreignRoot, Origin: "foreign codex", Name: scan.SkillDirName, DuplicatePolicy: scan.RefuseDifferentBytesVerifiedException}},
		Changes: &diff.ChangeSet{Diffs: []diff.FileDiff{{Path: path, Type: "skill", Tool: "pi", Scope: "local", Artifact: name, Version: "1.0.0", Action: diff.Skip, Before: raw, After: raw}}},
	}
	if err := harness.Admit(d, in); err != nil {
		t.Fatalf("prepared/foreign reference inventory refused: %v", err)
	}
	if len(in.Inventory) != 1 || len(in.Inventory[0].Content) == 0 {
		t.Fatalf("prepared/foreign copies did not reconcile once with bytes: %+v", in.Inventory)
	}
}

func TestManagedReferenceFreshSecondHostInstall(t *testing.T) {
	for _, name := range []string{"rust-style-reference", "writing-editorial"} {
		for _, pair := range [][2]string{{"claude", "opencode"}, {"codex", "opencode"}, {"codex", "pi"}} {
			for _, scope := range []string{"local", "global"} {
				t.Run(name+"/"+strings.Join(pair[:], "+")+"/"+scope, func(t *testing.T) {
					root, home := fixtureCatalog(t), t.TempDir()
					t.Chdir(root)
					t.Setenv("HOME", home)
					t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
					t.Setenv("CLAUDE_CONFIG_DIR", "")
					for _, key := range []string{"OPENCODE_CONFIG_DIR", "OPENCODE_CONFIG", "OPENCODE_CONFIG_CONTENT", "XDG_CONFIG_HOME"} {
						t.Setenv(key, "")
					}
					t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(home, ".pi", "agent"))
					t.Setenv("PI_PACKAGE_DIR", "")
					t.Setenv("PI_OFFLINE", "true")

					dir := filepath.Join(root, "artifacts", "skills", name)
					codexWrite(t, filepath.Join(dir, "patronus.yaml"), []byte("apiVersion: patronus/v2\nfamily: artifact\ntype: skill\nrole: context\nname: "+name+"\ndescription: invented neutral reference\nversion: 1.0.0\nentry: SKILL.md\nfiles: [body.md]\ntargets: [claude, codex, opencode, pi]\ndefaults: {scope: project}\n"))
					codexWrite(t, filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\ndescription: invented neutral reference\n---\nRead body.md.\n"))
					codexWrite(t, filepath.Join(dir, "body.md"), []byte("invented shared guidance\n"))

					for _, tool := range pair {
						out, stderr, err := runInstall(t, name, "--target", tool, "--"+scope, "--deploy", "--yes")
						if err != nil {
							t.Fatalf("install %s after %s: %v\n%s\n%s", tool, pair[0], err, out, stderr)
						}
					}
					stateRoot := root
					if scope == "global" {
						stateRoot = home
					}
					installed, err := state.Load(filepath.Join(stateRoot, ".patronus", "state.json"))
					if err != nil {
						t.Fatal(err)
					}
					for _, tool := range pair {
						rows := installed.Find(name, tool, scope)
						if len(rows) != 1 || rows[0].ItemVersion != "1.0.0" || len(rows[0].Files) != 2 {
							t.Fatalf("incomplete second-host ownership for %s: %+v", tool, rows)
						}
						for _, file := range rows[0].Files {
							raw := mustRead(t, file.Path)
							if file.Action != "CREATE" || file.Checksum != state.Checksum(raw) {
								t.Fatalf("invalid second-host ownership for %s at %s: %+v", tool, file.Path, file)
							}
						}
					}
				})
			}
		}
	}
}
