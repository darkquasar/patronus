package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darkquasar/patronus/internal/adapter"
	"github.com/darkquasar/patronus/internal/diff"
	"github.com/darkquasar/patronus/internal/lock"
	"github.com/darkquasar/patronus/internal/state"
	"github.com/darkquasar/patronus/internal/toolpath"
)

func codexFixture(t *testing.T) (string, string, toolpath.Resolver, *diff.ChangeSet) {
	t.Helper()
	home, project := t.TempDir(), t.TempDir()
	t.Setenv("CODEX_HOME", "")
	if err := os.Mkdir(filepath.Join(project, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	res := toolpath.New(os.LookupEnv, home, project)
	cs := &diff.ChangeSet{Diffs: []diff.FileDiff{{Path: filepath.Join(project, ".agents/skills/fixture-cx/SKILL.md"), Action: diff.Create, Artifact: "fixture-cx", Type: "skill", Tool: "codex", Scope: "local", After: []byte("---\nname: fixture-cx\ndescription: invented Codex test\n---\nfixture\n")}}}
	return home, project, res, cs
}
func codexWrite(t *testing.T, path string, raw []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCodexPreflightRefusesLegacyInstallWithoutWrites(t *testing.T) {
	home, project, res, cs := codexFixture(t)
	legacy := filepath.Join(home, ".codex/skills/fixture-cx/SKILL.md")
	codexWrite(t, legacy, cs.Diffs[0].After)
	if err := codexPreflightPlan(cs, res, home, project); err == nil || !strings.Contains(err.Error(), "migration") {
		t.Fatalf("legacy install not blocked: %v", err)
	}
	if got, err := os.ReadFile(legacy); err != nil || string(got) != string(cs.Diffs[0].After) {
		t.Fatalf("legacy changed: %s %v", got, err)
	}
	if _, err := os.Stat(cs.Diffs[0].Path); !os.IsNotExist(err) {
		t.Fatalf("preflight wrote destination: %v", err)
	}
}

func TestCodexPreflightRefusesUnownedResourceAndDrift(t *testing.T) {
	home, project, res, cs := codexFixture(t)
	d := &cs.Diffs[0]
	d.Before = append([]byte(nil), d.After...)
	d.Action = diff.Skip
	codexWrite(t, d.Path, d.Before)
	if err := codexPreflightPlan(cs, res, home, project); err == nil {
		t.Fatal("unowned equal resource adopted")
	}
	codexOwn(t, project, "codex", *d, "")
	if err := codexPreflightPlan(cs, res, home, project); err != nil {
		t.Fatal(err)
	}
	d.Before = append(d.Before, []byte("edited\n")...)
	codexWrite(t, d.Path, d.Before)
	if err := codexPreflightPlan(cs, res, home, project); err == nil || !strings.Contains(err.Error(), "drift") {
		t.Fatalf("owned drift admitted: %v", err)
	}
}

func codexOwn(t *testing.T, root, tool string, d diff.FileDiff, section string) {
	t.Helper()
	s := &state.State{Version: state.Version, Items: []state.Item{{Artifact: d.Artifact, Type: d.Type, Tool: tool, Scope: d.Scope, Files: []state.FileState{{Path: d.Path, Checksum: codexChecksum(d.Before), Section: section, Action: "CREATE"}}}}}
	if err := state.Save(filepath.Join(root, ".patronus/state.json"), s); err != nil {
		t.Fatal(err)
	}
}

func TestCodexPreflightAdmitsVerifiedOtherScopeVersion(t *testing.T) {
	home, project, res, cs := codexFixture(t)
	other := cs.Diffs[0]
	other.Path = filepath.Join(home, ".agents/skills/fixture-cx/SKILL.md")
	other.Scope = "global"
	other.Before = append(append([]byte(nil), other.After...), []byte("older version\n")...)
	codexWrite(t, other.Path, other.Before)
	codexOwn(t, home, "codex", other, "")
	if err := codexPreflightPlan(cs, res, home, project); err != nil {
		t.Fatalf("verified separately scoped version refused: %v", err)
	}
}

func TestCodexPreflightOtherScopeVersionRequiresOwnership(t *testing.T) {
	for _, tool := range []string{"pi", "codex"} {
		t.Run(tool, func(t *testing.T) {
			home, project, res, cs := codexFixture(t)
			other := cs.Diffs[0]
			other.Path = filepath.Join(home, ".agents/skills/fixture-cx/SKILL.md")
			other.Scope = "global"
			other.Before = append(append([]byte(nil), other.After...), []byte("older version\n")...)
			codexOwn(t, home, tool, other, "")
			codexWrite(t, other.Path, append(other.Before, []byte("user edit\n")...))
			if err := codexPreflightPlan(cs, res, home, project); err == nil {
				t.Fatal("foreign or edited other-scope version admitted")
			}
		})
	}
}

func TestCodexMixedContextPreflightRefusesUnknownOwnership(t *testing.T) {
	for _, mode := range []string{"unknown", "pi", "shadow", "unknown-section", "drift", "empty", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			home, project, res, cs := codexFixture(t)
			d := &cs.Diffs[0]
			d.Path = filepath.Join(project, "AGENTS.md")
			d.Type = "instruction"
			d.Action = diff.Append
			d.Section = &diff.SectionEdit{Name: "fixture-cx", Body: []byte("new")}
			d.Before = []byte("User prose\n\n<!-- patronus:start fixture-cx -->\nold\n<!-- patronus:end fixture-cx -->\n")
			d.After = adapter.AppendSection(d.Before, "fixture-cx", []byte("new"))
			codexWrite(t, d.Path, d.Before)
			if mode == "empty" {
				d.Before = nil
				codexWrite(t, d.Path, nil)
			}
			if mode == "duplicate" {
				d.Before = append(d.Before, d.Before...)
				codexWrite(t, d.Path, d.Before)
			}
			if mode != "unknown" && mode != "empty" {
				tool := "codex"
				if mode == "pi" {
					tool = "pi"
				}
				codexOwn(t, project, tool, *d, "fixture-cx")
			}
			if mode == "shadow" {
				codexWrite(t, filepath.Join(project, "AGENTS.override.md"), []byte("shadow"))
			}
			if mode == "unknown-section" {
				d.Before = append(d.Before, []byte("<!-- patronus:start unknown -->\nother\n<!-- patronus:end unknown -->\n")...)
				codexOwn(t, project, "codex", *d, "fixture-cx")
			}
			if mode == "drift" {
				d.Before = append(d.Before, []byte("edited prose\n")...)
			}
			if err := codexPreflightPlan(cs, res, home, project); err == nil {
				t.Fatal("unsafe shared context admitted", mode)
			}
		})
	}
}

func TestCodexPiCoexistenceRefusesCrossTargetLockOverwrite(t *testing.T) {
	home, project, _, _ := codexFixture(t)
	t.Setenv("HOME", home)
	t.Chdir(project)
	for _, pair := range [][2]string{{"pi", "codex"}, {"codex", "pi"}, {"codex", "claude"}, {"claude", "codex"}} {
		t.Run(pair[0]+"-"+pair[1], func(t *testing.T) {
			prior := &lock.Lock{Version: lock.TargetVersion, Target: pair[0], Profile: "invented", Entries: []lock.Entry{}}
			if err := lock.Save(filepath.Join(project, "patronus.lock"), prior); err != nil {
				t.Fatal(err)
			}
			before := mustRead(t, filepath.Join(project, "patronus.lock"))
			_, _, err := runLock(t, "--profile", "invented", "--target", pair[1])
			if err == nil || !strings.Contains(err.Error(), "lock target") {
				t.Fatalf("cross-target lock not explicitly refused: %v", err)
			}
			if string(mustRead(t, filepath.Join(project, "patronus.lock"))) != string(before) {
				t.Fatal("prior lock changed")
			}
		})
	}
}

func TestCodexPreflightCoreProfileRejectsForeignTargets(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, target := range []string{"all", "claude", "pi"} {
		t.Run(target, func(t *testing.T) {
			for _, operation := range []string{"install", "lock", "update"} {
				var err error
				switch operation {
				case "install":
					_, _, err = runInstall(t, "--profile", "core-profile-cx", "--target", target, "--global")
				case "lock":
					_, _, err = runLock(t, "--profile", "core-profile-cx", "--target", target)
				case "update":
					_, _, err = runUpdate(t, "core-profile-cx", "--target", target, "--global")
				}
				if err == nil || !strings.Contains(err.Error(), "core-profile-cx requires --target codex") {
					t.Fatalf("%s target %s not explicitly refused: %v", operation, target, err)
				}
			}
		})
	}
}

func TestCodexPreflightNormalInstallRefusesLegacyEvenForce(t *testing.T) {
	root := fixtureCatalog(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Chdir(root)
	raw := mustRead(t, filepath.Join(root, "artifacts/skills/fix-skill/SKILL.md"))
	legacy := filepath.Join(home, ".codex/skills/fix-skill/SKILL.md")
	codexWrite(t, legacy, raw)
	_, _, err := runInstall(t, "fix-skill", "--target", "codex", "--global", "--deploy", "--force")
	if err == nil || !strings.Contains(err.Error(), "migration") {
		t.Fatalf("normal install bypassed admission: %v", err)
	}
	if string(mustRead(t, legacy)) != string(raw) {
		t.Fatal("legacy mutated")
	}
}

func TestCodexPiCoexistenceSameTargetLockRegeneration(t *testing.T) {
	root := fixtureCatalog(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(root)
	codexWrite(t, filepath.Join(root, "profiles/core-profile-cx.yaml"), []byte("apiVersion: patronus/v2\nfamily: profile\nname: core-profile-cx\nversion: 1.0.0\nrole: lifecycle\nlayers:\n  guardrails: [fix-skill]\n"))
	for i := 0; i < 2; i++ {
		if _, _, err := runLock(t, "--profile", "core-profile-cx", "--target", "codex"); err != nil {
			t.Fatal(err)
		}
		l, err := lock.Load(filepath.Join(root, "patronus.lock"))
		if err != nil {
			t.Fatal(err)
		}
		if l.Target != "codex" || l.Version != lock.TargetVersion || len(l.Entries) != 1 || l.Entries[0].Name != "fix-skill" {
			t.Fatalf("wrong shared lock: %+v", l)
		}
	}
}

func TestCodexPreflightOwnedSkillLifecycle(t *testing.T) {
	root := fixtureCatalog(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Chdir(root)
	if _, _, err := runInstall(t, "fix-skill", "--target", "codex", "--global", "--deploy"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".agents/skills/fix-skill/SKILL.md")
	manifestPath := filepath.Join(root, "artifacts/skills/fix-skill/patronus.yaml")
	codexWrite(t, manifestPath, []byte(strings.ReplaceAll(string(mustRead(t, manifestPath)), "version: 1.0.0", "version: 2.0.0")))
	src := filepath.Join(root, "artifacts/skills/fix-skill/SKILL.md")
	raw := append(mustRead(t, src), []byte("updated\n")...)
	codexWrite(t, src, raw)
	if _, _, err := runUpdate(t, "fix-skill", "--target", "codex", "--deploy"); err != nil {
		t.Fatal(err)
	}
	if string(mustRead(t, path)) != string(raw) {
		t.Fatal("owned update failed")
	}
	codexWrite(t, path, append(raw, []byte("user edit\n")...))
	if _, _, err := execRemove(t, "fix-skill", "--target", "codex", "--deploy", "--force"); err == nil {
		t.Fatal("force removed edited Codex resource")
	}
	codexWrite(t, path, raw)
	if _, _, err := execRemove(t, "fix-skill", "--target", "codex", "--deploy"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("owned clean remove failed: %v", err)
	}
}

func TestCodexPreflightHookRemovalPreservesUserConfig(t *testing.T) {
	root := fixtureCatalog(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Chdir(root)
	config := filepath.Join(home, ".codex/config.toml")
	if _, _, err := runInstall(t, "fix-hook-2", "--target", "codex", "--global", "--deploy"); err != nil {
		t.Fatal(err)
	}
	installed := mustRead(t, config)
	codexWrite(t, config, append([]byte("model = 'user-model'\n"), installed...))
	if _, _, err := execRemove(t, "fix-hook-2", "--target", "codex", "--deploy"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mustRead(t, config)), "user-model") || strings.Contains(string(mustRead(t, config)), "fix-hook-2") {
		t.Fatalf("config removal not surgical: %s", mustRead(t, config))
	}
	if _, err := os.Stat(filepath.Join(home, ".codex/hooks/fix-hook-2.sh")); !os.IsNotExist(err) {
		t.Fatalf("script not removed: %v", err)
	}
}

func TestCodexPreflightKnownSectionsPreserveProse(t *testing.T) {
	home, project, res, cs := codexFixture(t)
	d := &cs.Diffs[0]
	d.Path = filepath.Join(project, "AGENTS.md")
	d.Type = "instruction"
	d.Action = diff.Append
	d.Section = &diff.SectionEdit{Name: "fixture-cx", Body: []byte("new")}
	d.Before = adapter.AppendSection([]byte("User prose\n"), "fixture-cx", []byte("old"))
	d.Before = adapter.AppendSection(d.Before, "sibling-cx", []byte("sibling"))
	codexWrite(t, d.Path, d.Before)
	owners := &state.State{Version: state.Version}
	for _, name := range []string{"fixture-cx", "sibling-cx"} {
		owners.Items = append(owners.Items, state.Item{Artifact: name, Tool: "codex", Scope: "local", Type: "instruction", Files: []state.FileState{{Path: d.Path, Section: name, Checksum: codexChecksum(d.Before), Action: "APPEND"}}})
	}
	if err := state.Save(filepath.Join(project, ".patronus/state.json"), owners); err != nil {
		t.Fatal(err)
	}
	d.After = adapter.AppendSection(d.Before, "fixture-cx", []byte("new"))
	if err := codexPreflightPlan(cs, res, home, project); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(d.After), "User prose") || !strings.Contains(string(d.After), "sibling-cx") {
		t.Fatal("sibling/prose lost")
	}
}

func TestCodexPreflightOwnsSidecarsAndRetiresAlreadyAbsent(t *testing.T) {
	home, project, res, cs := codexFixture(t)
	d := cs.Diffs[0]
	codexWrite(t, d.Path, d.After)
	d.Before = d.After
	d.Action = diff.Skip
	sidecar := d
	sidecar.Path = filepath.Join(filepath.Dir(d.Path), "helper.md")
	sidecar.Before = []byte("old helper")
	sidecar.After = []byte("new helper")
	sidecar.Action = diff.Conflict
	codexWrite(t, sidecar.Path, sidecar.Before)
	s := &state.State{Version: state.Version, Items: []state.Item{{Artifact: d.Artifact, Tool: "codex", Type: "skill", Scope: "local", Files: []state.FileState{{Path: d.Path, Checksum: codexChecksum(d.Before), Action: "CREATE"}, {Path: sidecar.Path, Checksum: codexChecksum(sidecar.Before), Action: "CREATE"}}}}}
	if err := state.Save(filepath.Join(project, ".patronus/state.json"), s); err != nil {
		t.Fatal(err)
	}
	cs.Diffs = []diff.FileDiff{d, sidecar}
	if err := codexPreflightPlan(cs, res, home, project); err != nil {
		t.Fatal("owned sidecar update rejected", err)
	}
	if err := os.Remove(sidecar.Path); err != nil {
		t.Fatal(err)
	}
	cs.Diffs = []diff.FileDiff{{Path: sidecar.Path, Artifact: d.Artifact, Tool: "codex", Scope: "local", Action: diff.Skip}}
	if err := codexPreflightPlan(cs, res, home, project); err != nil {
		t.Fatal("missing owned effect cannot retire", err)
	}
}

func TestCodexPreflightEmptyUnownedFileCannotBypassOwnership(t *testing.T) {
	home, project, res, cs := codexFixture(t)
	d := &cs.Diffs[0]
	d.Type = "hook"
	d.Path = filepath.Join(project, ".codex/hooks/fixture-cx.sh")
	d.After = []byte("#!/bin/sh\n")
	codexWrite(t, d.Path, nil)
	if err := codexPreflightPlan(cs, res, home, project); err == nil {
		t.Fatal("empty unowned source admitted")
	}
}

func TestCodexPreflightMultiSectionRemovalRefusesDriftEvenForce(t *testing.T) {
	root := fixtureCatalog(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Chdir(root)
	if _, _, err := runInstall(t, "fix-instruction-global", "fix-instruction-2", "--target", "codex", "--global", "--deploy"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".codex/AGENTS.md")
	sp := filepath.Join(home, ".patronus/state.json")
	codexWrite(t, path, append(mustRead(t, path), []byte("Invented external edit\n")...))
	before, ownership := mustRead(t, path), mustRead(t, sp)
	for _, force := range []bool{false, true} {
		args := []string{"fix-instruction-global", "--target", "codex", "--deploy"}
		if force {
			args = append(args, "--force")
		}
		_, _, err := execRemove(t, args...)
		if err == nil {
			t.Fatalf("multi-section removal admitted checksum drift (force=%v)", force)
		}
		if string(mustRead(t, path)) != string(before) || string(mustRead(t, sp)) != string(ownership) {
			t.Fatal("drift refusal mutated sections/ownership")
		}
	}
}

func TestCodexMCPPreservesUserAuthByRefusingUnownedServer(t *testing.T) {
	root := fixtureCatalog(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Chdir(root)
	config := filepath.Join(home, ".codex/config.toml")
	prior := []byte("model = 'user-model'\n[mcp_servers.fix-mcp-two]\nurl = 'https://user.invalid/mcp'\nbearer_token_env_var = 'INVENTED_TOKEN_ENV'\n[mcp_servers.fix-mcp-two.http_headers]\nX-Invented = 'fixture-reference'\n")
	codexWrite(t, config, prior)
	if _, _, err := runInstall(t, "fix-mcp-two", "--target", "codex", "--global", "--deploy", "--force"); err == nil {
		t.Fatal("unowned authenticated MCP server overwritten")
	}
	if string(mustRead(t, config)) != string(prior) {
		t.Fatal("user auth/sibling changed")
	}
}

func TestCodexPreflightLegacyOwnershipCannotBeRetiredImplicitly(t *testing.T) {
	home, project, res, cs := codexFixture(t)
	old := cs.Diffs[0]
	old.Path = filepath.Join(home, ".codex/skills/fixture-cx/SKILL.md")
	old.Scope = "global"
	old.Before = old.After
	codexOwn(t, home, "codex", old, "")
	if err := codexPreflightPlan(cs, res, home, project); err == nil || !strings.Contains(err.Error(), "migration") {
		t.Fatalf("legacy missing-file ownership silently relocated: %v", err)
	}
}

func TestCodexPreflightInverseSettingRefusesForeignOverlap(t *testing.T) {
	home, project, res, cs := codexFixture(t)
	d := &cs.Diffs[0]
	d.Path = filepath.Join(project, ".codex/config.toml")
	d.Type = ""
	d.Action = diff.Restore
	d.Before = []byte("[mcp_servers.fixture]\nurl = 'https://fixture.invalid/mcp'\n")
	d.After = nil
	codexWrite(t, d.Path, d.Before)
	own := &diff.SettingEdit{Target: diff.FileTargetRef{File: ".codex/config.toml", Format: "toml"}, Dotted: "mcp_servers.fixture", ScalarValue: map[string]any{"url": "https://fixture.invalid/mcp"}}
	foreign := &diff.SettingEdit{Target: own.Target, Dotted: "mcp_servers.fixture.url", ScalarValue: "https://fixture.invalid/mcp"}
	s := &state.State{Version: state.Version, Items: []state.Item{{Artifact: d.Artifact, Tool: "codex", Scope: "local", Files: []state.FileState{{Path: d.Path, Action: "MERGE", Setting: own}}}, {Artifact: "foreign", Tool: "pi", Scope: "local", Files: []state.FileState{{Path: d.Path, Action: "MERGE", Setting: foreign}}}}}
	if err := state.Save(filepath.Join(project, ".patronus/state.json"), s); err != nil {
		t.Fatal(err)
	}
	if err := codexPreflightPlan(cs, res, home, project); err == nil {
		t.Fatal("forced inverse can destroy foreign configuration")
	}
}
